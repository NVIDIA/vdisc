// Copyright © 2019 NVIDIA Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ram_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	stdurl "net/url"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
	"github.com/NVIDIA/vdisc/pkg/storage/ram"
)

func TestRamDriver(t *testing.T) {
	bg := context.Background()

	m := ramdriver.NewDriver()

	wf1, err := m.Create(bg, "ram:foo")
	assert.NotNil(t, wf1)
	assert.Nil(t, err)
	io.WriteString(wf1, "test")

	wf2, err := m.Create(bg, "ram:foo")
	assert.NotNil(t, wf2)
	assert.Nil(t, err)
	io.WriteString(wf2, "bad")

	// shouldn't be visible yet
	_, err = m.Open(bg, "ram:foo", -1)
	assert.True(t, os.IsNotExist(err))
	_, err = wf2.Commit()
	assert.Nil(t, err)
	// last one to close wins
	_, err = wf1.Commit()
	assert.Nil(t, err)

	rf, err := m.Open(bg, "ram:foo", -1)
	assert.NotNil(t, rf)
	assert.Nil(t, err)

	buf, err := ioutil.ReadAll(rf)
	assert.Nil(t, err)
	assert.EqualValues(t, "test", string(buf))

	rf2, err := m.Open(bg, "ram:foo", -1)
	assert.NotNil(t, rf2)
	assert.Nil(t, err)

	buf2, err := ioutil.ReadAll(rf2)
	assert.Nil(t, err)
	assert.EqualValues(t, "test", string(buf2))

	s, err := m.Stat(bg, "ram:foo")
	assert.NotNil(t, s)
	assert.Nil(t, err)
	assert.EqualValues(t, 4, s.Size())

	assert.NotNil(t, m.Remove(bg, "ram:bar"))
	assert.Nil(t, m.Remove(bg, "ram:foo"))
}

func TestRamDriverDirs(t *testing.T) {
	bg := context.Background()

	m := ramdriver.NewDriver()

	assert.Nil(t, writeFile(m, "ram:///foo/bar/baz.txt", bytes.NewReader([]byte("test"))))
	assert.Nil(t, writeFile(m, "ram:///foo/bar.txt", bytes.NewReader([]byte("test2"))))
	assert.Nil(t, writeFile(m, "ram:///foo/bar/baz/more.txt", bytes.NewReader([]byte("test22"))))
	files, err := readDirRecursiveMap(m, "ram:///")
	assert.Nil(t, err)
	assert.EqualValues(t, 3, len(files))
	assert.EqualValues(t, 4, files["ram:///foo/bar/baz.txt"].Size())
	assert.EqualValues(t, 5, files["ram:///foo/bar.txt"].Size())
	assert.EqualValues(t, 6, files["ram:///foo/bar/baz/more.txt"].Size())
	// Remove
	assert.Nil(t, m.Remove(bg, "ram:foo/bar.txt"))
	files, err = readDirRecursiveMap(m, "ram:///")
	assert.EqualValues(t, 2, len(files))
	assert.Nil(t, files["ram:///foo/bar.txt"])
	assert.Nil(t, err)
	// Remove
	assert.Nil(t, m.Remove(bg, "ram:///foo/bar/baz.txt"))
	files, err = readDirRecursiveMap(m, "ram:///")
	assert.EqualValues(t, 1, len(files))
	assert.Nil(t, files["foo/bar/baz.txt"])
	assert.Nil(t, err)
	// Remove
	assert.Nil(t, m.Remove(bg, "ram:///foo/bar/baz/more.txt"))
	files, err = readDirRecursiveMap(m, "ram:///")
	assert.EqualValues(t, 0, len(files))
}

func writeFile(drvr driver.Creator, name string, r io.Reader) error {
	w, err := drvr.Create(context.Background(), name)
	if err != nil {
		return err
	}
	defer w.Abort()
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	_, err = w.Commit()
	if err != nil {
		return err
	}
	return nil
}

type mapVisitor struct {
	m    sync.Mutex
	wg   sync.WaitGroup
	urls map[string]os.FileInfo
}

func (d *mapVisitor) VisitDir(baseURL string, files []os.FileInfo) error {
	for _, f := range files {
		if !f.IsDir() {
			d.m.Lock()
			d.urls[urlJoin(baseURL, f.Name())] = f
			d.m.Unlock()
		}
	}
	return nil
}

func readDirRecursiveMap(drvr driver.Readdirer, root string) (map[string]os.FileInfo, error) {
	v := &mapVisitor{
		urls: make(map[string]os.FileInfo),
	}
	err := driver.Visit(context.Background(), drvr, root, v)
	return v.urls, err
}

func urlJoin(baseURL, name string) string {
	base, err := stdurl.Parse(baseURL)
	if err != nil {
		panic(err)
	}
	if len(base.Opaque) == 0 {
		base.Path = path.Clean(path.Join(base.Path, name))
	} else {
		base.Opaque = path.Clean(path.Join(base.Opaque, name))
	}

	return base.String()
}
