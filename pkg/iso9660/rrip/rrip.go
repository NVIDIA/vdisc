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

import (
	"encoding/binary"
	"io"

	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

// Rock Ridge Interchange Protocol (RRIP, IEEE P1282)
// See ftp://ftp.ymi.com/pub/rockridge/rrip112.ps

const (
	RockRidgeExtensionVersion    = 1
	RockRidgeExtensionIdentifier = "IEEE_P1282"
	RockRidgeExtensionDescriptor = "THE IEEE P1282 PROTOCOL PROVIDES SUPPORT FOR POSIX FILE SYSTEM SEMANTICS."
	RockRidgeExtensionSource     = "PLEASE CONTACT THE IEEE STANDARDS DEPARTMENT, PISCATAWAY, NJ, USA FOR THE P1282 SPECIFICATION."
)

const (
	RockRidgeExtensionVersionLegacy    = 1
	RockRidgeExtensionIdentifierLegacy = "RRIP_1991A"
	RockRidgeExtensionDescriptorLegacy = "THE ROCK RIDGE INTERCHANGE PROTOCOL PROVIDES SUPPORT FOR POSIX FILE SYSTEM SEMANTICS"
	RockRidgeExtensionSourceLegacy     = "PLEASE CONTACT DISC PUBLISHER FOR SPECIFICATION SOURCE.  SEE PUBLISHER IDENTIFIER IN PRIMARY VOLUME DESCRIPTOR FOR CONTACT INFORMATION."
)

var (
	ExtensionsReference       susp.SystemUseEntry
	ExtensionsReferenceLegacy susp.SystemUseEntry
)

func init() {
	var err error
	ExtensionsReference, err = susp.NewExtensionsReferenceEntry(
		RockRidgeExtensionVersion,
		RockRidgeExtensionIdentifier,
		RockRidgeExtensionDescriptor,
		RockRidgeExtensionSource)
	if err != nil {
		panic(err)
	}

	ExtensionsReferenceLegacy, err = susp.NewExtensionsReferenceEntry(
		RockRidgeExtensionVersionLegacy,
		RockRidgeExtensionIdentifierLegacy,
		RockRidgeExtensionDescriptorLegacy,
		RockRidgeExtensionSourceLegacy)
	if err != nil {
		panic(err)
	}
}

func writeByte(w io.Writer, b byte) (err error) {
	_, err = w.Write([]byte{b})
	return
}

func putBothUint32(w io.Writer, v uint32) error {
	if err := binary.Write(w, binary.LittleEndian, v); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return err
	}

	return nil
}
