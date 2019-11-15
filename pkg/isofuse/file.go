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
	"context"
	"io"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"go.uber.org/zap"
)

// OpenFile is isoFS openFile ops called in response to a user space file open.
// This method setups a file resource indicated by the inode for a subesquent
// call to ReadFile ops.
func (fs *isoFS) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	fs.finfosMU.RLock()
	entry, ok := fs.finfos[op.Inode]
	if !ok {
		fs.finfosMU.RUnlock()
		fs.logger.Info("open file", zap.Uint64("ino", uint64(op.Inode)), zap.Error(errUnknownInode))
		return fuse.EINVAL
	}
	fs.finfosMU.RUnlock()

	obj, err := fs.volume.OpenExtent(entry.Info.Extent())
	if err != nil {
		fs.logger.Error("open extent", zap.Error(err))
		return fuse.EINVAL
	}

	fs.fileHandlesMU.Lock()
	defer fs.fileHandlesMU.Unlock()
	fs.fileHandles[fs.nextFileHandle] = obj
	op.Handle = fs.nextFileHandle
	op.KeepPageCache = true
	fs.nextFileHandle++
	return nil
}

// ReleaseFileHandle releses a file handle which is setup during OpenFile ops.
// This method this called on the last close of the file.
func (fs *isoFS) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	fs.fileHandlesMU.Lock()
	defer fs.fileHandlesMU.Unlock()

	obj, ok := fs.fileHandles[op.Handle]
	if !ok {
		fs.logger.Warn("release of unknown file handle", zap.Uint64("handle", uint64(op.Handle)))
	} else {
		delete(fs.fileHandles, op.Handle)
	}
	if err := obj.Close(); err != nil {
		fs.logger.Error("close file handle", zap.Uint64("handle", uint64(op.Handle)), zap.Error(err))
	}
	return nil
}

// ReadFile is isoFS read ops. It reads data from a file previously
// opened using OpenFile ops. The inode of the file to be read is
// provided in op.Inode
func (fs *isoFS) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) (err error) {
	fs.fileHandlesMU.RLock()
	obj, ok := fs.fileHandles[op.Handle]
	if !ok {
		fs.fileHandlesMU.RUnlock()
		fs.logger.Warn("read from unknown file handle", zap.Uint64("handle", uint64(op.Handle)))
		return fuse.EINVAL
	}
	fs.fileHandlesMU.RUnlock()

	op.BytesRead, err = obj.ReadAt(op.Dst, op.Offset)
	if err != nil && err != io.EOF {
		fs.logger.Error("read", zap.Uint64("inode", uint64(op.Inode)), zap.Error(err))
		err = fuse.EIO
		return
	}

	if err == io.EOF {
		err = nil
	}

	return
}

// FlushFile is not implemented. It is not required for isoFS which is
// ReadOnly FS
func (fs *isoFS) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) (err error) {
	// ReadOnly FS Flush is not relevent
	return
}
