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

package zerodriver

import (
	"fmt"
	"io"
)

type object struct {
	url  string
	size int64
	pos  int64
}

func (o *object) URL() string {
	return o.url
}

func (o *object) Close() error {
	return nil
}

func (o *object) Size() int64 {
	return o.size
}

func (o *object) Read(p []byte) (n int, err error) {
	n, err = o.ReadAt(p, o.pos)
	o.pos += int64(n)
	return
}

func (o *object) ReadAt(p []byte, off int64) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if off >= o.size {
		return 0, io.EOF
	}

	max := o.size - off
	if int64(len(p)) < max {
		max = int64(len(p))
	}

	for n = 0; int64(n) < max; n++ {
		p[n] = 0
	}
	return
}

func (o *object) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		o.pos = o.pos + offset
	case io.SeekStart:
		o.pos = offset
	case io.SeekEnd:
		if o.size < 0 {
			return 0, fmt.Errorf("unknown length")
		}
		o.pos = o.size + offset
	}

	if o.pos < 0 {
		o.pos = 0
	} else if o.size >= 0 && o.pos > o.size {
		o.pos = o.size
	}

	return o.pos, nil
}
