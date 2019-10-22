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
	"os"
	"sync"

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
	registerDefaultsOnce.Do(registerDefaults)

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
	registerDefaultsOnce.Do(registerDefaults)

	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}
	return drvr.Create(ctx, url)
}

// Remove an object
func Remove(url string) error {
	return RemoveContext(context.Background(), url)
}

// Remove an object
func RemoveContext(ctx context.Context, url string) error {
	registerDefaultsOnce.Do(registerDefaults)

	drvr, err := driver.Find(url)
	if err != nil {
		return err
	}
	return drvr.Remove(ctx, url)
}

// Stat returns a FileInfo describing the Object
func Stat(url string) (os.FileInfo, error) {
	return StatContext(context.Background(), url)
}

// Stat returns a FileInfo describing the Object
func StatContext(ctx context.Context, url string) (os.FileInfo, error) {
	registerDefaultsOnce.Do(registerDefaults)

	drvr, err := driver.Find(url)
	if err != nil {
		return nil, err
	}
	return drvr.Stat(ctx, url)
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

var disableRegisterDefaults bool
var registerDefaultsOnce sync.Once

// DisableDefaultDrivers is typically used in tests to disable registring the default storage drivers
func DisableDefaultDrivers() {
	disableRegisterDefaults = true
}

func registerDefaults() {
	if !disableRegisterDefaults {
		datadriver.RegisterDefaultDriver()
		filedriver.RegisterDefaultDriver()
		httpdriver.RegisterDefaultDriver()
		s3driver.RegisterDefaultDriver()
		swiftdriver.RegisterDefaultDriver()
		zerodriver.RegisterDefaultDriver()
	}
}
