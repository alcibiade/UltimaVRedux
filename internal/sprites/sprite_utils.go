package sprites

import (
	"bytes"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// NewPngSprite decodes embedded PNG bytes into an Ebitengine image, panicking on invalid data.
func NewPngSprite(rawBytes []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		panic(fmt.Sprintf("decoding embedded PNG sprite: %v", err))
	}

	return ebiten.NewImageFromImage(img)
}
