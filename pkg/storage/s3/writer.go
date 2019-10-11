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
package s3driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdurl "net/url"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"

	"github.com/NVIDIA/vdisc/pkg/s3util"
	"github.com/NVIDIA/vdisc/pkg/storage"
)

const (
	writerStateOpened int32 = iota
	writerStatePiped
	writerStateDirect
)

func NewObjectWriter(svc s3iface.S3API, bucket, key, canonicalURL string) storage.ObjectWriter {
	return &objectWriter{
		uploader:     s3util.NewS3Uploader(svc),
		wg:           &sync.WaitGroup{},
		bucket:       bucket,
		key:          key,
		canonicalURL: canonicalURL,
	}
}

type objectWriter struct {
	uploader     *s3util.S3Uploader
	pipeReader   *io.PipeReader
	pipeWriter   *io.PipeWriter
	wg           *sync.WaitGroup
	bucket       string
	key          string
	canonicalURL string
	state        int32
	err          error
	version      string
}

func (ow *objectWriter) startPiped() {
	ow.wg.Add(1)
	go func() {
		defer ow.wg.Done()

		resp, err := ow.uploader.Upload(&s3util.S3UploadInput{
			Bucket: aws.String(ow.bucket),
			Key:    aws.String(ow.key),
			Body:   ow.pipeReader,
		})
		if err != nil {
			ow.err = err
			ow.pipeReader.CloseWithError(err)
		} else {
			ow.pipeReader.Close()
			if resp.VersionID != nil {
				ow.storeVersion(*resp.VersionID)
			}
		}
	}()
}

func (ow *objectWriter) ReadFromContext(ctx context.Context, r io.Reader, counter *int64) error {
	if atomic.CompareAndSwapInt32(&ow.state, writerStateOpened, writerStateDirect) {
		resp, err := ow.uploader.UploadWithContext(aws.Context(ctx), &s3util.S3UploadInput{
			Bucket: aws.String(ow.bucket),
			Key:    aws.String(ow.key),
			Body:   r,
		}, func(u *s3util.S3Uploader) {
			u.ProgressCounter = counter
		})
		if err != nil {
			return err
		}
		if resp.VersionID != nil {
			ow.storeVersion(*resp.VersionID)
		}
		return nil
	} else if atomic.LoadInt32(&ow.state) == writerStatePiped {
		return fmt.Errorf("can't mix ReadFrom and Write")
	}
	return fmt.Errorf("only one ReadFrom call supported")
}

func (ow *objectWriter) ReadFrom(r io.Reader) (int64, error) {
	count := int64(0)
	err := ow.ReadFromContext(context.Background(), r, &count)
	return atomic.LoadInt64(&count), err
}

func (ow *objectWriter) Write(b []byte) (int, error) {
	if atomic.CompareAndSwapInt32(&ow.state, writerStateOpened, writerStatePiped) {
		ow.pipeReader, ow.pipeWriter = io.Pipe()
		go ow.startPiped()
	} else if atomic.LoadInt32(&ow.state) == writerStateDirect {
		return 0, fmt.Errorf("can't mix ReadFrom and Write")
	}
	return ow.pipeWriter.Write(b)
}

func (ow *objectWriter) Commit() (storage.CommitInfo, error) {
	if atomic.CompareAndSwapInt32(&ow.state, writerStateOpened, writerStatePiped) {
		// Nothing written yet, start the writer now to produce a zero-byte file.
		ow.pipeReader, ow.pipeWriter = io.Pipe()
		go ow.startPiped()
	}
	if atomic.LoadInt32(&ow.state) == writerStatePiped {
		ow.pipeWriter.Close()
		ow.wg.Wait()
	}

	if ow.err != nil {
		return nil, ow.err
	}
	return storage.NewCommitInfo(ow.canonicalURL), nil
}

func (ow *objectWriter) Abort() {
	if atomic.LoadInt32(&ow.state) == writerStatePiped {
		// TODO: log error
		ow.pipeWriter.CloseWithError(errors.New("abort file creation"))
		ow.wg.Wait()
	}
}

func (ow *objectWriter) storeVersion(version string) {
	ow.version = version
	u, err := stdurl.Parse(ow.canonicalURL)
	if err != nil {
		ow.err = err
		return
	}
	if version != "" && (u.Scheme == "s3" || u.Scheme == "swift") {
		q := u.Query()
		q.Set("versionId", version)
		u.RawQuery = q.Encode()
		ow.canonicalURL = u.String()
	}
}
