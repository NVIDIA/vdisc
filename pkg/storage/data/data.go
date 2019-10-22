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
	"context"
	"errors"
	"io"
	"os"

	"github.com/vincent-petithory/dataurl"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

// Driver is the data URI scheme storage driver.
// See https://tools.ietf.org/html/rfc2397.
type Driver struct{}

func (d *Driver) Open(ctx context.Context, url string, size int64) (driver.Object, error) {
	du, err := dataurl.DecodeString(url)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(du.Data)
	sr := io.NewSectionReader(r, 0, int64(len(du.Data)))
	return &object{
		url: url,
		sr:  sr,
	}, nil
}

func (d *Driver) Create(ctx context.Context, url string) (driver.ObjectWriter, error) {
	du, err := dataurl.DecodeString(url)
	if err != nil {
		return nil, err
	}

	return &objectWriter{
		buf: bytes.NewBuffer(du.Data),
	}, nil
}

func (d *Driver) Remove(ctx context.Context, url string) error {
	_, err := dataurl.DecodeString(url)
	if err != nil {
		return err
	}
	return errors.New("datadriver: remove not implemented")
}

func (d *Driver) Stat(ctx context.Context, url string) (os.FileInfo, error) {
	du, err := dataurl.DecodeString(url)
	if err != nil {
		return nil, err
	}

	return &finfo{du}, nil
}

func RegisterDefaultDriver() {
	driver.Register("data", &Driver{})
}
