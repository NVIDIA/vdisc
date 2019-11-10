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
	"encoding/binary"
	"io"
	"io/ioutil"
)

type PathTableRecord struct {
	Identifier                    string
	ExtendedAttributeRecordLength byte
	Location                      LogicalBlockAddress
	ParentIndex                   uint16
}

type PathTable struct {
	Records []PathTableRecord
}

type PathTableEncoder struct {
	byteOrder binary.ByteOrder
	w         *CountingWriter
}

func PathTableEncodedLen(pt *PathTable) int64 {
	enc := NewPathTableEncoder(binary.LittleEndian, ioutil.Discard)
	n, err := enc.Encode(pt)
	if err != nil {
		panic(err)
	}
	return n
}

func NewPathTableEncoder(byteOrder binary.ByteOrder, w io.Writer) *PathTableEncoder {
	return &PathTableEncoder{byteOrder, newCountingWriter(w)}
}

func (e *PathTableEncoder) Encode(pt *PathTable) (int64, error) {
	for _, rec := range pt.Records {
		if err := writeByte(e.w, byte(len(rec.Identifier))); err != nil {
			return e.w.Written(), err
		}

		if err := writeByte(e.w, rec.ExtendedAttributeRecordLength); err != nil {
			return e.w.Written(), err
		}

		if err := binary.Write(e.w, e.byteOrder, rec.Location); err != nil {
			return e.w.Written(), err
		}

		if err := binary.Write(e.w, e.byteOrder, rec.ParentIndex); err != nil {
			return e.w.Written(), err
		}

		if _, err := io.WriteString(e.w, rec.Identifier); err != nil {
			return e.w.Written(), err
		}

		if len(rec.Identifier)%2 != 0 {
			if err := pad(e.w, 1); err != nil {
				return e.w.Written(), err
			}
		}
	}

	return e.w.Written(), nil
}
