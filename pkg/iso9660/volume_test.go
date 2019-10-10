package iso9660_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/storage"
	_ "github.com/NVIDIA/vdisc/pkg/storage/zero"
)

type walkEntry struct {
	path string
	name string
	size int64
	mode os.FileMode
}

func TestVolume(t *testing.T) {
	var expectedWalk []walkEntry

	add := func(path string, name string, size int64, mode os.FileMode) {
		expectedWalk = append(expectedWalk, walkEntry{
			path: path,
			name: name,
			size: size,
			mode: mode,
		})
	}

	initDirs := func(sizeDir, segmentDir, symlinkDir string) {
		expectedWalk = nil
		expectedWalk = make([]walkEntry, 0)
		add("/", sizeDir, 2048, os.ModeDir|0555)
		add("/", segmentDir, 2048, os.ModeDir|0555)
		add("/", symlinkDir, 2048, os.ModeDir|0555)
	}

	var tests = []struct {
		sizeDir    string
		segmentDir string
		symlinkDir string
		//mode       iso9660.DirectoryInodeNamingMode
		volume     *iso9660.Volume
		badSizeDir bool
	}{
		{
			"a", "b", "c",
			iso9660.NewPosixPortableVolume(),
			false,
		},
		{
			"a", "b", "c",
			iso9660.NewNvidiaExtendedVolume(),
			false,
		},
		{
			"key=value", "b", "c",
			iso9660.NewPosixPortableVolume(),
			true,
		},
		{
			"key=value", "b#", "c~",
			iso9660.NewNvidiaExtendedVolume(),
			false,
		},
		{
			"no one likes spaces", "b", "c",
			iso9660.NewNvidiaExtendedVolume(),
			true,
		},
	}

	for k, test := range tests {
		msg := fmt.Sprintf("Case: %d", k)

		volume := test.volume
		assert.NotNil(t, volume)
		initDirs(test.sizeDir, test.segmentDir, test.symlinkDir)
		// Create files of various sizes
		for i, sz := range []int64{1, 2048, 2049, iso9660.MaxPartSize, iso9660.MaxPartSize + 1} {
			r, err := storage.Open(fmt.Sprintf("zero:%d", sz))
			if err != nil {
				t.Fatal(err)
			}

			name := fmt.Sprintf("%d.data", i)
			pth := fmt.Sprintf("%s/%s", test.sizeDir, name)
			err = volume.AddFile(pth, r)
			if !test.badSizeDir {
				assert.Nil(t, err, msg)
			} else {
				assert.NotNil(t, err, msg)
				break
			}
			add(fmt.Sprintf("/%s", test.sizeDir), name, sz, 0444)
		}
		if test.badSizeDir {
			break
		}
		// Create files of various path segment lengths
		lens := []int64{1, 10, 249, 250, 251}
		r, err := storage.Open("zero:1")
		if err != nil {
			t.Fatal(err)
		}
		for i, l := range lens {
			name := fmt.Sprintf(fmt.Sprintf("%%0%dd.txt", l), i)
			pth := fmt.Sprintf("%s/%s", test.segmentDir, name)

			err = volume.AddFile(pth, r)
			assert.Nil(t, err)
			add(fmt.Sprintf("/%s", test.segmentDir), name, 1, 0444)
		}

		// Create symlinks of various sizes
		for i, l := range lens {
			name := fmt.Sprintf("%d.txt", i)
			pth := fmt.Sprintf("%s/%s", test.symlinkDir, name)
			target := fmt.Sprintf(fmt.Sprintf("../%s/%%0%dd.txt", test.segmentDir, l), i)
			err = volume.AddSymlink(pth, target)
			assert.Nil(t, err)
			add(fmt.Sprintf("/%s", test.symlinkDir), name, 0, os.ModeSymlink|0777)
		}

		isow := bytes.NewBuffer(nil)
		n, err := volume.WriteMetadataTo(isow)
		if err != nil {
			t.Fatal(err)
		}

		iso := bytes.NewReader(isow.Bytes())

		assert.Equal(t, int64(0), n%iso9660.LogicalBlockSize, "partial sector")
		assert.Equal(t, int64(29), n/iso9660.LogicalBlockSize, "sectors written")

		i := 0
		walker := iso9660.NewWalker(iso)
		err = walker.Walk("/", func(path string, info os.FileInfo, err error) error {
			if info.Name() == "." || info.Name() == ".." {
				return nil
			}
			assert.Nil(t, err)
			assert.Equal(t, expectedWalk[i].path, path, "path")
			assert.Equal(t, expectedWalk[i].name, info.Name(), "name")
			assert.Equal(t, expectedWalk[i].size, info.Size(), "size "+filepath.Join(path, info.Name()))
			assert.Equal(t, expectedWalk[i].mode, info.Mode(), "mode "+filepath.Join(path, info.Name()))
			i++
			return nil
		})
		assert.Nil(t, err)
	}
}
