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

package rrip

import "unicode"

//isValid is fail-safe validation for Rock Ridge names and parts.
//The naming validation is done during in iso9660 when adding children to a directory.
//This valid must be equal or less restrictive than any implementation of the NameValidator interface defined in iso9660.
func isValid(s string) bool {
	for _, r := range s {
		//UNIX-compatible names
		if !unicode.IsPrint(r) || r == '/' {
			return false
		}
	}
	return true
}
