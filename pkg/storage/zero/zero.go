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
	"errors"
	"fmt"
	stdurl "net/url"
	"strconv"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

// Driver is the data URI scheme storage driver.
// See https://tools.ietf.org/html/rfc2397.
type Driver struct{}

func (d *Driver) Open(ctx context.Context, url string, size int64) (storage.Object, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "zero" {
		return nil, fmt.Errorf("zerodriver: unsupported URI scheme %q", u.Scheme)
	}

	usize, err := strconv.ParseInt(u.Opaque, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("zerodriver: invalid URI %q", url)
	}

	return &object{
		url:  url,
		size: usize,
	}, nil
}

func (d *Driver) Create(ctx context.Context, url string) (storage.ObjectWriter, error) {
	return nil, errors.New("zerodriver: create not implemented")
}

func (d *Driver) Remove(ctx context.Context, url string) error {
	return errors.New("zerodriver: remove not implemented")
}

func init() {
	storage.Register("zero", &Driver{})
}
