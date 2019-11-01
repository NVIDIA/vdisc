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
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/storage"
	_ "github.com/NVIDIA/vdisc/pkg/storage/zero"
)

func TestReadDir(t *testing.T) {
	v := iso9660.NewPosixPortableVolume()

	expected := make(map[string]int64)
	expected["."] = 129024
	expected[".."] = 129024

	for i := int64(0); i < 1000; i++ {
		name := fmt.Sprintf("file-%04d", i)
		url := fmt.Sprintf("zero:%d", i)
		obj, err := storage.Open(url)
		if err != nil {
			t.Fatal(err)
		}
		v.AddFile("/"+name, obj)
		expected[name] = i
	}

	isow := bytes.NewBuffer(nil)
	_, err := v.WriteMetadataTo(isow)
	if err != nil {
		t.Fatal(err)
	}
	iso := bytes.NewReader(isow.Bytes())

	var pvd iso9660.PrimaryVolumeDescriptor
	pvdSector := io.NewSectionReader(iso, 16*iso9660.LogicalBlockSize, iso9660.LogicalBlockSize)
	if err := iso9660.DecodePrimaryVolumeDescriptor(pvdSector, &pvd); err != nil {
		t.Fatal(err)
	}

	it := iso9660.NewReadDirIterator(iso, pvd.RootStart, int64(pvd.RootLength), 0)
	var resume int64
	for i := 0; i < 500; i++ {
		assert.True(t, it.Next(), "iterator exhausted")
		finfo, pos := it.FileInfoAndLen()
		assert.Equal(t, expected[finfo.Name()], finfo.Size())
		resume += pos
	}
	assert.Nil(t, it.Err())

	it = iso9660.NewReadDirIterator(iso, pvd.RootStart, int64(pvd.RootLength), resume)

	for i := 500; i < 1002; i++ {
		assert.True(t, it.Next(), "iterator exhausted")
		finfo, _ := it.FileInfoAndLen()
		assert.Equal(t, expected[finfo.Name()], finfo.Size())
	}

	assert.False(t, it.Next(), "iterator not exhausted")
	assert.Nil(t, it.Err())
}
