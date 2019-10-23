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
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	stdurl "net/url"
	"os"
	"strings"
	"time"
)

var isoDate = "2006-01-02T15:04:05.000Z"

type ObjectVersion struct {
	Key          string
	LastModified string
	ETag         string
	Size         int64
	StorageClass string
	VersionId    string
	IsLatest     bool
}

func (ov *ObjectVersion) Modified() (time.Time, error) {
	return time.Parse(isoDate, ov.LastModified)
}

type DirEntry struct {
	Prefix string
}

type ListBucketPage struct {
	XMLName             xml.Name `xml:"ListVersionsResult"`
	Name                string
	Prefix              string
	KeyCount            int
	MaxKeys             int
	Version             []ObjectVersion
	CommonPrefixes      []DirEntry
	NextKeyMarker       *string
	NextVersionIdMarker *string
	IsTruncated         bool
}

type ListBucketCallback func(page *ListBucketPage) error

func ListBucket(client *http.Client, url string, callback ListBucketCallback) error {
	v := stdurl.Values{}
	v.Set("delimiter", "/")

	u, err := stdurl.Parse(url)
	if err != nil {
		return err
	}

	dirpath := u.Path
	if strings.HasPrefix(dirpath, "/") {
		dirpath = dirpath[1:]
	}
	if !strings.HasSuffix(dirpath, "/") {
		dirpath += "/"
	}
	u.Path = "/"
	if dirpath != "/" {
		v.Set("prefix", dirpath)
	}

	u.RawQuery = "versions&" + v.Encode()
	url = u.String()

	var nextVersionId *string
	var nextKey *string
	for {
		nextUrl := url
		if nextVersionId != nil || nextKey != nil {
			if nextKey != nil && nextVersionId != nil {
				nextUrl = fmt.Sprintf("%s&key-marker=%s&verion-id-marker%s", url, stdurl.QueryEscape(*nextKey), stdurl.QueryEscape(*nextVersionId))
			} else if nextKey != nil {
				nextUrl = fmt.Sprintf("%s&key-marker=%s", url, stdurl.QueryEscape(*nextKey))
			} else if nextVersionId != nil {
				nextUrl = fmt.Sprintf("%s&version-id-marker=%s", url, stdurl.QueryEscape(*nextVersionId))
			}
		}

		resp, err := client.Get(nextUrl)
		if err != nil {
			return err
		} else if resp == nil {
			return fmt.Errorf("s3 readdir: nil response without error")
		}
		defer resp.Body.Close()

		if (nextVersionId == nil && nextKey == nil) && resp.StatusCode == 404 {
			io.Copy(ioutil.Discard, resp.Body)
			return os.ErrNotExist
		}

		if resp.StatusCode != 200 {
			bodyText, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("s3 readdir: bad bucket listing %d %s", resp.StatusCode, bodyText)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var page ListBucketPage
		if err := xml.Unmarshal(body, &page); err != nil {
			return err
		}

		nextVersionId = page.NextVersionIdMarker
		nextKey = page.NextKeyMarker

		if err := callback(&page); err != nil {
			return err
		}
		if !page.IsTruncated {
			break
		}
	}

	return nil
}
