package countio

import (
	"io"
	"sync/atomic"
)

type Reader struct {
	r  io.Reader
	pn *int64
	n  int64
}

func NewReader(r io.Reader, counter ...*int64) *Reader {
	return &Reader{
		r: r,
	}
}

func NewReaderWithAtomicCounter(r io.Reader, counter *int64) *Reader {
	return &Reader{
		r:  r,
		pn: counter,
	}
}

func (cr *Reader) Read(p []byte) (n int, err error) {
	n, err = cr.r.Read(p)
	cr.n += int64(n)
	if cr.pn != nil {
		atomic.AddInt64(cr.pn, int64(n))
	}
	return
}

func (cr *Reader) BytesRead() int64 {
	return cr.n
}
