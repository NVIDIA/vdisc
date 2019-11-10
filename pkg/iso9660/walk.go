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
	"github.com/NVIDIA/vdisc/pkg/safecast"
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
	it := NewReadDirIterator(iso, start, size, 0)
	for it.Next() {
		fi, _ := it.FileInfoAndLen()
		if cont := visit(fi); !cont {
			break
		}
	}
	return it.Err()
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

// NewReadDirIterator creates an iterator of FileInfos for a directory
// at start of size and off offset into the directory.
func NewReadDirIterator(iso io.ReaderAt, start LogicalBlockAddress, size int64, off int64) *ReadDirIterator {
	recIt := NewDirectoryRecordIterator(iso, start, size, off)
	return &ReadDirIterator{
		iso:       iso,
		recIt:     recIt,
		finfoLen:  -1,
		peekedLen: -1,
	}
}

type ReadDirIterator struct {
	iso       io.ReaderAt
	recIt     *DirectoryRecordIterator
	finfo     *FileInfo
	finfoLen  int64
	peeked    *DirectoryRecord
	peekedLen int64
	err       error
	exhausted bool
}

// Err should be checked once Next returns false
func (it *ReadDirIterator) Err() error {
	return it.err
}

// Next reads one or more records from the directory, aggregating them
// as necessary.
func (it *ReadDirIterator) Next() bool {
	if it.exhausted || it.err != nil {
		return false
	}

	var curr *DirectoryRecord
	var currLen int64
	if it.peeked == nil {
		if it.recIt.Next() {
			curr, currLen = it.recIt.RecordAndLen()
		} else {
			it.exhausted = true
			it.err = it.recIt.Err()
			return false
		}
	} else {
		curr = it.peeked
		currLen = it.peekedLen
		it.peeked = nil
		it.peekedLen = -1
	}

	size := int64(curr.Length)

	for {
		// look ahead to see if there is a next record, and if so, does it
		// have the same identifier.
		if it.recIt.Next() {
			var next *DirectoryRecord
			var nextLen int64
			next, nextLen = it.recIt.RecordAndLen()
			if next.Identifier == curr.Identifier {
				// This record is a continuation of the current record, so
				// we just aggregate the lengths.
				currLen += nextLen
				size += int64(next.Length)
			} else {
				it.peeked = next
				it.peekedLen = nextLen
				break
			}
		} else {
			it.exhausted = true
			it.err = it.recIt.Err()
			break
		}
	}

	// Now we actually construct the FileInfo
	systemUse, err := allSystemUseEntries(it.iso, curr)
	if err != nil {
		it.err = err
		return false
	}

	name, hasName := rrip.DecodeName(systemUse)
	if !hasName {
		if curr.Identifier == string([]byte{0x0}) {
			name = "."
		} else if curr.Identifier == string([]byte{0x1}) {
			name = ".."
		} else {
			name = curr.Identifier
		}
	}

	isDir := curr.Flags&FileFlagDir != 0
	target, isSymlink := rrip.DecodeSymlink(systemUse)
	if isSymlink {
		size = int64(len(target))
	}
	var mode os.FileMode
	nlink := uint32(1)
	var uid uint32
	var gid uint32
	ino := uint32(curr.Start)

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

	modTime := time.Unix(0, 0).UTC()

	if ts, ok := rrip.DecodeTimestamps(systemUse); ok {
		modTime = ts.Modified.UTC()
	}

	it.finfo = &FileInfo{
		name:    name,
		size:    size,
		mode:    mode,
		nlink:   nlink,
		uid:     uid,
		gid:     gid,
		ino:     ino,
		modTime: modTime,
		isDir:   isDir,
		extent:  curr.Start,
		target:  target,
	}
	it.finfoLen = currLen
	return true
}

// FileInfoAndLen returns the FileInfo consumed by calling Next()
// along with the number of bytes consumed in the directory.
func (it *ReadDirIterator) FileInfoAndLen() (*FileInfo, int64) {
	return it.finfo, it.finfoLen
}

// NewDirectoryRecordIterator creates a new iterator of
// DirectoryRecords for a directory with start and size and off offset
// into the directory.
func NewDirectoryRecordIterator(iso io.ReaderAt, start LogicalBlockAddress, size int64, off int64) *DirectoryRecordIterator {
	dir := io.NewSectionReader(iso, int64(start*LogicalBlockSize), size)
	_, err := dir.Seek(off, io.SeekStart)
	return &DirectoryRecordIterator{
		dir:    dir,
		recLen: -1,
		err:    err,
	}
}

type DirectoryRecordIterator struct {
	dir    *io.SectionReader
	rec    *DirectoryRecord
	recLen int64
	err    error
}

// Err should be checked once Next returns false
func (it *DirectoryRecordIterator) Err() error {
	return it.err
}

// The current position in the directory
func (it *DirectoryRecordIterator) Tell() int64 {
	pos, _ := it.dir.Seek(0, io.SeekCurrent)
	return pos
}

// Next reads the next DirectoryRecord. It returns true if another
// record was present, and false on EOF or if an error occurred.
func (it *DirectoryRecordIterator) Next() bool {
	if it.err != nil {
		return false
	}
	start := it.Tell()
	for {
		rlen, err := readByte(it.dir)
		if err != nil {
			it.rec = nil
			it.recLen = -1
			if err == io.EOF {
				it.err = nil
				return false
			}
			it.err = err
		}

		if rlen == 0 {
			// The rest of this sector is padding. Consume, and move on the the next sector.
			pos := it.Tell()
			padding := safecast.Uint64ToInt64(sectorsToBytes(bytesToSectors(safecast.Int64ToUint32(pos)))) - pos
			if err := unpad(it.dir, safecast.Int64ToInt(padding)); err != nil {
				it.rec = nil
				it.recLen = -1
				it.err = err
				return false
			}
			continue
		}

		var rec DirectoryRecord
		if err := DecodeDirectoryRecord(io.LimitReader(it.dir, int64(rlen)-1), &rec); err != nil {
			it.rec = nil
			it.recLen = -1
			it.err = err
			return false
		}

		it.rec = &rec
		it.recLen = it.Tell() - start
		it.err = nil
		return true
	}
}

// RecordAndLen returns the DirectoryRecord that was read by Next and
// the number of bytes that were read from the directory.
func (it *DirectoryRecordIterator) RecordAndLen() (*DirectoryRecord, int64) {
	return it.rec, it.recLen
}
