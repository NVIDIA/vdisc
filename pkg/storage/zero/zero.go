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
package zerodriver

import (
	"context"
	"fmt"
	stdurl "net/url"
	"os"
	"strconv"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

// Driver is the data URI scheme storage driver.
// See https://tools.ietf.org/html/rfc2397.
type Driver struct{}

func (d *Driver) Name() string {
	return "zerodriver"
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (driver.Object, error) {
	usize, err := urlToSize(url)
	if err != nil {
		return nil, err
	}

	return &object{
		url:  url,
		size: usize,
	}, nil
}

func (d *Driver) Stat(ctx context.Context, url string) (os.FileInfo, error) {
	usize, err := urlToSize(url)
	if err != nil {
		return nil, err
	}

	return &finfo{usize}, nil
}

func urlToSize(url string) (int64, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return -1, err
	}

	if u.Scheme != "zero" {
		return -1, fmt.Errorf("zerodriver: unsupported URI scheme %q", u.Scheme)
	}

	usize, err := strconv.ParseInt(u.Opaque, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("zerodriver: invalid URI %q", url)
	}
	return usize, nil
}

func RegisterDefaultDriver() {
	driver.Register("zero", &Driver{})
}
