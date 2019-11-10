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
	"os"
	"syscall"
	"time"

	"github.com/NVIDIA/vdisc/pkg/iso9660/rrip"
	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

const (
	MaxPartSize = 4294965248 // (1 << 32) - 2048
)

type InodeType byte

const (
	InodeTypeFile InodeType = iota
	InodeTypeDirectory
	InodeTypeSymlink
)

var (
	ErrInodesExhausted = errors.New("inodes exhausted")
)

// A file, directory, or symlink
type Inode interface {
	// The type of inode
	Type() InodeType

	// The unique number assigned to this inode
	InodeNumber() InodeNumber

	// Each child consists of one or more parts of up to MaxPartSize bytes
	Parts() []InodePart

	// Perm returns the Unix permissions for this inode
	Perm() os.FileMode

	// SetPerm sets the Unix permissions for this inode
	SetPerm(perm os.FileMode)

	Created() time.Time

	SetCreated(time.Time)

	Modified() time.Time

	SetModified(time.Time)

	AddParent(*DirectoryInode)

	Nlink() uint32

	IsRoot() bool

	// Additional SUSP entries unique to this inode
	AdditionalSystemUseEntries() ([]susp.SystemUseEntry, error)
}

// A portion of a file or directory up to MaxPartSize bytes long
type InodePart interface {
	Start() LogicalBlockAddress
	Size() uint32
}

type part struct {
	start LogicalBlockAddress
	size  uint32
}

func (s *part) Start() LogicalBlockAddress {
	return s.start
}
func (s *part) Size() uint32 {
	return s.size
}

// Calculate the parts of an inode, each up to MaxPartSize size.
func InodeParts(start LogicalBlockAddress, size uint64) []InodePart {
	var parts []InodePart

	remaining := size
	for remaining >= 0 {
		var sz uint32
		if remaining > MaxPartSize {
			sz = MaxPartSize
		} else {
			sz = uint32(remaining)
		}

		parts = append(parts, &part{start, sz})
		remaining -= uint64(sz)

		if remaining > 0 {
			// calculate the logical block address of the next part
			lba := start + LogicalBlockAddress(bytesToSectors(sz))
			if lba <= start {
				panic("overflow")
			}
			start = lba
		} else {
			break
		}
	}

	return parts
}

type InodeNumber uint32

// A producer of inode numbers
type InodeAllocator interface {
	Next() (InodeNumber, error)
}

// Create a new inode number allocator
func NewInodeAllocator() InodeAllocator {
	// The root inode must be 1
	return &inodeAllocator{1, false}
}

type inodeAllocator struct {
	next      InodeNumber
	exhausted bool
}

func (ia *inodeAllocator) Next() (InodeNumber, error) {
	if ia.exhausted {
		return 0, ErrInodesExhausted
	}
	num := ia.next
	ia.next++
	ia.exhausted = (ia.next == 0)
	return num, nil
}

func InodeSystemUseEntries(identifier string, name string, inode Inode) ([]susp.SystemUseEntry, error) {
	var result []susp.SystemUseEntry

	if inode.IsRoot() && identifier == "\x00" {
		result = append(result, susp.NewSharingProtocolEntry(0))
	}
	if inode.Type() == InodeTypeDirectory {
		result = append(result, rrip.ExtensionsReferenceLegacy)
		result = append(result, rrip.ExtensionsReference)
	}

	nms, err := rrip.NewName(name)
	if err != nil {
		return nil, err
	}
	result = append(result, nms...)

	mode := inode.Perm()
	switch inode.Type() {
	case InodeTypeFile:
		mode |= syscall.S_IFREG
	case InodeTypeDirectory:
		mode |= syscall.S_IFDIR
	case InodeTypeSymlink:
		mode |= syscall.S_IFLNK
	}

	result = append(result,
		&rrip.PosixEntry{
			Mode:  mode,
			Nlink: inode.Nlink(),
			Uid:   0, // root
			Gid:   0, // root
			Ino:   uint32(inode.InodeNumber()),
		})

	// TF
	created := inode.Created()
	modified := inode.Modified()
	result = append(result,
		&rrip.Timestamps{
			Created:  &created,
			Modified: &modified,
		})

	additional, err := inode.AdditionalSystemUseEntries()
	if err != nil {
		return nil, err
	}
	result = append(result, additional...)

	return result, nil
}
