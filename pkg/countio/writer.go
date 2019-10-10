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
