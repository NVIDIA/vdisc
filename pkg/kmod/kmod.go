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

package kmod

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
)

func Modprobe(module string) error {
	modbytes, err := ioutil.ReadFile("/proc/modules")
	if err != nil {
		return fmt.Errorf("reading /proc/modules: %s", err)
	}

	inserted, err := regexp.Match(fmt.Sprintf(".*\\b%s\\b.*", module), modbytes)
	if err != nil {
		return fmt.Errorf("Examining contents of /proc/modules: %s", err)
	}

	if !inserted {
		err := exec.Command("modprobe", module).Run()
		if err != nil {
			return fmt.Errorf("inserting %#v kernel module: %s", module, err)
		}
	}

	return nil
}
