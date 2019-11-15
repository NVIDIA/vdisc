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
	"errors"
	"io"
	"sync"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/safecast"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

// Config is used to configure a Server
type Options struct {
	AllowOtherUsers bool `help:"Allow other users to access the fuse mount"`
}

type Volume interface {
	Image() storage.AnonymousObject
	OpenExtent(lba iso9660.LogicalBlockAddress) (storage.Object, error)
}

// NewServer creates an instance of an isofuse server
func NewWithOptions(mountpoint string, volume Volume, opts Options) (*Server, error) {
	var pvd iso9660.PrimaryVolumeDescriptor
	pvdSector := io.NewSectionReader(volume.Image(), 16*iso9660.LogicalBlockSize, iso9660.LogicalBlockSize)
	if err := iso9660.DecodePrimaryVolumeDescriptor(pvdSector, &pvd); err != nil {
		return nil, err
	}

	l := zap.L().Named("isofuse")
	fs, err := newIsoFS(l, volume, &pvd)
	if err != nil {
		return nil, err
	}

	return &Server{
		name:        pvd.VolumeIdentifier,
		mountpoint:  mountpoint,
		logger:      l,
		allowOthers: opts.AllowOtherUsers,
		fs:          fs,
		joined:      make(chan interface{}),
		err:         make(chan error),
	}, nil
}

type Server struct {
	name        string      // Name of the mounted volume (OS X only)
	mountpoint  string      // Mount dir
	logger      *zap.Logger // Logger used for logging
	allowOthers bool        // security override, allow all users to access the files
	fs          *isoFS
	joined      chan interface{}
	err         chan error // Error channel between serve thread and close
}

// Start mounts and starts an isofuse server
func (s *Server) Start() error {
	fserver := fuseutil.NewFileSystemServer(s.fs)

	elog, err := zap.NewStdLogAt(s.logger, zapcore.ErrorLevel)
	if err != nil {
		return err
	}

	dlog, err := zap.NewStdLogAt(s.logger, zapcore.DebugLevel)
	if err != nil {
		return err
	}

	cfg := &fuse.MountConfig{
		FSName:      "isofuse",
		ReadOnly:    true,
		ErrorLogger: elog,
		DebugLogger: dlog,
		VolumeName:  s.name,
	}
	if s.allowOthers == true {
		// user_allow_other must be set in /etc/fuse.conf
		cfg.Options = map[string]string{"allow_other": ""}
	}
	mfs, err := fuse.Mount(s.mountpoint, fserver, cfg)
	if err != nil {
		return err
	}
	s.fs.mfs = mfs
	go s.serve()
	go s.join()

	return nil

}

// Close unmounts isofuse and stops the server
func (s *Server) Close() error {
	if s.fs.mfs == nil {
		// in case close is called even if start failed
		s.logger.Info("nil mountedfs", zap.String("mountpoint", s.mountpoint))
		return nil
	}
	s.logger.Info("Unmount", zap.String("mountpoint", s.fs.mfs.Dir()))
	err := fuse.Unmount(s.fs.mfs.Dir())
	if err != nil {
		s.logger.Info("unmount", zap.Error(err))
		return err
	}
	return <-s.err
}

func (s *Server) Join() chan interface{} {
	return s.joined
}

func (s *Server) serve() {
	s.logger.Info("Started")

	// Wait for it to be unmounted.
	err := s.fs.mfs.Join(context.Background())
	if err != nil {
		s.logger.Info("Join: ", zap.Error(err))
	}
	s.err <- err
	s.logger.Info("Done.")

	return
}

func (s *Server) join() {
	s.fs.mfs.Join(context.Background())
	close(s.joined)
}


var errUnknownInode = errors.New("unknown inode")

func newIsoFS(logger *zap.Logger, v Volume, pvd *iso9660.PrimaryVolumeDescriptor) (*isoFS, error) {
	c, err := NewFileInfoCache(100000)
	if err != nil {
		return nil, err
	}
	fs := &isoFS{
		logger:         logger,
		volume:         v,
		finfos:         make(map[fuseops.InodeID]*finfosEntry),
		finfoCache:     c,
		nextFileHandle: 1,
		fileHandles:    make(map[fuseops.HandleID]storage.Object),
	}

	// prime the root directory inode info
	it := iso9660.NewReadDirIterator(v.Image(), pvd.RootStart, int64(pvd.RootLength), 0)
	if !it.Next() {
		return nil, errors.New("bad iso9660 root directory")
	}
	root, _ := it.FileInfoAndLen()
	fs.finfos[1] = &finfosEntry{
		Info: root,
	}
	return fs, nil
}

type isoFS struct {
	fuseutil.NotImplementedFileSystem // To support default implementation of ops not implemented by this FS

	logger *zap.Logger

	volume Volume

	mfs *fuse.MountedFileSystem

	finfosMU sync.RWMutex
	finfos   map[fuseops.InodeID]*finfosEntry

	finfoCache FileInfoCache

	fileHandlesMU  sync.RWMutex
	nextFileHandle fuseops.HandleID
	fileHandles    map[fuseops.HandleID]storage.Object
}

// StartFS returns information about file system capacity and resources
func (fs *isoFS) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	op.BlockSize = uint32(iso9660.LogicalBlockSize)
	op.Blocks = safecast.Int64ToUint64(fs.volume.Image().Size()) / uint64(iso9660.LogicalBlockSize)
	op.IoSize = 4194304
	return nil
}

// ReadSymlink returns the target of a symlink inode
func (fs *isoFS) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	fs.finfosMU.RLock()
	entry, ok := fs.finfos[op.Inode]
	if !ok {
		fs.finfosMU.RUnlock()
		fs.logger.Info("read symlink", zap.Uint64("ino", uint64(op.Inode)), zap.Error(errUnknownInode))
		return fuse.EINVAL
	}
	fs.finfosMU.RUnlock()
	op.Target = entry.Info.Target()
	return nil
}

func inoToExtent(ino fuseops.InodeID) iso9660.LogicalBlockAddress {
	return iso9660.LogicalBlockAddress(safecast.Uint64ToUint32(uint64(ino)))
}

type finfosEntry struct {
	RefCnt uint64
	Info   *iso9660.FileInfo
}
