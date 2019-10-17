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
	"context"
	"errors"
	"io"
	stdurl "net/url"
	"os"
	"strings"

	"github.com/NVIDIA/vdisc/pkg/safecast"
	"github.com/NVIDIA/vdisc/pkg/storage"
	"github.com/NVIDIA/vdisc/pkg/vdisc/types/v1"
)

type extent struct {
	blockSize uint16
	baseURL   *stdurl.URL
	uris      vdisc_types_v1.ITrie_List
	extents   vdisc_types_v1.Extent_List
	idx       int
	pos       int64
	closed    bool
}

func (e *extent) Close() error {
	e.closed = true
	return nil
}

func (e *extent) URL() string {
	extent := e.extents.At(e.idx)
	uri, err := extent.UriSuffix()
	if err != nil {
		panic(err)
	}

	parent := extent.UriPrefix()
	for {
		node := e.uris.At(safecast.Uint32ToInt(parent))
		prefix, err := node.Content()
		if err != nil {
			panic(err)
		}

		uri = prefix + uri
		if node.Parent() == parent {
			break
		}
		parent = node.Parent()
	}

	// Possibly evaluate relative to baseURL
	u, err := stdurl.Parse(uri)
	if err != nil {
		panic(err)
	}

	resolved := e.baseURL.ResolveReference(u)
	if !strings.HasPrefix(e.baseURL.Path, "/") {
		resolved.Path = strings.TrimPrefix(resolved.Path, "/")
	}

	return resolved.String()
}

func (e *extent) Size() int64 {
	ext := e.extents.At(e.idx)
	blocks := ext.Blocks()
	padding := ext.Padding()
	if blocks == 0 {
		return 0
	}

	return int64(blocks)*int64(e.blockSize) - int64(padding)
}

func (e *extent) Read(p []byte) (n int, err error) {
	n, err = e.ReadAt(p, e.pos)
	e.pos += int64(n)
	return
}

func (e *extent) ReadAt(p []byte, off int64) (n int, err error) {
	if e.closed {
		err = os.ErrClosed
		return
	}

	var obj storage.Object
	obj, err = storage.OpenContextSize(context.Background(), e.URL(), e.Size())
	if err != nil {
		return
	}
	defer obj.Close()
	return obj.ReadAt(p, off)
}

func (e *extent) Seek(offset int64, whence int) (int64, error) {
	if e.closed {
		return 0, os.ErrClosed
	}

	size := e.Size()

	switch whence {
	case io.SeekCurrent:
		e.pos = e.pos + offset
	case io.SeekStart:
		e.pos = offset
	case io.SeekEnd:
		if size < 0 {
			return 0, errors.New("unknown length")
		}
		e.pos = size + offset
	}

	if e.pos < 0 {
		e.pos = 0
	} else if size >= 0 && e.pos > size {
		e.pos = size
	}

	return e.pos, nil
}
