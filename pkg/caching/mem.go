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

package caching

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

func NewMemorySlicer(bsize int64, bcount int64) (Slicer, error) {
	pool, err := newMemPool(bsize, bcount)
	if err != nil {
		return nil, err
	}
	return &memSlicer{bsize, pool}, nil
}

type memSlicer struct {
	bsize int64
	pool  *memPool
}

func (ms *memSlicer) Bsize() int64 {
	return ms.bsize
}

func (ms *memSlicer) Slice(obj storage.Object, offset int64) Slice {
	size := obj.Size() - offset
	if ms.bsize < size {
		size = ms.bsize
	}
	key := fmt.Sprintf("%s[%d,%d)", obj.URL(), offset, offset+size)

	return &memSlice{
		bsize: ms.bsize,
		pool:  ms.pool,
		size:  size,
		key:   key,
		ff: func(buf []byte) (int, error) {
			if int64(len(buf)) < size {
				panic(fmt.Sprintf("never expected=%d, got=%d", size, len(buf)))
			}

			src := io.NewSectionReader(obj, offset, size)
			n, err := io.ReadFull(src, buf[:size])
			if err != nil {
				return 0, err
			}
			return n, nil
		},
	}
}

type memSlice struct {
	bsize int64
	pool  *memPool
	size  int64
	key   string
	ff    memBufFillFunc
	pos   int64
}

func (ms *memSlice) Close() error {
	return nil
}

func (ms *memSlice) Size() int64 {
	return ms.size
}

func (ms *memSlice) Read(p []byte) (n int, err error) {
	n, err = ms.ReadAt(p, ms.pos)
	ms.pos += int64(n)
	return
}

func (ms *memSlice) ReadAt(p []byte, off int64) (n int, err error) {
	buffer := ms.pool.Get(ms.key)
	n, err = buffer.FillAndCopyAt(ms.key, ms.ff, p, off)
	return
}

func (ms *memSlice) ReadAhead() {
	buffer := ms.pool.Get(ms.key)
	if err := buffer.Fill(ms.key, ms.ff); err != nil {
		cacheLogger().Error("read-ahead", zap.Error(err))
	}
}

func (ms *memSlice) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		ms.pos = ms.pos + offset
	case io.SeekStart:
		ms.pos = offset
	case io.SeekEnd:
		if ms.size < 0 {
			return 0, errors.New("unknown length")
		}
		ms.pos = ms.size + offset
	}

	if ms.pos < 0 {
		ms.pos = 0
	} else if ms.size >= 0 && ms.pos > ms.size {
		ms.pos = ms.size
	}

	return ms.pos, nil
}

func newMemPool(bsize, bcount int64) (*memPool, error) {
	lru, err := simplelru.NewLRU(int(bcount), nil)
	if err != nil {
		return nil, err
	}
	return &memPool{
		bsize:  bsize,
		bcount: bcount,
		lru:    lru,
	}, nil
}

type memPool struct {
	bsize  int64
	bcount int64

	mu   sync.Mutex
	next int64
	lru  *simplelru.LRU
}

func (mp *memPool) Get(key string) (buffer *memBuf) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	ent, hit := mp.lru.Get(key)
	if hit {
		buffer = ent.(*memBuf)
	} else if mp.next < mp.bcount {
		mp.next++
		buffer = newMemBuf(mp.bsize)
		mp.lru.Add(key, buffer)
	} else {
		_, ent, _ = mp.lru.RemoveOldest()
		buffer = ent.(*memBuf)
		mp.lru.Add(key, buffer)
	}

	return
}

type memBufFillFunc func(buf []byte) (int, error)

type memBuf struct {
	mu  sync.Mutex
	buf []byte
	key string
	n   int
}

func newMemBuf(bsize int64) *memBuf {
	return &memBuf{
		buf: make([]byte, bsize),
	}
}

func (b *memBuf) Fill(key string, ff memBufFillFunc) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.key != key {
		b.key = key
		b.n, err = ff(b.buf)
	}
	return
}

func (b *memBuf) FillAndCopyAt(key string, ff memBufFillFunc, p []byte, off int64) (n int, err error) {
	if len(p) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.key != key {
		b.key = key
		b.n, err = ff(b.buf)
	}

	if err != nil {
		// Reset the buffer on error. The next caller will attempt to
		// fill the buffer.
		b.key = ""
		b.n = 0
	} else {
		// Do the memcpy
		if off < int64(b.n) {
			n = copy(p, b.buf[off:b.n])
		} else {
			err = io.EOF
		}
	}

	return
}
