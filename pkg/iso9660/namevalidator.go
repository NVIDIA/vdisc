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

import (
	"fmt"
	"unicode"
)

//NameValidator is an interface for validating directory names
type NameValidator interface {
	//IsValid returns an error if name is not valid, else return nil
	IsValid(name string) error
}

//PosixPortableNameValidator enforces POSIX portable directory names
type PosixPortableNameValidator struct {
	allowedRunes map[rune]struct{}
}

//NewPosixPortableNameValidator creates a *PosixPortableNameValidator
func NewPosixPortableNameValidator() *PosixPortableNameValidator {
	//Expect only one Validator per vdisc Builder, thus a very small footprint
	//If there are many instatiated Validators, the posixPortableRunes map could be made a global
	posixPortableRunes := make(map[rune]struct{})
	allowed := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I',
		'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U',
		'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g',
		'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's',
		't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4',
		'5', '6', '7', '8', '9', '_', '-', '.'}
	for _, r := range allowed {
		posixPortableRunes[r] = struct{}{}
	}
	return &PosixPortableNameValidator{allowedRunes: posixPortableRunes}
}

//IsValid validates name against POSIX portable character set
func (ppnv *PosixPortableNameValidator) IsValid(s string) error {
	for i, r := range s {
		if _, ok := ppnv.allowedRunes[r]; !ok {
			return fmt.Errorf("PosixPortableNameValidator: invalid rune %v (%c) at position %d", r, r, i)
		}
	}
	return nil
}

//NvidiaExtendedNameValidator is directory NameValidator with an extend character set
type NvidiaExtendedNameValidator struct {
	invalidRunes map[rune]struct{}
}

//NewNvidiaExtendedNameValidator return a *NvidiaExtendedNameValidator
func NewNvidiaExtendedNameValidator() *NvidiaExtendedNameValidator {
	//Inspired by https://en.wikipedia.org/wiki/Filename#Reserved_characters_and_words
	//Same comment about memory and global map as for posixPortableRunes
	nvidiaExtendedInvalidRunes := make(map[rune]struct{})
	nvidiaInvalid := []rune{'/', '\\', '?', '%', '*', ':', '|', '"', '<', '>',
		' ', '\n', '\t', '$', '!'}
	for _, r := range nvidiaInvalid {
		nvidiaExtendedInvalidRunes[r] = struct{}{}
	}
	return &NvidiaExtendedNameValidator{invalidRunes: nvidiaExtendedInvalidRunes}
}

//IsValid validates name against custom NVIDIA character set
func (nenv *NvidiaExtendedNameValidator) IsValid(s string) error {
	for i, r := range s {
		if _, isInvalid := nenv.invalidRunes[r]; isInvalid {
			return fmt.Errorf("NvidiaExtendedNameValidator: invalid rune %v (%c) at position %d", r, r, i)
		}
		//!isPrint ensures control characters are excluded
		if !unicode.IsPrint(r) {
			return fmt.Errorf("NvidiaExtendedNameValidator: non-printable rune %v at position %d", r, i)
		}
	}
	return nil
}
