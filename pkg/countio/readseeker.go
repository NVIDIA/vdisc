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
