package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// canonicalAudioCoordinate supports repeated wrapping on small dungeon grids;
// audio scanning may extend farther than a single map width in either direction.
func canonicalAudioCoordinate(value, size references.Coordinate) references.Coordinate {
	return (value%size + size) % size
}

// projectAudioCoordinate chooses the nearest image of a canonical source. The
// negative displacement wins an exact half-map tie, independent of scan order.
func projectAudioCoordinate(listener, source, size references.Coordinate) references.Coordinate {
	delta := canonicalAudioCoordinate(source-listener+size/2, size) - size/2

	return listener + delta
}

func (g *GameScene) projectAmbientSources(listener references.Position, sources []audio.AmbientSoundSource) {
	current := g.gameState.GetLayeredMapByCurrentLocation()
	location := g.gameState.MapState.PlayerLocation
	for index := range sources {
		source := &sources[index]
		position := source.Position
		if current != nil && current.IsWrappedMap() {
			position.X = canonicalAudioCoordinate(position.X, current.XMaxTilesPerMap)
			position.Y = canonicalAudioCoordinate(position.Y, current.YMaxTilesPerMap)
			source.Position.X = projectAudioCoordinate(listener.X, position.X, current.XMaxTilesPerMap)
			source.Position.Y = projectAudioCoordinate(listener.Y, position.Y, current.YMaxTilesPerMap)
		}
		// Identity uses canonical coordinates and map ownership, never the nearest
		// projected image, so crossing a seam cannot restart/duplicate the emitter.
		source.ID = fmt.Sprintf("%d/%d/%s/%d/%d", location.Location, location.Floor, source.Emitter, position.X, position.Y)
	}
}
