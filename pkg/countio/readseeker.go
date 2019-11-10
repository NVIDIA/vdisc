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

package countio

import (
	"io"
	"sync/atomic"
)

type ReaderAtSeeker interface {
	io.ReaderAt
	io.ReadSeeker
}

type CountingReaderAtSeeker struct {
	r  ReaderAtSeeker
	n  int64
	pn *int64
}

func NewReaderAtSeeker(r ReaderAtSeeker) *CountingReaderAtSeeker {
	return &CountingReaderAtSeeker{
		r: r,
	}
}

func NewReaderAtSeekerWithAtomicCounter(r ReaderAtSeeker, counter *int64) *CountingReaderAtSeeker {
	return &CountingReaderAtSeeker{
		r:  r,
		pn: counter,
	}
}

func (r *CountingReaderAtSeeker) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	r.n += int64(n)
	if r.pn != nil {
		atomic.AddInt64(r.pn, int64(n))
	}
	return
}

func (r *CountingReaderAtSeeker) Seek(offset int64, whence int) (int64, error) {
	return r.r.Seek(offset, whence)
}

func (r *CountingReaderAtSeeker) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = r.r.ReadAt(p, off)
	r.n += int64(n)
	if r.pn != nil {
		atomic.AddInt64(r.pn, int64(n))
	}
	return
}

func (r *CountingReaderAtSeeker) BytesRead() int64 {
	return r.n
}
