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
