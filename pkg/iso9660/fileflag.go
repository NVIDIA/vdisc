// Copyright © 2018 NVIDIA Corporation

package iso9660

type FileFlag byte

const (
	FileFlagHidden FileFlag = 1 << iota
	FileFlagDir
	FileFlagAssociated
	FileFlagExtendedFormatInfo
	FileFlagExtendedPermissions
	FileFlagReserved1
	FileFlagReserved2
	FileFlagNonTerminal // this is not the final directory record for this file (for files spanning several extents, for example files over 4GiB long.
)
