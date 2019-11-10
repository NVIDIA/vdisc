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

package s3util

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/NVIDIA/vdisc/pkg/countio"
)

// Forked from the AWS implementation.

type readerAtSeeker interface {
	io.ReaderAt
	io.ReadSeeker
}

type S3UploadInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`
	Key    *string `location:"uri" locationName:"Key" type:"string" required:"true"`
	Body   io.Reader
}

type S3UploadOutput struct {
	Location  string
	VersionID *string
	UploadID  string
	ETag      *string
}

type S3Uploader struct {
	PartSize        int64
	Concurrency     int
	MaxUploadParts  int
	ProgressCounter *int64
	S3              s3iface.S3API
}

func NewS3Uploader(svc s3iface.S3API, options ...func(*S3Uploader)) *S3Uploader {
	u := &S3Uploader{
		S3:             svc,
		PartSize:       s3manager.DefaultUploadPartSize,
		Concurrency:    s3manager.DefaultUploadConcurrency,
		MaxUploadParts: s3manager.MaxUploadParts,
	}

	for _, option := range options {
		option(u)
	}

	return u
}

func (u S3Uploader) Upload(input *S3UploadInput, options ...func(*S3Uploader)) (*S3UploadOutput, error) {
	return u.UploadWithContext(aws.BackgroundContext(), input, options...)
}

func (u S3Uploader) UploadWithContext(ctx aws.Context, input *S3UploadInput, opts ...func(*S3Uploader)) (*S3UploadOutput, error) {
	i := uploader{in: input, cfg: u, ctx: ctx}

	for _, opt := range opts {
		opt(&i.cfg)
	}

	return i.upload()
}

// internal structure to manage an upload to S3.
type uploader struct {
	ctx aws.Context
	cfg S3Uploader

	in *S3UploadInput

	readerPos int64 // current reader position
	totalSize int64 // set to -1 if the size is not known

	bufferPool sync.Pool
}

// internal logic for deciding whether to upload a single part or use a
// multipart upload.
func (u *uploader) upload() (*S3UploadOutput, error) {
	u.init()

	if u.cfg.PartSize < s3manager.MinUploadPartSize {
		msg := fmt.Sprintf("part size must be at least %d bytes", s3manager.MinUploadPartSize)
		return nil, awserr.New("ConfigError", msg, nil)
	}

	// Do one read to determine if we have more than one part
	reader, _, part, err := u.nextReader()
	if err == io.EOF { // single part
		return u.singlePart(reader)
	} else if err != nil {
		return nil, awserr.New("ReadRequestBody", "read upload data failed", err)
	}

	mu := multiuploader{uploader: u}
	return mu.upload(reader, part)
}

// init will initialize all default options.
func (u *uploader) init() {
	if u.cfg.Concurrency == 0 {
		u.cfg.Concurrency = s3manager.DefaultUploadConcurrency
	}
	if u.cfg.PartSize == 0 {
		u.cfg.PartSize = s3manager.DefaultUploadPartSize
	}
	if u.cfg.MaxUploadParts == 0 {
		u.cfg.MaxUploadParts = s3manager.MaxUploadParts
	}

	u.bufferPool = sync.Pool{
		New: func() interface{} { return make([]byte, u.cfg.PartSize) },
	}

	// Try to get the total size for some optimizations
	u.initSize()
}

// initSize tries to detect the total stream size, setting u.totalSize. If
// the size is not known, totalSize is set to -1.
func (u *uploader) initSize() {
	u.totalSize = -1

	switch r := u.in.Body.(type) {
	case io.Seeker:
		n, err := aws.SeekerLen(r)
		if err != nil {
			return
		}
		u.totalSize = n

		// Try to adjust partSize if it is too small and account for
		// integer division truncation.
		if u.totalSize/u.cfg.PartSize >= int64(u.cfg.MaxUploadParts) {
			// Add one to the part size to account for remainders
			// during the size calculation. e.g odd number of bytes.
			u.cfg.PartSize = (u.totalSize / int64(u.cfg.MaxUploadParts)) + 1
		}
	}
}

// nextReader returns a seekable reader representing the next packet of data.
// This operation increases the shared u.readerPos counter, but note that it
// does not need to be wrapped in a mutex because nextReader is only called
// from the main thread.
func (u *uploader) nextReader() (readerAtSeeker, int, []byte, error) {
	type readerAtSeeker interface {
		io.ReaderAt
		io.ReadSeeker
	}
	switch r := u.in.Body.(type) {
	case readerAtSeeker:
		var err error

		n := u.cfg.PartSize
		if u.totalSize >= 0 {
			bytesLeft := u.totalSize - u.readerPos

			if bytesLeft <= u.cfg.PartSize {
				err = io.EOF
				n = bytesLeft
			}
		}
		reader := io.NewSectionReader(r, u.readerPos, n)
		u.readerPos += n

		return reader, int(n), nil, err

	default:
		part := u.bufferPool.Get().([]byte)
		n, err := readFillBuf(r, part)
		u.readerPos += int64(n)

		return bytes.NewReader(part[0:n]), n, part, err
	}
}

func readFillBuf(r io.Reader, b []byte) (offset int, err error) {
	for offset < len(b) && err == nil {
		var n int
		n, err = r.Read(b[offset:])
		offset += n
	}

	return offset, err
}

func wrapReader(cfg S3Uploader, r readerAtSeeker) readerAtSeeker {
	if cfg.ProgressCounter != nil {
		return countio.NewReaderAtSeekerWithAtomicCounter(r, cfg.ProgressCounter)
	}
	return r
}

// singlePart contains upload logic for uploading a single chunk via
// a regular PutObject request. Multipart requests require at least two
// parts, or at least 5MB of data.
func (u *uploader) singlePart(buf readerAtSeeker) (*S3UploadOutput, error) {
	params := &s3.PutObjectInput{}
	awsutil.Copy(params, u.in)
	params.Body = wrapReader(u.cfg, buf)

	// Need to use request form because URL generated in request is
	// used in return.
	req, out := u.cfg.S3.PutObjectRequest(params)

	err := computeHashes(buf, req)
	if err != nil {
		return nil, err
	}

	req.SetContext(u.ctx)
	if err := req.Send(); err != nil {
		return nil, err
	}

	url := req.HTTPRequest.URL.String()
	return &S3UploadOutput{
		Location:  url,
		VersionID: out.VersionId,
		ETag:      out.ETag,
	}, nil
}

// internal structure to manage a specific multipart upload to S3.
type multiuploader struct {
	*uploader
	wg       sync.WaitGroup
	m        sync.Mutex
	err      error
	uploadID string
	parts    completedParts
}

// keeps track of a single chunk of data being sent to S3.
type chunk struct {
	buf  readerAtSeeker
	part []byte
	num  int64
}

// completedParts is a wrapper to make parts sortable by their part number,
// since S3 required this list to be sent in sorted order.
type completedParts []*s3.CompletedPart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return *a[i].PartNumber < *a[j].PartNumber }

// upload will perform a multipart upload using the firstBuf buffer containing
// the first chunk of data.
func (u *multiuploader) upload(firstBuf readerAtSeeker, firstPart []byte) (*S3UploadOutput, error) {
	params := &s3.CreateMultipartUploadInput{}
	awsutil.Copy(params, u.in)

	// Create the multipart
	resp, err := u.cfg.S3.CreateMultipartUploadWithContext(u.ctx, params)
	if err != nil {
		return nil, err
	}
	u.uploadID = *resp.UploadId

	// Create the workers
	ch := make(chan chunk, u.cfg.Concurrency)
	for i := 0; i < u.cfg.Concurrency; i++ {
		u.wg.Add(1)
		go u.readChunk(ch)
	}

	// Send part 1 to the workers
	var num int64 = 1
	ch <- chunk{buf: firstBuf, part: firstPart, num: num}

	// Read and queue the rest of the parts
	for u.geterr() == nil && err == nil {
		num++
		// This upload exceeded maximum number of supported parts, error now.
		if num > int64(u.cfg.MaxUploadParts) || num > int64(s3manager.MaxUploadParts) {
			var msg string
			if num > int64(u.cfg.MaxUploadParts) {
				msg = fmt.Sprintf("exceeded total allowed configured MaxUploadParts (%d). Adjust PartSize to fit in this limit",
					u.cfg.MaxUploadParts)
			} else {
				msg = fmt.Sprintf("exceeded total allowed S3 limit MaxUploadParts (%d). Adjust PartSize to fit in this limit",
					s3manager.MaxUploadParts)
			}
			u.seterr(awserr.New("TotalPartsExceeded", msg, nil))
			break
		}

		var reader readerAtSeeker
		var nextChunkLen int
		var part []byte
		reader, nextChunkLen, part, err = u.nextReader()

		if err != nil && err != io.EOF {
			u.seterr(awserr.New(
				"ReadRequestBody",
				"read multipart upload data failed",
				err))
			break
		}

		if nextChunkLen == 0 {
			// No need to upload empty part, if file was empty to start
			// with empty single part would of been created and never
			// started multipart upload.
			break
		}

		ch <- chunk{buf: reader, part: part, num: num}
	}

	// Close the channel, wait for workers, and complete upload
	close(ch)
	u.wg.Wait()
	complete := u.complete()

	if err := u.geterr(); err != nil {
		return nil, fmt.Errorf("multipart upload failed: %s", err)
	}
	return &S3UploadOutput{
		Location:  aws.StringValue(complete.Location),
		VersionID: complete.VersionId,
		UploadID:  u.uploadID,
		ETag:      complete.ETag,
	}, nil
}

// readChunk runs in worker goroutines to pull chunks off of the ch channel
// and send() them as UploadPart requests.
func (u *multiuploader) readChunk(ch chan chunk) {
	defer u.wg.Done()
	for {
		data, ok := <-ch

		if !ok {
			break
		}

		if u.geterr() == nil {
			if err := u.send(data); err != nil {
				u.seterr(err)
			}
		}
	}
}

// send performs an UploadPart request and keeps track of the completed
// part information.
func (u *multiuploader) send(c chunk) error {
	params := &s3.UploadPartInput{
		Bucket:     u.in.Bucket,
		Key:        u.in.Key,
		Body:       wrapReader(u.cfg, c.buf),
		UploadId:   &u.uploadID,
		PartNumber: &c.num,
	}

	req, resp := u.cfg.S3.UploadPartRequest(params)

	err := computeHashes(c.buf, req)
	if err != nil {
		return err
	}

	req.SetContext(u.ctx)
	err = req.Send()
	if err != nil {
		return err
	}

	// put the byte array back into the pool to conserve memory
	u.bufferPool.Put(c.part)
	if err != nil {
		return err
	}

	n := c.num
	completed := &s3.CompletedPart{ETag: resp.ETag, PartNumber: &n}

	u.m.Lock()
	u.parts = append(u.parts, completed)
	u.m.Unlock()

	return nil
}

// geterr is a thread-safe getter for the error object
func (u *multiuploader) geterr() error {
	u.m.Lock()
	defer u.m.Unlock()

	return u.err
}

// seterr is a thread-safe setter for the error object
func (u *multiuploader) seterr(e error) {
	u.m.Lock()
	defer u.m.Unlock()

	u.err = e
}

// fail will abort the multipart unless LeavePartsOnError is set to true.
func (u *multiuploader) fail() {
	params := &s3.AbortMultipartUploadInput{
		Bucket:   u.in.Bucket,
		Key:      u.in.Key,
		UploadId: &u.uploadID,
	}
	u.cfg.S3.AbortMultipartUploadWithContext(u.ctx, params)
}

// complete successfully completes a multipart upload and returns the response.
func (u *multiuploader) complete() *s3.CompleteMultipartUploadOutput {
	if u.geterr() != nil {
		u.fail()
		return nil
	}

	// Parts must be sorted in PartNumber order.
	sort.Sort(u.parts)

	var resp *s3.CompleteMultipartUploadOutput
	var err error

	for i := 0; i < 20; i++ {
		params := &s3.CompleteMultipartUploadInput{
			Bucket:          u.in.Bucket,
			Key:             u.in.Key,
			UploadId:        &u.uploadID,
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: u.parts},
		}
		resp, err = u.cfg.S3.CompleteMultipartUploadWithContext(u.ctx, params)
		if err == nil {
			break
		}
	}

	if err != nil {
		u.seterr(err)
		u.fail()
	}

	return resp
}

func computeHashes(r readerAtSeeker, req *request.Request) error {
	hsha := sha256.New()
	if _, err := copySeekableBody(hsha, r); err != nil {
		return fmt.Errorf("failed to compute body hashes: %s", err)
	}
	req.HTTPRequest.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(hsha.Sum(nil)))
	return nil
}

func copySeekableBody(dst io.Writer, src io.ReadSeeker) (int64, error) {
	curPos, err := src.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	// hash the body.  seek back to the first position after reading to reset
	// the body for transmission.  copy errors may be assumed to be from the
	// body.
	n, err := io.Copy(dst, src)
	if err != nil {
		return n, err
	}

	_, err = src.Seek(curPos, io.SeekStart)
	if err != nil {
		return n, err
	}

	return n, nil
}
