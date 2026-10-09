package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// MirageTerrainSprite applies the original terrain-buffer substitution; ref:TILE-0008.
// Squared distance from the view center gives the range on the 11x11
// viewport. In particular its threshold 5 is squared distance, not five tiles.
// This is presentation only; callers must retain the authored collision tile.
func MirageTerrainSprite(tile indexes.SpriteIndex, position, viewer Position, mapType GeneralMapType) indexes.SpriteIndex {
	if mapType != SmallMapType && mapType != LargeMapType {
		return tile
	}
	if tile != indexes.Oasis && tile != indexes.Desert3 {
		return tile
	}
	dx, dy := int(position.X-viewer.X), int(position.Y-viewer.Y)
	if mapType == LargeMapType {
		dx, dy = position.GetWrappedAxisDistances(viewer)
	}
	const mirageNearRange = 5
	if dx*dx+dy*dy <= mirageNearRange {
		return indexes.Desert3
	}

	return indexes.Oasis
}
