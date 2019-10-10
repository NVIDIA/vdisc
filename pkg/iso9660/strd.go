// Copyright © 2018 NVIDIA Corporation

package iso9660

import (
	"fmt"
	"strings"
)

var (
	StrDRunes map[rune]struct{}
)

func StrD(input string, length int) string {
	var b strings.Builder
	for i, r := range strings.ToUpper(input) {
		if i >= length {
			break
		}
		if _, ok := StrDRunes[r]; ok {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}

	return fmt.Sprintf(fmt.Sprintf("%%-%ds", length), b.String())
}

func init() {
	StrDRunes = make(map[rune]struct{})
	allowed := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H',
		'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U',
		'V', 'W', 'X', 'Y', 'Z', '0', '1', '2', '3', '4', '5', '6', '7',
		'8', '9', '_'}
	for _, r := range allowed {
		StrDRunes[r] = struct{}{}
	}
}
