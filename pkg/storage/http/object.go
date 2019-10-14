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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	stdurl "net/url"
	"os"

	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/safecast"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

// NewObject opens an HTTP URL as an Object. If size is negative,
// a HEAD request will be performed to determine the actual size.
func NewObject(client *http.Client, u *stdurl.URL, size int64) (storage.AnonymousObject, error) {
	if size < 0 {
		resp, err := client.Head(u.String())
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		io.Copy(ioutil.Discard, resp.Body)

		if resp.StatusCode == 404 {
			return nil, os.ErrNotExist
		}

		if resp.StatusCode == 405 {
			// Server doesn't support HEAD, download the whole resource up front.
			resp, err := client.Get(u.String())
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				return nil, fmt.Errorf("io: http %d for %s", resp.StatusCode, u)
			}
			body, err := ioutil.ReadAll(resp.Body)
			return &static{bytes.NewReader(body)}, nil
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("io: http %d for %s", resp.StatusCode, u)
		}

		if resp.ContentLength < 0 {
			return nil, fmt.Errorf("io: bad content length %d for %s", resp.ContentLength, u)
		}
		size = resp.ContentLength
	}

	return &object{
		client: client,
		u:      u,
		size:   size,
	}, nil
}

type static struct {
	*bytes.Reader
}

func (s *static) Close() error {
	return nil
}

type object struct {
	client *http.Client
	u      *stdurl.URL
	size   int64
	pos    int64
	closed bool
}

func (o *object) Close() error {
	o.closed = true
	return nil
}

func (o *object) Size() int64 {
	return o.size
}

func (o *object) Read(p []byte) (n int, err error) {
	n, err = o.ReadAt(p, o.pos)
	o.pos += int64(n)
	return
}

func (o *object) ReadAt(p []byte, off int64) (n int, err error) {
	if o.closed {
		err = os.ErrClosed
		return
	}

	if len(p) == 0 {
		return
	}

	if off >= o.size {
		err = io.EOF
		return
	}

	req, err := http.NewRequest("GET", o.u.String(), nil)
	if err != nil {
		err = fmt.Errorf("io: bad url?: %#v: %s", o.u, err)
		return
	}

	end := off + int64(len(p))
	if end > o.size {
		end = o.size
	}

	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", off, end-1))
	req.Header.Add("Accept-Encoding", "identity")

	resp, err := o.client.Do(req)
	if err != nil {
		err = fmt.Errorf("io: http get: %s for %s", err, o.u)
		return
	}
	defer resp.Body.Close()

	logger().Debug("GET", zap.String("url", o.u.String()), zap.String("range", fmt.Sprintf("bytes=%d-%d", off, end-1)), zap.Int("status", resp.StatusCode))

	if resp.StatusCode != 206 {
		// throw away the body so the connection can be reused
		io.Copy(ioutil.Discard, resp.Body)
		if resp.StatusCode == 404 {
			err = os.ErrNotExist
			return
		}
		err = fmt.Errorf("io: http get: http %d for %s", resp.StatusCode, o.u)
		return
	}

	contentRange, err := httputil.GetContentRange(resp)
	if err != nil {
		err = fmt.Errorf("io: http get: %s for %s", err, o.u)
		return
	}

	if contentRange.Total < safecast.Int64ToUint64(o.size) {
		err = fmt.Errorf("io: http get: content-range total %d less than expected size %d for %s", err, contentRange.Total, o.size, o.u)
		return
	}

	if safecast.Uint64ToInt64(contentRange.Len()) != resp.ContentLength {
		err = fmt.Errorf("io: http get: mismatch: Content-Range total %d, Content-Length %d for %s", err, contentRange.Total, resp.ContentLength, o.u)
		return
	}

	if contentRange.First != safecast.Int64ToUint64(off) || contentRange.Last != safecast.Int64ToUint64(end-1) {
		err = fmt.Errorf("io: http get: range/content-range mismatch for %s: \"range=%d-%d\" vs %q", o.u, off, end-1, resp.Header.Get("Content-Range"))
		return
	}

	n, err = io.ReadFull(resp.Body, p[:resp.ContentLength])
	if err == nil && safecast.IntToUint64(n) < contentRange.Len() {
		err = fmt.Errorf("io: http get: Content-Length=%d, read=%d, %s for %s", resp.ContentLength, n, io.ErrUnexpectedEOF, o.u)
		return
	}

	return
}

func (o *object) Seek(offset int64, whence int) (int64, error) {
	if o.closed {
		return 0, os.ErrClosed
	}

	switch whence {
	case io.SeekCurrent:
		o.pos = o.pos + offset
	case io.SeekStart:
		o.pos = offset
	case io.SeekEnd:
		if o.size < 0 {
			return 0, fmt.Errorf("unknown length")
		}
		o.pos = o.size + offset
	}

	if o.pos < 0 {
		o.pos = 0
	} else if o.size >= 0 && o.pos > o.size {
		o.pos = o.size
	}

	return o.pos, nil
}
