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

package caching_test

import (
	"sync"
	"testing"

	"golang.org/x/sync/semaphore"

	"github.com/NVIDIA/vdisc/pkg/caching"
	"github.com/NVIDIA/vdisc/pkg/storage/mock"
)

func TestReadAheadDisabled(t *testing.T) {
	obj := &mockdriver.Object{}
	obj.On("Size").Return(int64(1024 * 1024))
	slicer := &mockSlicer{}
	slicer.On("Bsize").Return(int64(1024))
	rac := caching.NewReadAheadController(0, semaphore.NewWeighted(0), slicer, obj)
	for i := int64(0); i < 64*1024; i++ {
		rac.Update(i, 1)
	}
}

func TestReadAheadMany(t *testing.T) {
	obj := &mockdriver.Object{}
	obj.On("Size").Return(int64(1024 * 1024))
	slicer := &mockSlicer{}
	slicer.On("Bsize").Return(int64(1024))
	rac := caching.NewReadAheadController(32, semaphore.NewWeighted(64), slicer, obj)

	blk := int64(1)
	for i := 0; i < 1024; i++ {
		var wg sync.WaitGroup
		slice := &mockSlice{}
		if blk < 1024 {
			slice.On("ReadAhead").Return(func() {
				wg.Done()
			})
		}

		count := i + 1
		if count > 32 {
			count = 32
		}

		for j := 0; j < count; j++ {
			if blk < 1024 && blk <= int64(i)+32 {
				wg.Add(1)
				slicer.On("Slice", obj, blk*1024).Return(slice)
				blk++
			}
		}

		rac.Update(int64(i)*1024, 1024)
		wg.Wait()
		slice.AssertExpectations(t)
	}
	slicer.AssertExpectations(t)
	obj.AssertExpectations(t)
}
