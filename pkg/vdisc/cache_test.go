// Copyright © 2019 NVIDIA Corporation
package vdisc_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/NVIDIA/vdisc/pkg/storage"
	_ "github.com/NVIDIA/vdisc/pkg/storage/data"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

func TestBufferCache(t *testing.T) {
	bsizes := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	bcounts := []int{1, 2, 3, 4, 5}

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
			bcache, err := vdisc.NewBufferCache(vdisc.BufferCacheConfig{bsize, bcount})
			if err != nil {
				t.Fatal(err)
			}
			sra := bcache.Wrap(storage.WithURL(srax, "test:srax"))

			for i := 0; i < 10; i++ {
				for start := 0; start < len(full); start++ {
					for end := start; end < len(full); end++ {
						want := full[start:end]

						buf := make([]byte, end-start)
						n, err := sra.ReadAt(buf, int64(start))
						if err != nil {
							t.Fatal(err)
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

func TestBufferCacheRace(t *testing.T) {
	s, err := storage.Open("data:text/plain,aaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	bcache, err := vdisc.NewBufferCache(vdisc.BufferCacheConfig{8, 2})
	if err != nil {
		t.Fatal(err)
	}
	cached := bcache.Wrap(s)

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
