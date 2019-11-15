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

	"github.com/badgerodon/collections/queue"
	"github.com/emirpasic/gods/maps/treemap"

	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

type dirEntry struct {
	name  string
	child Inode
}

//DirectoryInode implements Inode interface for directories
type DirectoryInode struct {
	ino           InodeNumber
	perm          os.FileMode
	created       time.Time
	modified      time.Time
	parent        *DirectoryInode
	start         LogicalBlockAddress
	idAlloc       *IdentifierAllocator
	names         map[string]string // name to identifier
	children      *treemap.Map      // map[identifier]dirEntry
	size          uint32
	nameValidator NameValidator
}

//NewDirectoryInode returns a new *DirectoryInode
func NewDirectoryInode(ia InodeAllocator, validator NameValidator) (*DirectoryInode, error) {
	ino, err := ia.Next()
	if err != nil {
		return nil, err
	}

	return &DirectoryInode{
		ino:           ino,
		perm:          0555,
		created:       time.Unix(0, 0).UTC(),
		modified:      time.Unix(0, 0).UTC(),
		idAlloc:       NewIdentifierAllocator(),
		names:         make(map[string]string),
		children:      treemap.NewWithStringComparator(),
		nameValidator: validator,
	}, nil
}

//Type return InodeTypeDirectory
func (d *DirectoryInode) Type() InodeType {
	return InodeTypeDirectory
}

//InodeNumber returns the inode number
func (d *DirectoryInode) InodeNumber() InodeNumber {
	return d.ino
}

// Perm returns the Unix permissions for this inode
func (d *DirectoryInode) Perm() os.FileMode {
	return d.perm
}

// SetPerm sets the Unix permissions for this inode
func (d *DirectoryInode) SetPerm(perm os.FileMode) {
	d.perm = perm & os.ModePerm
}

//Created returns the creation time
func (d *DirectoryInode) Created() time.Time {
	return d.created
}

//SetCreated sets the creation time
func (d *DirectoryInode) SetCreated(created time.Time) {
	d.created = created
}

//Modified returns the modified time
func (d *DirectoryInode) Modified() time.Time {
	return d.modified
}

//SetModified returns the modified time
func (d *DirectoryInode) SetModified(modified time.Time) {
	d.modified = modified
}

//AddChild adds a child Inode to the DirectoryInode.
//It validates the name and checks for collisions before adding the child
func (d *DirectoryInode) AddChild(name string, child Inode) error {
	if len(name) < 1 {
		return errors.New("Inode names may not be empty")
	}

	if err := d.nameValidator.IsValid(name); err != nil {
		return errors.New("AddChild: " + err.Error())
	}

	if name == "." || name == ".." {
		return errors.New("Inode names '.' and '..' are reserved")
	}

	if _, ok := d.names[name]; ok {
		return errors.New("Directory entry collision")
	}

	ident := d.idAlloc.Next()
	if child.Type() != InodeTypeDirectory {
		ident += ";1"
	}
	d.names[name] = ident
	d.children.Put(ident, &dirEntry{name, child})
	child.AddParent(d)
	return nil
}

//GetChild returns a child Inode based on the name. If no child is found, it returns (nil, false)
func (d *DirectoryInode) GetChild(name string) (Inode, bool) {
	ident, ok := d.names[name]
	if ok {
		dent, ok := d.children.Get(ident)
		if ok {
			return dent.(*dirEntry).child, true
		}
	}
	return nil, false
}

//AddParent adds a parent of the DirectoryInode
func (d *DirectoryInode) AddParent(parent *DirectoryInode) {
	if d.parent != nil {
		panic("Directories can only have one parent")
	}
	if d != parent {
		d.parent = parent
	}
}

//Nlink returns the number of links from the DirectoryInode
func (d *DirectoryInode) Nlink() uint32 {
	nlink := uint32(2)
	it := d.children.Iterator()
	for it.Next() {
		dent := it.Value().(*dirEntry)
		if dent.child.Type() == InodeTypeDirectory {
			nlink++
		}
	}
	return nlink
}

//IsRoot returns true if the DirectoryInode has no parent
func (d *DirectoryInode) IsRoot() bool {
	return d.parent == nil
}

//SetSize sets the size
func (d *DirectoryInode) SetSize(size uint32) {
	d.size = size
}

//Parts returns the inode parts
func (d *DirectoryInode) Parts() []InodePart {
	return InodeParts(d.start, uint64(d.size))
}

//AdditionalSystemUseEntries is a noop
func (d *DirectoryInode) AdditionalSystemUseEntries() ([]susp.SystemUseEntry, error) {
	return nil, nil
}

//Start returns the starting address
func (d *DirectoryInode) Start() LogicalBlockAddress {
	return d.start
}

//SetStart sets the starting address
func (d *DirectoryInode) SetStart(start LogicalBlockAddress) {
	d.start = start
}

//VisitDirectories walks the directory structure, visiting only directories, in level order.
func (d *DirectoryInode) VisitDirectories(visit Visitor) error {
	q := queue.New()
	q.Enqueue(Relationship{
		Parent:     d,
		Identifier: "\x00",
		Name:       ".",
		Child:      d,
	})

	for q.Len() > 0 {
		rel := q.Dequeue().(Relationship)
		if err := visit(rel); err != nil {
			return err
		}

		parent := rel.Child.(*DirectoryInode)
		it := parent.children.Iterator()
		for it.Next() {
			ident := it.Key().(string)
			dent := it.Value().(*dirEntry)
			if dent.child.Type() == InodeTypeDirectory {
				q.Enqueue(Relationship{
					Parent:     parent,
					Identifier: ident,
					Name:       dent.name,
					Child:      dent.child,
				})
			}
		}
	}

	return nil
}

//VisitFiles walks the directory structure, visiting only files, in level order.
func (d *DirectoryInode) VisitFiles(visit Visitor) error {
	q := queue.New()
	q.Enqueue(d)

	for q.Len() > 0 {
		parent := q.Dequeue().(*DirectoryInode)
		it := parent.children.Iterator()
		for it.Next() {
			ident := it.Key().(string)
			dent := it.Value().(*dirEntry)
			switch dent.child.Type() {
			case InodeTypeDirectory:
				q.Enqueue(dent.child)
			case InodeTypeFile:
				visit(Relationship{
					Parent:     parent,
					Identifier: ident,
					Name:       dent.name,
					Child:      dent.child,
				})
			}
		}
	}

	return nil
}

//ToDirectory returns the Directory and ContinurationArea of the inode
func (d *DirectoryInode) ToDirectory() (Directory, ContinuationArea, error) {
	parts := d.Parts()
	sectors := bytesToSectors(parts[0].Size())
	return d.toDirectory(d.start + LogicalBlockAddress(sectors))
}

func (d *DirectoryInode) toDirectory(contStart LogicalBlockAddress) (Directory, ContinuationArea, error) {
	var dir Directory
	cont := NewContinuationArea(contStart)

	splitInode := func(identifier string, name string, inode Inode) error {
		systemUseEntries, err := InodeSystemUseEntries(identifier, name, inode)
		if err != nil {
			return err
		}

		parts := inode.Parts()
		for i, part := range parts {
			record := &DirectoryRecord{
				Identifier: identifier,
				Start:      part.Start(),
				Length:     part.Size(),
				Recorded:   inode.Modified(),
				SystemUse:  systemUseEntries,
				VolumeID:   1,
			}

			if inode.Type() == InodeTypeDirectory {
				record.Flags |= FileFlagDir
			}

			if i < len(parts)-1 {
				record.Flags |= FileFlagNonTerminal
			}

			// We pack as many system use entries directly into the
			// record as possible, keeping the record less than one
			// sector.
			var overflow []susp.SystemUseEntry
			baseLen := record.Len()
			var extraLen int
			for (baseLen + extraLen) > MaxDirectoryRecordLen {
				// pop off the last system use entry and move it to the continuation area
				lastIdx := len(record.SystemUse) - 1
				last := record.SystemUse[lastIdx]
				record.SystemUse = record.SystemUse[:lastIdx]
				overflow = append([]susp.SystemUseEntry{last}, overflow...)
				baseLen -= last.Len()
				extraLen = susp.ContinuationAreaEntryLength
			}

			if len(overflow) > 0 {
				ce := cont.Append(overflow)
				record.SystemUse = append(record.SystemUse, ce)
			}

			dir.Records = append(dir.Records, *record)
		}
		return nil
	}

	parent := d.parent
	if parent == nil {
		parent = d
	}
	err := splitInode("\x00", ".", d)
	if err != nil {
		return dir, cont, err
	}
	err = splitInode("\x01", "..", parent)
	if err != nil {
		return dir, cont, err
	}

	// children
	it := d.children.Iterator()
	for it.Next() {
		ident := it.Key().(string)
		dent := it.Value().(*dirEntry)

		err := splitInode(ident, dent.name, dent.child)
		if err != nil {
			return dir, cont, err
		}
	}

	return dir, cont, nil
}

//ToPathTable computes the PathTable for the inode
func (d *DirectoryInode) ToPathTable() PathTable {
	var idx uint16
	indices := make(map[Inode]uint16)
	var table PathTable

	d.VisitDirectories(func(rel Relationship) error {
		indices[rel.Child] = idx
		idx++

		parentIdx, ok := indices[rel.Parent]
		if !ok {
			panic("never")
		}

		table.Records = append(table.Records, PathTableRecord{
			Identifier:                    rel.Identifier,
			ExtendedAttributeRecordLength: 0,
			Location:                      rel.Child.(*DirectoryInode).Start(),
			ParentIndex:                   parentIdx,
		})

		return nil
	})

	return table
}
