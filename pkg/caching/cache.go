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
	"errors"
	"io"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"github.com/NVIDIA/vdisc/pkg/interval"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

type Slice interface {
	storage.AnonymousObject
	ReadAhead()
}

type Slicer interface {
	// Bsize returns the maximum slice size
	Bsize() int64

	// Slice returns a read-through cached slice of an object
	Slice(obj storage.Object, offset int64) Slice
}

type Cache interface {
	// WithCaching applies a read-through caching layer to obj
	WithCaching(obj storage.Object) storage.Object
}

var NopCache = nopCache{}

type nopCache struct{}

func (nc nopCache) WithCaching(obj storage.Object) storage.Object {
	return obj
}

func NewCache(slicer Slicer, readAheadTokens int64) Cache {
	return &cache{slicer, semaphore.NewWeighted(readAheadTokens)}
}

type cache struct {
	slicer          Slicer
	readAheadTokens *semaphore.Weighted
}

func (c *cache) WithCaching(obj storage.Object) storage.Object {
	return &withCaching{
		obj:                 obj,
		slicer:              c.slicer,
		readAheadController: NewReadAheadController(c.readAheadTokens, c.slicer, obj),
	}
}

type withCaching struct {
	obj                 storage.Object
	slicer              Slicer
	readAheadController *ReadAheadController

	pos int64
}

func (w *withCaching) Close() error {
	return w.obj.Close()
}

func (w *withCaching) URL() string {
	return w.obj.URL()
}

func (w *withCaching) Size() int64 {
	return w.obj.Size()
}

func (w *withCaching) Read(p []byte) (n int, err error) {
	n, err = w.ReadAt(p, w.pos)
	w.pos += int64(n)
	return
}

func (w *withCaching) ReadAt(p []byte, off int64) (n int, err error) {
	if len(p) == 0 {
		return
	}

	full := interval.Interval{0, w.obj.Size()}
	read := interval.Interval{off, off + int64(len(p))}
	overlap, ok := interval.Intersection(full, read)
	if !ok {
		err = io.EOF
		return
	}

	bstart := overlap.Start / w.slicer.Bsize()
	boff := overlap.Start % w.slicer.Bsize()
	bend := overlap.End / w.slicer.Bsize()

	var parts []storage.AnonymousObject
	for block := bstart; block <= bend; block++ {
		offset := block * w.slicer.Bsize()
		part := w.slicer.Slice(w.obj, offset)
		parts = append(parts, part)
	}

	var joined storage.AnonymousObject
	if len(parts) > 1 {
		joined = storage.ConcurrentConcat(parts...)
	} else {
		joined = parts[0]
	}

	n, err = joined.ReadAt(p[:overlap.End-overlap.Start], boff)

	w.readAheadController.Update(off, n)

	return
}

func (w *withCaching) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		w.pos = w.pos + offset
	case io.SeekStart:
		w.pos = offset
	case io.SeekEnd:
		if w.Size() < 0 {
			return 0, errors.New("unknown length")
		}
		w.pos = w.Size() + offset
	}

	if w.pos < 0 {
		w.pos = 0
	} else if w.Size() >= 0 && w.pos > w.Size() {
		w.pos = w.Size()
	}

	return w.pos, nil
}

func cacheLogger() *zap.Logger {
	return zap.L().Named("cache")
}
