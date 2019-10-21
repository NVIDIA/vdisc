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
	"os"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/safecast"
)

// GetInodeAttribute returns the attribute of an inode and
// attribute expiration time
func (fs *isoFS) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	fs.finfosMU.RLock()
	entry, ok := fs.finfos[op.Inode]
	if !ok {
		fs.finfosMU.RUnlock()
		fs.logger.Error("get inode attributes", zap.Uint64("ino", uint64(op.Inode)), zap.Error(errUnknownInode))
		return fuse.EINVAL
	}
	fs.finfosMU.RUnlock()

	op.Attributes = fuseops.InodeAttributes{
		Size:  safecast.Int64ToUint64(entry.Info.Size()),
		Nlink: entry.Info.Nlink(),
		Mode:  entry.Info.Mode(),
		Ctime: entry.Info.ModTime(),
		Mtime: entry.Info.ModTime(),
		Uid:   entry.Info.Uid(),
		Gid:   entry.Info.Gid(),
	}
	op.AttributesExpiration = time.Now().Add(1 * time.Minute)
	return nil
}

// LookUpInode looks up a child by name within a parent directory.
// Returns ChildInodeEntry which contains information about a child
// inode within its parent directory. Kernel sends this when resolving
// user paths to dentry structs and setup a dcache entry.
func (fs *isoFS) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	set := func(child *iso9660.FileInfo) {
		childIno := fuseops.InodeID(child.Ino())
		fs.finfosMU.Lock()
		defer fs.finfosMU.Unlock()
		centry, ok := fs.finfos[childIno]
		if !ok {
			centry = &finfosEntry{
				Info: child,
			}
			fs.finfos[childIno] = centry
		}
		centry.RefCnt++
		op.Entry = fuseops.ChildInodeEntry{
			Child: childIno,
			Attributes: fuseops.InodeAttributes{
				Size:  safecast.Int64ToUint64(child.Size()),
				Nlink: child.Nlink(),
				Mode:  child.Mode(),
				Ctime: child.ModTime(),
				Mtime: child.ModTime(),
				Uid:   child.Uid(),
				Gid:   child.Gid(),
			},
			AttributesExpiration: time.Now().Add(1 * time.Minute),
			EntryExpiration:      time.Now().Add(1 * time.Hour),
		}
	}

	cachedFi, ok := fs.finfoCache.Get(op.Parent, op.Name)
	if ok {
		set(cachedFi)
		return nil
	}

	fs.finfosMU.RLock()
	entry, ok := fs.finfos[op.Parent]
	if !ok {
		fs.finfosMU.RUnlock()
		fs.logger.Info("lookup inode", zap.Uint64("parent", uint64(op.Parent)), zap.String("name", op.Name), zap.Error(errUnknownInode))
		return fuse.EINVAL
	}
	fs.finfosMU.RUnlock()

	// Sequentially scan the directory looking for Name
	it := iso9660.NewReadDirIterator(fs.vdisc.Image(), entry.Info.Extent(), entry.Info.Size(), 0)
	for it.Next() {
		child, _ := it.FileInfoAndLen()
		fs.finfoCache.Put(op.Parent, child.Name(), child)
		if child.Name() == op.Name {
			set(child)
			return nil
		}
	}
	if err := it.Err(); err != nil {
		fs.logger.Error("lookup inode", zap.Uint64("parent", uint64(op.Parent)), zap.String("name", op.Name), zap.Error(err))
		return fuse.EIO
	}

	return fuse.ENOENT
}

// ForgetInode is called by FS to decrement inode reference count.
func (fs *isoFS) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	if op.Inode == 1 {
		return nil
	}

	fs.finfosMU.Lock()
	defer fs.finfosMU.Unlock()

	entry, ok := fs.finfos[op.Inode]
	if !ok {
		fs.logger.Error("forget inode", zap.Uint64("ino", uint64(op.Inode)), zap.Error(errUnknownInode))
		return fuse.EINVAL
	}

	if entry.RefCnt > op.N {
		entry.RefCnt = entry.RefCnt - op.N
	} else {
		delete(fs.finfos, op.Inode)
	}

	return nil
}

// OpenDir opens a Dir inode
func (fs *isoFS) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return nil
}

// ReleaseDirHandle is a nop added to avoid error logs. sent by kernel when there are
// no more references to dir handle and all file descriptor are closed.
func (fs isoFS) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	return nil
}

// ReadDir returns directory entries for a directory represetend by the inode
// Prior to calling this method inode/dir is opened by calling OpenDir ops.
func (fs *isoFS) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	fs.finfosMU.RLock()
	entry, ok := fs.finfos[op.Inode]
	if !ok {
		fs.finfosMU.RUnlock()
		fs.logger.Info("open dir", zap.Uint64("ino", uint64(op.Inode)), zap.Error(errUnknownInode))
		return fuse.EINVAL
	}
	fs.finfosMU.RUnlock()

	it := iso9660.NewReadDirIterator(fs.vdisc.Image(), entry.Info.Extent(), entry.Info.Size(), safecast.Uint64ToInt64(uint64(op.Offset)))

	off := op.Offset
	for it.Next() {
		fi, fiLen := it.FileInfoAndLen()
		fs.finfoCache.Put(op.Inode, fi.Name(), fi)

		var dEntryType fuseutil.DirentType
		if fi.IsDir() {
			dEntryType = fuseutil.DT_Directory
		} else if fi.Mode()&os.ModeSymlink != 0 {
			dEntryType = fuseutil.DT_Link
		} else {
			dEntryType = fuseutil.DT_File
		}

		eino := fi.Ino()
		l := fuseops.DirOffset(safecast.Int64ToUint64(fiLen))
		m := fuseutil.WriteDirent(op.Dst[op.BytesRead:], fuseutil.Dirent{
			Offset: off + l,
			Inode:  fuseops.InodeID(eino),
			Name:   fi.Name(),
			Type:   dEntryType,
		})
		if m == 0 {
			break
		}
		off += l
		op.BytesRead += m
	}

	if err := it.Err(); err != nil {
		fs.logger.Error("readdir", zap.Uint64("inode", uint64(op.Inode)), zap.Error(err))
		return fuse.EIO
	}

	return nil
}
