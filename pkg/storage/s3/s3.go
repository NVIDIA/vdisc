// Copyright © 2019 NVIDIA Corporation
package s3driver

import (
	"context"
	"fmt"
	"net/http"
	stdurl "net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/NVIDIA/vdisc/pkg/httputil"
	"github.com/NVIDIA/vdisc/pkg/s3util"
	"github.com/NVIDIA/vdisc/pkg/storage"
	"github.com/NVIDIA/vdisc/pkg/storage/http"
)

type Driver struct {
	sess             *session.Session
	defaultTransport http.RoundTripper

	mu                sync.Mutex
	bucketRegionCache map[string]regionPromise
}

func (d *Driver) Open(ctx context.Context, url string, size int64) (storage.Object, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if !u.IsAbs() {
		return nil, fmt.Errorf("s3: URIs must be absolute")
	}

	bucket := u.Hostname()
	bucketRegion, err := d.getBucketRegion(bucket)
	if err != nil {
		return nil, err
	}

	u.Scheme = "https"
	u.Host = fmt.Sprintf("%s.s3-%s.amazonaws.com", bucket, bucketRegion)

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

	anon, err := httpdriver.NewObject(c, u, size)
	if err != nil {
		return nil, err
	}

	return storage.WithURL(anon, url), err
}

func (d *Driver) Create(ctx context.Context, url string) (storage.ObjectWriter, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if !u.IsAbs() {
		return nil, fmt.Errorf("s3: URIs must be absolute")
	}

	bucket := u.Hostname()
	bucketRegion, err := d.getBucketRegion(bucket)
	if err != nil {
		return nil, err
	}

	config := aws.NewConfig().
		WithRegion(bucketRegion).
		WithMaxRetries(100).
		WithS3DisableContentMD5Validation(true)

	if creds, ok := CredentialsFromCtx(ctx); ok {
		config = config.WithCredentials(creds)
	}

	svc := s3.New(d.sess, config)

	return NewObjectWriter(svc, bucket, u.Path, url), nil
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
		c.Transport = httputil.WithRetries(s3util.NewSigningRoundTripper(d.defaultTransport, creds, lreg))
		rp = getBucketRegion(c, bucketName)
		d.bucketRegionCache[bucketName] = rp
	}
	d.mu.Unlock()

	return rp.Apply()
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

	storage.Register("s3", &Driver{
		sess:              session.Must(session.NewSession()),
		defaultTransport:  httputil.WithMetrics(t, "s3"),
		bucketRegionCache: make(map[string]regionPromise),
	})
}
