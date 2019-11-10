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

package blockdev

import (
	"encoding/hex"
	"fmt"
	"path"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/tnarg/go-tcmu"

	"github.com/NVIDIA/vdisc/pkg/kmod"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

type TCMUConfig struct {
	Workers          int    `help:"Number of goroutines handling SCSI commands" default:"12"`
	WorkerBufferSize int    `help:"Size of the static buffer used by each goroutine" default:"4194304"`
	DeviceNamespace  string `help:"The namespace under /dev where devices are created" default:"vdisc"`
	HBA              int    `help:"The SCSI Host Bus Adapter identifier" default:"30"`
	BlockXferMin     uint16 `help:"Advertise minimum blocks to transfer" default:"128"`   // 256KB
	BlockXferMax     uint32 `help:"Advertise maximum blocks to transfer" default:"16384"` // 32MB
	BlockXferOpt     uint32 `help:"Advertise optimal block transfer size" default:"2048"` // 4MB
}

func NewTCMUBlockDeviceManager(cfg TCMUConfig) (BlockDeviceManager, error) {
	if err := kmod.Modprobe("configfs"); err != nil {
		return nil, fmt.Errorf("failed to load configfs kernel module: %s", err)
	}

	if err := kmod.Modprobe("target_core_user"); err != nil {
		return nil, fmt.Errorf("failed to load target_core_user kernel module: %s", err)
	}

	return &tcmuBlockDeviceManager{
		cfg:  cfg,
		pool: NewCmdPool(cfg.Workers, cfg.WorkerBufferSize),
	}, nil
}

type tcmuBlockDeviceManager struct {
	cfg  TCMUConfig
	pool *CmdPool
}

func (d *tcmuBlockDeviceManager) Open(obj storage.AnonymousObject, volumeName uuid.UUID, blockSize int64) (BlockDevice, error) {
	wwn := generateWWN(volumeName)

	//path.Join(scsiDir, d.scsi.WWN.DeviceID(), "tpgt_1")

	handler := &tcmu.SCSIHandler{
		HBA: d.cfg.HBA,
		//LUN:        lun,
		WWN:        wwn,
		VolumeName: volumeName.String(),
		DataSizes: tcmu.DataSizes{
			VolumeSize:   obj.Size(),
			BlockSize:    blockSize,
			BlockXferMin: d.cfg.BlockXferMin,
			BlockXferMax: d.cfg.BlockXferMax, // 32MB
			BlockXferOpt: d.cfg.BlockXferOpt, // 4MB
		},
		DevReady: d.pool.DevReady(
			readerAtCmdHandler{
				R: obj,
			}),
	}

	prefix := path.Join(devfs, d.cfg.DeviceNamespace)
	dev, err := tcmu.OpenTCMUDevice(prefix, handler)
	if err != nil {
		return nil, fmt.Errorf("opening tcmu device: %s", err)
	}

	devpath := path.Join(prefix, volumeName.String())
	devpath, err = canonicalizeBlockDevice(devpath)
	if err != nil {
		if e := dev.Close(); e != nil {
			err = multierror.Append(err, e)
		}
		return nil, fmt.Errorf("canonicalizing backing device path: %s", err)
	}

	return &tcmuBlockDevice{dev, devpath}, nil
}

func (d *tcmuBlockDeviceManager) Close() error {
	return d.pool.Close()
}

func generateWWN(id uuid.UUID) tcmu.WWN {
	return tcmu.NaaWWN{
		OUI:         "00044B", // NVIDIA's IEEE Organizationally Unique Identifier
		VendorID:    hex.EncodeToString(id[:4]),
		VendorIDExt: hex.EncodeToString(id[4:12]),
	}

}

type tcmuBlockDevice struct {
	dev     *tcmu.Device
	devpath string
}

func (d *tcmuBlockDevice) Close() error {
	err := d.dev.Close()
	return err
}

func (d *tcmuBlockDevice) DevicePath() string {
	return d.devpath
}
