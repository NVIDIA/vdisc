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
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
)

var contentRangeRE = regexp.MustCompile("^bytes (\\d+)-(\\d+)/(\\d+)$")

type ContentRange struct {
	First uint64
	Last  uint64
	Total uint64
}

func (cr ContentRange) Len() uint64 {
	return cr.Last - cr.First + 1
}

func (cr ContentRange) Terminal() bool {
	return cr.Last == cr.Total-1
}

func GetContentRange(resp *http.Response) (*ContentRange, error) {
	cr := resp.Header.Get("Content-Range")
	if cr == "" {
		return nil, errors.New("empty/missing Content-Range header")
	}
	parts := contentRangeRE.FindStringSubmatch(cr)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid Content-Range header %q", cr)
	}
	first, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Range header %q", cr)
	}
	last, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Range header %q", cr)
	}
	total, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Range header %q", cr)
	}
	if total > 0 && (first > last || first >= total || last >= total) {
		return nil, fmt.Errorf("invalid Content-Range header %q", cr)
	}

	return &ContentRange{first, last, total}, nil
}
