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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	stdurl "net/url"
	"os"
)

func Delete(c *http.Client, url string, u *stdurl.URL) error {
	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return fmt.Errorf("delete %q: %+v", url, err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("delete %q: %+v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		io.Copy(os.Stderr, resp.Body)
		return fmt.Errorf("delete %q: HTTP Status %d", url, resp.StatusCode)
	}

	if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
		return fmt.Errorf("delete %q: %+v", url, err)
	}

	return nil
}
