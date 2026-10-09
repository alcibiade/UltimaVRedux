package sprites

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

type ImageAnimation struct {
	images                    []*ebiten.Image
	millisecondsBetweenChange int64
}

func NewImageAnimation(images []*ebiten.Image, millisecondsBetweenChange int64) *ImageAnimation {
	return &ImageAnimation{images: images, millisecondsBetweenChange: millisecondsBetweenChange}
}

func NewSpriteSlice(rawSprites [][]byte) []*ebiten.Image {
	images := make([]*ebiten.Image, len(rawSprites))
	for i, sprite := range rawSprites {
		img, err := png.Decode(bytes.NewReader(sprite))
		if err != nil {
			panic(fmt.Sprintf("decoding embedded animation sprite %d: %v", i, err))
		}
		ebitenSprite := ebiten.NewImageFromImage(img)
		images[i] = ebitenSprite
	}

	return images
}

// GetCurrentImageAtTime returns the animation frame for a specific time (deterministic)
// Use this variant for core game logic that needs to be reproducible
func (s *ImageAnimation) GetCurrentImageAtTime(elapsedMs int64) *ebiten.Image {
	nFrame := int((elapsedMs / s.millisecondsBetweenChange) % int64(len(s.images)))

	return s.images[nFrame]
}
