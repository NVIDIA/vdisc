// Copyright © 2018 NVIDIA Corporation

package iso9660

type Visitor func(Relationship) error

type Relationship struct {
	Parent     *DirectoryInode
	Identifier string
	Name       string
	Child      Inode
}
