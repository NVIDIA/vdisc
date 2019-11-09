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
package ramdriver

import (
	"bytes"
	"os"
)

type object struct {
	url    string
	br     *bytes.Reader
	closed bool
}

func (o *object) Close() error {
	o.closed = true
	return nil
}

func (o *object) Read(p []byte) (n int, err error) {
	if o.closed {
		err = os.ErrClosed
		return
	}

	n, err = o.br.Read(p)
	return
}

func (o *object) ReadAt(p []byte, off int64) (n int, err error) {
	if o.closed {
		err = os.ErrClosed
		return
	}
	n, err = o.br.ReadAt(p, off)
	return
}

func (o *object) Seek(offset int64, whence int) (n int64, err error) {
	if o.closed {
		err = os.ErrClosed
		return
	}
	n, err = o.br.Seek(offset, whence)
	return
}

func (o *object) Size() int64 {
	return o.br.Size()
}

func (o *object) URL() string {
	return o.url
}
