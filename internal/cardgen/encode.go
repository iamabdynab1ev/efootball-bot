package cardgen

import (
	"bytes"
	"image"
	"image/png"
)

func encodePNGStdlib(buf *bytes.Buffer, img image.Image) error {
	return png.Encode(buf, img)
}
