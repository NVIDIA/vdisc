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
	"sync"

	"golang.org/x/sync/semaphore"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

func NewReadAheadController(window int, readAheadTokens *semaphore.Weighted, slicer Slicer, obj storage.Object) *ReadAheadController {
	return &ReadAheadController{
		window:          int64(window),
		readAheadTokens: readAheadTokens,
		slicer:          slicer,
		obj:             obj,
		numBlocks:       (obj.Size() + slicer.Bsize() - 1) / slicer.Bsize(),
		nextBlock:       1,
	}
}

type ReadAheadController struct {
	window          int64
	readAheadTokens *semaphore.Weighted
	slicer          Slicer
	obj             storage.Object
	numBlocks       int64

	mu        sync.Mutex
	pos       int64 // the expected position for the next read to be considered sequential
	runCount  int   // number of consecutive sequential reads
	runLength int64 // the length of the current run
	nextBlock int64
}

func (rac *ReadAheadController) Update(off int64, n int) {
	rac.mu.Lock()
	defer rac.mu.Unlock()

	currBlock := off / rac.slicer.Bsize()

	if rac.pos != off {
		// Not a sequential read, reset
		rac.runCount = 1
		rac.runLength = int64(n)
		rac.nextBlock = currBlock + 1
	} else {
		rac.runCount++
		rac.runLength += int64(n)
		if rac.nextBlock <= currBlock {
			rac.nextBlock = currBlock + 1
		}
	}
	rac.pos = off + int64(n)

	// We only read-ahead as many blocks as we've read sequentially.
	damper := (rac.runLength + rac.slicer.Bsize() - 1) / rac.slicer.Bsize()
	limit := currBlock + rac.window
	for rac.nextBlock <= limit && damper > 0 && rac.nextBlock < rac.numBlocks && rac.readAheadTokens.TryAcquire(1) {
		go rac.readBlock(rac.nextBlock)
		rac.nextBlock++
		damper--
	}
}

func (rac *ReadAheadController) readBlock(block int64) {
	defer rac.readAheadTokens.Release(1)

	bsize := rac.slicer.Bsize()
	off := block * bsize
	part := rac.slicer.Slice(rac.obj, off)
	part.ReadAhead()
}
