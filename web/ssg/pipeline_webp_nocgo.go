//go:build !cgo

package ssg

import (
	"fmt"
)

func encodeWebp(b []byte, quality int) ([]byte, error) {
	return nil, fmt.Errorf("webp: format conversion requires CGO or a dedicated webp plugin")
}
