// build +darwin
// Copyright © 2019 NVIDIA Corporation
package vdisc_cli

import (
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

func (cmd *MountCmd) doTcmu(v vdisc.VDisc) {
	zap.L().Fatal("TCMU is not supported on macOS")
}
