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

type Writer struct {
	w  io.Writer
	pn *int64
	n  int64
}

func NewWriter(w io.Writer, counter ...*int64) *Writer {
	return &Writer{
		w: w,
	}
}

func NewWriterWithAtomicCounter(w io.Writer, counter *int64) *Writer {
	return &Writer{
		w:  w,
		pn: counter,
	}
}

func (cw *Writer) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.n += int64(n)
	if cw.pn != nil {
		atomic.AddInt64(cw.pn, int64(n))
	}
	return
}

func (cw *Writer) BytesWritten() int64 {
	return cw.n
}
