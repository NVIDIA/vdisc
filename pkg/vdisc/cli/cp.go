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
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type CpCmd struct {
	Url  string `short:"u" help:"The URL of the vdisc" required:"true"`
	Path string `short:"p" help:"The path in the vdisc to list" required:"true"`
	Out  string `short:"o" help:"Output file" required:"true"`
}

func (cmd *CpCmd) Run(globals *Globals) error {
	v, err := vdisc.Load(cmd.Url, globalCache(&globals.Cache))
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
