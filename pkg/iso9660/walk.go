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
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/badgerodon/collections/queue"
	lru "github.com/hashicorp/golang-lru"

	"github.com/NVIDIA/vdisc/pkg/iso9660/rrip"
	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

// Walker provides an API for traversing a serialized ISO 9660 file
// system.
type Walker struct {
	iso        io.ReaderAt
	lstatCache *lru.ARCCache
}

func NewWalker(iso io.ReaderAt) *Walker {
	lstatCache, err := lru.NewARC(256 * 1024)
	if err != nil {
		panic(err)
	}
	return &Walker{
		iso:        iso,
		lstatCache: lstatCache,
	}
}

// Walk walks the file tree rooted at root, calling walkFn for each
// file or directory in the tree, including root. All errors that
// arise visiting files and directories are filtered by walkFn. The
// files are walked in lexical order, which makes the output
// deterministic but means that for very large directories Walk can be
// inefficient. Walk does not follow symbolic links.
func (w *Walker) Walk(root string, walkFn filepath.WalkFunc) (err error) {
	rootParts := w.pathParts(root)
	rootFi, rootErr := w.lstat(rootParts)
	if err = walkFn(root, rootFi, rootErr); err != nil {
		return
	}

	q := queue.New()

	if rootFi != nil && rootFi.IsDir() {
		q.Enqueue(&walkItem{
			parts: rootParts,
			fi:    rootFi,
		})
	}

	for q.Len() > 0 {
		item := q.Dequeue().(*walkItem)
		err = iterDir(w.iso, item.fi.Extent(), item.fi.Size(), func(fi *FileInfo) bool {
			path := "/" + strings.Join(item.parts, "/")
			if err = walkFn(path, fi, nil); err != nil {
				return false
			}

			if fi.IsDir() && fi.Name() != "." && fi.Name() != ".." {
				q.Enqueue(&walkItem{
					parts: append(item.parts, fi.Name()),
					fi:    fi,
				})
			}
			return true
		})
		if err != nil {
			return
		}
	}

	return
}

// Stat returns a FileInfo describing the named file.
func (w *Walker) Stat(path string) (*FileInfo, error) {
	parts := w.pathParts(path)
	return w.stat(parts)
}

// Lstat returns a FileInfo describing the named file. If the file is
// a symbolic link, the returned FileInfo describes the symbolic
// link. Lstat makes no attempt to follow the link.
func (w *Walker) Lstat(path string) (*FileInfo, error) {
	parts := w.pathParts(path)
	return w.lstat(parts)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
func (w *Walker) Open(name string) (*File, error) {
	parts := w.pathParts(name)
	fi, err := w.stat(parts) // follows symlinks
	if err != nil {
		return nil, fmt.Errorf("open %s: %+v", name, err)
	}

	return &File{
		name: name,
		fi:   fi,
		r:    io.NewSectionReader(w.iso, int64(fi.Extent())*LogicalBlockSize, fi.Size()),
	}, nil
}

// ReadDir reads the directory named by dirname and returns a list of
// directory entries sorted by filename.
func (w *Walker) ReadDir(dirname string) ([]*FileInfo, error) {
	var entries []*FileInfo

	fi, err := w.Lstat(dirname)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, syscall.ENOTDIR
	}

	err = iterDir(w.iso, fi.Extent(), fi.Size(), func(fi *FileInfo) bool {
		cacheKey := filepath.Join(filepath.Clean("/"+dirname), fi.Name())
		w.lstatCache.Add(cacheKey, fi)
		entries = append(entries, fi)
		return true
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

const SymlinkRecursionLimit = 40

func (w *Walker) stat(parts []string) (*FileInfo, error) {
	for i := 0; i < SymlinkRecursionLimit; i++ {
		fi, err := w.lstat(parts)
		if err != nil {
			return nil, err
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			path := "/" + strings.Join(parts, "/")
			if !fi.IsDir() {
				// If the symlink is a file, only join the directory part of the path
				path = filepath.Dir(path)
			}
			joined := filepath.Clean(filepath.Join(path, fi.Target()))
			parts = w.pathParts(joined)
		} else {
			return fi, nil
		}
	}
	return nil, syscall.ELOOP
}

func (w *Walker) lstat(parts []string) (*FileInfo, error) {
	var start LogicalBlockAddress
	var size int64
	var cached *FileInfo
	cached, parts = w.lstatCacheSearch(append([]string{"."}, parts...))
	if cached == nil {
		var pvd PrimaryVolumeDescriptor
		pvdSector := io.NewSectionReader(w.iso, 16*LogicalBlockSize, LogicalBlockSize)
		if err := DecodePrimaryVolumeDescriptor(pvdSector, &pvd); err != nil {
			return nil, err
		}

		start = pvd.RootStart
		size = int64(pvd.RootLength)
	} else if len(parts) > 0 {
		start = cached.Extent()
		size = cached.Size()
	} else {
		return cached, nil
	}

	var path []string
	for len(parts) > 0 {
		part := parts[0]
		parts = parts[1:]

		var partInfo *FileInfo
		err := iterDir(w.iso, start, size, func(fi *FileInfo) bool {
			if fi.IsDir() {
				w.lstatCacheAdd(append(path, fi.Name()), fi)
			}

			if fi.Name() == part {
				partInfo = fi
				return false
			}
			return true
		})
		if err != nil {
			return nil, err
		}

		if partInfo == nil {
			return nil, syscall.ENOENT
		}

		if len(parts) == 0 {
			return partInfo, nil
		}

		if !partInfo.IsDir() {
			return nil, syscall.ENOTDIR
		}

		path = append(path, part)
		start = partInfo.Extent()
		size = partInfo.Size()
	}
	panic("never")
}

func (w *Walker) lstatCacheAdd(parts []string, fi *FileInfo) {
	key := filepath.Clean("/" + strings.Join(parts, "/"))
	w.lstatCache.Add(key, fi)
}

func (w *Walker) lstatCacheSearch(parts []string) (*FileInfo, []string) {
	for i := len(parts); i >= 0; i-- {
		key := filepath.Clean("/" + strings.Join(parts[:i], "/"))
		if fi, ok := w.lstatCache.Get(key); ok {
			return fi.(*FileInfo), parts[i:]
		}
	}

	return nil, parts
}

func iterDir(iso io.ReaderAt, start LogicalBlockAddress, size int64, visit func(fi *FileInfo) bool) error {
	var dir io.Reader
	dir = io.NewSectionReader(iso, int64(start*LogicalBlockSize), size)
	minSize := int64(1024 * 1024)
	if size < minSize {
		minSize = size
	}
	dir = bufio.NewReaderSize(dir, int(minSize))

	var directory Directory
	err := DecodeDirectory(dir, &directory)
	if err != nil {
		return err
	}

	var prevIdentifier string

	// accumulate multiple records with the same identifier (i.e. files larger than 4GB).
	var aggregateRecords []*aggregateRec
	for i, rec := range directory.Records {
		if i > 0 {
			prev := aggregateRecords[len(aggregateRecords)-1]
			if prev.rec.Identifier == rec.Identifier {
				prev.length += int64(rec.Length)
				continue
			}
		}

		aggregateRecords = append(aggregateRecords, &aggregateRec{
			rec:    rec,
			length: int64(rec.Length),
		})
	}

	for _, arec := range aggregateRecords {
		rec := arec.rec
		if rec.Identifier == prevIdentifier {
			continue
		}

		systemUse, err := allSystemUseEntries(iso, &rec)
		if err != nil {
			return err
		}

		name, hasName := rrip.DecodeName(systemUse)
		if !hasName {
			if rec.Identifier == string([]byte{0x0}) {
				name = "."
			} else if rec.Identifier == string([]byte{0x1}) {
				name = ".."
			} else {
				name = rec.Identifier
			}
		}

		isDir := rec.Flags&FileFlagDir != 0
		target, isSymlink := rrip.DecodeSymlink(systemUse)
		var mode os.FileMode
		nlink := uint32(1)
		var uid uint32
		var gid uint32
		ino := uint32(rec.Start)

		if pe, ok := rrip.DecodePosixEntry(systemUse); ok {
			mode = pe.Mode & os.ModePerm
			if pe.Mode&syscall.S_IFDIR == syscall.S_IFDIR {
				mode |= os.ModeDir
			} else if pe.Mode&syscall.S_IFLNK == syscall.S_IFLNK {
				mode |= os.ModeSymlink
			}
			nlink = pe.Nlink
			uid = pe.Uid
			gid = pe.Gid
			ino = pe.Ino
		} else if isDir {
			mode = os.ModeDir | 0555
		} else if isSymlink {
			mode = os.ModeSymlink | 0777
		} else {
			mode = 0444
		}

		fi := &FileInfo{
			name:    name,
			size:    arec.length,
			mode:    mode,
			nlink:   nlink,
			uid:     uid,
			gid:     gid,
			ino:     ino,
			modTime: time.Unix(0, 0).UTC(), // TODO: rrip tf
			isDir:   isDir,
			extent:  rec.Start,
			target:  target,
		}

		if cont := visit(fi); !cont {
			break
		}

		prevIdentifier = rec.Identifier
	}

	return nil
}

func (w *Walker) pathParts(path string) []string {
	var parts []string
	cleaned := filepath.Clean("/" + path)
	if len(cleaned) > 1 {
		parts = strings.Split(cleaned[1:], "/")
	}
	return parts
}

type walkItem struct {
	parts []string
	fi    *FileInfo
}

type aggregateRec struct {
	rec    DirectoryRecord
	length int64
}

func allSystemUseEntries(iso io.ReaderAt, rec *DirectoryRecord) ([]susp.SystemUseEntry, error) {
	unconsumed := rec.SystemUse
	var systemUse []susp.SystemUseEntry
	for len(unconsumed) > 0 {
		entry := unconsumed[0]
		unconsumed = unconsumed[1:]
		switch v := entry.(type) {
		case *susp.ContinuationAreaEntry:
			ceStart := v.ContinuationStart()*LogicalBlockSize + v.ContinuationOffset()
			ceLen := v.ContinuationLength()
			ce := io.NewSectionReader(iso, int64(ceStart), int64(ceLen))
			more, err := DecodeSystemUseEntries(ce)
			if err != nil {
				return nil, err
			}

			unconsumed = append(unconsumed, more...)
		default:
			systemUse = append(systemUse, entry)
		}
	}

	return systemUse, nil
}
