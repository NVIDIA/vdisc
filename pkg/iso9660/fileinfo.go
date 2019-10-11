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
