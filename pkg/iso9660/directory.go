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
	"errors"
	"io"
	"io/ioutil"
)

var (
	ErrDirectoryRecordTooBig = errors.New("DirectoryRecord larger than sector")
	ErrDirectoryTooBig       = errors.New("Directory larger than MaxUint32")
)

// The contents of a directory extent.
type Directory struct {
	Records []DirectoryRecord
}

func (d *Directory) Size() int64 {
	n, err := d.WriteTo(ioutil.Discard)
	if err != nil {
		panic(err)
	}
	return int64(sectorsToBytes(bytesToSectors(uint32(n))))
}

// Write Records to w sequentially, ensure that no record is split
// across a sector boundary.
func (d *Directory) WriteTo(w io.Writer) (int64, error) {
	cw := newCountingWriter(w)
	for _, rec := range d.Records {
		l := rec.Len()
		if l > LogicalBlockSize {
			return cw.Written(), ErrDirectoryRecordTooBig
		}
		sectorCapacity := int(LogicalBlockSize - (cw.Written() % LogicalBlockSize))
		if l > sectorCapacity {
			if err := pad(cw, sectorCapacity); err != nil {
				return cw.Written(), err
			}
		}

		if _, err := rec.WriteTo(cw); err != nil {
			return cw.Written(), err
		}
	}

	if cw.Written() > MaxPartSize {
		return cw.Written(), ErrDirectoryTooBig
	}
	return cw.Written(), nil
}

func DecodeDirectory(r io.Reader, dir *Directory) (err error) {
	cr := newCountingReader(r)

	for {
		var rlen byte
		if rlen, err = readByte(cr); err != nil {
			if err == io.EOF {
				return nil
			}
			return
		}

		if rlen == 0 {
			// The rest of this sector is padding. Consume, and move on the the next sector.
			padding := sectorsToBytes(bytesToSectors(uint32(cr.Consumed()))) - cr.Consumed()
			err = unpad(cr, int(padding))
			if err != nil {
				return
			}
			continue
		}

		var rec DirectoryRecord
		err = DecodeDirectoryRecord(io.LimitReader(cr, int64(rlen)-1), &rec)
		if err != nil {
			return
		}

		dir.Records = append(dir.Records, rec)
	}
}
