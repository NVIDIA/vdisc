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
	"io"
	"time"

	"github.com/NVIDIA/vdisc/pkg/iso9660/datetime"
	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

// RRIP "TF" timestamps
type Timestamps struct {
	Created    *time.Time
	Modified   *time.Time
	Access     *time.Time
	Attributes *time.Time
	Backup     *time.Time
	Expiration *time.Time
	Effective  *time.Time
	LongForm   bool
}

func (tf *Timestamps) Len() int {
	result := 5

	var tslen int
	if tf.LongForm {
		tslen = 17
	} else {
		tslen = 7
	}

	if tf.Created != nil {
		result += tslen
	}
	if tf.Modified != nil {
		result += tslen
	}
	if tf.Access != nil {
		result += tslen
	}
	if tf.Attributes != nil {
		result += tslen
	}
	if tf.Backup != nil {
		result += tslen
	}
	if tf.Expiration != nil {
		result += tslen
	}
	if tf.Effective != nil {
		result += tslen
	}

	return result
}

func (tf *Timestamps) WriteTo(w io.Writer) (n int64, err error) {
	var m int

	m, err = io.WriteString(w, "TF")
	n += int64(m)
	if err != nil {
		return
	}

	expectedLength := byte(tf.Len())
	if err = writeByte(w, expectedLength); err != nil {
		return
	}
	n += 1
	if err = writeByte(w, 1); err != nil {
		return
	}
	n += 1

	var flags byte
	if tf.Created != nil {
		flags |= 0x1
	}
	if tf.Modified != nil {
		flags |= 0x2
	}
	if tf.Access != nil {
		flags |= 0x4
	}
	if tf.Attributes != nil {
		flags |= 0x8
	}
	if tf.Backup != nil {
		flags |= 0x10
	}
	if tf.Expiration != nil {
		flags |= 0x20
	}
	if tf.Effective != nil {
		flags |= 0x40
	}
	if tf.LongForm {
		flags |= 0x80
	}

	if err = writeByte(w, flags); err != nil {
		return
	}
	n += 1

	writeTimestamp := func(t time.Time) {
		var et []byte
		if tf.LongForm {
			dd := datetime.NewDecDateTime(t)
			et = dd[:]
		} else {
			ed := datetime.NewEntryDateTime(t)
			et = ed[:]
		}
		m, err = w.Write(et)
		n += int64(m)
		return
	}

	if tf.Created != nil {
		writeTimestamp(*tf.Created)
		if err != nil {
			return
		}
	}
	if tf.Modified != nil {
		writeTimestamp(*tf.Modified)
		if err != nil {
			return
		}
	}
	if tf.Access != nil {
		writeTimestamp(*tf.Access)
		if err != nil {
			return
		}
	}
	if tf.Attributes != nil {
		writeTimestamp(*tf.Attributes)
		if err != nil {
			return
		}
	}
	if tf.Backup != nil {
		writeTimestamp(*tf.Backup)
		if err != nil {
			return
		}
	}
	if tf.Expiration != nil {
		writeTimestamp(*tf.Expiration)
		if err != nil {
			return
		}
	}
	if tf.Effective != nil {
		writeTimestamp(*tf.Effective)
		if err != nil {
			return
		}
	}

	if int64(expectedLength) != n {
		panic("never")
	}
	return
}

func DecodeTimestamps(entries []susp.SystemUseEntry) (*Timestamps, bool) {
	for _, entry := range entries {
		switch e := entry.(type) {
		case *Timestamps:
			return e, true
		}
	}
	return nil, false
}
