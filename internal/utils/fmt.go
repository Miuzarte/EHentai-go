package utils

import (
	"fmt"
	"strings"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~uintptr
}

type Number interface {
	Integer | ~float32 | ~float64
}

func FormatBytes[T Integer](bytes T) string {
	b := uint64(bytes)
	const unit uint64 = 1024
	const sizeUnits = "KMGTPE"
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := unit, uint64(0)
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	format := fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), sizeUnits[exp])
	return strings.ReplaceAll(format, ".00", "")
}
