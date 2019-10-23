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
	"io/ioutil"
	stdurl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

// Driver is the file URI scheme storage driver.
type Driver struct{}

func (d *Driver) Open(ctx context.Context, url string, size int64) (driver.Object, error) {
	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if size < 0 {
		finfo, err := f.Stat()
		if err != nil {
			return nil, err
		}
		size = finfo.Size()
	}
	return &object{
		url:  url,
		f:    f,
		size: size,
	}, nil
}

func (d *Driver) Create(ctx context.Context, url string) (driver.ObjectWriter, error) {
	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	f, err := ioutil.TempFile(dir, ".tmp.filedriver")
	if err != nil {
		return nil, err
	}

	return &objectWriter{
		path: path,
		f:    f,
	}, nil
}

func (d *Driver) Remove(ctx context.Context, url string) error {
	path, err := urlToPath(url)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

func (d *Driver) Stat(ctx context.Context, url string) (os.FileInfo, error) {
	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	return os.Stat(path)
}

func (d *Driver) Readdir(ctx context.Context, url string) ([]os.FileInfo, error) {
	path, err := urlToPath(url)
	if err != nil {
		return nil, err
	}

	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	return dir.Readdir(0)
}

func urlToPath(url string) (string, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return "", err
	}

	if u.Scheme == "" {
		u.Scheme = "file"
		if !strings.HasPrefix(u.Path, "/") {
			u.Opaque = u.Path
			u.Path = ""
			u.RawPath = ""
		}
	}

	var path string
	if len(u.Opaque) == 0 {
		path = u.Path
	} else {
		path = u.Opaque
	}

	return filepath.Clean(path), nil
}

func RegisterDefaultDriver() {
	driver.Register("file", &Driver{})
}
