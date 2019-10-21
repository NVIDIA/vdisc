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

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/blockdev"
	"github.com/NVIDIA/vdisc/pkg/isofuse"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type MountCmd struct {
	Url            string                  `short:"u" help:"The URL of the vdisc" required:"true"`
	Mountpoint     string                  `short:"p" help:"The path to mount the vdisc" required:"true" type:"existingdir"`
	Mode           string                  `short:"m" help:"The mount mode" enum:"fuse,tcmu" default:"fuse"`
	Fuse           isofuse.Options         `embed prefix:"fuse-"`
	Tcmu           blockdev.TCMUConfig     `embed prefix:"tcmu-"`
	TcmuVolumeName uuid.UUID               `help:"The name of the tcmu volume"`
	Bcache         vdisc.BufferCacheConfig `embed prefix:"bcache-"`
}

func (cmd *MountCmd) Run(globals *Globals) error {
	bcache, err := vdisc.NewBufferCache(cmd.Bcache)
	if err != nil {
		zap.L().Fatal("creating buffer cache", zap.Error(err))
	}
	v, err := vdisc.Load(cmd.Url, bcache)
	if err != nil {
		zap.L().Fatal("loading vdisc", zap.Error(err))
	}
	defer v.Close()

	switch cmd.Mode {
	case "fuse":
		cmd.doFuse(v)
	case "tcmu":
		cmd.doTcmu(v)
	default:
		panic("never")
	}

	return nil
}

func (cmd *MountCmd) doFuse(v vdisc.VDisc) {
	fs, err := isofuse.NewWithOptions(cmd.Mountpoint, v, cmd.Fuse)
	if err != nil {
		zap.L().Fatal("new isofuse", zap.Error(err))
	}

	// Make signal channel and register notifiers for Interrupt and Terminate
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, syscall.SIGTERM)

	// Run the file system
	if err := fs.Start(); err != nil {
		zap.L().Fatal("start", zap.Error(err))
	}

	// Block until we receive a signal on the channel
	<-sigchan

	// Shutdown now that we've received the signal
	if err := fs.Close(); err != nil {
		zap.L().Fatal("shutdown error", zap.Error(err))
	}
}
