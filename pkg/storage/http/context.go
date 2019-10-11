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
