// Copyright © 2018 NVIDIA Corporation

package iso9660

import (
	"encoding/base32"
	"encoding/binary"
)

var (
	shortEncoding = base32.StdEncoding.WithPadding('_')
)

// Assign unique (hidden) names to directory entries
type IdentifierAllocator struct {
	count uint64
}

func NewIdentifierAllocator() *IdentifierAllocator {
	return &IdentifierAllocator{0}
}

func (ia *IdentifierAllocator) Next() string {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, ia.count)
	ia.count += 1
	src := buf[:n]

	dst := make([]byte, shortEncoding.EncodedLen(len(src)))
	shortEncoding.Encode(dst, src)
	return string(dst)
}
