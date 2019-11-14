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

package caching

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/OneOfOne/xxhash"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/storage"
	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

const (
	XattrKey      = "user.vdisc-cache-key"
	XattrChecksum = "user.vdisc-cache-checksum"
)

type GcThreshold interface {
	GcNeeded(st *syscall.Statfs_t) bool
}

type DiskSlicer struct {
	root    string
	bsize   int64
	bufPool *sync.Pool
	wg      sync.WaitGroup
}

func NewDiskSlicer(root string, bsize int64) *DiskSlicer {
	return &DiskSlicer{
		root:  root,
		bsize: bsize,
		bufPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, bsize)
			},
		},
	}
}

func (ds *DiskSlicer) Bsize() int64 {
	return ds.bsize
}

func (ds *DiskSlicer) Slice(obj storage.Object, offset int64) Slice {
	size := obj.Size() - offset
	if ds.bsize < size {
		size = ds.bsize
	}

	ckey := diskKey{obj.URL(), offset, size}
	key, err := json.Marshal(&ckey)
	if err != nil {
		panic(err)
	}

	return &diskSlice{
		root:    ds.root,
		bsize:   ds.bsize,
		bufPool: ds.bufPool,
		wg:      &ds.wg,
		obj:     obj,
		offset:  offset,
		size:    size,
		key:     key,
	}
}

func (ds *DiskSlicer) Gc(threshold GcThreshold) {
	cacheLogger().Info("starting garbage collection")

	defer cacheLogger().Info("garbage collection complete")
	var st syscall.Statfs_t
	it := &diskIter{root: ds.root}
	for {
		if err := syscall.Statfs(ds.root, &st); err != nil {
			cacheLogger().Error("cache statfs", zap.String("cacheDir", ds.root), zap.Error(err))
			return
		}

		if !threshold.GcNeeded(&st) {
			return
		}
		victim, ok := it.Next()
		if !ok {
			cacheLogger().Warn("victims exhausted")
			return
		}

		// Unlink the victim and then fsync the parent dir
		if err := storage.Remove(victim); err != nil {
			cacheLogger().Error("failed to evict victim", zap.String("victim", victim), zap.Error(err))
			return
		}

		cacheLogger().Debug("garbage collected", zap.String("victim", victim))
	}
}

func (ds *DiskSlicer) CheckIntegrity() error {
	it := &diskIter{root: ds.root}
	for {
		fname, ok := it.Next()
		if !ok {
			return nil
		}

		cobj, err := storage.Open(fname)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		defer cobj.Close()

		var expected uint64
		switch xobj := cobj.(type) {
		case driver.XattrObject:
			csumEnc, err := xobj.GetXattr(XattrChecksum)
			if err != nil {
				return err
			}

			expected = binary.LittleEndian.Uint64(csumEnc)
		default:
			return fmt.Errorf("object missing checksum xattr: %q", fname)
		}

		csum := xxhash.New64()
		if _, err := io.Copy(csum, cobj); err != nil {
			return err
		}

		if csum.Sum64() != expected {
			return fmt.Errorf("Bad checksum: %q", fname)
		}
	}
	return nil
}

func (ds *DiskSlicer) Wait() {
	ds.wg.Wait()
}

type diskSlice struct {
	root    string
	bsize   int64
	bufPool *sync.Pool
	wg      *sync.WaitGroup
	obj     storage.Object
	offset  int64
	size    int64
	key     []byte
	pos     int64
}

func (ds *diskSlice) Close() error {
	return nil
}

func (ds *diskSlice) Size() int64 {
	return ds.size
}

func (ds *diskSlice) Read(p []byte) (n int, err error) {
	n, err = ds.ReadAt(p, ds.pos)
	ds.pos += int64(n)
	return
}

func (ds *diskSlice) ReadAt(p []byte, off int64) (n int, err error) {
	if len(p) == 0 {
		return
	}

	for {
		cobj, cerr := storage.Open(ds.cacheFname())
		if cerr == nil {
			defer cobj.Close()

			switch xobj := cobj.(type) {
			case driver.XattrObject:
				ckey, xerr := xobj.GetXattr(XattrKey)
				if xerr == nil {
					if bytes.Equal(ckey, ds.key) {
						// TODO: increment cache hit
						n, err = cobj.ReadAt(p, off)
						return
					} else {
						cacheLogger().Debug("cache collision", zap.String("expected", string(ds.key)), zap.String("actual", string(ckey)))
					}
				} else {
					cacheLogger().Error("getxattr", zap.Error(err))
				}
			}
		}
		if !os.IsNotExist(cerr) {
			cacheLogger().Error("cache read error", zap.Error(cerr))
		}

		n, err = ds.fill(p, off)
		if err != nil {
			return
		} else if n > 0 {
			// TODO: increment cache miss
			return
		}
	}
}

func (ds *diskSlice) ReadAhead() {
	_, err := ds.fill(nil, 0)
	if err != nil {
		cacheLogger().Error("read-ahead", zap.Error(err))
	}
}

func (ds *diskSlice) fill(p []byte, off int64) (n int, err error) {
	// Grab a lock for writing to the cache
	fname := ds.cacheFname()
	dir := filepath.Dir(fname)
	base := filepath.Base(fname)
	lockc, lerr := storage.Lock(filepath.Join(dir, ".lock."+base))
	if lerr != nil {
		cacheLogger().Error("cache lock", zap.Error(lerr))
		return 0, lerr
	}

	unlock := func() {
		if err := lockc.Close(); err != nil {
			cacheLogger().Error("cache unlock", zap.Error(err))
		}
	}

	// Double check that we didn't lose the race trying to write to the cache
	cobj, serr := storage.Open(fname)
	if serr == nil {
		switch xobj := cobj.(type) {
		case driver.XattrObject:
			ckey, xerr := xobj.GetXattr(XattrKey)
			if xerr == nil {
				if bytes.Equal(ckey, ds.key) {
					// we lost the race, let's start over
					unlock()
					return
				}
			}
		}
	} else if !os.IsNotExist(serr) {
		// There was some deeper issue checking the cache
		unlock()
		err = serr
		return
	}

	// Okay, it is our job to try to fill the cache.  Grab a
	// temporary buffer from the pool and fill it from the source.
	b := ds.bufPool.Get().([]byte)
	src := io.NewSectionReader(ds.obj, ds.offset, ds.bsize) // TNARG
	m, rerr := io.ReadFull(src, b[:ds.size])
	if rerr != nil {
		// Reading from the source failed, nothing we can do.
		unlock()
		err = rerr
		return
	} else if int64(m) < ds.size {
		// Uh oh. The source wasn't as big as we thought?!
		unlock()
		err = io.ErrUnexpectedEOF
		return
	}

	// We've fulfilled the request at this point.
	if p != nil {
		n = copy(p, b[off:ds.size])
	}

	// Queue a write-back to the cache and then return immediately.
	ds.wg.Add(1)
	go func() {
		defer ds.wg.Done()
		// The lock and temporary buffer are released whether this succeeds
		// or not
		defer ds.bufPool.Put(b)
		defer unlock()

		w, err := storage.Create(ds.cacheFname())
		if err != nil {
			cacheLogger().Error("cache write error", zap.Error(err))
			return
		}
		defer w.Abort()

		csum := xxhash.New64()
		mw := io.MultiWriter(w, csum)

		n, err := mw.Write(b[:ds.size])
		if err != nil {
			cacheLogger().Error("cache write error", zap.Error(err))
			return
		}
		if int64(n) < ds.size {
			cacheLogger().Error("cache write error", zap.Error(io.ErrShortWrite))
			return
		}

		switch xw := w.(type) {
		case driver.XattrObjectWriter:
			if err := xw.SetXattr(XattrKey, []byte(ds.key)); err != nil {
				cacheLogger().Error("cache setxattr error", zap.String("xattr", XattrKey), zap.Error(err))
				return
			}
			var csumEnc [8]byte
			binary.LittleEndian.PutUint64(csumEnc[:], csum.Sum64())
			if err := xw.SetXattr(XattrChecksum, csumEnc[:]); err != nil {
				cacheLogger().Error("cache setxattr error", zap.String("xattr", XattrChecksum), zap.Error(err))
				return
			}
		default:
			cacheLogger().Error("cache setxattr not supported")
			return
		}

		if _, err := w.Commit(); err != nil {
			cacheLogger().Error("cache write error", zap.Error(err))
		}
	}()
	return
}

func (ds *diskSlice) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		ds.pos = ds.pos + offset
	case io.SeekStart:
		ds.pos = offset
	case io.SeekEnd:
		if ds.size < 0 {
			return 0, errors.New("unknown length")
		}
		ds.pos = ds.size + offset
	}

	if ds.pos < 0 {
		ds.pos = 0
	} else if ds.size >= 0 && ds.pos > ds.size {
		ds.pos = ds.size
	}

	return ds.pos, nil
}

func (ds *diskSlice) cacheFname() string {
	bkey := fmt.Sprintf("%x", md5.Sum(ds.key))
	prefix := bkey[:2]
	return filepath.Join(ds.root, "v0", prefix, bkey[2:])
}

type diskKey struct {
	Url string `json:"url"`
	Off int64  `json:"off"`
	Len int64  `json:"len"`
}

type diskIter struct {
	root    string
	prime   sync.Once
	parentQ []string
	victimQ []string
}

func (it *diskIter) Next() (string, bool) {
	it.prime.Do(func() {
		root := filepath.Join(it.root, "v0")
		finfos, err := storage.Readdir(root)
		if err != nil {
			cacheLogger().Error("failed to list cache root", zap.Error(err))
			return
		}

		for _, finfo := range finfos {
			if finfo.IsDir() && len(finfo.Name()) == 2 {
				it.parentQ = append(it.parentQ, filepath.Join(root, finfo.Name()))
			}
		}

		// Randomize the order we walk the subdirectories.
		rand.Shuffle(len(it.parentQ), func(i, j int) {
			it.parentQ[i], it.parentQ[j] = it.parentQ[j], it.parentQ[i]
		})
	})

	for {
		if len(it.victimQ) > 0 {
			victim := it.victimQ[0]
			it.victimQ = it.victimQ[1:]
			return victim, true
		}

		if len(it.parentQ) == 0 {
			return "", false
		}

		parent := it.parentQ[0]
		it.parentQ = it.parentQ[1:]
		finfos, err := storage.Readdir(parent)
		if err != nil {
			cacheLogger().Error("failed to list cache parent", zap.String("parent", parent), zap.Error(err))
			continue
		}

		for _, finfo := range finfos {
			if !finfo.IsDir() && !strings.HasPrefix(finfo.Name(), ".lock") && !strings.HasPrefix(finfo.Name(), ".tmp") {
				it.victimQ = append(it.victimQ, filepath.Join(parent, finfo.Name()))
			}
		}

		rand.Shuffle(len(it.victimQ), func(i, j int) {
			it.victimQ[i], it.victimQ[j] = it.victimQ[j], it.victimQ[i]
		})
	}
}
