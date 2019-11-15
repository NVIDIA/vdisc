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
	"encoding/base32"
	"encoding/binary"
)

var (
	shortEncoding = base32.StdEncoding.WithPadding('_')
)

// Assign unique (hidden) names to directory entries
type IdentifierAllocator struct {
	count uint64
}

func NewIdentifierAllocator() *IdentifierAllocator {
	return &IdentifierAllocator{0}
}

func (ia *IdentifierAllocator) Next() string {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, ia.count)
	ia.count++
	src := buf[:n]

	dst := make([]byte, shortEncoding.EncodedLen(len(src)))
	shortEncoding.Encode(dst, src)
	return string(dst)
}
