package iso9660

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

type File struct {
	name string
	fi   *FileInfo
	r    *io.SectionReader
	pos  int64
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if f.r == nil {
		return 0, os.ErrClosed
	}

	if f.fi.IsDir() {
		return 0, fmt.Errorf("read %s: %+v", f.name, syscall.EISDIR)
	}

	return f.r.ReadAt(p, off)
}

func (f *File) Read(p []byte) (n int, err error) {
	if f.r == nil {
		return 0, os.ErrClosed
	}

	if f.fi.IsDir() {
		return 0, fmt.Errorf("read %s: %+v", f.name, syscall.EISDIR)
	}

	return f.r.Read(p)
}

func (f *File) ReadDir() ([]*FileInfo, error) {
	if f.r == nil {
		return nil, os.ErrClosed
	}

	if !f.fi.IsDir() {
		return nil, fmt.Errorf("readdir %s: %+v", f.name, syscall.ENOTDIR)
	}

	var entries []*FileInfo
	err := iterDir(f.r, 0, f.r.Size(), func(fi *FileInfo) bool {
		entries = append(entries, fi)
		return true
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.r == nil {
		return 0, os.ErrClosed
	}

	if f.fi.IsDir() {
		return 0, fmt.Errorf("seek %s: %+v", f.name, syscall.EISDIR)
	}

	return f.r.Seek(offset, whence)
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Size() int64 {
	return f.fi.Size()
}

func (f *File) Close() error {
	f.r = nil
	return nil
}
