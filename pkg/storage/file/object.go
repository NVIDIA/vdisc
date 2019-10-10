// Copyright © 2019 NVIDIA Corporation
package filedriver

import (
	"os"
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

func (o *object) URL() string {
	return o.url
}
