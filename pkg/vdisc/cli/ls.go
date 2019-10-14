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
	"os/user"
	"strconv"

	"github.com/fatih/color"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type LsCmd struct {
	Url       string `short:"u" help:"The URL of the vdisc" required:"true"`
	Path      string `short:"p" help:"The path in the vdisc to list" required:"true"`
	Long      bool   `short:"l" help:"Long listing"`
	Recursive bool   `short:"r" help:"Recursive listing"`
}

func (cmd *LsCmd) Run(globals *Globals) error {
	v, err := vdisc.Load(cmd.Url, globalCache(&globals.Cache))
	if err != nil {
		zap.L().Fatal("loading vdisc", zap.Error(err))
	}
	defer v.Close()

	w := iso9660.NewWalker(v.Image())
	if cmd.Recursive {
		prevPath := cmd.Path
		w.Walk(cmd.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				zap.L().Fatal("recursive listing", zap.String("path", path), zap.Error(err))
			}
			if path != prevPath {
				fmt.Println("\n" + path + ":")
				prevPath = path
			}

			cmd.listFile(info)
			return nil
		})
	} else {
		fi, err := w.Lstat(cmd.Path)
		if err != nil {
			zap.L().Fatal("lstat", zap.String("path", cmd.Path), zap.Error(err))
		}

		if fi.IsDir() {
			infos, err := w.ReadDir(cmd.Path)
			if err != nil {
				zap.L().Fatal("readdir", zap.String("path", cmd.Path), zap.Error(err))
			}

			for _, fi := range infos {
				cmd.listFile(fi)
			}
		} else {
			cmd.listFile(fi)
		}
	}

	return nil
}

func (cmd *LsCmd) listFile(info os.FileInfo) {
	if cmd.Long {
		cmd.listLong(info)
	} else {
		cmd.listShort(info)
	}
}

func (cmd *LsCmd) listShort(info os.FileInfo) {
	fi := info.Sys().(*iso9660.FileInfo)

	name := fi.Name()

	if name == "." || name == ".." {
		return
	}

	if fi.IsDir() {
		name = color.New(color.FgBlue, color.Bold).Sprintf("%s", name)
	} else if fi.Mode()&os.ModeSymlink != 0 {
		name = color.New(color.FgRed, color.Bold).Sprintf("%s", name)
	}

	target := fi.Target()

	if len(target) > 0 {
		fmt.Println(fmt.Sprintf("%s@", name))
	} else {
		fmt.Println(fmt.Sprintf("%s", name))
	}
}

func (cmd *LsCmd) listLong(info os.FileInfo) {
	fi := info.Sys().(*iso9660.FileInfo)

	name := fi.Name()
	if name == "." || name == ".." {
		return
	}

	mode := os.FileMode.String(fi.Mode())
	if fi.IsDir() {
		mode = "d" + mode[1:]
		name = color.New(color.FgBlue, color.Bold).Sprintf("%s", name)
	} else if fi.Mode()&os.ModeSymlink != 0 {
		mode = "l" + mode[1:]
		name = color.New(color.FgRed, color.Bold).Sprintf("%s", name)
	}

	t := fi.ModTime()
	target := fi.Target()

	uname := strconv.Itoa(int(fi.Uid()))
	if u, err := user.LookupId(uname); err == nil {
		uname = u.Username
	}

	gname := strconv.Itoa(int(fi.Gid()))
	if g, err := user.LookupId(gname); err == nil {
		gname = g.Name
	}

	if len(target) > 0 {
		fmt.Println(fmt.Sprintf("%s %s %s %9d %.3s %02d %02d:%02d %.4d %s -> %s", mode, uname, gname, fi.Size(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Year(), name, target))
	} else {
		fmt.Println(fmt.Sprintf("%s %s %s %9d %.3s %02d %02d:%02d %.4d %s", mode, uname, gname, fi.Size(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Year(), name))
	}
}
