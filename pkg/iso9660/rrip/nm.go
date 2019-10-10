// Copyright © 2018 NVIDIA Corporation

package rrip

import (
	"errors"
	"fmt"
	"io"

	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

//Name RRIP "NM" Alternate name
type Name struct {
	data string
	cont bool
}

//NewNamePart returns valids data and returns a new Name
func NewNamePart(data string, cont bool) (susp.SystemUseEntry, error) {

	if !isValid(data) {
		return nil, fmt.Errorf("Invalid RockRidge file name component (%s)", data)
	}
	if len(data) > 250 {
		return nil, errors.New("RockRidge file name component too long")
	}

	return &Name{data, cont}, nil
}

//NewName creates a sequence of RRIP System Use Entries representing a file name
func NewName(name string) ([]susp.SystemUseEntry, error) {

	if !isValid(name) {
		return nil, fmt.Errorf("Invalid RockRidge file name(%s)", name)
	}
	var entries []susp.SystemUseEntry
	remaining := name
	for len(remaining) > 0 {
		var chunk string
		if len(remaining) > 250 {
			chunk = remaining[:250]
			remaining = remaining[250:]
		} else {
			chunk = remaining
			remaining = ""
		}
		cont := len(remaining) > 0
		entries = append(entries, &Name{chunk, cont})
	}
	return entries, nil
}

func (nm *Name) Data() string {
	return nm.data
}

func (nm *Name) Continue() bool {
	return nm.cont
}

func (nm *Name) Len() int {
	return len(nm.data) + 5
}

func (nm *Name) WriteTo(w io.Writer) (n int64, err error) {
	var m int
	m, err = io.WriteString(w, "NM")
	n += int64(m)
	if err != nil {
		return
	}

	expectedLength := byte(nm.Len())
	if err = writeByte(w, expectedLength); err != nil {
		return
	}
	n++

	if err = writeByte(w, 1); err != nil {
		return
	}
	n++

	flags := byte(0)
	if nm.cont {
		flags |= 0x1
	}
	if err = writeByte(w, flags); err != nil {
		return
	}
	n++

	m, err = io.WriteString(w, nm.data)
	n += int64(m)

	if n != int64(expectedLength) {
		panic("never")
	}

	return
}

func DecodeName(entries []susp.SystemUseEntry) (string, bool) {
	var result string
	for _, entry := range entries {
		switch v := entry.(type) {
		case *Name:
			result = result + v.Data()
			if !v.Continue() {
				return result, true
			}
		}
	}
	return "", false
}
