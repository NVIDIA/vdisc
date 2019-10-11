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
