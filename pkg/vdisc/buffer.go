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
	"io"
	"sync"
)

type FillFunc func(buf []byte) (int, error)

type Buffer struct {
	mu  *sync.Mutex
	buf []byte
	key string
	n   int
}

func NewBuffer(bsize int) *Buffer {
	return &Buffer{
		mu:  &sync.Mutex{},
		buf: make([]byte, bsize),
	}
}

func (b *Buffer) ReadAtCached(p []byte, off int64, key string, ff FillFunc) (n int, err error) {
	if len(p) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.key != key {
		b.key = key
		b.n, err = ff(b.buf)
	}

	if err == nil {
		if off < int64(b.n) {
			n = copy(p, b.buf[off:b.n])
		} else {
			err = io.EOF
		}
	} else {
		// Reset the buffer on error. The next caller will attempt to
		// fill the buffer.
		b.key = ""
		b.n = 0
	}

	return
}
