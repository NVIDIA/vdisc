// Copyright © 2019 NVIDIA Corporation
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
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if !u.IsAbs() {
		return nil, fmt.Errorf("swiftdriver: URIs must be absolute")
	}

	u.Scheme = "https"

	parts := strings.Split(u.Path, "/")
	if len(parts) < 4 || len(parts[0]) != 0 {
		return nil, fmt.Errorf("swiftdriver: invalid URL: %q", url)
	}

	// extract the account portion of the path.
	account := parts[1]
	u.Path = "/" + strings.Join(parts[2:], "/")

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

	anon, err := httpdriver.NewObject(c, u, size)
	if err != nil {
		return nil, err
	}
	return storage.WithURL(anon, url), nil
}

func (d *Driver) Create(ctx context.Context, url string) (storage.ObjectWriter, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if !u.IsAbs() {
		return nil, fmt.Errorf("swiftdriver: URIs must be absolute")
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) < 4 || len(parts[0]) != 0 {
		return nil, fmt.Errorf("swiftdriver: invalid URL: %q", url)
	}

	// extract the account portion of the path.
	account := parts[1]
	container := parts[2]
	key := "/" + strings.Join(parts[3:], "/")

	creds := credentials.NewCredentials(NewSwiftEnvProvider(account))
	config := aws.NewConfig().
		WithRegion(GetSwiftRegion()).
		WithEndpoint(fmt.Sprintf("https://%s", u.Host)).
		WithS3ForcePathStyle(true).
		WithCredentials(creds).
		WithMaxRetries(100)

	svc := s3.New(d.sess, config)

	return s3driver.NewObjectWriter(svc, container, key, url), nil

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
