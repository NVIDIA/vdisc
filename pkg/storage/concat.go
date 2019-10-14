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
package storage

import (
	"errors"
	"io"
	"sync"

	"github.com/google/btree"

	"github.com/NVIDIA/vdisc/pkg/interval"
)

type part struct {
	Offset int64
	Obj    AnonymousObject
}

func (p part) Less(than btree.Item) bool {
	return p.Offset < than.(part).Offset
}

func ConcurrentConcat(objects ...AnonymousObject) AnonymousObject {
	return concat(true, objects)
}

func Concat(objects ...AnonymousObject) AnonymousObject {
	return concat(false, objects)
}

func concat(concurrent bool, objects []AnonymousObject) AnonymousObject {
	if len(objects) == 1 {
		return objects[0]
	}

	var off int64
	parts := btree.New(24)
	for _, obj := range objects {
		parts.ReplaceOrInsert(part{off, obj})
		off += obj.Size()
	}

	return &concatenated{
		parts:      parts,
		size:       off,
		concurrent: concurrent,
	}
}

type concatenated struct {
	parts      *btree.BTree
	size       int64
	concurrent bool
	pos        int64
}

func (c *concatenated) Close() (err error) {
	c.parts.Ascend(func(i btree.Item) bool {
		err = i.(part).Obj.Close()
		return err == nil
	})
	return
}

func (c *concatenated) Size() int64 {
	return c.size
}

func (c *concatenated) Read(p []byte) (n int, err error) {
	n, err = c.ReadAt(p, c.pos)
	c.pos += int64(n)
	return
}

func (c *concatenated) ReadAt(p []byte, off int64) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if off >= c.size {
		return 0, io.EOF
	}

	start := off
	end := off + int64(len(p))
	if c.size < end {
		end = c.size
	}
	wantN := int(end - start)

	x := interval.Interval{start, end}

	var wg sync.WaitGroup
	var workItems []*workItem

	c.parts.DescendLessOrEqual(part{off + int64(len(p)) - 1, nil}, func(i btree.Item) bool {
		part := i.(part)

		y := interval.Interval{part.Offset, part.Offset + part.Obj.Size()}
		z, ok := interval.Intersection(x, y)
		if !ok {
			return false
		}

		workItem := &workItem{
			obj: part.Obj,
			dst: p[z.Start-off : z.End-off],
			off: z.Start - part.Offset,
		}

		if c.concurrent {
			wg.Add(1)
			go func() {
				defer wg.Done()
				nn, err := workItem.obj.ReadAt(workItem.dst, workItem.off)
				workItem.n = nn
				workItem.err = err
			}()
		} else {
			nn, err := workItem.obj.ReadAt(workItem.dst, workItem.off)
			workItem.n = nn
			workItem.err = err
		}

		workItems = append(workItems, workItem)

		return true
	})

	wg.Wait()

	for i := len(workItems) - 1; i >= 0; i-- {
		workItem := workItems[i]
		n += workItem.n
		if workItem.err != nil && workItem.err != io.EOF {
			err = workItem.err
			return
		}

		if workItem.n < len(workItem.dst) {
			err = io.ErrUnexpectedEOF
			return
		}
	}

	if n != wantN {
		err = io.ErrUnexpectedEOF
	}
	return
}

func (c *concatenated) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		c.pos = c.pos + offset
	case io.SeekStart:
		c.pos = offset
	case io.SeekEnd:
		if c.size < 0 {
			return 0, errors.New("unknown length")
		}
		c.pos = c.size + offset
	}

	if c.pos < 0 {
		c.pos = 0
	} else if c.size >= 0 && c.pos > c.size {
		c.pos = c.size
	}

	return c.pos, nil
}

type workItem struct {
	obj AnonymousObject
	dst []byte
	off int64
	n   int
	err error
}
