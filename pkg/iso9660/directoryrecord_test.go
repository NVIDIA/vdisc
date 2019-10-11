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
package iso9660_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

func TestDirectoryRecord(t *testing.T) {
	expected := iso9660.DirectoryRecord{
		Identifier:               "ASDF;txt",
		Start:                    17,
		Length:                   5000,
		Flags:                    0,
		Recorded:                 time.Unix(1, 0).UTC(),
		SystemUse:                []susp.SystemUseEntry{susp.NewSharingProtocolEntry(0)},
		ExtendedAttrRecordLength: 0,
		FileUnitSize:             0,
		InterleaveGap:            0,
		VolumeID:                 1,
	}

	buf := bytes.NewBuffer(nil)
	expected.WriteTo(buf)

	size, err := buf.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	r := io.LimitReader(buf, int64(size))

	var actual iso9660.DirectoryRecord
	iso9660.DecodeDirectoryRecord(r, &actual)

	assert.Equal(t, expected, actual)
}
