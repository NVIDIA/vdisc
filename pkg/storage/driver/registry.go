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
package driver

import (
	"fmt"
	stdurl "net/url"
	"sort"
	"strings"
	"sync"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

// Register makes a storage driver available by the provided URL scheme.
// If Register is called twice with the same scheme or if driver is nil,
// it panics.
func Register(scheme string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("storage: Register driver is nil")
	}
	if _, dup := drivers[scheme]; dup {
		panic("storage: Register called twice for driver " + scheme)
	}
	drivers[scheme] = driver
}

// Drivers returns a sorted list of the URL schemes of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var list []string
	for scheme := range drivers {
		list = append(list, scheme)
	}
	sort.Strings(list)
	return list
}

func Find(url string) (Driver, error) {
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		u.Scheme = "file"
		if !strings.HasPrefix(u.Path, "/") {
			u.Opaque = u.Path
			u.Path = ""
			u.RawPath = ""
		}
	}

	driversMu.RLock()
	defer driversMu.RUnlock()

	drvr, ok := drivers[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("storage: unknown driver %q (forgotten import?)", u.Scheme)
	}
	return drvr, nil
}
