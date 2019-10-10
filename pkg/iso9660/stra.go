// Copyright © 2018 NVIDIA Corporation

package iso9660

import (
	"fmt"
	"strings"
)

var (
	StrARunes map[rune]struct{}
)

func StrA(input string, length int) string {
	var b strings.Builder
	for i, r := range strings.ToUpper(input) {
		if i >= length {
			break
		}
		if _, ok := StrARunes[r]; ok {
			b.WriteRune(r)
		} else {
			b.WriteRune('?')
		}
	}
	return fmt.Sprintf(fmt.Sprintf("%%-%ds", length), b.String())
}

func init() {
	StrARunes = make(map[rune]struct{})
	allowed := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H',
		'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U',
		'V', 'W', 'X', 'Y', 'Z', '0', '1', '2', '3', '4', '5', '6', '7',
		'8', '9', '_', '!', '"', '%', '&', '\'', '(', ')', '*', '+', ',',
		'-', '.', '/', ':', ';', '<', '=', '>', '?'}
	for _, r := range allowed {
		StrARunes[r] = struct{}{}
	}
}
