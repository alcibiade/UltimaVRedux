package sprites

import (
	"bytes"
	"fmt"
	"image/png"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

// ImageAnimation owns game frame selection; image borrows retain their existing owners.
type ImageAnimation struct {
	images                    []*ebiten.Image
	millisecondsBetweenChange int64
}

// NewImageAnimation freezes a supplied frame list without taking image ownership.
func NewImageAnimation(images []*ebiten.Image, millisecondsBetweenChange int64) *ImageAnimation {
	if len(images) == 0 || millisecondsBetweenChange <= 0 {
		panic("invalid game image animation")
	}

	return &ImageAnimation{images: slices.Clone(images), millisecondsBetweenChange: millisecondsBetweenChange}
}

// NewSpriteSlice decodes game-owned embedded PNG animation frames.
func NewSpriteSlice(rawSprites [][]byte) []*ebiten.Image {
	images := make([]*ebiten.Image, len(rawSprites))
	for index, sprite := range rawSprites {
		img, err := png.Decode(bytes.NewReader(sprite))
		if err != nil {
			panic(fmt.Sprintf("decoding embedded animation sprite %d: %v", index, err))
		}
		ebitenSprite := ebiten.NewImageFromImage(img)
		images[index] = ebitenSprite
	}

	return images
}

// GetCurrentImageAtTime returns the animation frame for a specific time (deterministic)
// Use this variant for core game logic that needs to be reproducible
func (s *ImageAnimation) GetCurrentImageAtTime(elapsedMs int64) *ebiten.Image {
	nFrame := cycleIndex(len(s.images), elapsedMs/s.millisecondsBetweenChange, 0)

	return s.images[nFrame]
}
