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
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
)

func TestRoundRobinTransport(t *testing.T) {
	var wg sync.WaitGroup

	tnet := newTestNetwork()
	defer tnet.Shutdown()

	var ips []string
	expectedBodies := make(map[string]struct{})

	for i := 0; i < 1000; i++ {
		c := i / 253
		d := (i % 253) + 1
		ip := fmt.Sprintf("169.254.%d.%d", c, d)
		ips = append(ips, ip)
		address := ip + ":80"
		expectedBodies[address] = struct{}{}

		l, err := tnet.Listen("tcp", address)
		if err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			wg.Done()
			w.Write([]byte(address))
		}))
	}

	c := &http.Client{
		Transport: NewRoundRobinTransport(RoundRobinTransportConfig{
			Dial: tnet.DialContext,
			LookupHost: func(host string) (addrs []string, err error) {
				return ips, nil
			},
		}),
	}

	for i := 0; i < 1000; i++ {
		resp, err := c.Get("http://example.com/")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatal(errors.New("not ok"))
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		actualBody := string(data)
		_, ok := expectedBodies[actualBody]
		if !ok {
			t.Fatal(fmt.Errorf("unexpected body: %q", actualBody))
		}
		delete(expectedBodies, actualBody)
	}

	wg.Wait()
}
