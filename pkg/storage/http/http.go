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
	"context"
	"fmt"
	"net/http"
	stdurl "net/url"
	"os"
	stdpath "path"
	"time"

	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/storage/driver"
)

const (
	CtxAuthorization = "CTX_AUTHORIZATION"
	CtxTimeout       = "CTX_TIMEOUT"
)

type Driver struct {
	defaultTransport http.RoundTripper
}

func (d *Driver) Name() string {
	return "httpdriver"
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (driver.Object, error) {
	u, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	c := d.newClient(ctx)
	return NewObject(c, url, u, size), nil
}

func (d *Driver) Remove(ctx context.Context, url string) error {
	u, err := d.parseURL(url)
	if err != nil {
		return err
	}

	c := d.newClient(ctx)
	return Delete(c, url, u)
}

func (d *Driver) Stat(ctx context.Context, url string) (os.FileInfo, error) {
	u, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	name := stdpath.Base(u.Path)

	c := d.newClient(ctx)

	size, err := Stat(c, url)
	if err != nil {
		return nil, err
	}

	return NewFileInfo(name, size), nil
}

func (d *Driver) parseURL(url string) (*stdurl.URL, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("httpdriver: parsing url: %+v", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("httpdriver: unsupported URI scheme %q", u.Scheme)
	}

	return u, nil
}

func (d *Driver) newClient(ctx context.Context) *http.Client {
	c := &http.Client{}
	if timeout, ok := TimeoutFromCtx(ctx); ok {
		c.Timeout = *timeout
	} else {
		c.Timeout = 30 * time.Second
	}

	if authz, ok := AuthzFromCtx(ctx); ok {
		c.Transport = httputil.WithAuthz(d.defaultTransport, *authz)
	} else {
		c.Transport = d.defaultTransport
	}
	return c
}

func RegisterDefaultDriver() {
	t := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          1024,
		MaxIdleConnsPerHost:   1024,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	httputil.AddDNSCache(t)

	d := &Driver{
		defaultTransport: httputil.WithMetrics(t, "http"),
	}
	driver.Register("http", d)
	driver.Register("https", d)
}

func logger() *zap.Logger {
	return zap.L().Named("httpdriver")
}
