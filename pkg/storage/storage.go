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
	"fmt"
	"io"
	stdurl "net/url"
	"sort"
	"strings"
	"sync"
)

// AnonymousObject represents a read-only, fixed size, random access object.
type AnonymousObject interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	Size() int64
}

// Object represents a AnonymousObject with a URL
type Object interface {
	AnonymousObject

	// URL is the location of this object.
	URL() string
}

// ObjectWriter is a handle for creating an Object
type ObjectWriter interface {
	io.Writer
	Abort()
	Commit() (CommitInfo, error)
}

type CommitInfo interface {
	// ObjectURL returns the final URL of the committed object
	ObjectURL() string
}

var CommitOnAbortedObjectWriter = errors.New("commit on aborted ObjectWriter")

// NewCommitInfo is a helper function for storage drivers
func NewCommitInfo(url string) CommitInfo {
	return &commitInfo{url}
}

type commitInfo struct {
	url string
}

func (ci *commitInfo) ObjectURL() string {
	return ci.url
}

// Driver is the interface that must be implemented by a storage driver.
type Driver interface {
	// Open opens the Object for reading.
	Open(ctx context.Context, url string, size int64) (Object, error)

	// Create an ObjectWriter handle
	Create(ctx context.Context, url string) (ObjectWriter, error)

	// Remove an object
	Remove(ctx context.Context, url string) error
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

// Register makes a storage driver available by the provided URL scheme.
// If Register is called twice with the same scheme or if driver is nil,
// it panics.
func Register(scheme string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("storage: Register driver is nil")
	}
	if _, dup := drivers[scheme]; dup {
		panic("storage: Register called twice for driver " + scheme)
	}
	drivers[scheme] = driver
}

// Drivers returns a sorted list of the URL schemes of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var list []string
	for scheme := range drivers {
		list = append(list, scheme)
	}
	sort.Strings(list)
	return list
}

// Open opens the Object for reading.
func Open(url string) (Object, error) {
	return OpenContextSize(context.Background(), url, -1)
}

// Open opens the Object with the context and declared size.
func OpenContextSize(ctx context.Context, url string, size int64) (Object, error) {
	drvr, err := findDriver(url)
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
	drvr, err := findDriver(url)
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
	drvr, err := findDriver(url)
	if err != nil {
		return err
	}
	return drvr.Remove(ctx, url)
}

func findDriver(url string) (Driver, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		u.Scheme = "file"
		if !strings.HasPrefix(u.Path, "/") {
			u.Opaque = u.Path
			u.Path = ""
			u.RawPath = ""
		}
	}

	driversMu.RLock()
	defer driversMu.RUnlock()

	drvr, ok := drivers[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("storage: unknown driver %q (forgotten import?)", u.Scheme)
	}
	return drvr, nil
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
