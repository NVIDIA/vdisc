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
	SharingProtocolEntryLength = 7
)

// The purpose of the "SP" System Use Entry is to provide an
// identifier that the System Use Sharing Protocol is being used
// within the given volume.
type SharingProtocolEntry struct {
	lenSkp byte
}

func NewSharingProtocolEntry(lenSkp byte) SystemUseEntry {
	return &SharingProtocolEntry{lenSkp}
}

func (sp *SharingProtocolEntry) Len() int {
	return SharingProtocolEntryLength
}

func (sp *SharingProtocolEntry) WriteTo(w io.Writer) (n int64, err error) {
	var m int

	m, err = io.WriteString(w, "SP")
	n += int64(m)
	if err != nil {
		return
	}
	if err = writeByte(w, byte(sp.Len())); err != nil {
		return
	}
	n += 1
	// Version 1 Always
	if err = writeByte(w, 0x01); err != nil {
		return
	}
	n += 1
	// Check Byte BE
	if err = writeByte(w, 0xBE); err != nil {
		return
	}
	n += 1
	// Check Byte EF
	if err = writeByte(w, 0xEF); err != nil {
		return
	}
	n += 1
	// LEN_SKP
	if err = writeByte(w, sp.lenSkp); err != nil {
		return
	}
	n += 1
	return
}
