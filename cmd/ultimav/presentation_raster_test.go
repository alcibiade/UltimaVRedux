package main

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"

	"github.com/hajimehoshi/ebiten/v2"
)

// newPresentationRaster is an owned offscreen test target, not sprite artwork.
// Standalone storage keeps a large transient canvas out of atlases shared with
// persistent tiny sprites. Callers retain their existing Deallocate cleanup.
func newPresentationRaster(width, height int) *ebiten.Image {
	return ebiten.NewImageWithOptions(image.Rect(0, 0, width, height), &ebiten.NewImageOptions{Unmanaged: true})
}

// Independent scene fixtures are separate application lifetimes. Retaining their
// shared caches until the end of a 1,000-test package is unrelated to the tested
// warm-draw contract; reset reuse references when each fixture retires instead.
func retirePresentationFixtureCaches(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		text.ClearTextCache()
		sprites.ClearBorderCache()
	})
}
