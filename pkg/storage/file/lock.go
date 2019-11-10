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
	"context"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func lock(ctx context.Context, url string, flag int) (io.Closer, error) {
	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, os.FileMode(0600))
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), flag); err != nil {
		f.Close()
		return nil, err
	}
	return &unlock{f}, nil
}

type unlock struct {
	f *os.File
}

func (u *unlock) Close() error {
	defer u.f.Close()
	return syscall.Flock(int(u.f.Fd()), syscall.LOCK_UN)
}
