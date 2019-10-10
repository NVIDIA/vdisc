// Copyright © 2019 NVIDIA Corporation
package blockdev

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"syscall"
	"unsafe"

	"github.com/NVIDIA/vdisc/pkg/unixcompat"
)

var (
	devfs string
	sysfs string
)

func init() {
	if dpath := os.Getenv("DEVFS"); dpath != "" {
		devfs = dpath
	} else {
		devfs = "/dev"
	}

	if spath := os.Getenv("SYSFS"); spath != "" {
		sysfs = spath
	} else {
		sysfs = "/sys"
	}
}

func TuneDeviceQueue(bdev BlockDevice) (err error) {
	devpath, err := canonicalizeBlockDevice(bdev.DevicePath())
	if err != nil {
		return
	}
	base := path.Base(devpath)
	queue := path.Join(sysfs, "block", base, "queue")

	if e := sysfsWriteFull(path.Join(queue, "scheduler"), []byte("noop")); e != nil {
		log.Printf("WARNING: Failed to set scheduler: %s", e)
	}
	if e := sysfsWriteFull(path.Join(queue, "read_ahead_kb"), []byte("4096")); e != nil {
		log.Printf("WARNING: Failed to set read_ahead_kb: %s", e)
	}
	if e := sysfsWriteFull(path.Join(queue, "rotational"), []byte("0")); e != nil {
		log.Printf("WARNING: Failed to set rotational: %s", e)
	}

	dev := path.Join(devfs, base)
	f, err := os.Open(dev)
	if err != nil {
		return
	}
	defer f.Close()

	isrdonly := 1
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), unixcompat.BLKROSET, uintptr(unsafe.Pointer(&isrdonly)))
	if e != 0 {
		err = e
		return
	}

	return
}

func canonicalizeBlockDevice(devpath string) (string, error) {
	stat, err := os.Stat(devpath)
	if err != nil {
		return "", err
	}

	sys, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		panic("never")
	}

	major := unixcompat.Major(sys.Rdev)
	minor := unixcompat.Minor(sys.Rdev)

	symlink := path.Join(sysfs, "dev", "block", fmt.Sprintf("%d:%d", major, minor))
	dst, err := os.Readlink(symlink)
	if err != nil {
		return "", err
	}
	sysfsDevPath := path.Clean(path.Join(symlink, dst))
	base := path.Base(sysfsDevPath)
	return path.Join(devfs, base), nil
}

func sysfsWriteFull(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0200)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}
