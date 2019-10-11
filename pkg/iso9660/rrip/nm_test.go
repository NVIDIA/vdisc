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
package rrip_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660/rrip"
)

func TestName(t *testing.T) {
	expected := strings.Repeat("x", 300)

	entries, err := rrip.NewName(expected)
	if err != nil {
		t.Fatal(err)
	}

	actual, ok := rrip.DecodeName(entries)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	actual, ok = rrip.DecodeName(nil)
	assert.False(t, ok)
	assert.Equal(t, "", actual)
}
