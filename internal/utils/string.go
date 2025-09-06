package utils

import (
	"fmt"
	"strconv"
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

// StringToUint converts a string to uint
// Returns 0 if the string is empty
// Returns error if the string cannot be converted to uint
func StringToUint(s string) (uint, error) {
	if s == "" {
		return 0, nil
	}
	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid uint value: %w", err)
	}
	return uint(val), nil
}
