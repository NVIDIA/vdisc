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

package iso9660

import (
	"io"

	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

// A continuation of a directory record's system use area.
type ContinuationArea struct {
	start   LogicalBlockAddress
	sectors []continuationSector
}

func NewContinuationArea(start LogicalBlockAddress) ContinuationArea {
	return ContinuationArea{start, nil}
}

func (ca *ContinuationArea) Append(entries []susp.SystemUseEntry) *susp.ContinuationAreaEntry {
	if len(entries) == 0 {
		return nil
	}

	if len(ca.sectors) == 0 {
		ca.sectors = append(ca.sectors, continuationSector{LogicalBlockSize, nil})
	}

	for {
		tmpEntries := entries

		lastSector := len(ca.sectors) - 1
		var baseLen int
		var extraLen int
		for _, entry := range tmpEntries {
			baseLen += entry.Len()
		}

		var overflow []susp.SystemUseEntry
		for len(tmpEntries) > 0 && (baseLen+extraLen) > ca.sectors[lastSector].room {
			lastIdx := len(tmpEntries) - 1
			last := tmpEntries[lastIdx]
			tmpEntries = tmpEntries[:lastIdx]
			overflow = append([]susp.SystemUseEntry{last}, overflow...)
			baseLen -= last.Len()
			extraLen = susp.ContinuationAreaEntryLength
		}

		if len(tmpEntries) == 0 {
			// We couldn't fit any of the entries in the last sector, lets append a new sector and try again
			ca.sectors = append(ca.sectors, continuationSector{LogicalBlockSize, nil})
			continue
		}

		if len(overflow) == 0 {
			// We can fit all the entries in this sector. Just append tmpEntries, and we're done
			ceOffset := LogicalBlockSize - ca.sectors[lastSector].room
			ca.sectors[lastSector].room -= baseLen
			ca.sectors[lastSector].entries = append(ca.sectors[lastSector].entries, tmpEntries...)

			return susp.NewContinuationAreaEntry(uint32(ca.start)+uint32(lastSector), uint32(ceOffset), uint32(baseLen))
		} else {
			// We're going to spill over into the next sector. We
			// append all the entries that will fit and update the
			// room to include a CE pointing to the next sector. Once
			// we've reserved space, we recursively call Append with
			// the overflowing entries, effectively chaining system
			// use areas together with CE entries.
			ceOffset := LogicalBlockSize - ca.sectors[lastSector].room
			ca.sectors[lastSector].room -= baseLen + extraLen
			ca.sectors[lastSector].entries = append(ca.sectors[lastSector].entries, tmpEntries...)

			nextCE := ca.Append(overflow)
			ca.sectors[lastSector].entries = append(ca.sectors[lastSector].entries, nextCE)

			return susp.NewContinuationAreaEntry(uint32(ca.start)+uint32(lastSector), uint32(ceOffset), uint32(baseLen+extraLen))
		}
	}
}

func (ca *ContinuationArea) Len() (n int) {
	return len(ca.sectors) * LogicalBlockSize
}

func (ca *ContinuationArea) WriteTo(w io.Writer) (int64, error) {
	cw := newCountingWriter(w)
	padOutSector := func() error {
		remainder := int(cw.Written() % LogicalBlockSize)
		if remainder > 0 {
			return pad(cw, LogicalBlockSize-remainder)
		}
		return nil
	}

	for _, sector := range ca.sectors {
		for _, entry := range sector.entries {
			if _, err := entry.WriteTo(cw); err != nil {
				return cw.Written(), err
			}
		}

		if err := padOutSector(); err != nil {
			return cw.Written(), err
		}
	}

	return cw.Written(), nil
}

type continuationSector struct {
	room    int
	entries []susp.SystemUseEntry
}
