package main

import (
	stdcolor "image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const tavernStatusOverlayDurationMs = 2000

type tavernMealTableOverlay struct {
	location references.Location
	floor    references.FloorNumber
	layer    *map_state.LayeredMap
	position references.Position
	sprite   indexes.SpriteIndex
}

func (g *GameScene) showTavernStatusOverlay(message string, textColor stdcolor.Color) {
	overlay := widgets.NewOverlayDialog("").
		AsOverlayFeedback().
		WithExternallyManagedDuration(tavernStatusOverlayDurationMs).
		Build()
	overlay.AppendLineWithColor(message, textColor)
	g.installTimedOverlay(overlay)
}

func (g *GameScene) showTavernSoberedOverlay() {
	g.showTavernStatusOverlay("The party has sobered up.", pkgcolor.EGALightBlue)
}

func (g *GameScene) installTavernMealTableVisual(visual game_state.TavernMealTableVisual) {
	if !visual.Show || g.gameState == nil {
		return
	}
	current := g.gameState.MapState.PlayerLocation
	g.tavernMealTableOverlay = &tavernMealTableOverlay{
		location: current.Location,
		floor:    current.Floor,
		layer:    g.gameState.GetLayeredMapByCurrentLocation(),
		position: visual.Position,
		sprite:   visual.SpriteIndex,
	}
}

func (g *GameScene) currentTavernMealTableOverlay() *tavernMealTableOverlay {
	if g.tavernMealTableOverlay == nil || g.gameState == nil {
		return nil
	}
	current := g.gameState.MapState.PlayerLocation
	if current.Location != g.tavernMealTableOverlay.location ||
		current.Floor != g.tavernMealTableOverlay.floor ||
		g.gameState.GetLayeredMapByCurrentLocation() != g.tavernMealTableOverlay.layer {
		g.tavernMealTableOverlay = nil

		return nil
	}

	return g.tavernMealTableOverlay
}

func (g *GameScene) drawTavernMealTableOverlay(screen *ebiten.Image) {
	overlay := g.currentTavernMealTableOverlay()
	if screen == nil || overlay == nil || overlay.layer == nil || g.spriteSheet == nil || g.mapImage == nil {
		return
	}
	player := g.gameState.MapState.PlayerLocation.Position
	tileX := overlay.position.X - player.X + xCenter
	tileY := overlay.position.Y - player.Y + yCenter
	margin := g.surfaceOverscanTiles()
	if tileX < -margin || tileX >= xTilesVisibleOnGameScreen+margin || tileY < -margin || tileY >= yTilesVisibleOnGameScreen+margin ||
		!g.isPositionVisible(overlay.layer, &overlay.position) {
		return
	}
	sprite := g.spriteSheet.GetSprite(overlay.sprite)
	if sprite == nil {
		return
	}
	screenX, screenY := sprites.ConvertPosition16to18(int(tileX), int(tileY))
	drawOptions := &ebiten.DrawImageOptions{}
	drawOptions.GeoM.Translate(screenX, screenY)
	g.applyLightTinting(overlay.position, drawOptions, g.getElapsedMs())
	mapOptions := g.worldDrawOptions()
	drawOptions.GeoM.Concat(mapOptions.GeoM)
	screen.DrawImage(sprite, drawOptions)
}
