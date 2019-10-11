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
	"time"

	"github.com/NVIDIA/vdisc/pkg/iso9660/rrip"
	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

// A posix symbolic link
type SymlinkInode struct {
	ino      InodeNumber
	perm     os.FileMode
	created  time.Time
	modified time.Time
	parent   *DirectoryInode
	target   string
}

func NewSymlinkInode(ia InodeAllocator, target string) (*SymlinkInode, error) {
	ino, err := ia.Next()
	if err != nil {
		return nil, err
	}

	if len(target) < 1 {
		return nil, errors.New("symlink target cannot be empty")
	}

	// TODO: Validate characters in target
	return &SymlinkInode{
		ino:      ino,
		perm:     0777,
		created:  time.Unix(0, 0).UTC(),
		modified: time.Unix(0, 0).UTC(),
		target:   target,
	}, nil
}

func (s *SymlinkInode) Type() InodeType {
	return InodeTypeSymlink
}

func (s *SymlinkInode) InodeNumber() InodeNumber {
	return s.ino
}

// Perm returns the Unix permissions for this inode
func (s *SymlinkInode) Perm() os.FileMode {
	return s.perm
}

// SetPerm sets the Unix permissions for this inode
func (s *SymlinkInode) SetPerm(perm os.FileMode) {
	s.perm = perm & os.ModePerm
}

func (s *SymlinkInode) Created() time.Time {
	return s.created
}

func (s *SymlinkInode) SetCreated(created time.Time) {
	s.created = created
}

func (s *SymlinkInode) Modified() time.Time {
	return s.modified
}

func (s *SymlinkInode) SetModified(modified time.Time) {
	s.modified = modified
}

func (s *SymlinkInode) AddParent(parent *DirectoryInode) {
	if s.parent != nil {
		panic("Symlinks can only have one parent")
	}
	s.parent = parent
}

func (s *SymlinkInode) Nlink() uint32 {
	if s.parent == nil {
		return 0
	} else {
		return 1
	}
}

func (s *SymlinkInode) IsRoot() bool {
	return false
}

func (s *SymlinkInode) Parts() []InodePart {
	// Symlinks don't have any content
	return InodeParts(0, 0)
}

func (s *SymlinkInode) AdditionalSystemUseEntries() ([]susp.SystemUseEntry, error) {
	return rrip.NewSymlink(s.target)
}
