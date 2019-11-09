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
package ramdriver

import (
	"bytes"
	"errors"
	"sync/atomic"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

const (
	stateCreating int32 = iota
	stateClosed
	stateAborted
)

var ErrCommitAbortedWriter = errors.New("commit aborted object writer")

type writer struct {
	url    string
	path   string
	buf    bytes.Buffer
	state  int32
	commit func(string, []byte) error
}

func (w *writer) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *writer) Commit() (driver.CommitInfo, error) {
	if atomic.CompareAndSwapInt32(&w.state, stateCreating, stateClosed) {
		if err := w.commit(w.path, w.buf.Bytes()); err != nil {
			return nil, err
		}
		return driver.NewCommitInfo(w.url), nil
	} else if atomic.LoadInt32(&w.state) == stateAborted {
		return nil, ErrCommitAbortedWriter
	}
	return driver.NewCommitInfo(w.url), nil
}

func (w *writer) Abort() {
	atomic.CompareAndSwapInt32(&w.state, stateCreating, stateAborted)
}
