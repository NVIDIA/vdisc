// Copyright © 2019 NVIDIA Corporation
package datadriver

import (
	"bytes"
	"context"
	"io"

	"github.com/vincent-petithory/dataurl"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

// Driver is the data URI scheme storage driver.
// See https://tools.ietf.org/html/rfc2397.
type Driver struct{}

func (d *Driver) Open(ctx context.Context, url string, size int64) (storage.Object, error) {
	dataURL, err := dataurl.DecodeString(url)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(dataURL.Data)
	sr := io.NewSectionReader(r, 0, int64(len(dataURL.Data)))
	return &object{
		url: url,
		sr:  sr,
	}, nil
}

func (d *Driver) Create(ctx context.Context, url string) (storage.ObjectWriter, error) {
	dataURL, err := dataurl.DecodeString(url)
	if err != nil {
		return nil, err
	}

	return &objectWriter{
		buf: bytes.NewBuffer(dataURL.Data),
	}, nil
}

func init() {
	storage.Register("data", &Driver{})
}
