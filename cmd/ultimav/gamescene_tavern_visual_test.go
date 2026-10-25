package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestTavernMealTableOverlayRendersNorthSignalAndTracksMapVisit(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantTavernBritain)
	layer := fixture.state.GetLayeredMapByCurrentLocation()
	require.NotNil(t, layer)
	player := fixture.state.MapState.PlayerLocation.Position
	north := player.GetPositionUp()
	south := player.GetPositionDown()
	layer.SetTileByLayer(map_state.MapLayer, north, indexes.TableMiddle)
	layer.SetTileByLayer(map_state.MapLayer, south, indexes.TableMiddle)
	layer.UnSetTileByLayer(map_state.MapOverrideLayer, north)
	layer.UnSetTileByLayer(map_state.MapOverrideLayer, south)

	fixture.scene.StartTalkWorkflow(fixture.direction)
	dialog := waitForMerchantDialog[*widgets.TavernShopDialog](t, fixture)
	completeTavernAction(t, fixture, dialog, ebiten.KeyEnter)
	overlay := fixture.scene.currentTavernMealTableOverlay()
	require.NotNil(t, overlay)
	require.Equal(t, references.Britain, overlay.location)
	require.Equal(t, fixture.state.MapState.PlayerLocation.Floor, overlay.floor)
	require.Same(t, layer, overlay.layer)
	require.Equal(t, *north, overlay.position, "authority must preserve its north-first table signal")
	require.Equal(t, indexes.SpriteIndex(indexes.TableFoodBottom), overlay.sprite)
	require.True(t, layer.GetTileTopMapOnlyTile(north).Is(indexes.TableMiddle), "presentation must not mutate the map")

	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	require.NotNil(t, fixture.scene.currentTavernMealTableOverlay(), "closing the Tavern must retain the table")

	fixture.scene.mapImage = ebiten.NewImage(
		sprites.TileSize*xTilesVisibleOnGameScreen,
		sprites.TileSize*yTilesVisibleOnGameScreen,
	)
	fixture.scene.spriteSheet.SpriteImage.Fill(color.RGBA{R: 0xf1, G: 0x22, B: 0x33, A: 0xff})
	fixture.scene.xRayVisionActive = true
	fixture.scene.screenShakeOffsetX = 3
	fixture.scene.screenShakeOffsetY = 2
	width, height := ebiten.WindowSize()
	screen := newPresentationRaster(width, height)
	fixture.scene.drawTavernMealTableOverlay(screen)
	sourceX, sourceY := sprites.ConvertPosition16to18(int(xCenter), int(yCenter-1))
	mapOptions := sprites.GetDrawOptionsFromPercentsForWholeScreen(fixture.scene.mapImage, gameScreenPercents)
	mapOptions.GeoM.Translate(fixture.scene.screenShakeOffsetX, fixture.scene.screenShakeOffsetY)
	sampleX, sampleY := mapOptions.GeoM.Apply(sourceX+8, sourceY+8)
	_, _, _, alpha := screen.At(int(sampleX), int(sampleY)).RGBA()
	require.NotZero(t, alpha, "the overlay must use the composed map scale and screen shake transform")

	fixture.state.MapState.PlayerLocation.Position.X++
	require.NotNil(t, fixture.scene.currentTavernMealTableOverlay(), "movement within the same map visit must retain the table")
	fixture.state.MapState.PlayerLocation.Floor++
	require.Nil(t, fixture.scene.currentTavernMealTableOverlay(), "changing floor must clear the table")

	fixture.state.MapState.PlayerLocation.Floor--
	visual := game_state.TavernMealTableVisual{
		Show: true, Position: *north, SpriteIndex: indexes.TableFoodBottom,
	}
	fixture.scene.installTavernMealTableVisual(visual)
	beforeReload := fixture.state.GetLayeredMapByCurrentLocation()
	fixture.state.UpdateSmallMap(
		fixture.state.GameReferences.TileReferences,
		fixture.state.GameReferences.LocationReferences,
	)
	require.NotSame(t, beforeReload, fixture.state.GetLayeredMapByCurrentLocation())
	require.Nil(t, fixture.scene.currentTavernMealTableOverlay(), "reloading the same floor must clear the old pointer identity")

	fixture.scene.installTavernMealTableVisual(visual)
	fixture.state.MapState.PlayerLocation.Location = references.Paws
	require.Nil(t, fixture.scene.currentTavernMealTableOverlay(), "changing location must clear the table")
}
