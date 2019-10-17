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
package swiftdriver

import (
	"context"
	"fmt"
	"net/http"
	stdurl "net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/s3util"
	"github.com/NVIDIA/vdisc/pkg/storage"
	"github.com/NVIDIA/vdisc/pkg/storage/http"
	"github.com/NVIDIA/vdisc/pkg/storage/s3"
)

type Driver struct {
	sess             *session.Session
	defaultTransport http.RoundTripper
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (storage.Object, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	c := d.newClient(ctx, parsed.Account)

	return httpdriver.NewObject(c, url, parsed.URL, size), nil
}

func (d *Driver) Create(ctx context.Context, url string) (storage.ObjectWriter, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	creds := credentials.NewCredentials(NewSwiftEnvProvider(parsed.Account))

	region, ok := RegionFromCtx(ctx)
	if !ok {
		region = GetSwiftRegion()
	}

	config := aws.NewConfig().
		WithRegion(region).
		WithEndpoint(fmt.Sprintf("https://%s", parsed.URL.Host)).
		WithS3ForcePathStyle(true).
		WithCredentials(creds).
		WithMaxRetries(100)

	svc := s3.New(d.sess, config)

	return s3driver.NewObjectWriter(svc, parsed.Container, parsed.Key, url), nil
}

func (d *Driver) Remove(ctx context.Context, url string) error {
	parsed, err := d.parseURL(url)
	if err != nil {
		return err
	}

	c := d.newClient(ctx, parsed.Account)

	return httpdriver.Delete(c, url, parsed.URL)
}

func (d *Driver) parseURL(url string) (*parsedURL, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "swift" {
		return nil, fmt.Errorf("swiftdriver: unexpected scheme %q", u.Scheme)
	}

	if !u.IsAbs() {
		return nil, fmt.Errorf("swiftdriver: url must be absolute: %q", url)
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) < 4 || len(parts[0]) != 0 {
		return nil, fmt.Errorf("swiftdriver: invalid url: %q", url)
	}

	u.Scheme = "https"
	u.Path = "/" + strings.Join(parts[2:], "/")

	return &parsedURL{
		URL:       u,
		Account:   parts[1],
		Container: parts[2],
		Key:       "/" + strings.Join(parts[3:], "/"),
	}, nil
}

func (d *Driver) newClient(ctx context.Context, account string) *http.Client {
	c := &http.Client{}
	if timeout, ok := TimeoutFromCtx(ctx); ok {
		c.Timeout = *timeout
	} else {
		c.Timeout = 30 * time.Second
	}

	var prov credentials.Provider
	if creds, ok := CredentialsFromCtx(ctx); ok {
		prov = NewSwiftCtxProvider(account, creds)
	} else {
		prov = NewSwiftEnvProvider(account)
	}

	region, ok := RegionFromCtx(ctx)
	if !ok {
		region = GetSwiftRegion()
	}

	c.Transport = httputil.WithRetries(s3util.NewSigningRoundTripper(d.defaultTransport, credentials.NewCredentials(prov), region))
	return c
}

type parsedURL struct {
	URL       *stdurl.URL
	Account   string
	Container string
	Key       string
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

	storage.Register("swift", &Driver{
		sess:             session.Must(session.NewSession()),
		defaultTransport: httputil.WithMetrics(t, "swift"),
	})
}
