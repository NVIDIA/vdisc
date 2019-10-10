// Copyright © 2019 NVIDIA Corporation
package filedriver

import (
	"context"
	"io/ioutil"
	stdurl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

// Driver is the file URI scheme storage driver.
type Driver struct{}

func (d *Driver) Open(ctx context.Context, url string, size int64) (storage.Object, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
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

func (d *Driver) Create(ctx context.Context, url string) (storage.ObjectWriter, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
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

	path = filepath.Clean(path)

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

func init() {
	storage.Register("file", &Driver{})
}
