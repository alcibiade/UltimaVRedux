package sprites

import (
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

// GetCurrentImageAtTime returns the animation frame for a specific time (deterministic)
// Use this variant for core game logic that needs to be reproducible
func (s *ImageAnimation) GetCurrentImageAtTime(elapsedMs int64) *ebiten.Image {
	nFrame := cycleIndex(len(s.images), elapsedMs/s.millisecondsBetweenChange, 0)

	return s.images[nFrame]
}
