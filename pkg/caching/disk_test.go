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
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/NVIDIA/vdisc/pkg/caching"
	"github.com/NVIDIA/vdisc/pkg/storage"
	"github.com/NVIDIA/vdisc/pkg/storage/data"
	"github.com/NVIDIA/vdisc/pkg/storage/driver"
	"github.com/NVIDIA/vdisc/pkg/storage/mock"
	"github.com/NVIDIA/vdisc/pkg/storage/zero"
)

type mockCloser struct {
	mock.Mock
}

func (mc *mockCloser) Close() error {
	ret := mc.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type fakeObj struct {
	mockdriver.XattrObject
	s string
}

func (fo *fakeObj) ReadAt(p []byte, off int64) (int, error) {
	return strings.NewReader(fo.s).ReadAt(p, off)
}

func TestDiskCacheMiss0(t *testing.T) {
	undo := driver.ClearRegistry()
	defer undo()

	drvr := &mockdriver.ComprehensiveDriver{}
	datadriver.RegisterDefaultDriver()
	zerodriver.RegisterDefaultDriver()
	driver.Register("file", drvr)

	obj, err := storage.Open("data:,Hello%2C%20World!")
	if err != nil {
		t.Fatal(err)
	}

	slicer := caching.NewDiskSlicer("/test", 4096)
	defer slicer.Wait()
	cache := caching.NewCache(slicer, 0)
	obj = cache.WithCaching(obj)

	// first, we expect the cache to look for the object
	drvr.On("Open", context.Background(), "/test/v0/60/db8a29f0b095e1b5135740e98ff420", int64(-1)).Return(nil, os.ErrNotExist)

	// when the object isn't found, we expect the cache to acquire a lock for the object
	unlock := &mockCloser{}
	drvr.On("Lock", context.Background(), "/test/v0/60/.lock.db8a29f0b095e1b5135740e98ff420").Return(unlock, nil)

	// Once the lock is acquired, the cache will create a temporary file
	w := &mockdriver.XattrObjectWriter{}
	drvr.On("Create", context.Background(), "/test/v0/60/db8a29f0b095e1b5135740e98ff420").Return(w, nil)

	// Then the cache will write the content to the object and set the appropriate xattrs
	w.On("Write", []byte("Hello, World!")).Return(13, nil)
	w.On("SetXattr", caching.XattrKey, []byte("{\"url\":\"data:,Hello%2C%20World!\",\"off\":0,\"len\":13}")).Return(nil)
	w.On("SetXattr", caching.XattrChecksum, []byte{0x7f, 0xe4, 0x0f, 0x08, 0xf8, 0xac, 0x9a, 0xc4}).Return(nil)

	// Finally, the cache will commit the object, and
	commitInfo := &mockdriver.CommitInfo{}
	w.On("Commit").Return(commitInfo, nil)
	w.On("Abort").Return(nil)

	// And the lock will be released
	unlock.On("Close").Return(nil)

	actual, err := ioutil.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []byte("Hello, World!"), actual)
}

func TestDiskCacheMiss1(t *testing.T) {
	undo := driver.ClearRegistry()
	defer undo()

	drvr := &mockdriver.ComprehensiveDriver{}
	datadriver.RegisterDefaultDriver()
	zerodriver.RegisterDefaultDriver()
	driver.Register("file", drvr)

	obj, err := storage.Open("data:,Hello%2C%20World!")
	if err != nil {
		t.Fatal(err)
	}

	slicer := caching.NewDiskSlicer("/test", 4096)
	defer slicer.Wait()
	cache := caching.NewCache(slicer, 0)
	obj = cache.WithCaching(obj)

	// first, we expect the cache to look for the object
	xobj := &fakeObj{s: "Hello, World!"}
	drvr.On("Open", context.Background(), "/test/v0/60/db8a29f0b095e1b5135740e98ff420", int64(-1)).Return(xobj, nil)

	// Since the cache found an object, it needs to check the key xattr
	xobj.On("GetXattr", caching.XattrKey).Return([]byte("garbage"), nil)
	xobj.On("Close").Return(nil)

	// since the cache object isn't a match, we expect the cache to acquire a lock for the object
	unlock := &mockCloser{}
	drvr.On("Lock", context.Background(), "/test/v0/60/.lock.db8a29f0b095e1b5135740e98ff420").Return(unlock, nil)

	// Once the lock is acquired, the cache will create a temporary file
	w := &mockdriver.XattrObjectWriter{}
	drvr.On("Create", context.Background(), "/test/v0/60/db8a29f0b095e1b5135740e98ff420").Return(w, nil)

	// Then the cache will write the content to the object and set the appropriate xattrs
	w.On("Write", []byte("Hello, World!")).Return(13, nil)
	w.On("SetXattr", caching.XattrKey, []byte("{\"url\":\"data:,Hello%2C%20World!\",\"off\":0,\"len\":13}")).Return(nil)
	w.On("SetXattr", caching.XattrChecksum, []byte{0x7f, 0xe4, 0x0f, 0x08, 0xf8, 0xac, 0x9a, 0xc4}).Return(nil)

	// Finally, the cache will commit the object, and
	commitInfo := &mockdriver.CommitInfo{}
	w.On("Commit").Return(commitInfo, nil)
	w.On("Abort").Return(nil)

	// And the lock will be released
	unlock.On("Close").Return(nil)

	actual, err := ioutil.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("Hello, World!"), actual)
}

func TestDiskCacheHit0(t *testing.T) {
	undo := driver.ClearRegistry()
	defer undo()

	drvr := &mockdriver.ComprehensiveDriver{}
	datadriver.RegisterDefaultDriver()
	zerodriver.RegisterDefaultDriver()
	driver.Register("file", drvr)

	obj, err := storage.Open("data:,Hello%2C%20World!")
	if err != nil {
		t.Fatal(err)
	}

	slicer := caching.NewDiskSlicer("/test", 4096)
	defer slicer.Wait()
	cache := caching.NewCache(slicer, 0)
	obj = cache.WithCaching(obj)

	// first, we expect the cache to look for the object
	xobj := &fakeObj{s: "Hello, World!"}
	drvr.On("Open", context.Background(), "/test/v0/60/db8a29f0b095e1b5135740e98ff420", int64(-1)).Return(xobj, nil)

	// Since the cache found an object, it needs to check the key xattr
	xobj.On("GetXattr", caching.XattrKey).Return([]byte("{\"url\":\"data:,Hello%2C%20World!\",\"off\":0,\"len\":13}"), nil)
	xobj.On("Close").Return(nil)

	actual, err := ioutil.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("Hello, World!"), actual)
}

func TestDiskCacheHit1(t *testing.T) {
	undo := driver.ClearRegistry()
	defer undo()

	drvr := &mockdriver.ComprehensiveDriver{}
	datadriver.RegisterDefaultDriver()
	zerodriver.RegisterDefaultDriver()
	driver.Register("file", drvr)

	obj, err := storage.Open("data:,Hello%2C%20World!")
	if err != nil {
		t.Fatal(err)
	}

	slicer := caching.NewDiskSlicer("/test", 10)
	defer slicer.Wait()
	cache := caching.NewCache(slicer, 0)
	obj = cache.WithCaching(obj)

	// first, we expect the cache to look for the object
	xobj := &fakeObj{s: "Hello, Wor"}
	drvr.On("Open", context.Background(), "/test/v0/46/a075896c3d16a41e4000bf5ba9b79d", int64(-1)).Return(xobj, nil)

	// Since the cache found an object, it needs to check the key xattr
	xobj.On("GetXattr", caching.XattrKey).Return([]byte("{\"url\":\"data:,Hello%2C%20World!\",\"off\":0,\"len\":10}"), nil)
	xobj.On("Close").Return(nil)

	// first, we expect the cache to look for the object
	xobj2 := &fakeObj{s: "ld!"}
	drvr.On("Open", context.Background(), "/test/v0/67/260b3bab0abe40bd657ba0bf92729a", int64(-1)).Return(xobj2, nil)

	// Since the cache found an object, it needs to check the key xattr
	xobj2.On("GetXattr", caching.XattrKey).Return([]byte("{\"url\":\"data:,Hello%2C%20World!\",\"off\":10,\"len\":3}"), nil)
	xobj2.On("Close").Return(nil)

	actual, err := ioutil.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("Hello, World!"), actual)
}
