// Copyright © 2019 NVIDIA Corporation
package filedriver

import (
	"os"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

type objectWriter struct {
	path string
	f    *os.File
}

func (ow *objectWriter) Abort() {
	os.Remove(ow.f.Name())
	// TODO: log err
	ow.f.Close()
}

func (ow *objectWriter) Commit() (storage.CommitInfo, error) {
	if err := ow.f.Sync(); err != nil {
		return nil, err
	}

	if err := os.Rename(ow.f.Name(), ow.path); err != nil {
		return nil, err
	}

	return storage.NewCommitInfo(ow.path), nil
}

func (ow *objectWriter) Write(p []byte) (n int, err error) {
	n, err = ow.f.Write(p)
	return
}
