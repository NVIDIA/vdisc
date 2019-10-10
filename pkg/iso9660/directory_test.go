package iso9660_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
	"github.com/NVIDIA/vdisc/pkg/iso9660/susp"
)

func TestDirectory(t *testing.T) {
	expected := iso9660.Directory{}

	for i := iso9660.LogicalBlockAddress(0); i < 1024; i++ {
		expected.Records = append(expected.Records, iso9660.DirectoryRecord{
			Identifier:               "ASDF;txt",
			Start:                    17 + (i + 3),
			Length:                   5000,
			Flags:                    0,
			Recorded:                 time.Unix(1, 0).UTC(),
			SystemUse:                []susp.SystemUseEntry{susp.NewSharingProtocolEntry(0)},
			ExtendedAttrRecordLength: 0,
			FileUnitSize:             0,
			InterleaveGap:            0,
			VolumeID:                 1,
		})
	}

	buf := bytes.NewBuffer(nil)
	expected.WriteTo(buf)

	var actual iso9660.Directory
	iso9660.DecodeDirectory(buf, &actual)

	assert.Equal(t, expected, actual)
}
