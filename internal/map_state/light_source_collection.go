package map_state

import (
	"cmp"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (l *LayeredMap) getVisibleStaticLightSources(center references.Position) LightSources {
	slices.SortFunc(l.visiblePositions, func(a, b references.Position) int {
		if order := cmp.Compare(a.X, b.X); order != 0 {
			return order
		}

		return cmp.Compare(a.Y, b.Y)
	})
	l.visiblePositions = slices.Compact(l.visiblePositions)
	sources := make(LightSources, 0)
	for _, position := range l.visiblePositions {
		// Preserve the original half-open scan domain and raw overflow identities.
		if position.X < center.X-l.XMaxTilesPerMap || position.X >= center.X+l.XMaxTilesPerMap ||
			position.Y < center.Y-l.YMaxTilesPerMap || position.Y >= center.Y+l.YMaxTilesPerMap ||
			!l.visibleFlags[position.X][position.Y] {
			continue
		}
		if tile := l.GetTileTopMapOnlyTile(&position); tile != nil && tile.GetLightEmission() > 0 {
			sources = append(sources, LightSource{Tile: tile, Pos: position})
		}
	}

	return sources
}
