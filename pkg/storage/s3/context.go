// Copyright © 2019 NVIDIA Corporation
package s3driver

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

var (
	ctxCredentialsKey int
	ctxTimeoutKey     int
)

func CtxWithCredentials(ctx context.Context, creds *credentials.Credentials) context.Context {
	return context.WithValue(ctx, ctxCredentialsKey, creds)
}

func CredentialsFromCtx(ctx context.Context) (*credentials.Credentials, bool) {
	v, ok := ctx.Value(ctxCredentialsKey).(*credentials.Credentials)
	return v, ok
}

func CtxWithTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, ctxTimeoutKey, &timeout)
}

func TimeoutFromCtx(ctx context.Context) (*time.Duration, bool) {
	v, ok := ctx.Value(ctxTimeoutKey).(*time.Duration)
	return v, ok
}
