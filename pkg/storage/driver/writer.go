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
	"errors"
	"io"
)

// ObjectWriter is a handle for creating an Object
type ObjectWriter interface {
	io.Writer
	Abort()
	Commit() (CommitInfo, error)
}

type XattrObjectWriter interface {
	ObjectWriter
	SetXattr(name string, value []byte) error
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
