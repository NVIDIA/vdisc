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
	"syscall"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type Options struct {
	AllowOtherUsers bool `help:"Allow other users to access this fuse mount"`
}

// FS implements the hello world file system.
type FS struct {
	mountpoint string
	vdisc      vdisc.VDisc
	w          *iso9660.Walker
	conn       *fuse.Conn // Hook to the FUSE connection object
	options    Options
}

func New(mountpoint string, v vdisc.VDisc) (*FS, error) {
	return NewWithOptions(mountpoint, v, Options{})
}

func NewWithOptions(mountpoint string, v vdisc.VDisc, options Options) (*FS, error) {
	return &FS{
		mountpoint: mountpoint,
		vdisc:      v,
		w:          iso9660.NewWalker(v.Image()),
		options:    options,
	}, nil
}

// Run the file system, mounting the MountPoint and connecting to FUSE
func (fs *FS) Run() error {
	var err error

	// Unmount the FS in case it was mounted with errors.
	fuse.Unmount(fs.mountpoint)

	// Create the mount options to pass to Mount.
	opts := []fuse.MountOption{
		fuse.FSName("ISO9660"),
		fuse.Subtype("vdisc"),
		fuse.VolumeName("MYVDISC"),
		fuse.DefaultPermissions(),
		fuse.MaxReadahead(64 * 1024 * 1024),
		fuse.ReadOnly(),
	}

	if fs.options.AllowOtherUsers {
		opts = append(opts, fuse.AllowOther())
	}

	// Mount the FS with the specified options
	if fs.conn, err = fuse.Mount(fs.mountpoint, opts...); err != nil {
		return err
	}

	// Ensure that the file system is shutdown
	defer fs.conn.Close()
	zap.L().Info("mounted iso", zap.String("mountpoint", fs.mountpoint))

	// Serve the file system
	if err = fusefs.Serve(fs.conn, fs); err != nil {
		return err
	}

	zap.L().Info("post serve")

	// Check if the mount process has an error to report
	<-fs.conn.Ready
	if fs.conn.MountError != nil {
		return fs.conn.MountError
	}

	return nil
}

func (fs *FS) Shutdown() error {
	zap.L().Info("shutting the file system down gracefully")

	if fs.conn == nil {
		return nil
	}

	if err := fuse.Unmount(fs.mountpoint); err != nil {
		return err
	}

	return nil
}

func (fs *FS) Close() error {
	return fs.Shutdown()
}

func (fs *FS) Root() (fusefs.Node, error) {
	finfo, err := fs.w.Lstat("")
	if err != nil {
		switch err {
		case syscall.ENOENT:
			return nil, fuse.ENOENT
		case syscall.ENOTDIR:
			return nil, fuse.Errno(syscall.ENOTDIR)
		default:
			zap.L().Error("lstat file system root", zap.Error(err))
			return nil, fuse.EIO
		}
	}

	fi := finfo.Sys().(*iso9660.FileInfo)

	return &Dir{
		vdisc: fs.vdisc,
		w:     fs.w,
		path:  "/",
		fi:    fi,
	}, nil
}
