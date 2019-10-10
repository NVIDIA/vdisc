// Copyright © 2019 NVIDIA Corporation
package datadriver

import (
	"bytes"

	"github.com/vincent-petithory/dataurl"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

type objectWriter struct {
	buf *bytes.Buffer
}

func (ow *objectWriter) Abort() {
	ow.buf = nil
}

func (ow *objectWriter) Commit() (storage.CommitInfo, error) {
	if ow.buf == nil {
		return nil, storage.CommitOnAbortedObjectWriter
	}

	durl := dataurl.New(ow.buf.Bytes(), "binary/octet-stream")

	return storage.NewCommitInfo(durl.String()), nil
}

func (ow *objectWriter) Write(p []byte) (n int, err error) {
	n, err = ow.buf.Write(p)
	return
}
