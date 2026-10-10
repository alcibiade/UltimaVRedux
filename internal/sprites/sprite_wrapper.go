// Package sprites provides sprite rendering and manipulation functionality.
// This file implements the 18x18 sprite wrapper system that adds a 1-pixel transparent
// border around 16x16 sprites to allow for complete outline rendering.
package sprites

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// OriginalTileSize is the original sprite size
	OriginalTileSize = 16
	// ExpandedTileSize is the size of sprites after adding the 1-pixel border
	ExpandedTileSize = 18
)

// ExpandedSprite caches both original and expanded versions of a sprite
type ExpandedSprite struct {
	original16x16 *ebiten.Image // Original 16x16 sprite
	expanded18x18 *ebiten.Image // Expanded 18x18 sprite with border
}

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

	// Create 18x18 image with transparent background
	expanded := ebiten.NewImage(ExpandedTileSize, ExpandedTileSize)

	// Draw original 16x16 sprite centered with 1-pixel offset
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(1, 1) // Center in the 18x18 canvas
	expanded.DrawImage(sprite16, op)

	return expanded
}

// GetOriginal returns the original 16x16 sprite
func (es *ExpandedSprite) GetOriginal() *ebiten.Image {
	return es.original16x16
}

// GetExpanded returns the expanded 18x18 sprite
func (es *ExpandedSprite) GetExpanded() *ebiten.Image {
	return es.expanded18x18
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
