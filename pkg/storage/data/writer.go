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
package datadriver

import (
	"bytes"

	"github.com/vincent-petithory/dataurl"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

type objectWriter struct {
	buf *bytes.Buffer
}

func (ow *objectWriter) Abort() {
	ow.buf = nil
}

func (ow *objectWriter) Commit() (storage.CommitInfo, error) {
	if ow.buf == nil {
		return nil, storage.CommitOnAbortedObjectWriter
	}

	durl := dataurl.New(ow.buf.Bytes(), "binary/octet-stream")

	return storage.NewCommitInfo(durl.String()), nil
}

func (ow *objectWriter) Write(p []byte) (n int, err error) {
	n, err = ow.buf.Write(p)
	return
}
