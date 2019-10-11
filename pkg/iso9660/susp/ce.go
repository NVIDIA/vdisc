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
