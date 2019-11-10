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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type InspectCmd struct {
	Url string `short:"u" help:"The URL of the vdisc" required:"true"`
}

func (cmd *InspectCmd) Run(globals *Globals) error {
	v, err := vdisc.Load(cmd.Url, globalCache(&globals.Cache))
	if err != nil {
		zap.L().Fatal("loading vdisc", zap.Error(err))
	}
	defer v.Close()

	fmt.Println("{")
	fmt.Printf("  \"FsType\": %q,\n", v.FsType())
	fmt.Printf("  \"BlockSize\": %d,\n", v.BlockSize())

	if v.FsType() == "iso9660" {
		fmt.Print("  \"PrimaryVolumeDescriptor\": ")

		var pvd iso9660.PrimaryVolumeDescriptor
		pvdSector := io.NewSectionReader(v.Image(), 16*iso9660.LogicalBlockSize, iso9660.LogicalBlockSize)
		if err := iso9660.DecodePrimaryVolumeDescriptor(pvdSector, &pvd); err != nil {
			zap.L().Fatal("decoding primary volume descriptor", zap.Error(err))
		}

		jenc := json.NewEncoder(os.Stdout)
		jenc.SetIndent("  ", "  ")
		if err := jenc.Encode(&pvd); err != nil {
			zap.L().Fatal("serializing primary volume descriptor", zap.Error(err))
		}
	}
	fmt.Println("}")
	return nil
}
