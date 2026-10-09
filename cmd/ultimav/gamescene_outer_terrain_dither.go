package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/profiling"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// Fade only the artificial terrain outside a small map, over three artwork tiles.
const outerTerrainDitherPixels = 3 * sprites.TileSize

type outerTerrainKey struct {
	origin, size image.Point
	mapBounds    image.Rectangle
}

func (g *GameScene) usesOuterTerrainDither() bool {
	return g.usesSurfaceMapFrame() && g.gameState.MapState.PlayerLocation.Location.GetMapType() == references.SmallMapType
}

// drawOuterTerrainDither replaces the outer terrain's tile-row darkening with
// world-anchored pixel rejection. Apply before scaling and entity composition:
// this decoration must not change LOS, admit hidden tiles, or mask entities.
// It also remains stable when workflows or X-ray disable the visibility mask.
func (g *GameScene) drawOuterTerrainDither() {
	if !g.usesOuterTerrainDither() {
		return
	}
	location := g.gameState.GetCurrentSmallLocationReference()
	mapBounds := image.Rect(0, 0, (int(location.GetMaxX())+1)*sprites.TileSize, (int(location.GetMaxY())+1)*sprites.TileSize)
	margin := g.surfaceOverscanTiles()
	player := g.gameState.MapState.PlayerLocation.Position
	origin := image.Pt(int(player.X-xCenter-margin)*sprites.TileSize, int(player.Y-yCenter-margin)*sprites.TileSize)
	target := g.unscaledMapImage
	if g.surfaceCamera.active {
		target = g.surfaceTerrainImage
	}
	width, height := target.Bounds().Dx(), target.Bounds().Dy()
	if (image.Rectangle{Min: origin, Max: origin.Add(image.Pt(width, height))}).In(mapBounds) {
		return
	}
	key := outerTerrainKey{origin: origin, size: image.Pt(width, height), mapBounds: mapBounds}
	if g.outerDitherValid && g.outerDitherKey == key {
		g.outerTerrainDither.Draw(target, ebiten.GeoM{})

		return
	}
	pixels := g.outerTerrainDither.Resize(width, height)
	const bytesPerPixel, opaque = 4, 255
	for y := range height {
		for x := range width {
			worldX, worldY := origin.X+x, origin.Y+y
			coverage := outerTerrainCoverage(worldX, worldY, mapBounds)
			if !sprites.DitherKeeps(coverage, worldX, worldY) {
				pixels[(y*width+x)*bytesPerPixel+3] = opaque
			}
		}
	}
	g.outerTerrainDither.Upload()
	profiling.Count(profiling.OuterDitherBuilds)
	g.outerDitherKey, g.outerDitherValid = key, true
	g.outerTerrainDither.Draw(target, ebiten.GeoM{})
}

func outerTerrainCoverage(worldX, worldY int, mapBounds image.Rectangle) uint16 {
	// Doubled pixel centers make the fade symmetric at opposite edges without
	// floating point. Max distance preserves the map's rectangular corner shape.
	const doubled = 2
	x, y := doubled*worldX+1, doubled*worldY+1
	distance := max(0, doubled*mapBounds.Min.X-x, x-doubled*mapBounds.Max.X,
		doubled*mapBounds.Min.Y-y, y-doubled*mapBounds.Max.Y)
	width := doubled * outerTerrainDitherPixels

	return uint16(max(0, width-distance) * sprites.DitherCoverageFull / width)
}
