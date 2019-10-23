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
package swiftdriver

import (
	"os"
	"time"
)

type finfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	etag    string
	version string
}

func (fi *finfo) Name() string {
	return fi.name
}

func (fi *finfo) Size() int64 {
	return fi.size
}

func (fi *finfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *finfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *finfo) IsDir() bool {
	return fi.isDir
}

func (fi *finfo) Sys() interface{} {
	return fi
}

func (fi *finfo) ETag() string {
	return fi.etag
}

func (fi *finfo) Version() string {
	if fi.version == "null" {
		return ""
	}
	return fi.version
}
