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

package s3driver

import (
	"context"
	"fmt"
	"net/http"
	stdurl "net/url"
	"os"
	stdpath "path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/s3util"
	"github.com/NVIDIA/vdisc/pkg/storage/driver"
	"github.com/NVIDIA/vdisc/pkg/storage/http"
)

type Driver struct {
	sess             *session.Session
	defaultTransport http.RoundTripper

	mu                sync.Mutex
	bucketRegionCache map[string]regionPromise
}

func (d *Driver) Name() string {
	return "s3driver"
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (driver.Object, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	c := d.newClient(ctx, parsed.BucketRegion)
	return httpdriver.NewObject(c, url, parsed.URL, size), nil
}

func (d *Driver) Create(ctx context.Context, url string) (driver.ObjectWriter, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	config := aws.NewConfig().
		WithRegion(parsed.BucketRegion).
		WithMaxRetries(100).
		WithS3DisableContentMD5Validation(true)

	if creds, ok := CredentialsFromCtx(ctx); ok {
		config = config.WithCredentials(creds)
	}

	svc := s3.New(d.sess, config)

	return NewObjectWriter(svc, parsed.Bucket, parsed.URL.Path, url), nil
}

func (d *Driver) Remove(ctx context.Context, url string) error {
	parsed, err := d.parseURL(url)
	if err != nil {
		return err
	}

	c := d.newClient(ctx, parsed.BucketRegion)

	return httpdriver.Delete(c, url, parsed.URL)
}

func (d *Driver) Stat(ctx context.Context, url string) (os.FileInfo, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	name := stdpath.Base(parsed.URL.Path)

	c := d.newClient(ctx, parsed.BucketRegion)

	size, err := httpdriver.Stat(c, parsed.URL.String())
	if err != nil {
		return nil, err
	}

	return &finfo{
		name:    name,
		size:    size,
		mode:    0644,
		modTime: time.Unix(0, 0),
		isDir:   false,
	}, nil
}

func (d *Driver) Readdir(ctx context.Context, url string) ([]os.FileInfo, error) {
	parsed, err := d.parseURL(url)
	if err != nil {
		return nil, err
	}

	c := d.newClient(ctx, parsed.BucketRegion)

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

	if u.Scheme != "s3" {
		return nil, fmt.Errorf("s3driver: unexpected scheme %q", u.Scheme)
	}

	if !u.IsAbs() {
		return nil, fmt.Errorf("s3driver: url must be absolute: %q", url)
	}

	bucket := u.Hostname()
	bucketRegion, err := d.getBucketRegion(bucket)
	if err != nil {
		return nil, err
	}

	u.Scheme = "https"
	u.Host = fmt.Sprintf("%s.s3-%s.amazonaws.com", bucket, bucketRegion)

	return &parsedURL{
		URL:          u,
		Bucket:       bucket,
		BucketRegion: bucketRegion,
	}, nil
}

func (d *Driver) getBucketRegion(bucketName string) (string, error) {
	var rp regionPromise
	var ok bool
	d.mu.Lock()
	rp, ok = d.bucketRegionCache[bucketName]
	if !ok {
		c := &http.Client{}

		lreg := getRegion()
		cfg := aws.NewConfig().
			WithEndpointResolver(endpoints.DefaultResolver()).
			WithRegion(lreg)
		creds := defaults.CredChain(cfg, defaults.Handlers())
		c.Transport = s3util.NewSigningRoundTripper(httputil.WithRetries(d.defaultTransport), creds, lreg)
		rp = getBucketRegion(c, bucketName)
		d.bucketRegionCache[bucketName] = rp
	}
	d.mu.Unlock()

	return rp.Apply()
}

func (d *Driver) newClient(ctx context.Context, bucketRegion string) *http.Client {
	c := &http.Client{}
	if timeout, ok := TimeoutFromCtx(ctx); ok {
		c.Timeout = *timeout
	} else {
		c.Timeout = 30 * time.Second
	}

	creds, ok := CredentialsFromCtx(ctx)
	if !ok {
		cfg := aws.NewConfig().
			WithEndpointResolver(endpoints.DefaultResolver()).
			WithRegion(bucketRegion)
		creds = defaults.CredChain(cfg, defaults.Handlers())
	}
	c.Transport = httputil.WithRetries(s3util.NewSigningRoundTripper(d.defaultTransport, creds, bucketRegion))

	return c
}

type parsedURL struct {
	URL          *stdurl.URL
	Bucket       string
	BucketRegion string
}

func RegisterDefaultDriver() {
	t := httputil.NewRoundRobinTransport(httputil.RoundRobinTransportConfig{
		MaxHosts:              1024,
		MaxIdleConnsPerHost:   1024,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	})

	driver.Register("s3", &Driver{
		sess:              session.Must(session.NewSession()),
		defaultTransport:  httputil.WithMetrics(t, "s3"),
		bucketRegionCache: make(map[string]regionPromise),
	})
}
