// Copyright © 2019 NVIDIA Corporation
package swiftdriver

import (
	"context"
	"time"
)

var (
	ctxCredentialsKey int
	ctxRegionKey      int
	ctxTimeoutKey     int
)

func CtxWithCredentials(ctx context.Context, creds *SwiftCreds) context.Context {
	return context.WithValue(ctx, ctxCredentialsKey, creds)
}

func CredentialsFromCtx(ctx context.Context) (*SwiftCreds, bool) {
	v, ok := ctx.Value(ctxCredentialsKey).(*SwiftCreds)
	return v, ok
}

func CtxWithTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, ctxTimeoutKey, &timeout)
}

func TimeoutFromCtx(ctx context.Context) (*time.Duration, bool) {
	v, ok := ctx.Value(ctxTimeoutKey).(*time.Duration)
	return v, ok
}

func CtxWithRegion(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, ctxTimeoutKey, region)
}

func RegionFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxRegionKey).(string)
	return v, ok
}
