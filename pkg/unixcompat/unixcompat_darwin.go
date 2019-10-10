// +build darwin
package unixcompat

import (
	"errors"
	"syscall"
)

var MNT_FORCE int
var BLKROSET uintptr
var MS_MGC_VAL uintptr
var MS_RDONLY uintptr

var errNotImpl = errors.New("not implemented on this OS")

func Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return errNotImpl
}

func Unmount(path string, flags int) error {
	return syscall.Unmount(path, flags)
}

func Major(dev int32) uint32 {
	panic(errNotImpl)
}

func Minor(dev int32) uint32 {
	panic(errNotImpl)
}
