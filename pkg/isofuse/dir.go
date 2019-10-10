package isofuse

import (
	"path/filepath"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	vdisc vdisc.VDisc
	w     *iso9660.Walker
	path  string
	fi    *iso9660.FileInfo
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(d.fi.Extent())
	a.Mode = d.fi.Mode()
	a.Size = uint64(d.fi.Size())
	a.Blocks = uint64((d.fi.Size() + 2047) / 2048)
	a.Ctime = d.fi.ModTime()
	a.Mtime = d.fi.ModTime()
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	pth := filepath.Join(d.path, name)
	finfo, err := d.w.Lstat(pth)
	if err != nil {
		switch err {
		case syscall.ENOENT:
			return nil, fuse.ENOENT
		case syscall.ENOTDIR:
			return nil, fuse.Errno(syscall.ENOTDIR)
		default:
			zap.L().Error("lookup", zap.Error(err))
			return nil, fuse.EIO
		}
	}

	fi := finfo.Sys().(*iso9660.FileInfo)

	if fi.IsDir() {
		return &Dir{d.vdisc, d.w, pth, fi}, nil
	}

	obj, err := d.vdisc.OpenExtent(fi.Extent(), fi.Size())
	if err != nil {
		zap.L().Error("lookup", zap.Error(err))
		return nil, fuse.EIO
	}

	return &File{fi, obj}, nil
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var result []fuse.Dirent
	finfos, err := d.w.ReadDir(d.path)
	if err != nil {
		switch err {
		case syscall.ENOENT:
			return nil, fuse.ENOENT
		case syscall.ENOTDIR:
			return nil, fuse.Errno(syscall.ENOTDIR)
		default:
			zap.L().Error("readdir", zap.Error(err))
			return nil, fuse.EIO
		}
	}

	for _, finfo := range finfos {
		fi := finfo.Sys().(*iso9660.FileInfo)

		var typ fuse.DirentType
		if fi.IsDir() {
			typ = fuse.DT_Dir
		} else {
			typ = fuse.DT_File
		}

		result = append(result, fuse.Dirent{
			Inode: uint64(fi.Extent()),
			Name:  fi.Name(),
			Type:  typ,
		})
	}

	return result, nil
}

func (d *Dir) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	resp.Flags |= fuse.OpenKeepCache
	return d, nil
}
