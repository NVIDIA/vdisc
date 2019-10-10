// Copyright © 2018 NVIDIA Corporation

package iso9660

import (
	"io"
	"os"
	"time"

	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

// A posix symbolic link
type FileInode struct {
	ino      InodeNumber
	perm     os.FileMode
	created  time.Time
	modified time.Time
	nlink    uint32
	start    LogicalBlockAddress
	o        storage.Object
}

func NewFileInode(ia InodeAllocator, o storage.Object) (*FileInode, error) {
	ino, err := ia.Next()
	if err != nil {
		return nil, err
	}

	return &FileInode{
		ino:      ino,
		perm:     0444,
		created:  time.Unix(0, 0).UTC(),
		modified: time.Unix(0, 0).UTC(),
		o:        o,
	}, nil
}

func (f *FileInode) Type() InodeType {
	return InodeTypeFile
}

func (f *FileInode) InodeNumber() InodeNumber {
	return f.ino
}

// Perm returns the Unix permissions for this inode
func (f *FileInode) Perm() os.FileMode {
	return f.perm
}

// SetPerm sets the Unix permissions for this inode
func (f *FileInode) SetPerm(perm os.FileMode) {
	f.perm = perm & os.ModePerm
}

func (f *FileInode) Created() time.Time {
	return f.created
}

func (f *FileInode) SetCreated(created time.Time) {
	f.created = created
}

func (f *FileInode) Modified() time.Time {
	return f.modified
}

func (f *FileInode) SetModified(modified time.Time) {
	f.modified = modified
}

func (f *FileInode) AddParent(parent *DirectoryInode) {
	f.nlink += 1
}

func (f *FileInode) Nlink() uint32 {
	return f.nlink
}

func (f *FileInode) IsRoot() bool {
	return false
}

func (f *FileInode) Parts() []InodePart {
	return InodeParts(f.start, uint64(f.o.Size()))
}

func (f *FileInode) AdditionalSystemUseEntries() ([]susp.SystemUseEntry, error) {
	return nil, nil
}

func (f *FileInode) Start() LogicalBlockAddress {
	return f.start
}

func (f *FileInode) SetStart(start LogicalBlockAddress) {
	f.start = start
}

func (f *FileInode) WriteTo(w io.Writer) (n int64, err error) {
	sr := io.NewSectionReader(f.o, 0, f.o.Size())
	return io.Copy(w, sr)
}

func (f *FileInode) Object() storage.Object {
	return f.o
}
