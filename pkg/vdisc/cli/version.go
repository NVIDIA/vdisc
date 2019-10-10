// Copyright © 2019 NVIDIA Corporation
package vdisc_cli

import (
	"fmt"
)

// Version is injected with git sha in build
var Version = ""

type VersionCmd struct{}

func (cmd *VersionCmd) Run(globals *Globals) error {
	fmt.Println(Version)
	return nil
}
