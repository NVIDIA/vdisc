package iso9660

import (
	"os"
	"time"
)

type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	nlink   uint32
	uid     uint32
	gid     uint32
	ino     uint32
	modTime time.Time
	isDir   bool
	extent  LogicalBlockAddress
	target  string
}

func (fi *FileInfo) Name() string {
	return fi.name
}

func (fi *FileInfo) Size() int64 {
	return fi.size
}

func (fi *FileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *FileInfo) IsDir() bool {
	return fi.isDir
}

func (fi *FileInfo) Sys() interface{} {
	return fi
}

func (fi *FileInfo) Extent() LogicalBlockAddress {
	return fi.extent
}

func (fi *FileInfo) Target() string {
	return fi.target
}

func (fi *FileInfo) Nlink() uint32 {
	return fi.nlink
}

func (fi *FileInfo) Uid() uint32 {
	return fi.uid
}

func (fi *FileInfo) Gid() uint32 {
	return fi.gid
}

func (fi *FileInfo) Ino() uint32 {
	return fi.ino
}
