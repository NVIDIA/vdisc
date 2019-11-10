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

package s3util

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"go.uber.org/zap"
)

// NewSigningRoundTripper augments an http.RoundTripper to sign HTTP requests for a given set of credentials and region.
func NewSigningRoundTripper(transport http.RoundTripper, creds *credentials.Credentials, region string) http.RoundTripper {
	v4s := v4.NewSigner(creds)
	v4s.DisableURIPathEscaping = true
	return &signer{transport, v4s, region}
}

// Signer implements the http.RoundTripper interface and houses an optional RoundTripper that will be called between
// signing and response.
type signer struct {
	transport http.RoundTripper
	v4        *v4.Signer
	region    string
}

func shouldEscapeEncodePath(c byte) bool {
	// §2.3 Unreserved characters (alphanum)
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}

	switch c {
	case '-', '_', '.', '~': // §2.3 Unreserved characters (mark)
		return false

	case '$', '&', '+', ',', '/', ':', ';', '=', '?', '@': // §2.2 Reserved characters (reserved)
		// The RFC allows : @ & = + $ but saves / ; , for assigning
		// meaning to individual path segments.
		return c != '/'
	}

	// Everything else must be escaped.
	return true
}

// encodePathCanonical escapes the path s, replacing special
// characters with %XX sequences as needed.
func encodePathCanonical(s string) string {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscapeEncodePath(c) {
			hexCount++
		}
	}

	if hexCount == 0 {
		return s
	}

	var buf [64]byte
	var t []byte

	required := len(s) + 2*hexCount
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	j := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case shouldEscapeEncodePath(c):
			t[j] = '%'
			t[j+1] = "0123456789ABCDEF"[c>>4]
			t[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

// RoundTrip implements the http.RoundTripper interface and is used to wrap HTTP requests in order to sign them for AWS
// API calls. The scheme for all requests will be changed to HTTPS.
func (s *signer) RoundTrip(req *http.Request) (*http.Response, error) {
	if h, ok := req.Header["Authorization"]; ok && len(h) > 0 && strings.HasPrefix(h[0], "AWS4") {
		debugln("Received request to sign that is already signed. Skipping.")
		return s.transport.RoundTrip(req)
	}
	debugf("Receiving request for signing: %+v", req)
	req.URL.Scheme = "https"
	req.URL.RawPath = encodePathCanonical(req.URL.Path)

	t := time.Now()
	req.Header.Set("Date", t.Format(time.RFC3339))
	debugf("Final request to be signed: %+v", req)
	var err error
	switch req.Body {
	case nil:
		debugln("Signing request with no body...")
		_, err = s.v4.Sign(req, nil, "s3", s.region, t)
	default:
		d, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(d))
		debugln("Signing request with body...")
		_, err = s.v4.Sign(req, bytes.NewReader(d), "s3", s.region, t)
	}
	if err != nil {
		debugf("Error while attempting to sign request: '%s'", err)
		return nil, err
	}
	debugf("Signing succesful. Set header to: '%+v'", req.Header)
	debugf("Sending signed request to RoundTripper: %+v", req)
	resp, err := s.transport.RoundTrip(req)
	if err != nil {
		debugf("Error from RoundTripper.\n\n\tResponse: %+v\n\n\tError: '%s'", resp, err)
		return resp, err
	}
	debugf("Successful response from RoundTripper: %+v\n", resp)
	return resp, nil
}

const loggerName = "s3util"

func debugln(v string) {
	zap.L().Sugar().Named(loggerName).Debug(v)
}

func debugf(v string, a ...interface{}) {
	zap.L().Sugar().Named(loggerName).Debugf(v, a...)
}
