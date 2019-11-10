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
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/units"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/caching"
	"github.com/NVIDIA/vdisc/pkg/safecast"
)

type CacheCmd struct {
	Fsck FsckCmd `cmd help:"Cache integrity check"`
	Gc   GcCmd   `cmd help:"Cache garbage collection daemon"`
}

type FsckCmd struct{}

func (cmd *FsckCmd) Run(globals *Globals) error {
	slicer := caching.NewDiskSlicer(globalCacheRoot(&globals.Cache), int64(globals.Cache.Bsize))

	zap.L().Info("Checking vdisc cache integrity")
	if err := slicer.CheckIntegrity(); err != nil {
		zap.L().Fatal("vdisc cache integrity error", zap.Error(err))
	}
	zap.L().Info("vdisc cache integrity verified")
	return nil
}

type GcCmd struct {
	Period    time.Duration `help:"Period to wait between collections" default:"30s"`
	Threshold GcThreshold   `help:"Goal for disk free space. Either a percentage (e.g. 10%) or an absolute number of bytes" default:"100MiB"`
}

func (cmd *GcCmd) Run(globals *Globals) error {
	slicer := caching.NewDiskSlicer(globalCacheRoot(&globals.Cache), int64(globals.Cache.Bsize))

	// Make signal channel and register notifiers for Interrupt and Terminate
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, syscall.SIGTERM)

	slicer.Gc(&cmd.Threshold)
	for {
		select {
		case <-time.After(cmd.Period):
			slicer.Gc(&cmd.Threshold)
		case signal := <-sigchan:
			zap.L().Info("received signal", zap.String("sig", signal.String()))
			return nil
		}
	}
}

type Percentage float64

type GcThreshold struct {
	absolute    bool
	free        units.SI
	percentFree Percentage
}

func (gt *GcThreshold) GcNeeded(st *syscall.Statfs_t) bool {
	if gt.absolute {
		avail := int64(st.Bsize) * safecast.Uint64ToInt64(st.Bavail)
		return units.SI(avail) < gt.free
	}
	return Percentage(float64(st.Bfree)/float64(st.Blocks)*100) < gt.percentFree
}

var percentageRE = regexp.MustCompile(`^(\d+([.]\d+)?)([%])$`)

func GcThresholdDecoder(ctx *kong.DecodeContext, target reflect.Value) error {
	var value string
	if err := ctx.Scan.PopValueInto("threshold", &value); err != nil {
		return err
	}

	var gct GcThreshold
	free, err := units.ParseStrictBytes(value)
	if err != nil {
		groups := percentageRE.FindStringSubmatch(value)
		if len(groups) != 4 {
			return fmt.Errorf("invalid threshold: %q", value)
		}

		v, err := strconv.ParseFloat(groups[1], 64)
		if err != nil {
			return err
		}

		if v < 0.0 || v > 100.0 {
			return fmt.Errorf("percentage out of range: %f", v)
		}
		gct = GcThreshold{
			absolute:    false,
			percentFree: Percentage(v),
		}
	} else {
		gct = GcThreshold{
			absolute: true,
			free:     units.SI(free),
		}
	}
	target.Set(reflect.ValueOf(gct))
	return nil
}

func GcThresholdTypeMapper() kong.Option {
	var gt GcThreshold
	return kong.TypeMapper(reflect.TypeOf(gt), kong.MapperFunc(GcThresholdDecoder))
}
