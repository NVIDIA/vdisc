// Copyright © 2018 NVIDIA Corporation

package iso9660

import (
	"io"
)

type Terminator struct{}

func (t *Terminator) WriteTo(w io.Writer) (int64, error) {
	cw := newCountingWriter(w)

	// Type Code (255 indicates Terminator)
	if err := writeByte(cw, 255); err != nil {
		return cw.Written(), err
	}

	// Standard Identifier
	if _, err := io.WriteString(cw, "CD001"); err != nil {
		return cw.Written(), err
	}

	// Version (always 1)
	if err := writeByte(cw, 1); err != nil {
		return cw.Written(), err
	}

	// Terminator Data (empty)
	if err := pad(cw, 2041); err != nil {
		return cw.Written(), err
	}
	return cw.Written(), nil
}
