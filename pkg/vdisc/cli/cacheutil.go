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
	"os/user"
	"path/filepath"
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/units"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/caching"
)

type CacheConfig struct {
	Mode            string   `help:"Cache mode" enum:"disabled,memory,disk" default:"disk"`
	Bsize           units.SI `help:"Cache buffer size" default:"4MiB"`
	Bcount          int64    `help:"Cache buffer count (memory mode only)" default:"16"`
	Root            string   `help:"Disk mode cache root directory" default:"/var/cache/vdisc"`
	ReadAheadTokens int64    `help:"Read-ahead tokens" default:"32"`
}

func SIDecoder(ctx *kong.DecodeContext, target reflect.Value) error {
	var value string
	if err := ctx.Scan.PopValueInto("si", &value); err != nil {
		return err
	}

	si, err := units.ParseStrictBytes(value)
	if err != nil {
		return err
	}
	target.Set(reflect.ValueOf(units.SI(si)))
	return nil
}

func SITypeMapper() kong.Option {
	var si units.SI
	return kong.TypeMapper(reflect.TypeOf(si), kong.MapperFunc(SIDecoder))
}

func globalCache(cfg *CacheConfig) (cache caching.Cache) {
	switch cfg.Mode {
	case "disabled":
		cache = caching.NopCache
	case "memory":
		slicer, err := caching.NewMemorySlicer(int64(cfg.Bsize), cfg.Bcount)
		if err != nil {
			zap.L().Fatal("creating cache memory slicer", zap.Error(err))
		}
		cache = caching.NewCache(slicer, cfg.ReadAheadTokens)
	case "disk":

		slicer := caching.NewDiskSlicer(globalCacheRoot(cfg), int64(cfg.Bsize))
		cache = caching.NewCache(slicer, cfg.ReadAheadTokens)
	default:
		panic("never")
	}
	return
}

func globalCacheRoot(cfg *CacheConfig) string {
	root := cfg.Root
	if root == "/var/cache/vdisc" {
		u, err := user.Current()
		if err != nil {
			zap.L().Error("cache failed to get current user", zap.Error(err))
		} else if u.Uid != "0" && u.HomeDir != "" {
			root = filepath.Join(u.HomeDir, ".cache", "vdisc")
		}
	}
	return root
}
