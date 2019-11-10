// Copyright © 2019 NVIDIA Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


package rrip

import (
	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

type SymlinkComponentFlag byte

const (
	SymlinkComponentFlagContinue SymlinkComponentFlag = 1 << iota
	SymlinkComponentFlagCurrent
	SymlinkComponentFlagParent
	SymlinkComponentFlagRoot
)

// Component portion of RRIP "SL" Symbolic link
type SymlinkComponent struct {
	flags SymlinkComponentFlag
	data  string
}

func NewSymlinkComponent(segment string) (components []SymlinkComponent, err error) {
	if !isValid(segment) {
		err = errors.New("symlink path segments must only contain characters from the POSIX portable file name character set")
		return
	}

	if segment == "." {
		components = append(components, SymlinkComponent{
			flags: SymlinkComponentFlagCurrent,
		})
		return
	} else if segment == ".." {
		components = append(components, SymlinkComponent{
			flags: SymlinkComponentFlagParent,
		})
		return
	} else if segment == "" {
		components = append(components, SymlinkComponent{
			flags: SymlinkComponentFlagRoot,
		})
		return
	}

	var chunks []string
	remaining := segment
	for {
		if len(remaining) > 248 {
			chunks = append(chunks, remaining[:248])
			remaining = remaining[248:]
		} else {
			chunks = append(chunks, remaining)
			break
		}
	}

	for _, chunk := range chunks {
		components = append(components, SymlinkComponent{
			flags: SymlinkComponentFlagContinue,
			data:  chunk,
		})
	}
	components[len(components)-1].flags ^= SymlinkComponentFlagContinue
	return
}

func (sc *SymlinkComponent) Flags() SymlinkComponentFlag {
	return sc.flags
}

func (sc *SymlinkComponent) Data() string {
	return sc.data
}

func (sc *SymlinkComponent) Len() int {
	if sc.flags&SymlinkComponentFlagCurrent != 0 {
		return 2
	} else if sc.flags&SymlinkComponentFlagParent != 0 {
		return 2
	} else if sc.flags&SymlinkComponentFlagRoot != 0 {
		return 2
	}

	return 2 + len(sc.data)
}

func (sc *SymlinkComponent) WriteTo(w io.Writer) (n int64, err error) {
	var m int

	if err = writeByte(w, byte(sc.flags)); err != nil {
		return
	}
	n++

	if err = writeByte(w, byte(len(sc.data))); err != nil {
		return
	}
	n++

	m, err = io.WriteString(w, sc.data)
	n += int64(m)

	return
}

// RRIP "SL" Symbolic link
type Symlink struct {
	comp SymlinkComponent
	cont bool
}

func NewSymlinkPart(flags SymlinkComponentFlag, data string, cont bool) (susp.SystemUseEntry, error) {
	return &Symlink{
		comp: SymlinkComponent{
			flags: flags,
			data:  data,
		},
		cont: cont,
	}, nil
}

// Create a sequence of RRIP System Use Entries representing a posix symlink
func NewSymlink(target string) ([]susp.SystemUseEntry, error) {
	var sls []*Symlink
	segments := strings.Split(path.Clean(target), string(os.PathSeparator))
	for _, segment := range segments {
		components, err := NewSymlinkComponent(segment)
		if err != nil {
			return nil, err
		}
		for _, component := range components {
			sls = append(sls, &Symlink{component, true})
		}
	}

	if len(sls) > 0 {
		// mark the final entry as terminal
		sls[len(sls)-1].cont = false
	}
	var entries []susp.SystemUseEntry
	for _, sl := range sls {
		entries = append(entries, sl)
	}
	return entries, nil
}

func (sl *Symlink) Component() SymlinkComponent {
	return sl.comp
}

func (sl *Symlink) Continue() bool {
	return sl.cont
}

func (sl *Symlink) Len() int {
	return 5 + sl.comp.Len()
}

func (sl *Symlink) WriteTo(w io.Writer) (n int64, err error) {
	var m int
	var nn int64

	m, err = io.WriteString(w, "SL")
	n += int64(m)
	if err != nil {
		return
	}

	expectedLength := byte(sl.Len())
	if err = writeByte(w, expectedLength); err != nil {
		return
	}
	n++

	if err = writeByte(w, 1); err != nil {
		return
	}
	n++

	flags := byte(0)
	if sl.cont {
		flags |= 1 << 0
	}

	if err = writeByte(w, flags); err != nil {
		return
	}
	n++

	nn, err = sl.comp.WriteTo(w)
	n += nn

	if n != int64(expectedLength) {
		panic("never")
	}

	return
}

func DecodeSymlink(entries []susp.SystemUseEntry) (string, bool) {
	var parts []string
	var partial string

	for _, entry := range entries {
		switch v := entry.(type) {
		case *Symlink:
			comp := v.Component()

			if comp.Flags()&SymlinkComponentFlagCurrent != 0 {
				parts = append(parts, ".")
				continue
			}

			if comp.Flags()&SymlinkComponentFlagParent != 0 {
				parts = append(parts, "..")
				continue
			}

			if comp.Flags()&SymlinkComponentFlagRoot != 0 {
				parts = append(parts, "")
				continue
			}

			partial = partial + comp.Data()
			if comp.Flags()&SymlinkComponentFlagContinue == 0 {
				parts = append(parts, partial)
				partial = ""
			}
			if !v.Continue() {
				return strings.Join(parts, string(os.PathSeparator)), true
			}
		}
	}
	return "", false
}
