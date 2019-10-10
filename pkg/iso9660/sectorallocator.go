// Copyright © 2018 NVIDIA Corporation

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
