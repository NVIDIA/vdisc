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
package vdisc

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	stdurl "net/url"
	"os"
	"runtime"
	"syscall"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/NVIDIA/vdisc/pkg/caching"
	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/storage"
	"github.com/NVIDIA/vdisc/pkg/vdisc/types"
	"github.com/NVIDIA/vdisc/pkg/vdisc/types/v1"
)

type VDisc interface {
	io.Closer
	FsType() string
	BlockSize() uint16
	Image() storage.AnonymousObject
	OpenExtent(lba iso9660.LogicalBlockAddress) (storage.Object, error)
	ExtentURL(lba iso9660.LogicalBlockAddress) (string, error)
}

func Load(url string, cache caching.Cache) (VDisc, error) {
	baseURL, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	raw, mmapHandle, err := downloadMemoryMapped(url)
	if err != nil {
		return nil, err
	}

	msg, err := capnp.Unmarshal(raw)
	if err != nil {
		mmapHandle.Close()
		return nil, err
	}
	msg.TraverseLimit = math.MaxUint64

	root, err := vdisc_types.ReadRootVDisc(msg)
	if err != nil {
		mmapHandle.Close()
		return nil, err
	}

	v1, err := root.V1()
	if err != nil {
		mmapHandle.Close()
		return nil, err
	}

	fstype, err := v1.FsType()
	if err != nil {
		mmapHandle.Close()
		return nil, err
	}

	blockSize := v1.BlockSize()

	var parts []storage.AnonymousObject

	uris, err := v1.Uris()
	if err != nil {
		mmapHandle.Close()
		return nil, err
	}

	extents, err := v1.Extents()
	if err != nil {
		mmapHandle.Close()
		return nil, err
	}

	extentIndices := make(map[iso9660.LogicalBlockAddress]int)
	pos := iso9660.LogicalBlockAddress(0)
	for i := 0; i < extents.Len(); i++ {
		extentIndices[pos] = i

		ext := extents.At(i)
		blocks := ext.Blocks()
		padding := ext.Padding()
		if blocks == 0 {
			continue
		}

		var obj storage.Object
		obj = &extent{
			blockSize: blockSize,
			baseURL:   baseURL,
			uris:      uris,
			extents:   extents,
			idx:       i,
		}

		obj = cache.WithCaching(obj)

		parts = append(parts, obj)
		if padding > 0 {
			padObj, err := storage.Open(fmt.Sprintf("zero:%d", padding))
			if err != nil {
				mmapHandle.Close()
				return nil, err
			}
			parts = append(parts, padObj)
		}

		pos += iso9660.LogicalBlockAddress(blocks)
	}

	return &vdisc{
		cache:         cache,
		baseURL:       baseURL,
		fsType:        fstype,
		blockSize:     blockSize,
		image:         storage.Concat(parts...),
		uris:          uris,
		extents:       extents,
		extentIndices: extentIndices,
		mmapHandle:    mmapHandle,
	}, nil
}

type mmapCloser struct {
	data []byte
}

// Close all the underlying closers, in sequence.
// Return the first error.
func (mc *mmapCloser) Close() error {
	if mc.data == nil {
		return nil
	}

	data := mc.data
	mc.data = nil

	runtime.SetFinalizer(mc, nil)
	return syscall.Munmap(data)
}

func downloadMemoryMapped(url string) ([]byte, io.Closer, error) {
	res, err := storage.Open(url)
	if err != nil {
		return nil, nil, err
	}

	r := io.NewSectionReader(res, 0, res.Size())

	brSize := res.Size()
	if brSize > 67108864 {
		brSize = 67108864
	}
	br := bufio.NewReaderSize(r, int(brSize))
	hdr, err := br.Peek(4)
	if err != nil {
		return nil, nil, err
	}

	var src io.Reader
	if bytes.Equal(hdr, []byte{0x1f, 0x8b, 0x08, 0x00}) {
		src, _ = gzip.NewReader(br)
	} else {
		src = br
	}

	dst, err := ioutil.TempFile("", "vdisc.")
	if err != nil {
		return nil, nil, err
	}
	defer dst.Close()
	os.Remove(dst.Name())

	n, err := io.Copy(dst, src)
	if err != nil {
		return nil, nil, err
	}

	data, err := syscall.Mmap(int(dst.Fd()), 0, int(n), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}

	return data, &mmapCloser{data}, nil
}

type vdisc struct {
	cache         caching.Cache
	baseURL       *stdurl.URL
	fsType        string
	blockSize     uint16
	image         storage.AnonymousObject
	uris          vdisc_types_v1.ITrie_List
	extents       vdisc_types_v1.Extent_List
	extentIndices map[iso9660.LogicalBlockAddress]int
	mmapHandle    io.Closer
}

func (v *vdisc) Close() error {
	if err := v.image.Close(); err != nil {
		return err
	}

	return v.mmapHandle.Close()
}

func (v *vdisc) FsType() string {
	return v.fsType
}
func (v *vdisc) BlockSize() uint16 {
	return v.blockSize
}

func (v *vdisc) Image() storage.AnonymousObject {
	return v.image
}

func (v *vdisc) OpenExtent(lba iso9660.LogicalBlockAddress) (storage.Object, error) {
	idx, ok := v.extentIndices[lba]
	if !ok {
		return nil, fmt.Errorf("unable to open file: invalid extent - %d", lba)
	}

	var obj storage.Object
	obj = &extent{
		blockSize: v.blockSize,
		baseURL:   v.baseURL,
		uris:      v.uris,
		extents:   v.extents,
		idx:       idx,
	}

	return v.cache.WithCaching(obj), nil
}

func (v *vdisc) ExtentURL(lba iso9660.LogicalBlockAddress) (string, error) {
	idx, ok := v.extentIndices[lba]
	if !ok {
		return "", fmt.Errorf("unable to open file: invalid extent - %d", lba)
	}

	ext := &extent{
		blockSize: v.blockSize,
		baseURL:   v.baseURL,
		uris:      v.uris,
		extents:   v.extents,
		idx:       idx,
	}
	return ext.URL(), nil
}
