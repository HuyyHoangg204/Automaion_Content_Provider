package utils

import (
	"fmt"
	"strings"
	"unicode"
)

// ExtractNumbersToUint extracts all numbers from a string and converts them to uint
func ExtractNumbersToUint(s string) (uint, error) {
	var numbers strings.Builder
	for _, char := range s {
		if unicode.IsDigit(char) {
			numbers.WriteRune(char)
		}
	}

	if numbers.Len() == 0 {
		return 0, fmt.Errorf("no numbers found in string: %s", s)
	}

	return StringToUint(numbers.String())
}
