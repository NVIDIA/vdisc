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
package countio

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummy struct {
	nextSize int
	maxSize  int
}

func (r *dummy) Read(p []byte) (n int, err error) {
	return r.nextSize, nil
}

func (r *dummy) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekCurrent {
		return 0, nil
	} else if whence == io.SeekEnd {
		return int64(r.maxSize), nil
	}
	return 0, nil
}

func (r *dummy) ReadAt(p []byte, off int64) (n int, err error) {
	if int(off)+r.nextSize > r.maxSize {
		return r.maxSize - int(off), nil
	}
	return r.nextSize, nil
}

func TestReadSeeker(t *testing.T) {
	d := &dummy{}
	c := NewReaderAtSeeker(d)
	d.nextSize = 16
	d.maxSize = 4097
	r, err := c.Read(nil)
	assert.Nil(t, err)
	assert.EqualValues(t, 16, r)
	assert.EqualValues(t, 16, c.BytesRead())
	d.nextSize = 8
	r, err = c.Read(nil)
	assert.Nil(t, err)
	assert.EqualValues(t, 8, r)
	assert.EqualValues(t, 24, c.BytesRead())
	d.nextSize = 1024
	r, err = c.ReadAt(nil, 1000)
	assert.Nil(t, err)
	assert.EqualValues(t, 1024, r)
	assert.EqualValues(t, 1048, c.BytesRead())
	sp, err := c.Seek(0, io.SeekEnd)
	assert.EqualValues(t, 4097, sp)
	d.nextSize = 8
	r, err = c.Read(nil)
	assert.EqualValues(t, 8, r)
	assert.EqualValues(t, 1056, c.BytesRead())
}
