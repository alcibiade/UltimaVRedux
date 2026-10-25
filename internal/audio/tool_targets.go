package audio

import (
	"slices"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TileTargets derives Redux source descriptors from its actual animation atlas.
func TileTargets() []TileTarget {
	var result []TileTarget
	for sprite, tile := range *references.NewTileReferences() {
		if sprite < 0 || indexes.CanonicalAnimationSprite(sprite) != sprite {
			continue
		}
		item := TileTarget{ID: strconv.Itoa(int(sprite)), Name: tile.Name, Frames: []int{int(sprite)}}
		if family, animated := indexes.AnimationFor(sprite); animated {
			item.Name = strings.TrimRight(tile.Name, "0123456789")
			item.Frames = nil
			for frame := range family.Frames {
				item.Frames = append(item.Frames, int(family.Base)+frame)
			}
		}
		result = append(result, item)
	}
	slices.SortFunc(result, func(a, b TileTarget) int { return a.Frames[0] - b.Frames[0] })

	return result
}
