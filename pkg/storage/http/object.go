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
	stdurl "net/url"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/safecast"
	"github.com/NVIDIA/vdisc/pkg/storage/driver"
	"github.com/NVIDIA/vdisc/pkg/unixcompat"
)

// NewObject opens an HTTP URL as an Object. If size is negative,
// a HEAD request will be performed to determine the actual size.
func NewObject(client *http.Client, url string, u *stdurl.URL, size int64) driver.Object {
	return &object{
		client: client,
		url:    url,
		u:      u,
		size:   size,
	}
}

type object struct {
	client   *http.Client
	url      string
	u        *stdurl.URL
	size     int64
	sizeOnce sync.Once
	sizeErr  error
	pos      int64
	closed   bool
}

func (o *object) URL() string {
	return o.url
}

func (o *object) Close() error {
	o.closed = true
	return nil
}

func (o *object) Size() int64 {
	o.sizeOnce.Do(func() {
		if o.size < 0 {
			o.size, o.sizeErr = Stat(o.client, o.u.String())
		}
	})
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

	// make sure we've computed the object size
	o.Size()

	if o.sizeErr != nil {
		err = o.sizeErr
		return
	}

	if off >= o.size {
		err = io.EOF
		return
	}

	req, err := http.NewRequest(http.MethodGet, o.u.String(), nil)
	if err != nil {
		err = fmt.Errorf("http get %q: %+v", o.url, err)
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
		err = fmt.Errorf("http get %q: %+v", o.url, err)
		return
	}
	defer resp.Body.Close()

	logger().Debug("GET", zap.String("url", o.url), zap.String("range", fmt.Sprintf("bytes=%d-%d", off, end-1)), zap.Int("status", resp.StatusCode))

	if resp.StatusCode != 206 {
		// throw away the body so the connection can be reused
		io.Copy(ioutil.Discard, resp.Body)
		if resp.StatusCode == 404 {
			err = os.ErrNotExist
			return
		}
		err = fmt.Errorf("http get %q: HTTP %d", o.url, resp.StatusCode)
		return
	}

	contentRange, err := httputil.GetContentRange(resp)
	if err != nil {
		err = fmt.Errorf("http get %q: %+v", o.url, err)
		return
	}

	if contentRange.Total < safecast.Int64ToUint64(o.size) {
		err = fmt.Errorf("http get %q: content-range total %d less than expected size %d", o.url, contentRange.Total, o.size)
		return
	}

	if safecast.Uint64ToInt64(contentRange.Len()) != resp.ContentLength {
		err = fmt.Errorf("http get %q: mismatch: Content-Range total %d, Content-Length %d", o.url, contentRange.Total, resp.ContentLength)
		return
	}

	if contentRange.First != safecast.Int64ToUint64(off) || contentRange.Last != safecast.Int64ToUint64(end-1) {
		err = fmt.Errorf("http get %q: range/content-range mismatch: \"range=%d-%d\" vs %q", o.url, off, end-1, resp.Header.Get("Content-Range"))
		return
	}

	n, err = sleepyReadFull(resp.Body, p[:resp.ContentLength])
	if err == nil && safecast.IntToUint64(n) < contentRange.Len() {
		err = fmt.Errorf("http get %q: Content-Length=%d, read=%d, %s", o.url, resp.ContentLength, n, io.ErrUnexpectedEOF)
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

// Like io.ReadFull but with nanosleeps
func sleepyReadFull(r io.Reader, buf []byte) (n int, err error) {
	min := len(buf)

	cond := n < min && err == nil
	for cond {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn

		cond = n < min && err == nil
		if cond {
			unixcompat.MaybeNanosleep(1 * time.Millisecond)
		}
	}
	if n >= min {
		err = nil
	} else if n > 0 && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}
