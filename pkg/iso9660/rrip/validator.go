// Copyright © 2018 NVIDIA Corporation

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
