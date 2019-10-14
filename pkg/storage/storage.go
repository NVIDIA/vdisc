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
	"context"
	"errors"
	"io"
	"os"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"

	"github.com/NVIDIA/vdisc/pkg/storage/data"
	"github.com/NVIDIA/vdisc/pkg/storage/file"
	"github.com/NVIDIA/vdisc/pkg/storage/http"
	"github.com/NVIDIA/vdisc/pkg/storage/s3"
	"github.com/NVIDIA/vdisc/pkg/storage/swift"
	"github.com/NVIDIA/vdisc/pkg/storage/zero"
)

// AnonymousObject represents a read-only, fixed size, random access object.
type AnonymousObject interface {
	driver.AnonymousObject
}

// Object represents a AnonymousObject with a URL
type Object interface {
	driver.Object
}

// ObjectWriter is a handle for creating an Object
type ObjectWriter interface {
	driver.ObjectWriter
}

type CommitInfo interface {
	driver.CommitInfo
}

// Open opens the Object for reading.
func Open(url string) (Object, error) {
	return OpenContextSize(context.Background(), url, -1)
}

// Open opens the Object with the context and declared size.
func OpenContextSize(ctx context.Context, url string, size int64) (Object, error) {
	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}
	return drvr.Open(ctx, url, size)
}

// Create an ObjectWriter handle
func Create(url string) (ObjectWriter, error) {
	return CreateContext(context.Background(), url)
}

// Create an ObjectWriter handle
func CreateContext(ctx context.Context, url string) (ObjectWriter, error) {
	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}

	switch mdrvr := drvr.(type) {
	case driver.Writable:
		return mdrvr.Create(ctx, url)
	default:
		return nil, errors.New(drvr.Name() + ": create not implemented")
	}
}

// Remove an object
func Remove(url string) error {
	return RemoveContext(context.Background(), url)
}

// Remove an object
func RemoveContext(ctx context.Context, url string) error {
	drvr, err := driver.Find(url)
	if err != nil {
		return err
	}

	switch mdrvr := drvr.(type) {
	case driver.Removable:
		return mdrvr.Remove(ctx, url)
	default:
		return errors.New(drvr.Name() + ": remove not implemented")
	}
}

// Stat returns a FileInfo describing the Object
func Stat(url string) (os.FileInfo, error) {
	return StatContext(context.Background(), url)
}

// Stat returns a FileInfo describing the Object
func StatContext(ctx context.Context, url string) (os.FileInfo, error) {
	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}
	return drvr.Stat(ctx, url)
}

// Readdir reads the contents of the directory and returns a slice of
// FileInfo values, as would be returned by Stat, in directory order.
func Readdir(url string) ([]os.FileInfo, error) {
	return ReaddirContext(context.Background(), url)
}

// Readdir reads the contents of the directory and returns a slice of
// FileInfo values, as would be returned by Stat, in directory order.
func ReaddirContext(ctx context.Context, url string) ([]os.FileInfo, error) {
	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}

	switch rdrvr := drvr.(type) {
	case driver.Readdirable:
		return rdrvr.Readdir(ctx, url)
	default:
		return nil, errors.New(drvr.Name() + ": readdir not implemented")
	}
}

// Lock acquires an advisory exclusive lock on the object
// specified, potentially creating it if it does not already
// exist.
func Lock(url string) (io.Closer, error) {
	return LockContext(context.Background(), url)
}

// Lock acquires an advisory exclusive lock on the object
// specified, potentially creating it if it does not already
// exist.
func LockContext(ctx context.Context, url string) (io.Closer, error) {
	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}

	switch ldrvr := drvr.(type) {
	case driver.Lockable:
		return ldrvr.Lock(ctx, url)
	default:
		return nil, errors.New(drvr.Name() + ": lock not implemented")
	}
}

// RLock acquires an advisory shared lock on the object specified,
// potentially creating it if it does not already exist.
func RLock(url string) (io.Closer, error) {
	return RLockContext(context.Background(), url)
}

// RLock acquires an advisory shared lock on the object specified,
// potentially creating it if it does not already exist.
func RLockContext(ctx context.Context, url string) (io.Closer, error) {
	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}
	switch ldrvr := drvr.(type) {
	case driver.Lockable:
		return ldrvr.RLock(ctx, url)
	default:
		return nil, errors.New(drvr.Name() + ": rlock not implemented")
	}
}

// WithURL gives a URL to an AnonymousObject
func WithURL(o AnonymousObject, url string) Object {
	return &withURL{o, url}
}

type withURL struct {
	o   AnonymousObject
	url string
}

func (wu *withURL) Close() error {
	return wu.o.Close()
}

func (wu *withURL) Read(p []byte) (int, error) {
	return wu.o.Read(p)
}

func (wu *withURL) ReadAt(p []byte, off int64) (int, error) {
	return wu.o.ReadAt(p, off)
}

func (wu *withURL) Seek(offset int64, whence int) (int64, error) {
	return wu.o.Seek(offset, whence)
}

func (wu *withURL) Size() int64 {
	return wu.o.Size()
}

func (wu *withURL) URL() string {
	return wu.url
}

func init() {
	datadriver.RegisterDefaultDriver()
	filedriver.RegisterDefaultDriver()
	httpdriver.RegisterDefaultDriver()
	s3driver.RegisterDefaultDriver()
	swiftdriver.RegisterDefaultDriver()
	zerodriver.RegisterDefaultDriver()
}
