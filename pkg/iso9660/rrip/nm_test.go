package rrip_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NVIDIA/vdisc/pkg/iso9660/rrip"
)

func TestName(t *testing.T) {
	expected := strings.Repeat("x", 300)

	entries, err := rrip.NewName(expected)
	if err != nil {
		t.Fatal(err)
	}

	actual, ok := rrip.DecodeName(entries)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	actual, ok = rrip.DecodeName(nil)
	assert.False(t, ok)
	assert.Equal(t, "", actual)
}
