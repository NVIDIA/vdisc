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
	"encoding/binary"
	"io"
)

// System Usage Sharing Protocol (SUSP, IEEE P1281)
// See ftp://ftp.ymi.com/pub/rockridge/susp112.ps

type SystemUseEntry interface {
	io.WriterTo
	Len() int
}

func writeByte(w io.Writer, b byte) error {
	n, err := w.Write([]byte{b})
	if err != nil {
		return err
	}
	if n != 1 {
		return io.ErrShortWrite
	}
	return nil
}

func putBothUint32(w io.Writer, v uint32) error {
	if err := binary.Write(w, binary.LittleEndian, v); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return err
	}

	return nil
}
