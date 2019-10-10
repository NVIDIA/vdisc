// Copyright © 2019 NVIDIA Corporation
package datadriver

import (
	"io"
	"os"
)

type object struct {
	url    string
	sr     *io.SectionReader
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

	n, err = o.sr.Read(p)
	return
}

func (o *object) ReadAt(p []byte, off int64) (n int, err error) {
	if o.closed {
		err = os.ErrClosed
		return
	}
	n, err = o.sr.ReadAt(p, off)
	return
}

func (o *object) Seek(offset int64, whence int) (n int64, err error) {
	if o.closed {
		err = os.ErrClosed
		return
	}
	n, err = o.sr.Seek(offset, whence)
	return
}

func (o *object) Size() int64 {
	return o.sr.Size()
}

func (o *object) URL() string {
	return o.url
}
