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
package httpdriver

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func Stat(c *http.Client, url string) (int64, error) {
	resp, err := c.Head(url)
	if err != nil {
		return -1, err
	}

	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	if resp.StatusCode == 404 {
		return -1, os.ErrNotExist
	}

	if resp.StatusCode == 405 {
		// Server doesn't support HEAD, download the whole resource up front.
		resp, err := c.Get(url)
		if err != nil {
			return -1, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return -1, fmt.Errorf("http get %q: HTTP %d", url, resp.StatusCode)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return -1, fmt.Errorf("http get %q: %+v", url, err)
		}

		return int64(len(body)), nil
	}

	if resp.StatusCode != 200 {
		return -1, fmt.Errorf("http head %q: HTTP %d", url, resp.StatusCode)
	}

	if resp.ContentLength < 0 {
		return -1, fmt.Errorf("http head %q: bad content length %d", url, resp.ContentLength)
	}
	return resp.ContentLength, nil
}

func NewFileInfo(name string, size int64) os.FileInfo {
	return &finfo{name, size}
}

type finfo struct {
	name string
	size int64
}

// base name of the file
func (fi *finfo) Name() string {
	return fi.name
}

// length in bytes for regular files; system-dependent for others
func (fi *finfo) Size() int64 {
	return fi.size
}

// file mode bits
func (fi *finfo) Mode() os.FileMode {
	return 0444
}

// modification time
func (fi *finfo) ModTime() time.Time {
	return time.Unix(0, 0).UTC()
}

// abbreviation for Mode().IsDir()
func (fi *finfo) IsDir() bool {
	return false
}

// underlying data source (can return nil)
func (fi *finfo) Sys() interface{} {
	return nil
}
