package utils

import (
	"fmt"
	"strconv"
	"strings"
)

type ByteSize int64

func (b *ByteSize) UnmarshalText(text []byte) error {
	str := strings.TrimSpace(string(text))
	if str == "" {
		*b = 0
		return nil
	}

	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}

	suffixes := []string{"GB", "MB", "KB", "B"}

	str = strings.ToUpper(str)

	for _, suffix := range suffixes {
		if strings.HasSuffix(str, suffix) {
			numPart := strings.TrimSuffix(str, suffix)
			value, err := strconv.ParseFloat(strings.TrimSpace(numPart), 64)
			if err != nil {
				return fmt.Errorf("invalid size value %q: %w", str, err)
			}
			*b = ByteSize(value * float64(multipliers[suffix]))
			return nil
		}
	}

	value, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid size %q: %w", str, err)
	}
	*b = ByteSize(value)
	return nil
}

func (b *ByteSize) Int64() int64 {
	return int64(*b)
}
