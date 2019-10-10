// Copyright © 2018 NVIDIA Corporation

package susp

import (
	"errors"
	"io"
	"math"
)

// The purpose of the "ER" System Use Entry is to store information
// which uniquely identifies a specification of system-specific
// extensions utilized on a specific Directory Hierarchy.
type ExtensionsReferenceEntry struct {
	version    byte
	identifier string
	descriptor string
	source     string
}

func NewExtensionsReferenceEntry(version byte, identifier string, descriptor string, source string) (SystemUseEntry, error) {
	if len(identifier) > math.MaxUint8 {
		return nil, errors.New("identifier too long")
	}
	if len(descriptor) > math.MaxUint8 {
		return nil, errors.New("descriptor too long")
	}
	if len(source) > math.MaxUint8 {
		return nil, errors.New("source too long")
	}
	return &ExtensionsReferenceEntry{version, identifier, descriptor, source}, nil
}

func (er *ExtensionsReferenceEntry) Len() int {
	return 8 + len(er.identifier) + len(er.descriptor) + len(er.source)
}

func (er *ExtensionsReferenceEntry) WriteTo(w io.Writer) (n int64, err error) {
	var m int

	m, err = io.WriteString(w, "ER")
	n += int64(m)
	if err != nil {
		return
	}
	if err = writeByte(w, byte(er.Len())); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, 1); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, byte(len(er.identifier))); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, byte(len(er.descriptor))); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, byte(len(er.source))); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, er.version); err != nil {
		return
	}
	n += 1
	m, err = io.WriteString(w, er.identifier)
	n += int64(m)
	if err != nil {
		return
	}
	m, err = io.WriteString(w, er.descriptor)
	n += int64(m)
	if err != nil {
		return
	}
	m, err = io.WriteString(w, er.source)
	n += int64(m)

	return
}
