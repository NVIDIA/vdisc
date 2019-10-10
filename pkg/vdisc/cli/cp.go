// Copyright © 2019 NVIDIA Corporation
package vdisc_cli

import (
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type CpCmd struct {
	Url    string                  `short:"u" help:"The URL of the vdisc" required:"true"`
	Path   string                  `short:"p" help:"The path in the vdisc to list" required:"true"`
	Out    string                  `short:"o" help:"Output file" required:"true"`
	Bcache vdisc.BufferCacheConfig `embed prefix:"bcache-"`
}

func (cmd *CpCmd) Run(globals *Globals) error {
	bcache, err := vdisc.NewBufferCache(cmd.Bcache)
	if err != nil {
		zap.L().Fatal("creating buffer cache", zap.Error(err))
	}
	v, err := vdisc.Load(cmd.Url, bcache)
	if err != nil {
		zap.L().Fatal("loading vdisc", zap.Error(err))
	}
	defer v.Close()

	var out io.WriteCloser
	if cmd.Out == "-" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(cmd.Out)
		if err != nil {
			zap.L().Fatal("creating out", zap.Error(err))
		}
		defer out.Close()
	}

	w := iso9660.NewWalker(v.Image())

	src, err := w.Open(cmd.Path)
	if err != nil {
		return err
	}

	defer src.Close()

	buf := make([]byte, 1024*1024)
	if _, err := io.CopyBuffer(out, src, buf); err != nil {
		return err
	}

	return nil
}
