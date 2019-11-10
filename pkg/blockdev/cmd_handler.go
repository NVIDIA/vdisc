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
	"io"

	"github.com/tnarg/go-tcmu"
	"github.com/tnarg/go-tcmu/scsi"
	"go.uber.org/zap"
)

type readerAtCmdHandler struct {
	R   io.ReaderAt
	Inq *tcmu.InquiryInfo
}

func (h readerAtCmdHandler) HandleCommand(cmd *tcmu.SCSICmd) (tcmu.SCSIResponse, error) {
	switch cmd.Command() {
	case scsi.Inquiry:
		if h.Inq == nil {
			h.Inq = &tcmu.InquiryInfo{
				VendorID:   "NVIDIA",
				ProductID:  "VDISC",
				ProductRev: "0001",
			}
		}
		return tcmu.EmulateInquiry(cmd, h.Inq)
	case scsi.TestUnitReady:
		return tcmu.EmulateTestUnitReady(cmd)
	case scsi.ServiceActionIn16:
		return tcmu.EmulateServiceActionIn(cmd)
	case scsi.ModeSense, scsi.ModeSense10:
		return tcmu.EmulateModeSense(cmd, false)
	case scsi.ModeSelect, scsi.ModeSelect10:
		return tcmu.EmulateModeSelect(cmd, false)
	case scsi.Read6, scsi.Read10, scsi.Read12, scsi.Read16:
		return tcmu.EmulateRead(cmd, h.R)
	default:
		zap.L().Sugar().Warnf("Ignore unknown SCSI command 0x%x", cmd.Command())
	}
	return cmd.NotHandled(), nil
}
