// Copyright © 2018 NVIDIA Corporation

package rrip

import (
	"io"
	"os"
	"os/user"
	"strconv"

	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

const (
	PosixEntryLength = 44
)

// RRIP "PX" POSIX file attributes
type PosixEntry struct {
	Mode  os.FileMode
	Nlink uint32
	Uid   uint32
	Gid   uint32
	Ino   uint32
}

func (px *PosixEntry) Len() int {
	return PosixEntryLength
}

func (px *PosixEntry) WriteTo(w io.Writer) (n int64, err error) {
	var m int

	m, err = io.WriteString(w, "PX")
	n += int64(m)
	if err != nil {
		return
	}
	if err = writeByte(w, byte(px.Len())); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, 1); err != nil {
		return
	}
	n += 1
	if err = putBothUint32(w, uint32(px.Mode)); err != nil {
		return
	}
	n += 8
	if err = putBothUint32(w, px.Nlink); err != nil {
		return
	}
	n += 8
	if err = putBothUint32(w, px.Uid); err != nil {
		return
	}
	n += 8
	if err = putBothUint32(w, px.Gid); err != nil {
		return
	}
	n += 8
	if err = putBothUint32(w, px.Ino); err != nil {
		return
	}
	n += 8

	if n != PosixEntryLength {
		panic("never")
	}

	return
}

func (pe *PosixEntry) UidString() string {
	u, err := user.LookupId(strconv.Itoa(int(pe.Uid)))
	if err != nil {
		return strconv.Itoa(int(pe.Uid))
	}
	return u.Username
}

func (pe *PosixEntry) GidString() string {
	g, err := user.LookupGroupId(strconv.Itoa(int(pe.Gid)))
	if err != nil {
		return strconv.Itoa(int(pe.Gid))
	}
	return g.Name
}

func (pe *PosixEntry) ModeString() string {
	return os.FileMode.String(pe.Mode)
}

func DecodePosixEntry(entries []susp.SystemUseEntry) (*PosixEntry, bool) {
	for _, entry := range entries {
		switch e := entry.(type) {
		case *PosixEntry:
			return e, true
		}
	}
	return nil, false
}
