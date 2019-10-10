// Copyright © 2018 NVIDIA Corporation

package susp

import (
	"io"
)

const (
	ContinuationAreaEntryLength = 28
)

// The purpose of the "CE" System Use Entry is to extend the System
// Use field to store additionalSystemUseEntries.
type ContinuationAreaEntry struct {
	start uint32
	off   uint32
	len   uint32
}

func NewContinuationAreaEntry(start uint32, off uint32, len uint32) *ContinuationAreaEntry {
	return &ContinuationAreaEntry{start, off, len}
}

func (ce *ContinuationAreaEntry) ContinuationStart() uint32 {
	return ce.start
}

func (ce *ContinuationAreaEntry) ContinuationOffset() uint32 {
	return ce.off
}

func (ce *ContinuationAreaEntry) ContinuationLength() uint32 {
	return ce.len
}

func (ce *ContinuationAreaEntry) Len() int {
	return ContinuationAreaEntryLength
}

func (ce *ContinuationAreaEntry) WriteTo(w io.Writer) (n int64, err error) {
	var nn int
	nn, err = io.WriteString(w, "CE")
	if err != nil {
		return
	}
	n += int64(nn)
	if err = writeByte(w, byte(ce.Len())); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, 1); err != nil {
		return
	}
	n += 1
	if err = putBothUint32(w, ce.start); err != nil {
		return
	}
	n += 8
	if err = putBothUint32(w, ce.off); err != nil {
		return
	}
	n += 8
	if err = putBothUint32(w, ce.len); err != nil {
		return
	}
	n += 8
	return
}
