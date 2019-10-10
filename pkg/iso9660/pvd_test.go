package iso9660_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
)

func TestPVD(t *testing.T) {
	expected := iso9660.PrimaryVolumeDescriptor{
		SystemIdentifier:            "SYSTEM_IDENTIFIER",
		VolumeIdentifier:            "VOLUME_IDENTIFIER",
		VolumeSetIdentifier:         "VOLUME_SET_IDENTIFIER",
		PublisherIdentifier:         "PUBLISHER_IDENTIFIER",
		DataPreparerIdentifier:      "DATA_PREPARER_IDENTIFIER",
		ApplicationIdentifier:       "APPLICATION_IDENTIFIER",
		CopyrightFileIdentifier:     "COPYRIGHT_FILE_IDENTIFIER",
		AbstractFileIdentifier:      "ABSTRACT_FILE_IDENTIFIER",
		BibliographicFileIdentifier: "BIBLIOGRAPHIC_FILE_IDENTIFIER",
		VolumeSpaceSize:             20,
		VolumeSetSize:               2,
		VolumeSequenceNumber:        1,
		PathTableSize:               3,
		LTableStart:                 17,
		OptionalLTableStart:         0,
		MTableStart:                 20,
		OptionalMTableStart:         0,
		RootStart:                   24,
		RootLength:                  2048,
		RootModified:                time.Unix(1, 0).UTC(),
		Created:                     time.Unix(2, 0).UTC(),
		Modified:                    time.Unix(3, 0).UTC(),
		Effective:                   time.Unix(4, 0).UTC(),
	}

	buf := bytes.NewBuffer(nil)
	expected.WriteTo(buf)

	var actual iso9660.PrimaryVolumeDescriptor
	iso9660.DecodePrimaryVolumeDescriptor(buf, &actual)

	assert.Equal(t, expected, actual)
}
