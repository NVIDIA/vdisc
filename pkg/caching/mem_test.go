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
	"bytes"
	"io"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/NVIDIA/vdisc/pkg/caching"
	"github.com/NVIDIA/vdisc/pkg/storage"
	_ "github.com/NVIDIA/vdisc/pkg/storage/data"
)

func TestMemoryCache(t *testing.T) {
	bsizes := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	bcounts := []int64{1, 2, 3, 4, 5}

	as, err := storage.Open("data:text/plain,aaa")
	if err != nil {
		t.Fatal(err)
	}
	bs, err := storage.Open("data:text/plain,bbb")
	if err != nil {
		t.Fatal(err)
	}
	cs, err := storage.Open("data:text/plain,ccc")
	if err != nil {
		t.Fatal(err)
	}

	srax := storage.Concat(as, bs, cs)
	if srax.Size() != 9 {
		t.Fatalf("Size = %d; want 9", srax.Size())
	}
	const full = "aaabbbccc"

	for _, bsize := range bsizes {
		for _, bcount := range bcounts {
			slicer, err := caching.NewMemorySlicer(bsize, bcount)
			if err != nil {
				t.Fatal(err)
			}
			cache := caching.NewCache(slicer, 0)
			sra := cache.WithCaching(storage.WithURL(srax, "test:srax"))

			for i := 0; i < 10; i++ {
				for start := 0; start < len(full); start++ {
					for end := start; end < len(full); end++ {
						want := full[start:end]

						buf := make([]byte, end-start)
						n, err := sra.ReadAt(buf, int64(start))
						if err != nil {
							t.Fatalf("start=%d, end=%d, full=%q, n=%d, err=%v", start, end, full, n, err)
						}

						if n != len(buf) {
							t.Fatalf("for start=%d, end=%d: ReadAt = %q; want %q", start, end, buf, want)
						}

						got, err := ioutil.ReadAll(io.NewSectionReader(sra, int64(start), int64(end-start)))
						if err != nil {
							t.Fatal(err)
						}
						if string(got) != want {
							t.Fatalf("for start=%d, end=%d: ReadAll = %q; want %q", start, end, got, want)
						}
					}
				}
			}
		}
	}
}

func TestMemoryCacheRace(t *testing.T) {
	tokens := []int64{0, 1, 2}

	for _, raTokens := range tokens {
		s, err := storage.Open("data:text/plain,aaaaaaaaaaaaaaaaaaaa")
		if err != nil {
			t.Fatal(err)
		}
		slicer, err := caching.NewMemorySlicer(8, 2)
		if err != nil {
			t.Fatal(err)
		}
		cache := caching.NewCache(slicer, raTokens)
		cached := cache.WithCaching(s)

		var wg sync.WaitGroup
		wg.Add(1000)
		for i := 0; i < 1000; i++ {
			go func() {
				defer wg.Done()
				src := io.NewSectionReader(cached, 0, cached.Size())
				dst := bytes.NewBuffer(nil)
				buf := make([]byte, 1)
				_, err := io.CopyBuffer(dst, src, buf)
				if err != nil {
					t.Fatal(err)
				}
			}()
		}
		wg.Wait()
	}
}
