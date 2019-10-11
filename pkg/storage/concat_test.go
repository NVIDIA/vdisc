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

package storage_test

import (
	//"errors"
	"io"
	"io/ioutil"
	"testing"

	//"github.com/stretchr/testify/assert"
	//"github.com/stretchr/testify/mock"

	//rmock "git.nvda.ai/nucleus/src/common/resources/mocks"
	"github.com/NVIDIA/vdisc/pkg/storage"
	_ "github.com/NVIDIA/vdisc/pkg/storage/data"
)

func TestConcat(t *testing.T) {
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
	sras := []storage.AnonymousObject{
		storage.Concat(as, bs, cs),
		storage.ConcurrentConcat(as, bs, cs),
	}

	const full = "aaabbbccc"
	for _, sra := range sras {
		if sra.Size() != 9 {
			t.Fatalf("Size = %d; want 9", sra.Size())
		}
		for start := 0; start < len(full); start++ {
			for end := start; end < len(full); end++ {
				want := full[start:end]

				buf := make([]byte, end-start)
				n, err := sra.ReadAt(buf, int64(start))
				if err != nil {
					t.Fatal(err)
				}

				if n != len(buf) {
					t.Errorf("for start=%d, end=%d: ReadAt = %q; want %q", start, end, buf, want)
				}

				got, err := ioutil.ReadAll(io.NewSectionReader(sra, int64(start), int64(end-start)))
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != want {
					t.Errorf("for start=%d, end=%d: ReadAll = %q; want %q", start, end, got, want)
				}
			}
		}

		buf := make([]byte, 4)
		n, err := sra.ReadAt(buf, 6)
		if err != nil {
			t.Fatal(err)
		}
		if n != 3 {
			t.Fatalf("overread: expected=3, got=%d", n)
		}
		got := string(buf[:n])
		if got != "ccc" {
			t.Fatalf("overread: expected=\"ccc\", got=%q", got)
		}
	}
}

//func TestConcatPartErr(t *testing.T) {
//	a := &rmock.UnnamedResource{}
//	b := &rmock.UnnamedResource{}
//	c := &rmock.UnnamedResource{}
//
//	a.On("Size").Return(int64(4))
//	b.On("Size").Return(int64(5))
//	c.On("Size").Return(int64(6))
//
//	expectedErr := errors.New("fake error")
//
//	a.On("ReadAt", mock.Anything, int64(0)).Return(4, nil)
//	b.On("ReadAt", mock.Anything, int64(0)).Return(1, expectedErr)
//	c.On("ReadAt", mock.Anything, int64(0)).Return(6, nil)
//
//	r := storage.Concat(a, b, c)
//
//	buf := make([]byte, 15)
//	n, actualErr := r.ReadAt(buf, 0)
//	assert.Equal(t, expectedErr, actualErr)
//	assert.Equal(t, 5, n)
//}
//
//func TestConcatIgnoreEOF(t *testing.T) {
//	a := &rmock.UnnamedResource{}
//	b := &rmock.UnnamedResource{}
//	c := &rmock.UnnamedResource{}
//
//	a.On("Size").Return(int64(4))
//	b.On("Size").Return(int64(5))
//	c.On("Size").Return(int64(6))
//
//	a.On("ReadAt", mock.Anything, int64(0)).Return(4, nil)
//	b.On("ReadAt", mock.Anything, int64(0)).Return(5, io.EOF)
//	c.On("ReadAt", mock.Anything, int64(0)).Return(6, nil)
//
//	r := storage.Concat(a, b, c)
//
//	buf := make([]byte, 15)
//	n, actualErr := r.ReadAt(buf, 0)
//	assert.Nil(t, actualErr)
//	assert.Equal(t, 15, n)
//}
//
//func TestConcatDontIgnoreEOF(t *testing.T) {
//	a := &rmock.UnnamedResource{}
//	b := &rmock.UnnamedResource{}
//	c := &rmock.UnnamedResource{}
//
//	a.On("Size").Return(int64(4))
//	b.On("Size").Return(int64(5))
//	c.On("Size").Return(int64(6))
//
//	a.On("ReadAt", mock.Anything, int64(0)).Return(4, nil)
//	b.On("ReadAt", mock.Anything, int64(0)).Return(4, io.EOF)
//	c.On("ReadAt", mock.Anything, int64(0)).Return(6, nil)
//
//	r := storage.Concat(a, b, c)
//
//	buf := make([]byte, 15)
//	n, actualErr := r.ReadAt(buf, 0)
//	assert.Equal(t, actualErr, io.ErrUnexpectedEOF)
//	assert.Equal(t, 8, n)
//}
