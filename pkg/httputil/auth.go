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

package httputil

import (
	"net/http"
)

// Wraps a RoundTripper with one that injects an HTTP Authorization header
func WithAuthz(transport http.RoundTripper, value string) http.RoundTripper {
	return &authz{transport, value}
}

type authz struct {
	transport http.RoundTripper
	value     string
}

func (a *authz) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := req.Header["Authorization"]; !ok {
		req.Header.Set("Authorization", a.value)
	}
	return a.transport.RoundTrip(req)
}
