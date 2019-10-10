// Copyright © 2019 NVIDIA Corporation
package blockdev

import (
	"io"

	"github.com/google/uuid"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

type BlockDevice interface {
	io.Closer
	DevicePath() string
}

type BlockDeviceManager interface {
	io.Closer
	Open(obj storage.AnonymousObject, volumeName uuid.UUID, blockSize int64) (BlockDevice, error)
}
