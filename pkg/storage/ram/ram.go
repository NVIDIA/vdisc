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
	"bytes"
	"context"
	"fmt"
	stdurl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

func NewDriver() *Driver {
	d := &Driver{
		nextIno: 2,
		inodes:  make(map[int64]*inode),
	}
	d.inodes[rootIno] = newDirInode(rootIno, rootIno)
	return d
}

// Driver is the ram URI scheme storage driver.
type Driver struct {
	mu      sync.Mutex
	nextIno int64
	inodes  map[int64]*inode
}

func (d *Driver) Name() string {
	return "ramdriver"
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (driver.Object, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	f, err := d.traverse("open", path)
	if err != nil {
		return nil, err
	}

	it := d.inodes[f.Ino]
	if it.Mode.IsDir() {
		return nil, os.NewSyscallError("open", syscall.EISDIR)
	}

	data := it.File
	if size > -1 && size < int64(len(data)) {
		data = data[:size]
	}

	return &object{
		url: pathToUrl(path),
		br:  bytes.NewReader(it.File),
	}, nil
}

func (d *Driver) Create(ctx context.Context, url string) (driver.ObjectWriter, error) {
	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	segments := strings.Split(path, "/")
	if segments[0] == "" {
		segments = segments[1:]
	}

	if len(segments) < 1 {
		// can't create the root directory
		return nil, os.ErrExist
	}

	return &writer{
		url:    pathToUrl(path),
		path:   path,
		commit: d.commit,
	}, nil
}

func (d *Driver) commit(path string, data []byte) error {
	segments := strings.Split(path, "/")
	if segments[0] == "" {
		segments = segments[1:]
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	currIno := int64(rootIno)
	curr := d.inodes[currIno]
	for len(segments) > 1 {
		if !curr.Mode.IsDir() {
			return os.NewSyscallError("create", syscall.ENOTDIR)
		}

		name := segments[0]
		segments = segments[1:]
		ino, ok := curr.Dir.Lookup(name)
		if !ok {
			ino = d.nextIno
			d.nextIno++
			d.inodes[ino] = newDirInode(currIno, ino)
			curr.Dir.Set(name, ino)
		}
		currIno = ino
		curr = d.inodes[ino]
	}

	if !curr.Mode.IsDir() {
		return os.NewSyscallError("create", syscall.ENOTDIR)
	}

	name := segments[0]
	ino, ok := curr.Dir.Lookup(name)
	if ok {
		existing := d.inodes[ino]
		if existing.Mode.IsDir() {
			return os.ErrExist
		}
		existing.File = data
		return nil
	}

	ino = d.nextIno
	d.nextIno++
	d.inodes[ino] = newFileInode(data)
	curr.Dir.Set(name, ino)
	return nil
}

func (d *Driver) Remove(ctx context.Context, url string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	path, err := urlToPath(url)
	if err != nil {
		return err
	}

	f, err := d.traverse("remove", path)
	if err != nil {
		return err
	}

	f.Parent.Dir.Delete(f.Name)
	delete(d.inodes, f.Ino)
	return nil
}

func (d *Driver) Stat(ctx context.Context, url string) (os.FileInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	f, err := d.traverse("stat", path)
	if err != nil {
		return nil, err
	}

	inode := d.inodes[f.Ino]
	return &finfo{
		name: f.Name,
		size: int64(len(inode.File)),
		mode: inode.Mode,
	}, nil
}

func (d *Driver) Readdir(ctx context.Context, url string) ([]os.FileInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	f, err := d.traverse("readdir", path)
	if err != nil {
		return nil, err
	}

	it := d.inodes[f.Ino]
	if !it.Mode.IsDir() {
		return nil, os.NewSyscallError("readdir", syscall.ENOTDIR)
	}

	var result []os.FileInfo
	for name, ino := range it.Dir.Entries() {
		child := d.inodes[ino]
		result = append(result, &finfo{
			name: name,
			size: int64(len(child.File)),
			mode: child.Mode,
		})
	}
	return result, nil
}

type found struct {
	Parent *inode
	Name   string
	Ino    int64
}

func (d *Driver) traverse(op string, path string) (*found, error) {
	segments := strings.Split(path, "/")

	curr := d.inodes[rootIno]
	for len(segments) > 1 {
		if !curr.Mode.IsDir() {
			return nil, os.NewSyscallError(op, syscall.ENOTDIR)
		}

		name := segments[0]
		segments = segments[1:]
		ino, ok := curr.Dir.Lookup(name)
		if !ok {
			return nil, os.NewSyscallError(op, syscall.ENOENT)
		}
		curr = d.inodes[ino]
	}

	if !curr.Mode.IsDir() {
		return nil, os.NewSyscallError(op, syscall.ENOTDIR)
	}

	name := segments[0]
	ino, ok := curr.Dir.Lookup(name)
	if !ok {
		return nil, os.NewSyscallError(op, syscall.ENOENT)
	}

	return &found{
		Parent: curr,
		Name:   name,
		Ino:    ino,
	}, nil
}

func urlToPath(url string) (string, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return "", err
	}

	if u.Scheme != "ram" {
		return "", fmt.Errorf("ramdriver: unsupported URI scheme %q", u.Scheme)
	}

	var path string
	if len(u.Opaque) == 0 {
		path = u.Path
	} else {
		path = u.Opaque
	}

	path = filepath.Clean(path)
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	return filepath.Clean(path), nil
}

func pathToUrl(path string) string {
	var u stdurl.URL
	u.Scheme = "ram"
	u.Path = path
	return u.String()
}

func RegisterDefaultDriver() {
	driver.Register("ram", NewDriver())
}
