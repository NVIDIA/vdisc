// Copyright © 2019 NVIDIA Corporation
package vdisc

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

type BufferCacheConfig struct {
	Bsize  int `help:"Size of each buffer cache buffer" default:"262144"`
	Bcount int `help:"Number of buffer cache buffers" default:"4"`
}

// BufferCache implements a fixed size, lru cache of fixed size
// buffers similar to a buffer cache used in operating system file
// systems. The cache is shared by wrapping multiple objects using
// the same BufferCache instance.
type BufferCache struct {
	bsize  int
	bcount int

	mu    *sync.Mutex
	lru   *simplelru.LRU
	bnext int
}

// NewBufferCache returns a BufferCache which consumes bsize * bcount bytes of memory.
func NewBufferCache(cfg BufferCacheConfig) (*BufferCache, error) {
	lru, err := simplelru.NewLRU(cfg.Bcount, nil)
	if err != nil {
		return nil, err
	}

	return &BufferCache{
		bsize:  cfg.Bsize,
		bcount: cfg.Bcount,

		mu:  &sync.Mutex{},
		lru: lru,
	}, nil
}

// Wrap splits a object into fixed-size parts, wraps each part in a
// read-through cache, and then concatenates the chunks. Reads issued
// to the wrapped object are always made in fixed-size increments
// except potentially for the last part.
func (bc *BufferCache) Wrap(obj storage.Object) storage.Object {
	if obj.URL() == "" {
		panic("never")
	}

	return &wrapped{bc, obj, 0}
}

func (bc *BufferCache) getBuffer(key string) (buffer *Buffer) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	ent, cacheHit := bc.lru.Get(key)
	if cacheHit {
		buffer = ent.(*Buffer)
	} else if bc.bnext < bc.bcount {
		bc.bnext++
		buffer = NewBuffer(bc.bsize)
		bc.lru.Add(key, buffer)
	} else {
		_, ent, _ = bc.lru.RemoveOldest()
		buffer = ent.(*Buffer)
		bc.lru.Add(key, buffer)
	}

	return
}

type wrapped struct {
	bc  *BufferCache
	obj storage.Object
	pos int64
}

func (w *wrapped) Close() error {
	return w.obj.Close()
}

func (w *wrapped) URL() string {
	return w.obj.URL()
}

func (w *wrapped) Size() int64 {
	return w.obj.Size()
}

func (w *wrapped) Read(p []byte) (n int, err error) {
	n, err = w.ReadAt(p, w.pos)
	w.pos += int64(n)
	return
}

func (w *wrapped) ReadAt(p []byte, off int64) (n int, err error) {
	full := interval{0, w.obj.Size()}
	read := interval{off, off + int64(len(p))}
	overlap, ok := intersection(full, read)
	if !ok {
		return
	}

	bsize := int64(w.bc.bsize)
	bstart := overlap.start / bsize
	boff := overlap.start % bsize
	bend := overlap.end / bsize

	var parts []storage.AnonymousObject

	for block := bstart; block <= bend; block++ {
		offset := block * bsize
		n := minI64(bsize, w.obj.Size()-offset)

		var part storage.AnonymousObject
		part = &cached{
			bc:     w.bc,
			obj:    w.obj,
			offset: offset,
			size:   n,
		}
		parts = append(parts, part)
	}

	var joined storage.AnonymousObject
	if len(parts) > 1 {
		joined = storage.ConcurrentConcat(parts...)
	} else {
		joined = parts[0]
	}

	sect := io.NewSectionReader(joined, boff, overlap.end-overlap.start)
	return sect.Read(p)
}

func (w *wrapped) Seek(offset int64, whence int) (int64, error) {
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

type cached struct {
	bc     *BufferCache
	obj    storage.Object
	offset int64
	size   int64
	pos    int64
}

func (cr *cached) Close() error {
	return nil
}

func (cr *cached) Size() int64 {
	return cr.size
}

func (cr *cached) Read(p []byte) (n int, err error) {
	n, err = cr.ReadAt(p, cr.pos)
	cr.pos += int64(n)
	return
}

func (cr *cached) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	key := fmt.Sprintf("%s[%d,%d)", cr.obj.URL(), cr.offset, cr.offset+cr.size)
	buffer := cr.bc.getBuffer(key)
	return buffer.ReadAtCached(p, off, key, func(buf []byte) (int, error) {
		if int64(len(buf)) < cr.size {
			panic(fmt.Sprintf("never expected=%d, got=%d", cr.size, len(buf)))
		}

		src := io.NewSectionReader(cr.obj, cr.offset, cr.size)
		n, err := io.ReadFull(src, buf[:cr.size])
		if err != nil {
			return 0, err
		}
		return n, nil
	})
}

func (cr *cached) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		cr.pos = cr.pos + offset
	case io.SeekStart:
		cr.pos = offset
	case io.SeekEnd:
		if cr.size < 0 {
			return 0, errors.New("unknown length")
		}
		cr.pos = cr.size + offset
	}

	if cr.pos < 0 {
		cr.pos = 0
	} else if cr.size >= 0 && cr.pos > cr.size {
		cr.pos = cr.size
	}

	return cr.pos, nil
}

type interval struct {
	start int64 // inclusive
	end   int64 // exclusive
}

func intersection(a, b interval) (*interval, bool) {
	start := maxI64(a.start, b.start)
	end := minI64(a.end, b.end)
	if start < end {
		return &interval{start, end}, true
	}
	return nil, false
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
