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
package driver

import (
	"context"
	"io"
	"os"
)

// Driver is the interface that must be implemented by a storage driver.
type Driver interface {
	// Name returns the display name of this driver
	Name() string

	// Open opens the Object for reading.
	Open(ctx context.Context, url string, size int64) (Object, error)

	// Stat returns a FileInfo describing the Object.
	Stat(ctx context.Context, url string) (os.FileInfo, error)
}

type Creator interface {
	// Create an ObjectWriter handle
	Create(ctx context.Context, url string) (ObjectWriter, error)
}

type Remover interface {
	// Remove an object
	Remove(ctx context.Context, url string) error
}

type Readdirer interface {
	// Readdir reads the contents of the directory and returns a slice
	// of FileInfo values, as would be returned by Stat, in directory
	// order.
	Readdir(ctx context.Context, url string) ([]os.FileInfo, error)
}

type Locker interface {
	// Lock acquires an advisory exclusive lock on the object
	// specified, potentially creating it if it does not already
	// exist.
	Lock(ctx context.Context, url string) (io.Closer, error)

	// RLock acquires an advisory shared lock on the object specified,
	// potentially creating it if it does not already exist.
	RLock(ctx context.Context, url string) (io.Closer, error)
}

// Useful for mocking
type ComprehensiveDriver interface {
	Driver
	Creator
	Remover
	Readdirer
	Locker
}
