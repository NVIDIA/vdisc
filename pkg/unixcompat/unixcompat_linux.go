// +build linux
package unixcompat

import (
	"syscall"

	"golang.org/x/sys/unix"
)

var MNT_FORCE = syscall.MNT_FORCE
var MS_MGC_VAL = uintptr(syscall.MS_MGC_VAL)
var MS_RDONLY = uintptr(syscall.MS_RDONLY)
var BLKROSET = uintptr(unix.BLKROSET)

func Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return syscall.Mount(source, target, fstype, flags, data)
}

func Unmount(path string, flags int) error {
	return syscall.Unmount(path, flags)
}

func Major(dev uint64) uint32 {
	return unix.Major(dev)
}

func Minor(dev uint64) uint32 {
	return unix.Minor(dev)
}
