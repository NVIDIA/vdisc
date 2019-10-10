// Copyright © 2018 NVIDIA Corporation

package susp

import (
	"encoding/binary"
	"io"
)

// System Usage Sharing Protocol (SUSP, IEEE P1281)
// See ftp://ftp.ymi.com/pub/rockridge/susp112.ps

type SystemUseEntry interface {
	io.WriterTo
	Len() int
}

func writeByte(w io.Writer, b byte) error {
	n, err := w.Write([]byte{b})
	if err != nil {
		return err
	}
	if n != 1 {
		return io.ErrShortWrite
	}
	return nil
}

func putBothUint32(w io.Writer, v uint32) error {
	if err := binary.Write(w, binary.LittleEndian, v); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return err
	}

	return nil
}
