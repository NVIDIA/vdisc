// Copyright © 2019 NVIDIA Corporation
package swiftdriver

import (
	"os"
	"sync"
)

const DefaultRegion = "us-east-1"

var (
	cachedRegion     string
	cachedRegionOnce sync.Once
)

func GetSwiftRegion() string {
	cachedRegionOnce.Do(func() {
		var reg string
		reg = os.Getenv("SWIFT_REGION")
		if len(reg) < 1 {
			reg = DefaultRegion
		}

		cachedRegion = reg
	})

	return cachedRegion
}
