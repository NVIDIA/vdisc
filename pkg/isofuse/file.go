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
package isofuse

import (
	"io"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

// File implements both Node and Handle for the hello file.
type File struct {
	fi  *iso9660.FileInfo
	obj storage.Object
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(f.fi.Extent())
	a.Mode = f.fi.Mode()
	a.Size = uint64(f.fi.Size())
	a.Blocks = uint64((f.fi.Size() + 2047) / 2048)
	a.Ctime = f.fi.ModTime()
	a.Mtime = f.fi.ModTime()
	return nil
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	r := io.NewSectionReader(f.obj, req.Offset, int64(req.Size))
	resp.Data = make([]byte, req.Size)
	n, err := r.Read(resp.Data)
	if err != nil && err != io.EOF {
		zap.L().Error("read", zap.Error(err))
		return fuse.EIO
	}
	resp.Data = resp.Data[:n]

	return nil
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	resp.Flags |= fuse.OpenKeepCache
	return f, nil
}
