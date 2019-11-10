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

package isofuse

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/dgraph-io/ristretto"
	"github.com/jacobsa/fuse/fuseops"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
)

type FileInfoCache interface {
	Put(parent fuseops.InodeID, name string, fi *iso9660.FileInfo)
	Get(parent fuseops.InodeID, name string) (*iso9660.FileInfo, bool)
}

func NewFileInfoCache(maxEntries int64) (FileInfoCache, error) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: maxEntries * 10,
		MaxCost:     maxEntries * 176,
		BufferItems: 64,
		OnEvict:     func(key uint64, value interface{}, cost int64) {},
		KeyToHash:   finfoHash,
		Cost:        finfoCost,
	})
	if err != nil {
		return nil, err
	}
	return &finfoCache{c}, nil
}

type finfoCache struct {
	c *ristretto.Cache
}

func (fc *finfoCache) Put(parent fuseops.InodeID, name string, fi *iso9660.FileInfo) {
	fc.c.Set(&finfoKey{parent, name}, fi, finfoCost(fi))
}

func (fc *finfoCache) Get(parent fuseops.InodeID, name string) (*iso9660.FileInfo, bool) {
	v, ok := fc.c.Get(&finfoKey{parent, name})
	if !ok {
		return nil, false
	}

	return v.(*iso9660.FileInfo), true
}

type finfoKey struct {
	parent fuseops.InodeID
	name   string
}

func finfoHash(key interface{}) uint64 {
	k := key.(*finfoKey)
	h := fnv.New64()
	binary.Write(h, binary.LittleEndian, k.parent)
	h.Write([]byte(k.name))
	return h.Sum64()
}

func finfoCost(value interface{}) int64 {
	v := value.(*iso9660.FileInfo)
	return int64(len(v.Name()) + len(v.Target()) + 65)
}
