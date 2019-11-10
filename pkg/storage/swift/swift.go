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
	"os"
	stdpath "path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/s3util"
	"github.com/NVIDIA/vdisc/pkg/storage/driver"
	"github.com/NVIDIA/vdisc/pkg/storage/http"
	"github.com/NVIDIA/vdisc/pkg/storage/s3"
)

type Driver struct {
	sess             *session.Session
	defaultTransport http.RoundTripper
}

func (d *Driver) Name() string {
	return "swiftdriver"
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (driver.Object, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	c := d.newClient(ctx, parsed.Account)

	return httpdriver.NewObject(c, url, parsed.URL, size), nil
}

func (d *Driver) Create(ctx context.Context, url string) (driver.ObjectWriter, error) {
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

func (d *Driver) Stat(ctx context.Context, url string) (os.FileInfo, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	name := stdpath.Base(parsed.URL.Path)

	c := d.newClient(ctx, parsed.Account)

	size, err := httpdriver.Stat(c, parsed.URL.String())
	if err != nil {
		return nil, err
	}

	return &finfo{
		name:    name,
		size:    size,
		mode:    0644,
		modTime: time.Unix(0, 0).UTC(),
	}, nil
}

func (d *Driver) Readdir(ctx context.Context, url string) ([]os.FileInfo, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	c := d.newClient(ctx, parsed.Account)

	var results []os.FileInfo
	err = s3util.ListBucket(c, parsed.URL.String(), func(page *s3util.ListBucketPage) error {
		for _, entry := range page.Version {
			if !entry.IsLatest {
				continue
			}
			parts := strings.Split(entry.Key, "/")
			name := parts[len(parts)-1]
			if name == "" {
				continue
			}
			modTime, err := entry.Modified()
			if err != nil {
				return err
			}
			results = append(results, &finfo{
				name:    name,
				size:    entry.Size,
				mode:    0644,
				modTime: modTime,
				isDir:   false,
				etag:    entry.ETag,
				version: entry.VersionId,
			})
		}

		for _, entry := range page.CommonPrefixes {
			parts := strings.Split(entry.Prefix, "/")
			results = append(results, &finfo{
				name:    parts[len(parts)-2],
				size:    4096,
				mode:    0755,
				modTime: time.Unix(0, 0).UTC(),
				isDir:   true,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
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

	c.Transport = s3util.NewSigningRoundTripper(httputil.WithRetries(d.defaultTransport), credentials.NewCredentials(prov), region)
	return c
}

type parsedURL struct {
	URL       *stdurl.URL
	Account   string
	Container string
	Key       string
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

	driver.Register("swift", &Driver{
		sess:             session.Must(session.NewSession()),
		defaultTransport: httputil.WithMetrics(t, "swift"),
	})
}
