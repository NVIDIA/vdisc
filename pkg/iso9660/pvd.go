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
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/NVIDIA/vdisc/pkg/iso9660/datetime"
)

const (
	CD001 = "CD001"
)

type PrimaryVolumeDescriptor struct {
	SystemIdentifier            string
	VolumeIdentifier            string
	VolumeSetIdentifier         string
	PublisherIdentifier         string
	DataPreparerIdentifier      string
	ApplicationIdentifier       string
	CopyrightFileIdentifier     string
	AbstractFileIdentifier      string
	BibliographicFileIdentifier string
	VolumeSpaceSize             uint32
	VolumeSetSize               uint16
	VolumeSequenceNumber        uint16
	PathTableSize               uint32
	LTableStart                 LogicalBlockAddress
	OptionalLTableStart         LogicalBlockAddress
	MTableStart                 LogicalBlockAddress
	OptionalMTableStart         LogicalBlockAddress
	RootStart                   LogicalBlockAddress
	RootLength                  uint32
	RootModified                time.Time
	Created                     time.Time
	Modified                    time.Time
	Effective                   time.Time
}

func (pvd *PrimaryVolumeDescriptor) WriteTo(w io.Writer) (int64, error) {
	cw := newCountingWriter(w)

	// Type Code (1 indicates Primary)
	if err := writeByte(cw, 1); err != nil {
		return cw.Written(), err
	}

	// Standard Identifier
	if _, err := io.WriteString(cw, CD001); err != nil {
		return cw.Written(), err
	}

	// Version (always 1)
	if err := writeByte(cw, 1); err != nil {
		return cw.Written(), err
	}

	// Unused
	if err := writeByte(cw, 0); err != nil {
		return cw.Written(), err
	}

	// System Identifier
	if _, err := io.WriteString(cw, StrA(pvd.SystemIdentifier, 32)); err != nil {
		return cw.Written(), err
	}

	// Volume Identifier
	if _, err := io.WriteString(cw, StrD(pvd.VolumeIdentifier, 32)); err != nil {
		return cw.Written(), err
	}

	// Unused
	if err := pad(cw, 8); err != nil {
		return cw.Written(), err
	}

	// Volume Space Size
	if err := putBothUint32(cw, pvd.VolumeSpaceSize); err != nil {
		return cw.Written(), err
	}

	// Unused

	if err := pad(cw, 32); err != nil {
		return cw.Written(), err
	}

	// Volume Set Size
	if err := putBothUint16(cw, pvd.VolumeSetSize); err != nil {
		return cw.Written(), err
	}

	// Volume Sequence Number
	if err := putBothUint16(cw, pvd.VolumeSequenceNumber); err != nil {
		return cw.Written(), err
	}

	// Logical BlockSize
	if err := putBothUint16(cw, LogicalBlockSize); err != nil {
		return cw.Written(), err
	}

	// Path Table Size
	if err := putBothUint32(cw, pvd.PathTableSize); err != nil {
		return cw.Written(), err
	}

	// Location of Type-L Path Table
	if err := binary.Write(cw, binary.LittleEndian, pvd.LTableStart); err != nil {
		return cw.Written(), err
	}

	// optional l-table lba
	if err := binary.Write(cw, binary.LittleEndian, pvd.OptionalLTableStart); err != nil {
		return cw.Written(), err
	}

	// Location of Type-M Path Table
	if err := binary.Write(cw, binary.BigEndian, pvd.MTableStart); err != nil {
		return cw.Written(), err
	}

	// optional m-table lba
	if err := binary.Write(cw, binary.BigEndian, pvd.OptionalMTableStart); err != nil {
		return cw.Written(), err
	}

	// Root Directory Entry
	root := &DirectoryRecord{
		Identifier: "\x00",
		Start:      pvd.RootStart,
		Length:     pvd.RootLength,
		Flags:      FileFlagDir,
		Recorded:   pvd.RootModified,
		VolumeID:   1,
	}

	{
		n, err := root.WriteTo(cw)
		if err != nil {
			return cw.Written(), err
		}
		if n != 34 {
			panic("never")
		}
	}

	if _, err := io.WriteString(cw, StrD(pvd.VolumeSetIdentifier, 128)); err != nil {
		return cw.Written(), err
	}

	if _, err := io.WriteString(cw, StrA(pvd.PublisherIdentifier, 128)); err != nil {
		return cw.Written(), err
	}

	if _, err := io.WriteString(cw, StrA(pvd.DataPreparerIdentifier, 128)); err != nil {
		return cw.Written(), err
	}

	if _, err := io.WriteString(cw, StrA(pvd.ApplicationIdentifier, 128)); err != nil {
		return cw.Written(), err
	}

	if _, err := io.WriteString(cw, StrD(pvd.CopyrightFileIdentifier, 38)); err != nil {
		return cw.Written(), err
	}

	if _, err := io.WriteString(cw, StrD(pvd.AbstractFileIdentifier, 36)); err != nil {
		return cw.Written(), err
	}

	if _, err := io.WriteString(cw, StrD(pvd.BibliographicFileIdentifier, 37)); err != nil {
		return cw.Written(), err
	}

	// Volume Creation Date and Time
	created := datetime.NewDecDateTime(pvd.Created)
	if _, err := cw.Write(created[:]); err != nil {
		return cw.Written(), err
	}

	// Volume Modification Date and Time
	modified := datetime.NewDecDateTime(pvd.Modified)
	if _, err := cw.Write(modified[:]); err != nil {
		return cw.Written(), err
	}

	// Volume Expiration Date and Time
	if _, err := cw.Write(datetime.MaxDecDateTime[:]); err != nil {
		return cw.Written(), err
	}

	// Volume Effective Date and Time
	effective := datetime.NewDecDateTime(pvd.Effective)
	if _, err := cw.Write(effective[:]); err != nil {
		return cw.Written(), err
	}

	// File Structure Version
	if err := writeByte(cw, 1); err != nil {
		return cw.Written(), err
	}

	// Unused
	if err := writeByte(cw, 0); err != nil {
		return cw.Written(), err
	}

	// Application Used
	if _, err := io.WriteString(cw, strings.Repeat(" ", 512)); err != nil {
		return cw.Written(), err
	}

	// Reserved
	if err := pad(cw, 653); err != nil {
		return cw.Written(), err
	}

	return cw.Written(), nil
}

func DecodePrimaryVolumeDescriptor(r io.Reader, pvd *PrimaryVolumeDescriptor) (err error) {
	err = readExpectedByte(r, 1, "Primary Volume Descriptor - Type Code")
	if err != nil {
		return
	}

	err = readExpectedString(r, CD001, "Primary Volume Descriptor - Standard Identifier")
	if err != nil {
		return
	}

	err = readExpectedByte(r, 1, "Primary Volume Descriptor - Version")
	if err != nil {
		return
	}

	err = readExpectedByte(r, 0, "Primary Volume Descriptor - Unused")
	if err != nil {
		return
	}

	if pvd.SystemIdentifier, err = readStrA(r, 32); err != nil {
		return
	}

	if pvd.VolumeIdentifier, err = readStrD(r, 32); err != nil {
		return
	}

	if err = unpad(r, 8); err != nil {
		return
	}

	if pvd.VolumeSpaceSize, err = getBothUint32(r); err != nil {
		return
	}

	if err = unpad(r, 32); err != nil {
		return
	}

	if pvd.VolumeSetSize, err = getBothUint16(r); err != nil {
		return
	}

	if pvd.VolumeSequenceNumber, err = getBothUint16(r); err != nil {
		return
	}

	var declaredLogicalBlockSize uint16
	if declaredLogicalBlockSize, err = getBothUint16(r); err != nil {
		return
	}
	if declaredLogicalBlockSize != LogicalBlockSize {
		err = fmt.Errorf("Unsupported Logical Block Size: %d", declaredLogicalBlockSize)
		return
	}

	if pvd.PathTableSize, err = getBothUint32(r); err != nil {
		return
	}

	if err = binary.Read(r, binary.LittleEndian, &pvd.LTableStart); err != nil {
		return
	}

	if err = binary.Read(r, binary.LittleEndian, &pvd.OptionalLTableStart); err != nil {
		return
	}

	if err = binary.Read(r, binary.BigEndian, &pvd.MTableStart); err != nil {
		return
	}

	if err = binary.Read(r, binary.BigEndian, &pvd.OptionalMTableStart); err != nil {
		return
	}

	if err = readExpectedByte(r, 34, "Directory Record Length"); err != nil {
		return
	}

	var root DirectoryRecord
	if err = DecodeDirectoryRecord(io.LimitReader(r, 33), &root); err != nil {
		return
	}

	pvd.RootStart = root.Start
	pvd.RootLength = root.Length
	pvd.RootModified = root.Recorded

	if pvd.VolumeSetIdentifier, err = readStrD(r, 128); err != nil {
		return
	}

	if pvd.PublisherIdentifier, err = readStrA(r, 128); err != nil {
		return
	}

	if pvd.DataPreparerIdentifier, err = readStrA(r, 128); err != nil {
		return
	}

	if pvd.ApplicationIdentifier, err = readStrA(r, 128); err != nil {
		return
	}

	if pvd.CopyrightFileIdentifier, err = readStrD(r, 38); err != nil {
		return
	}

	if pvd.AbstractFileIdentifier, err = readStrD(r, 36); err != nil {
		return
	}

	if pvd.BibliographicFileIdentifier, err = readStrD(r, 37); err != nil {
		return
	}

	var created datetime.DecDateTime
	if _, err = io.ReadFull(r, created[:]); err != nil {
		return
	}
	pvd.Created = created.Timestamp()

	var modified datetime.DecDateTime
	if _, err = io.ReadFull(r, modified[:]); err != nil {
		return
	}
	pvd.Modified = modified.Timestamp()

	var expires datetime.DecDateTime
	if _, err = io.ReadFull(r, expires[:]); err != nil {
		return
	}
	// TODO: do something with expires

	var effective datetime.DecDateTime
	if _, err = io.ReadFull(r, effective[:]); err != nil {
		return
	}
	pvd.Effective = effective.Timestamp()

	if err = readExpectedByte(r, 1, "file structure version"); err != nil {
		return
	}

	if err = readExpectedByte(r, 0, "unused"); err != nil {
		return
	}

	// Application Used
	application := make([]byte, 512)
	if _, err = io.ReadFull(r, application); err != nil {
		return
	}
	// TODO: do something with application data

	// Reserved
	reserved := make([]byte, 653)
	if _, err = io.ReadFull(r, reserved); err != nil {
		return
	}

	return
}
