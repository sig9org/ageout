// Package size parses file sizes accepted by ageout's command-line flags.
package size

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Parse converts a size such as "10K" or "100" to bytes. Unit suffixes are
// binary (K=1024, M=1024^2, G=1024^3) and are case-insensitive.
func Parse(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("size must not be empty")
	}
	multiplier := int64(1)
	number := s
	switch strings.ToLower(s[len(s)-1:]) {
	case "b":
		number = s[:len(s)-1]
	case "k":
		number = s[:len(s)-1]
		multiplier = 1024
	case "m":
		number = s[:len(s)-1]
		multiplier = 1024 * 1024
	case "g":
		number = s[:len(s)-1]
		multiplier = 1024 * 1024 * 1024
	}
	if number == "" {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	n, err := strconv.ParseInt(number, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	if n > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("size %q is too large", s)
	}
	return n * multiplier, nil
}
