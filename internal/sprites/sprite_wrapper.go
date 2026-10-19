// Package sprites provides sprite rendering and manipulation functionality.
// This file implements the 18x18 sprite wrapper system that adds a 1-pixel transparent
// border around 16x16 sprites to allow for complete outline rendering.
package sprites

import (
	"fmt"

	graphics "github.com/bradhannah/berad-sdk/modules/berad-graphics"
	graphicsbackend "github.com/bradhannah/berad-sdk/modules/berad-graphics/backend/ebitengine"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// OriginalTileSize is the original sprite size
	OriginalTileSize = 16
	// ExpandedTileSize is the size of sprites after adding the 1-pixel border
	ExpandedTileSize = 18
)

// WrapSpriteWith18x18Border takes a 16x16 sprite and returns an 18x18 version
// with a 1-pixel transparent border on all sides. This allows outlines to
// completely wrap around sprites that have pixels at the edges.
func WrapSpriteWith18x18Border(sprite16 *ebiten.Image) *ebiten.Image {
	if sprite16 == nil {
		return nil
	}

	// Verify input is 16x16
	bounds := sprite16.Bounds()
	if bounds.Dx() != OriginalTileSize || bounds.Dy() != OriginalTileSize {
		// If not 16x16, return as-is (handles special cases)
		return sprite16
	}

	expanded, err := graphicsbackend.CopyRegion(sprite16, bounds, graphics.Insets{Left: 1, Top: 1, Right: 1, Bottom: 1})
	if err != nil {
		panic(fmt.Sprintf("pad original sprite: %v", err))
	}

	return expanded
}

// ConvertPosition16to18 converts a position for 16x16 tile grid to account
// for the -1 pixel offset needed for 18x18 sprites with overlap
func ConvertPosition16to18(tileX, tileY int) (float64, float64) {
	// Each tile is still positioned at 16-pixel intervals
	// but we subtract 1 to account for the border
	x := float64(tileX*OriginalTileSize - 1)
	y := float64(tileY*OriginalTileSize - 1)

	return x, y
}
