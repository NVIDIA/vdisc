// Copyright © 2019 NVIDIA Corporation
package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	stdurl "net/url"
	"time"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

const (
	CtxAuthorization = "CTX_AUTHORIZATION"
	CtxTimeout       = "CTX_TIMEOUT"
)

type Driver struct {
	defaultTransport http.RoundTripper
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (storage.Object, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("httpdriver: unsupported URI scheme %q", u.Scheme)
	}

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

	anon, err := NewObject(c, u, size)
	if err != nil {
		return nil, err
	}

	return storage.WithURL(anon, url), nil
}

func (d *Driver) Create(ctx context.Context, url string) (storage.ObjectWriter, error) {
	return nil, errors.New("httpdriver: create not implemented")
}

func init() {
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
	storage.Register("http", d)
	storage.Register("https", d)
}
