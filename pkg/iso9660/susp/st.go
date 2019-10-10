// Copyright © 2018 NVIDIA Corporation

package susp

import (
	"io"
)

const (
	TerminatorEntryLength = 4
)

// The purpose of the "ST" System Use Entry is to provide a terminator
// for the use of the System Use Sharing Protocol for a particular
// System Use field or Continuation Area.
type TerminatorEntry struct{}

func NewTerminatorEntry() SystemUseEntry {
	return &TerminatorEntry{}
}

func (st *TerminatorEntry) Len() int {
	return TerminatorEntryLength
}

func (st *TerminatorEntry) WriteTo(w io.Writer) (n int64, err error) {
	var m int

	m, err = io.WriteString(w, "ST")
	n += int64(m)
	if err != nil {
		return
	}
	if err = writeByte(w, byte(st.Len())); err != nil {
		return
	}
	n += 1
	// Version 1 Always
	if err = writeByte(w, 0x01); err != nil {
		return
	}
	n += 1
	return
}
