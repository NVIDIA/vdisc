// build +linux
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

package vdisc_cli

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/blockdev"
	"github.com/NVIDIA/vdisc/pkg/unixcompat"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

//var (
//	volumeNamespace = uuid.Must(uuid.Parse("32E59907-CD5D-4BA3-A5D6-FE5720509A8C"))
//)

func (cmd *MountCmd) doTcmu(v vdisc.VDisc) {
	blockdevMgr, err := blockdev.NewTCMUBlockDeviceManager(cmd.Tcmu)
	if err != nil {
		zap.L().Fatal("creating block device manager", zap.Error(err))
	}

	var volumeName uuid.UUID
	if cmd.TcmuVolumeName == uuid.Nil {
		volumeName = uuid.New()
	} else {
		volumeName = cmd.TcmuVolumeName
	}

	dev, err := blockdevMgr.Open(v.Image(), volumeName, int64(v.BlockSize()))
	if err != nil {
		zap.L().Fatal("opening tcmu device", zap.Error(err))
	}
	defer dev.Close()

	zap.L().Info("created tcmu device", zap.String("device", dev.DevicePath()))

	if err := blockdev.TuneDeviceQueue(dev); err != nil {
		zap.L().Error("tuning device", zap.Error(err))
	}

	time.Sleep(500 * time.Millisecond)
	if err := unixcompat.Mount(dev.DevicePath(), cmd.Mountpoint, v.FsType(), unixcompat.MS_MGC_VAL|unixcompat.MS_RDONLY, ""); err != nil {
		zap.L().Fatal("mounting tcmu device", zap.String("device", dev.DevicePath()), zap.String("mountpoint", cmd.Mountpoint), zap.Error(err))
	}

	zap.L().Info("mounted tcmu device", zap.String("device", dev.DevicePath()), zap.String("mountpoint", cmd.Mountpoint))

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, syscall.SIGTERM)

	// Block until we receive a signal on the channel
	<-sigchan

	if err := unixcompat.Unmount(cmd.Mountpoint, unixcompat.MNT_FORCE); err != nil {
		zap.L().Fatal("mounting tcmu device", zap.String("mountpoint", cmd.Mountpoint), zap.Error(err))
	}
	zap.L().Info("unmounted tcmu device", zap.String("device", dev.DevicePath()), zap.String("mountpoint", cmd.Mountpoint))
}
