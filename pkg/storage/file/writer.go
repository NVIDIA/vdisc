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

package filedriver

import (
	"os"

	"golang.org/x/sys/unix"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

type objectWriter struct {
	path string
	f    *os.File
}

func (ow *objectWriter) Abort() {
	os.Remove(ow.f.Name())
	// TODO: log err
	ow.f.Close()
}

func (ow *objectWriter) Commit() (driver.CommitInfo, error) {
	if err := ow.f.Sync(); err != nil {
		return nil, err
	}

	if err := os.Rename(ow.f.Name(), ow.path); err != nil {
		return nil, err
	}

	return driver.NewCommitInfo(ow.path), nil
}

func (ow *objectWriter) Write(p []byte) (n int, err error) {
	n, err = ow.f.Write(p)
	return
}

func (ow *objectWriter) SetXattr(name string, value []byte) error {
	return unix.Fsetxattr(int(ow.f.Fd()), name, value, 0)
}
