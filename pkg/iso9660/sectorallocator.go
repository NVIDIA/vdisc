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

const (
	LogicalBlockSize = 2048
)

type LogicalBlockAddress uint32

type SectorAllocator struct {
	next LogicalBlockAddress
}

func NewSectorAllocator() *SectorAllocator {
	return &SectorAllocator{0}
}

// Returns the start of the block and reserves enough sectors to store numBytes, at least one.
func (sa *SectorAllocator) Alloc(numBytes uint32) LogicalBlockAddress {
	result := sa.next
	sa.next += LogicalBlockAddress(bytesToSectors(numBytes))
	return result
}

// Returns the total number of sectors allocated
func (sa *SectorAllocator) Allocated() uint32 {
	return uint32(sa.next)
}
