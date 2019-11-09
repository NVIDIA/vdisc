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
package ramdriver

import (
	"os"
)

const rootIno = 1

func newDirInode(parent, self int64) *inode {
	return &inode{
		Mode: os.ModeDir | 0755,
		Dir:  newDirectory(parent, self),
	}
}

func newFileInode(data []byte) *inode {
	return &inode{
		Mode: 0644,
		File: data,
	}
}

type inode struct {
	Mode os.FileMode
	Dir  *directory
	File []byte
}

func newDirectory(parent, self int64) *directory {
	d := &directory{make(map[string]int64)}
	d.entries[".."] = parent
	d.entries["."] = self
	return d
}

type directory struct {
	entries map[string]int64
}

func (d *directory) Set(name string, ino int64) {
	d.entries[name] = ino
}

func (d *directory) Delete(name string) {
	delete(d.entries, name)
}

func (d *directory) Lookup(name string) (int64, bool) {
	ino, ok := d.entries[name]
	return ino, ok
}

func (d *directory) Entries() map[string]int64 {
	return d.entries
}
