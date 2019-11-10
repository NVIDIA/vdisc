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
)

type Terminator struct{}

func (t *Terminator) WriteTo(w io.Writer) (int64, error) {
	cw := newCountingWriter(w)

	// Type Code (255 indicates Terminator)
	if err := writeByte(cw, 255); err != nil {
		return cw.Written(), err
	}

	// Standard Identifier
	if _, err := io.WriteString(cw, "CD001"); err != nil {
		return cw.Written(), err
	}

	// Version (always 1)
	if err := writeByte(cw, 1); err != nil {
		return cw.Written(), err
	}

	// Terminator Data (empty)
	if err := pad(cw, 2041); err != nil {
		return cw.Written(), err
	}
	return cw.Written(), nil
}
