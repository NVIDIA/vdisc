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
	"os"
)

// Driver is the interface that must be implemented by a storage driver.
type Driver interface {
	// Open opens the Object for reading.
	Open(ctx context.Context, url string, size int64) (Object, error)

	// Create an ObjectWriter handle
	Create(ctx context.Context, url string) (ObjectWriter, error)

	// Remove an object
	Remove(ctx context.Context, url string) error

	// Stat returns a FileInfo describing the Object.
	Stat(ctx context.Context, url string) (os.FileInfo, error)
}
