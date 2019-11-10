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

package filedriver

import (
	"os"

	"golang.org/x/sys/unix"
)

type object struct {
	url  string
	f    *os.File
	size int64
}

func (o *object) Close() error {
	return o.f.Close()
}

func (o *object) Read(p []byte) (n int, err error) {
	n, err = o.f.Read(p)
	return
}

func (o *object) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = o.f.ReadAt(p, off)
	return
}

func (o *object) Seek(offset int64, whence int) (n int64, err error) {
	n, err = o.f.Seek(offset, whence)
	return
}

func (o *object) Size() int64 {
	return o.size
}

func (o *object) GetXattr(name string) ([]byte, error) {
	var value [4096]byte
	n, err := unix.Fgetxattr(int(o.f.Fd()), name, value[:])
	if err != nil {
		return nil, err
	}

	return value[:n], nil
}

func (o *object) URL() string {
	return o.url
}
