// Copyright © 2018 NVIDIA Corporation

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
