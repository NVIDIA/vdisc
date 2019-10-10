// Copyright © 2018 NVIDIA Corporation

package iso9660

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/NVIDIA/vdisc/pkg/storage"
)

type Volume struct {
	pvd           PrimaryVolumeDescriptor
	inodeAlloc    InodeAllocator
	root          *DirectoryInode
	now           time.Time
	nameValidator NameValidator
}

func NewVolume() *Volume {
	return NewPosixPortableVolume()
}

//NewPosixPortableVolume return *Volume that allows POSIX portable directory names
func NewPosixPortableVolume() *Volume {
	ia := NewInodeAllocator()
	validator := NewPosixPortableNameValidator()
	root, err := NewDirectoryInode(ia, validator)
	if err != nil {
		panic(err)
	}
	v := &Volume{
		inodeAlloc:    ia,
		root:          root,
		now:           time.Now(),
		nameValidator: validator,
	}

	root.SetCreated(v.now)
	root.SetModified(v.now)
	v.pvd.VolumeSetSize = 1
	v.pvd.VolumeSequenceNumber = 1
	v.pvd.Created = v.now
	v.pvd.Modified = v.now
	v.pvd.Effective = v.now
	return v
}

//NewPosixPortableVolume return *Volume that allows directory names in the NVIDIA extended character set
func NewNvidiaExtendedVolume() *Volume {
	ia := NewInodeAllocator()
	validator := NewNvidiaExtendedNameValidator()
	root, err := NewDirectoryInode(ia, validator)
	if err != nil {
		panic(err)
	}
	v := &Volume{
		inodeAlloc:    ia,
		root:          root,
		now:           time.Now(),
		nameValidator: validator,
	}

	root.SetCreated(v.now)
	root.SetModified(v.now)
	v.pvd.VolumeSetSize = 1
	v.pvd.VolumeSequenceNumber = 1
	v.pvd.Created = v.now
	v.pvd.Modified = v.now
	v.pvd.Effective = v.now
	return v
}

func (v *Volume) AddSymlink(pth string, target string) (err error) {
	var leaf Inode
	leaf, err = NewSymlinkInode(v.inodeAlloc, target)
	if err != nil {
		return
	}
	leaf.SetCreated(v.now)
	leaf.SetModified(v.now)

	err = v.addLeaf(pth, leaf)
	return
}

func (v *Volume) AddFile(pth string, o storage.Object) (err error) {
	var leaf *FileInode
	leaf, err = NewFileInode(v.inodeAlloc, o)
	if err != nil {
		return
	}
	leaf.SetCreated(v.now)
	leaf.SetModified(v.now)

	err = v.addLeaf(pth, leaf)
	return
}

func (v *Volume) addLeaf(pth string, leaf Inode) (err error) {
	cleaned := path.Clean(pth)
	if path.IsAbs(cleaned) {
		cleaned = cleaned[1:]
	}
	parts := strings.Split(cleaned, "/")

	parent := v.root
	for i, part := range parts {
		if i == len(parts)-1 {
			err = parent.AddChild(part, leaf)
			return
		}
		var dir *DirectoryInode
		inode, ok := parent.GetChild(part)
		if ok {
			if inode.Type() != InodeTypeDirectory {
				err = errors.New("Path segment exists and is not a directory")
				return
			}
			dir = inode.(*DirectoryInode)
		} else {
			dir, err = NewDirectoryInode(v.inodeAlloc, v.nameValidator)
			if err != nil {
				return
			}
			dir.SetCreated(v.now)
			dir.SetModified(v.now)

			err = parent.AddChild(part, dir)
			if err != nil {
				return
			}
		}
		parent = dir
	}
	return
}

func (v *Volume) VisitFiles(visit func(storage.Object) error) error {
	return v.root.VisitFiles(func(rel Relationship) error {
		finode := rel.Child.(*FileInode)
		return visit(finode.Object())
	})
}

func (v *Volume) VisitFileInodes(visit func(*FileInode) error) error {
	return v.root.VisitFiles(func(rel Relationship) error {
		finode := rel.Child.(*FileInode)
		return visit(finode)
	})
}

func (v *Volume) SetSystemIdentifier(val string) {
	v.pvd.SystemIdentifier = val
}

func (v *Volume) SetVolumeIdentifier(val string) {
	v.pvd.VolumeIdentifier = val
}

func (v *Volume) SetVolumeSetIdentifier(val string) {
	v.pvd.VolumeSetIdentifier = val
}

func (v *Volume) SetPublisherIdentifier(val string) {
	v.pvd.PublisherIdentifier = val
}

func (v *Volume) SetDataPreparerIdentifier(val string) {
	v.pvd.DataPreparerIdentifier = val
}

func (v *Volume) SetApplicationIdentifier(val string) {
	v.pvd.ApplicationIdentifier = val
}

func (v *Volume) SetCopyrightFileIdentifier(val string) {
	v.pvd.CopyrightFileIdentifier = val
}

func (v *Volume) SetAbstractFileIdentifier(val string) {
	v.pvd.AbstractFileIdentifier = val
}

func (v *Volume) SetBibliographicFileIdentifier(val string) {
	v.pvd.BibliographicFileIdentifier = val
}

func (v *Volume) assignLogicalBlockAddresses() error {
	sectors := NewSectorAllocator()

	sectors.Alloc(16 * 2048) // System Use Area
	sectors.Alloc(2048)      // Primary Volume Descriptor
	sectors.Alloc(2048)      // Terminator Volume Descriptor
	v.pvd.LTableStart = sectors.Alloc(v.pvd.PathTableSize)
	v.pvd.MTableStart = sectors.Alloc(v.pvd.PathTableSize)

	err := v.root.VisitDirectories(func(rel Relationship) error {
		dinode := rel.Child.(*DirectoryInode)
		d, c, err := dinode.ToDirectory()
		if err != nil {
			return err
		}

		// Allocate space for the directory
		dsize := uint32(d.Size())
		dinode.SetStart(sectors.Alloc(dsize))
		dinode.SetSize(dsize)

		if dinode.IsRoot() {
			v.pvd.RootStart = dinode.Start()
			v.pvd.RootLength = dsize
			v.pvd.RootModified = dinode.Modified()
		}

		// Allocate space for the continuation area if needed
		clen := c.Len()
		if clen > 0 {
			sectors.Alloc(uint32(clen))
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Allocate sectors for each FileInode. Only allocate sectors once
	// per FileInode.
	inodesVisited := make(map[InodeNumber]struct{})
	v.root.VisitFiles(func(rel Relationship) error {
		finode := rel.Child.(*FileInode)
		if _, ok := inodesVisited[finode.InodeNumber()]; !ok {
			inodesVisited[finode.InodeNumber()] = struct{}{}
			parts := finode.Parts()
			finode.SetStart(sectors.Alloc(parts[0].Size()))
			for i := 1; i < len(parts); i++ {
				sectors.Alloc(parts[i].Size())
			}
		}
		return nil
	})

	v.pvd.VolumeSpaceSize = sectors.Allocated()
	return nil
}

func (v *Volume) WriteMetadataTo(w io.Writer) (int64, error) {
	cw := newCountingWriter(w)

	padOutSector := func() error {
		remainder := int(cw.Written() % LogicalBlockSize)
		if remainder > 0 {
			return pad(cw, LogicalBlockSize-remainder)
		}
		return nil
	}

	assertLBA := func(expected LogicalBlockAddress) {
		if cw.Written()%LogicalBlockSize != 0 {
			panic(cw.Written())
		}
		actual := LogicalBlockAddress(cw.Written() / LogicalBlockSize)
		if expected != actual {
			panic(fmt.Errorf("expected=%d, actual=%d", expected, actual))
		}
	}

	pathTable := v.root.ToPathTable()
	v.pvd.PathTableSize = uint32(PathTableEncodedLen(&pathTable))

	if err := v.assignLogicalBlockAddresses(); err != nil {
		return cw.Written(), err
	}

	// regenerate the path table now that we've assigned blocks
	pathTable = v.root.ToPathTable()

	// System Use Area
	if err := pad(cw, 16*LogicalBlockSize); err != nil {
		return cw.Written(), err
	}

	// Primary Volume Descriptor
	if _, err := v.pvd.WriteTo(cw); err != nil {
		return cw.Written(), err
	}

	// Terminator
	var terminator Terminator
	if _, err := terminator.WriteTo(cw); err != nil {
		return cw.Written(), err
	}

	// L-Table
	assertLBA(v.pvd.LTableStart)
	lenc := NewPathTableEncoder(binary.LittleEndian, cw)
	if _, err := lenc.Encode(&pathTable); err != nil {
		return cw.Written(), err
	}
	if err := padOutSector(); err != nil {
		return cw.Written(), err
	}

	// M-Table
	assertLBA(v.pvd.MTableStart)
	menc := NewPathTableEncoder(binary.BigEndian, cw)
	if _, err := menc.Encode(&pathTable); err != nil {
		return cw.Written(), err
	}
	if err := padOutSector(); err != nil {
		return cw.Written(), err
	}

	// Directory Extents
	err := v.root.VisitDirectories(func(rel Relationship) error {
		dinode := rel.Child.(*DirectoryInode)
		assertLBA(dinode.Start())

		d, c, err := dinode.ToDirectory()
		if err != nil {
			return err
		}

		if _, err := d.WriteTo(cw); err != nil {
			return err
		}

		if err := padOutSector(); err != nil {
			return err
		}

		if c.Len() > 0 {
			if _, err := c.WriteTo(cw); err != nil {
				return err
			}
		}

		if err := padOutSector(); err != nil {
			return err
		}

		return nil
	})

	return cw.Written(), err
}

func (v *Volume) WriteTo(w io.Writer) (int64, error) {
	cw := newCountingWriter(w)

	padOutSector := func() error {
		remainder := int(cw.Written() % LogicalBlockSize)
		if remainder > 0 {
			return pad(cw, LogicalBlockSize-remainder)
		}
		return nil
	}

	assertLBA := func(expected LogicalBlockAddress) {
		if cw.Written()%LogicalBlockSize != 0 {
			panic(cw.Written())
		}
		actual := LogicalBlockAddress(cw.Written() / LogicalBlockSize)
		if expected != actual {
			panic(fmt.Errorf("expected=%d, actual=%d", expected, actual))
		}
	}

	if _, err := v.WriteMetadataTo(cw); err != nil {
		return cw.Written(), err
	}

	err := v.root.VisitFiles(func(rel Relationship) error {
		finode := rel.Child.(*FileInode)
		if finode.Start() == 0 {
			return nil
		}
		assertLBA(finode.Start())
		if _, err := finode.WriteTo(cw); err != nil {
			return err
		}

		if err := padOutSector(); err != nil {
			return err
		}

		return nil
	})

	return cw.Written(), err
}
