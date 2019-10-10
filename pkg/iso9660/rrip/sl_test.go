package rrip_test

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660/rrip"
)

func TestSymlink(t *testing.T) {
	target := "/../foo/./../x.txt"
	expected := path.Clean(target)

	entries, err := rrip.NewSymlink(target)
	if err != nil {
		t.Fatal(err)
	}

	actual, ok := rrip.DecodeSymlink(entries)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	actual, ok = rrip.DecodeSymlink(nil)
	assert.False(t, ok)
	assert.Equal(t, "", actual)
}
