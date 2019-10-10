// Copyright © 2018 NVIDIA Corporation

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
