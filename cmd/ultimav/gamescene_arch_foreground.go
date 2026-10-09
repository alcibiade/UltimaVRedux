package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// drawSurfaceArchForeground retains the original ARCH occlusion from
// ref:COMBAT-0001. Existing arch art includes its ground, so it needs
// no decorative-base override. Redrawing in world space also occludes actors
// crossing the arch during camera interpolation, without offsetting the HUD.
func (g *GameScene) drawSurfaceArchForeground(screen *ebiten.Image) {
	location := g.gameState.MapState.PlayerLocation
	if location.Location.GetMapType() != references.SmallMapType {
		return
	}
	layer := g.gameState.GetLayeredMapByCurrentLocation()
	margin := g.surfaceOverscanTiles()
	for y := -margin; y < references.Coordinate(yTilesVisibleOnGameScreen)+margin; y++ {
		for x := -margin; x < references.Coordinate(xTilesVisibleOnGameScreen)+margin; x++ {
			position := references.Position{X: location.Position.X + x - xCenter, Y: location.Position.Y + y - yCenter}
			if g.gameState.IsOutOfBounds(position) || !g.isPositionVisible(layer, &position) {
				continue
			}
			tile := layer.GetTileTopMapOnlyTile(&position)
			if tile == nil || tile.Index != indexes.BrickWallArchway {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			px, py := sprites.ConvertPosition16to18(int(x), int(y))
			op.GeoM.Translate(px, py)
			op.GeoM.Concat(g.worldDrawOptions().GeoM)
			g.applyLightTinting(position, op, g.getElapsedMs())
			g.drawTilePaletteSprite(screen, g.spriteSheet.GetAnimatedSprite(tile.Index, position.AnimationHash(), g.getElapsedMs()), op, tile.Index)
		}
	}
}
