package utils

import (
	"fmt"
	"strconv"
)

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
