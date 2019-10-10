// Copyright © 2019 NVIDIA Corporation
package httpdriver

import (
	"context"
	"time"
)

var (
	ctxAuthzKey   int
	ctxTimeoutKey int
)

func CtxWithAuthz(ctx context.Context, authz string) context.Context {
	return context.WithValue(ctx, ctxAuthzKey, &authz)
}

func CtxWithTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, ctxTimeoutKey, &timeout)
}

func AuthzFromCtx(ctx context.Context) (*string, bool) {
	v, ok := ctx.Value(ctxAuthzKey).(*string)
	return v, ok
}

func TimeoutFromCtx(ctx context.Context) (*time.Duration, bool) {
	v, ok := ctx.Value(ctxTimeoutKey).(*time.Duration)
	return v, ok
}
