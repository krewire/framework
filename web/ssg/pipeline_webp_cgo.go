//go:build cgo

package ssg

import (
	"bytes"
	"fmt"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

func encodeWebp(b []byte, quality int) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("webp decode: %w", err)
	}
	if quality <= 0 || quality > 100 {
		quality = 80
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: float32(quality), Exact: true}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
