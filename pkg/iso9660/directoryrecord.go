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
	"log"
	"os"
	"time"

	"github.com/NVIDIA/vdisc/pkg/iso9660/datetime"
	"github.com/NVIDIA/vdisc/pkg/iso9660/rrip"
	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

const (
	MaxDirectoryRecordLen           = 255
	MaxDirectoryRecordIdentifierLen = 30
)

var (
	ErrDirectoryRecordIdentifierOverflow = errors.New("DirectoryRecord.Identifier longer than 30 bytes")
)

type DirectoryRecord struct {
	Identifier               string
	Start                    LogicalBlockAddress
	Length                   uint32
	Flags                    FileFlag
	Recorded                 time.Time
	SystemUse                []susp.SystemUseEntry
	ExtendedAttrRecordLength byte
	FileUnitSize             byte
	InterleaveGap            byte
	VolumeID                 uint16
}

func (rec *DirectoryRecord) Len() int {
	idLen := len(rec.Identifier)
	n := 34 + idLen - (idLen % 2)
	for _, entry := range rec.SystemUse {
		n += entry.Len()
	}
	return n
}

func (rec *DirectoryRecord) WriteTo(w io.Writer) (int64, error) {
	// Check identifier length
	if len(rec.Identifier) > MaxDirectoryRecordIdentifierLen {
		return 0, ErrDirectoryRecordIdentifierOverflow
	}

	expectedLength := rec.Len()
	if expectedLength > MaxDirectoryRecordLen {
		panic("never")
	}

	cw := newCountingWriter(w)

	// length of the record
	if err := writeByte(cw, byte(expectedLength)); err != nil {
		return cw.Written(), err
	}

	// ExtendedAttrRecordLength
	if err := writeByte(cw, rec.ExtendedAttrRecordLength); err != nil {
		return cw.Written(), err
	}

	// Location of extent (LBA) in both-endian format.
	if err := putBothUint32(cw, uint32(rec.Start)); err != nil {
		return cw.Written(), err
	}

	// The length of all the entries in this directory excluding . an ..
	if err := putBothUint32(cw, rec.Length); err != nil {
		return cw.Written(), err
	}

	// Recording date and time.
	recorded := datetime.NewEntryDateTime(rec.Recorded)
	if _, err := cw.Write(recorded[:]); err != nil {
		return cw.Written(), err
	}

	// flags
	if err := writeByte(cw, byte(rec.Flags)); err != nil {
		return cw.Written(), err
	}

	// FileUnitSize for files recorded in interleaved mode, zero otherwise.
	if err := writeByte(cw, rec.FileUnitSize); err != nil {
		return cw.Written(), err
	}

	// InterleaveGap gap size for files recorded in interleaved mode, zero otherwise.
	if err := writeByte(cw, rec.InterleaveGap); err != nil {
		return cw.Written(), err
	}

	// the volume that this extent is recorded on, in 16 bit both-endian format.
	if err := putBothUint16(cw, rec.VolumeID); err != nil {
		return cw.Written(), err
	}

	// id length
	if err := writeByte(cw, byte(len(rec.Identifier))); err != nil {
		return cw.Written(), err
	}

	// id
	if _, err := io.WriteString(cw, rec.Identifier); err != nil {
		return cw.Written(), err
	}

	if len(rec.Identifier)%2 == 0 {
		if err := writeByte(cw, 0); err != nil {
			return cw.Written(), err
		}
	}

	// System Use Area
	for _, entry := range rec.SystemUse {
		if _, err := entry.WriteTo(cw); err != nil {
			return cw.Written(), err
		}
	}

	if cw.Written() != int64(expectedLength) {
		panic("never")
	}

	return cw.Written(), nil
}

// Assumes the length has already been consumed from r
func DecodeDirectoryRecord(r io.Reader, rec *DirectoryRecord) (err error) {
	if rec.ExtendedAttrRecordLength, err = readByte(r); err != nil {
		return
	}

	var start uint32
	if start, err = getBothUint32(r); err != nil {
		return
	}
	rec.Start = LogicalBlockAddress(start)

	if rec.Length, err = getBothUint32(r); err != nil {
		return
	}

	var recorded datetime.EntryDateTime
	if _, err = io.ReadFull(r, recorded[:]); err != nil {
		return
	}
	rec.Recorded = recorded.Timestamp()

	var flags byte
	if flags, err = readByte(r); err != nil {
		return
	}
	rec.Flags = FileFlag(flags)

	if rec.FileUnitSize, err = readByte(r); err != nil {
		return
	}

	if rec.InterleaveGap, err = readByte(r); err != nil {
		return
	}

	if rec.VolumeID, err = getBothUint16(r); err != nil {
		return
	}

	var idlen byte
	if idlen, err = readByte(r); err != nil {
		return
	}

	ident := make([]byte, int(idlen))
	if _, err = io.ReadFull(r, ident); err != nil {
		return
	}
	rec.Identifier = string(ident)

	if idlen%2 == 0 {
		if err = readExpectedByte(r, 0, "DirectoryRecord.Identifier pad byte"); err != nil {
			return
		}
	}

	rec.SystemUse, err = DecodeSystemUseEntries(r)
	if err != nil {
		return
	}

	return
}

func DecodeSystemUseEntries(r io.Reader) (entries []susp.SystemUseEntry, err error) {
	for {
		var nible0, nible1, entryLen byte

		if nible0, err = readByte(r); err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
		if nible1, err = readByte(r); err != nil {
			return
		}

		if entryLen, err = readByte(r); err != nil {
			return
		}

		body := io.LimitReader(r, int64(entryLen)-3)

		var entry susp.SystemUseEntry
		switch string([]byte{nible0, nible1}) {
		case "SP":
			if entry, err = decodeSP(body); err != nil {
				return
			}
		case "ST":
			if entry, err = decodeST(body); err != nil {
				return
			}
		case "CE":
			if entry, err = decodeCE(body); err != nil {
				return
			}
		case "ER":
			if entry, err = decodeER(body); err != nil {
				return
			}
		case "NM":
			if entry, err = decodeNM(body); err != nil {
				return
			}
		case "PX":
			if entry, err = decodePX(body); err != nil {
				return
			}
		case "SL":
			if entry, err = decodeSL(body); err != nil {
				return
			}
		case "TF":
			if entry, err = decodeTF(body); err != nil {
				return
			}
		}

		if entry != nil {
			entries = append(entries, entry)
		}

		// Discard the rest of the message
		var discarded int
		if _, err = io.Copy(ioutil.Discard, body); err != nil {
			return
		}

		if discarded > 0 {
			log.Printf("Ignoring %d bytes of SUSP[type=\"%s\", len=%d] entry.", discarded, string([]byte{nible0, nible1}), entryLen)
		}
	}
}

func decodeSP(body io.Reader) (susp.SystemUseEntry, error) {
	if err := readExpectedByte(body, 0x01, "SUSP SP Version"); err != nil {
		return nil, err
	}
	if err := readExpectedByte(body, 0xBE, "SUSP SP Check Byte 0xBE"); err != nil {
		return nil, err
	}
	if err := readExpectedByte(body, 0xEF, "SUSP SP Check Byte 0xEF"); err != nil {
		return nil, err
	}
	lenSkp, err := readByte(body)
	if err != nil {
		return nil, err
	}

	return susp.NewSharingProtocolEntry(lenSkp), nil
}

func decodeST(body io.Reader) (susp.SystemUseEntry, error) {
	if err := readExpectedByte(body, 0x01, "SUSP ST Version"); err != nil {
		return nil, err
	}

	return susp.NewTerminatorEntry(), nil
}

func decodeCE(body io.Reader) (susp.SystemUseEntry, error) {
	var err error
	var start, off, len uint32

	if err = readExpectedByte(body, 0x01, "SUSP CE Version"); err != nil {
		return nil, err
	}

	if start, err = getBothUint32(body); err != nil {
		return nil, err
	}
	if off, err = getBothUint32(body); err != nil {
		return nil, err
	}
	if len, err = getBothUint32(body); err != nil {
		return nil, err
	}
	return susp.NewContinuationAreaEntry(start, off, len), nil
}

func decodeER(body io.Reader) (susp.SystemUseEntry, error) {
	var err error
	var idLen, descLen, srcLen, version byte

	if err = readExpectedByte(body, 0x01, "SUSP ER Version"); err != nil {
		return nil, err
	}

	if idLen, err = readByte(body); err != nil {
		return nil, err
	}

	if descLen, err = readByte(body); err != nil {
		return nil, err
	}

	if srcLen, err = readByte(body); err != nil {
		return nil, err
	}

	if version, err = readByte(body); err != nil {
		return nil, err
	}

	identifier := make([]byte, int(idLen))
	if _, err := io.ReadFull(body, identifier); err != nil {
		return nil, err
	}
	descriptor := make([]byte, int(descLen))
	if _, err := io.ReadFull(body, descriptor); err != nil {
		return nil, err
	}
	source := make([]byte, int(srcLen))
	if _, err := io.ReadFull(body, source); err != nil {
		return nil, err
	}

	return susp.NewExtensionsReferenceEntry(version, string(identifier), string(descriptor), string(source))
}

func decodeNM(body io.Reader) (susp.SystemUseEntry, error) {
	if err := readExpectedByte(body, 0x01, "RRIP NM Version"); err != nil {
		return nil, err
	}

	flags, err := readByte(body)
	if err != nil {
		return nil, err
	}
	cont := (flags & 0x1) != 0

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	return rrip.NewNamePart(string(data), cont)
}

func decodePX(body io.Reader) (susp.SystemUseEntry, error) {
	if err := readExpectedByte(body, 0x01, "RRIP PX Version"); err != nil {
		return nil, err
	}

	mode, err := getBothUint32(body)
	if err != nil {
		return nil, err
	}
	nlink, err := getBothUint32(body)
	if err != nil {
		return nil, err
	}
	uid, err := getBothUint32(body)
	if err != nil {
		return nil, err
	}
	gid, err := getBothUint32(body)
	if err != nil {
		return nil, err
	}
	ino, err := getBothUint32(body)
	if err != nil {
		return nil, err
	}

	return &rrip.PosixEntry{
		Mode:  os.FileMode(mode),
		Nlink: nlink,
		Uid:   uid,
		Gid:   gid,
		Ino:   ino,
	}, nil
}

func decodeSL(body io.Reader) (susp.SystemUseEntry, error) {
	if err := readExpectedByte(body, 0x01, "RRIP SL Version"); err != nil {
		return nil, err
	}

	flags, err := readByte(body)
	if err != nil {
		return nil, err
	}
	cont := (flags & 0x1) != 0

	componentFlags, err := readByte(body)
	if err != nil {
		return nil, err
	}

	chunkLen, err := readByte(body)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	if len(data) != int(chunkLen) {
		return nil, errors.New("bad SL component data length")
	}

	return rrip.NewSymlinkPart(rrip.SymlinkComponentFlag(componentFlags), string(data), cont)
}

func decodeTF(body io.Reader) (susp.SystemUseEntry, error) {
	var tf rrip.Timestamps

	if err := readExpectedByte(body, 0x01, "RRIP TF Version"); err != nil {
		return nil, err
	}

	flags, err := readByte(body)
	if err != nil {
		return nil, err
	}

	tf.LongForm = flags&0x80 != 0
	var readTimestamp func() (*time.Time, error)
	if tf.LongForm {
		readTimestamp = func() (*time.Time, error) {
			var ddt datetime.DecDateTime
			_, err := io.ReadFull(body, ddt[:])
			if err != nil {
				return nil, err
			}
			ts := ddt.Timestamp()
			return &ts, nil
		}
	} else {
		readTimestamp = func() (*time.Time, error) {
			var edt datetime.EntryDateTime
			_, err := io.ReadFull(body, edt[:])
			if err != nil {
				return nil, err
			}
			ts := edt.Timestamp()
			return &ts, nil
		}
	}

	if flags&0x1 != 0 {
		ts, err := readTimestamp()
		if err != nil {
			return nil, err
		}
		tf.Created = ts
	}
	if flags&0x2 != 0 {
		ts, err := readTimestamp()
		if err != nil {
			return nil, err
		}
		tf.Modified = ts
	}
	if flags&0x4 != 0 {
		ts, err := readTimestamp()
		if err != nil {
			return nil, err
		}
		tf.Access = ts
	}
	if flags&0x8 != 0 {
		ts, err := readTimestamp()
		if err != nil {
			return nil, err
		}
		tf.Attributes = ts
	}
	if flags&0x10 != 0 {
		ts, err := readTimestamp()
		if err != nil {
			return nil, err
		}
		tf.Backup = ts
	}
	if flags&0x20 != 0 {
		ts, err := readTimestamp()
		if err != nil {
			return nil, err
		}
		tf.Expiration = ts
	}
	if flags&0x40 != 0 {
		ts, err := readTimestamp()
		if err != nil {
			return nil, err
		}
		tf.Effective = ts
	}

	return &tf, nil
}
