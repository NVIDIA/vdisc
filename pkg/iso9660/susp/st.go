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
	n++
	// Version 1 Always
	if err = writeByte(w, 0x01); err != nil {
		return
	}
	n++
	return
}
