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
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

const DefaultRegion = "us-east-1"

var (
	cachedRegion     string
	cachedRegionOnce sync.Once
)

func getRegion() string {
	cachedRegionOnce.Do(func() {
		var reg string
		reg = os.Getenv("AWS_REGION")
		if len(reg) < 1 {
			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))

			reg = sess.ClientConfig("s3").SigningRegion

			if len(reg) < 1 {
				c := ec2metadata.New(session.New())
				reg, _ = c.Region()
			}

		}
		if len(reg) < 1 {
			reg = DefaultRegion
		}

		cachedRegion = reg
	})

	return cachedRegion
}

type s3BucketRegionResult struct {
	XMLName  xml.Name `xml:"LocationConstraint"`
	Location string   `xml:",chardata"`
}

func getBucketRegion(c *http.Client, bucketName string) regionPromise {
	return newRegionPromise(func() (string, error) {
		lreg := getRegion()
		domain := "com"
		if strings.HasPrefix(lreg, "cn-") {
			domain = "com.cn"
		}

		// This url pattern is compatible with all s3 endpoints.
		// https://docs.aws.amazon.com/general/latest/gr/rande.html
		resp, err := c.Get(fmt.Sprintf("https://%s.s3.%s.amazonaws.%s/?location", bucketName, lreg, domain))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		if resp.StatusCode != 200 {
			return "", fmt.Errorf("s3driver: error detecting bucket location: bucket=%s, status=%d, %s", bucketName, resp.StatusCode, string(body))
		}

		var res s3BucketRegionResult
		err = xml.Unmarshal(body, &res)
		if err != nil {
			return "", err
		}

		return res.Location, nil
	})
}

type regionFunc func() (string, error)

type regionPromise interface {
	Apply() (string, error)
}

type regionFuture struct {
	mu       *sync.Mutex
	c        *sync.Cond
	complete bool
	value    string
	err      error
}

func (f *regionFuture) Apply() (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for !f.complete {
		f.c.Wait()
	}
	return f.value, f.err
}

func newRegionPromise(op regionFunc) regionPromise {
	mu := &sync.Mutex{}
	f := &regionFuture{mu, sync.NewCond(mu), false, "", nil}
	go func() {
		r, err := op()
		f.mu.Lock()
		f.value = r
		f.err = err
		f.complete = true
		f.mu.Unlock()
		f.c.Broadcast()
	}()
	return f
}
