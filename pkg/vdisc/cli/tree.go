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
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type TreeCmd struct {
	Url    string                  `short:"u" help:"The URL of the vdisc" required:"true"`
	Path   string                  `short:"p" help:"The path in the vdisc to list" default:"/"`
	Bcache vdisc.BufferCacheConfig `embed prefix:"bcache-"`
}

func (cmd *TreeCmd) Run(globals *Globals) error {
	bcache, err := vdisc.NewBufferCache(cmd.Bcache)
	if err != nil {
		zap.L().Fatal("creating buffer cache", zap.Error(err))
	}
	v, err := vdisc.Load(cmd.Url, bcache)
	if err != nil {
		zap.L().Fatal("loading vdisc", zap.Error(err))
	}
	defer v.Close()

	w := iso9660.NewWalker(v.Image())
	fmt.Println(cmd.Path)
	cmd.printTree(w, cmd.Path, nil, v)

	return nil
}

func (cmd *TreeCmd) printTree(walker *iso9660.Walker, path string, depth []bool, urlMap extentURLMapper) {
	finfos, err := walker.ReadDir(path)
	if err != nil {
		zap.L().Fatal("", zap.Error(err))
	}

	var maxSizeLen int
	for i, fi := range finfos {
		if i > 0 {
			l := len(fmt.Sprintf("%d", fi.Size()))
			if l > maxSizeLen {
				maxSizeLen = l
			}
		}
	}

	for i, fi := range finfos {
		name := fi.Name()
		if name == "." || name == ".." {
			continue
		}

		var prefix string
		for _, final := range depth {
			if final {
				prefix = prefix + "    "
			} else {
				prefix = prefix + "│   "
			}
		}

		final := i == len(finfos)-1
		if final {
			prefix = prefix + "└── "
		} else {
			prefix = prefix + "├── "
		}

		fmt.Print(prefix)
		fmt.Printf(fmt.Sprintf(" [%%%dd] ", maxSizeLen), fi.Size())

		if fi.IsDir() {
			color.New(color.FgBlue, color.Bold).Println(name)
		} else if fi.Mode()&os.ModeSymlink != 0 {
			color.New(color.FgRed, color.Bold).Print(name)
			fmt.Println(" → " + fi.Target())
		} else {
			url, err := urlMap.ExtentURL(fi.Extent())
			if err != nil {
				zap.L().Fatal("extent url lookup", zap.Uint32("lba", uint32(fi.Extent())), zap.Error(err))
			}
			color.New(color.FgGreen, color.Bold).Print(name)
			fmt.Println(" ⇒ " + url)
		}

		if fi.IsDir() {
			cmd.printTree(walker, filepath.Join(path, name), append(depth, final), urlMap)
		}
	}
}

//type aggregateRec struct {
//	rec    iso9660.DirectoryRecord
//	length int64
//}

type extentURLMapper interface {
	ExtentURL(lba iso9660.LogicalBlockAddress) (string, error)
}
