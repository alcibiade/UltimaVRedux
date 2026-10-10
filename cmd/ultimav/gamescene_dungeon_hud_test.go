package main

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	gameText "github.com/bradhannah/UltimaVRedux/internal/text"
)

func TestDungeonHUDFormatsOriginalLevelAndCardinalSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		facing    references.DungeonFacing
		cardinal  string
		direction string
	}{
		{name: "north", facing: references.DungeonFacingNorth, cardinal: "North", direction: "Dir: North"},
		{name: "east", facing: references.DungeonFacingEast, cardinal: "East", direction: "Dir: East"},
		{name: "south", facing: references.DungeonFacingSouth, cardinal: "South", direction: "Dir: South"},
		{name: "west", facing: references.DungeonFacingWest, cardinal: "West", direction: "Dir: West"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			status := dungeonHUDStatus{
				position: references.Position{X: 7, Y: 2},
				floor:    6,
				facing:   test.facing,
			}
			assert.Equal(t, test.cardinal, dungeonFacingCardinal(test.facing))
			assert.Equal(t, "L 7", status.levelText())
			assert.Equal(t, test.direction, status.directionText())
			assert.Equal(t, "X: 7 Y: 2 Floor: 6 Level: 7 Facing: "+test.cardinal, status.debugText())
		})
	}
}

func TestDungeonHUDLayoutStaysInsideMapPaneAtSupportedResolutions(t *testing.T) {
	t.Parallel()

	resolutions := []image.Point{
		{X: 1280, Y: 720},
		{X: 1366, Y: 768},
		{X: 1920, Y: 1080},
		{X: 2560, Y: 1440},
		{X: 3840, Y: 2160},
	}
	for _, resolution := range resolutions {
		bounds := image.Rectangle{Max: resolution}
		resolutionScale := float64(resolution.X) / 1920
		statusHeight := int(math.Ceil(18 * resolutionScale))
		debugHeight := int(math.Ceil(12 * resolutionScale))
		layout := calculateDungeonHUDLayout(bounds, statusHeight, statusHeight, debugHeight)
		railStart := int(math.Floor(float64(resolution.X) * dungeonHUDRightRailStart))

		assert.Equal(t, int(math.Round(float64(resolution.X)*gameScreenPercents.EndPercentX)), layout.frameBounds.Max.X)
		assert.Equal(t, railStart, layout.safePaneBounds.Max.X)
		assert.Equal(t, layout.frameBounds.Min.X, layout.levelPoint.X)
		assert.Equal(t, layout.frameBounds.Min.X, layout.directionPoint.X)
		assert.GreaterOrEqual(t, layout.levelPoint.Y, bounds.Min.Y)
		assert.LessOrEqual(t, layout.levelPoint.Y+statusHeight, bounds.Min.Y+layout.borderBandHeight)
		assert.GreaterOrEqual(t, layout.directionPoint.Y, bounds.Max.Y-layout.borderBandHeight)
		assert.LessOrEqual(t, layout.directionPoint.Y+statusHeight, bounds.Max.Y)
		assert.GreaterOrEqual(t, layout.debugBounds.Min.X, layout.safePaneBounds.Min.X)
		assert.LessOrEqual(t, layout.debugBounds.Max.X, railStart)
		assert.GreaterOrEqual(t, layout.debugBounds.Min.Y, bounds.Min.Y+layout.borderBandHeight)
		assert.LessOrEqual(t, layout.debugBounds.Max.Y, bounds.Max.Y)
	}
}

func TestDrawDungeonHUDRendersStatusAndDebugWithoutEnteringRightRail(t *testing.T) {
	font := gameText.NewUltimaFont(16)
	levelOutput := gameText.NewOutput(font, 16, 1, dungeonStatusMaxChars)
	directionOutput := gameText.NewOutput(font, 16, 1, dungeonStatusMaxChars)
	debugOutput := gameText.NewOutput(font, 16, 1, dungeonDebugMaxChars)
	scene := &GameScene{ //nolint:exhaustruct // Focused presentation fixture.
		gameState: &game_state.GameState{DebugOptions: references.DebugOptions{DialogBoundingBoxes: true}},
		dungeonHUD: dungeonHUDOutputs{
			level:     levelOutput,
			direction: directionOutput,
			debug:     debugOutput,
		},
	}
	screen := newPresentationRaster(1280, 720)
	status := dungeonHUDStatus{
		position: references.Position{X: 3, Y: 5},
		floor:    7,
		facing:   references.DungeonFacingWest,
	}

	scene.drawDungeonHUD(screen, status)

	levelHeight := int(math.Ceil(levelOutput.GetTotalHeightForLines(1)))
	directionHeight := int(math.Ceil(directionOutput.GetTotalHeightForLines(1)))
	debugHeight := int(math.Ceil(debugOutput.GetTotalHeightForLines(1)))
	layout := calculateDungeonHUDLayout(screen.Bounds(), levelHeight, directionHeight, debugHeight)
	require.True(t, dungeonHUDRegionHasVisiblePixel(screen, image.Rect(
		layout.frameBounds.Min.X,
		layout.levelPoint.Y,
		layout.frameBounds.Max.X,
		layout.levelPoint.Y+levelHeight,
	)))
	require.True(t, dungeonHUDRegionHasVisiblePixel(screen, image.Rect(
		layout.frameBounds.Min.X,
		layout.directionPoint.Y,
		layout.frameBounds.Max.X,
		layout.directionPoint.Y+directionHeight,
	)))
	require.True(t, dungeonHUDRegionHasVisiblePixel(screen, layout.debugBounds))
	assert.False(t, dungeonHUDRegionHasVisiblePixel(screen, image.Rect(
		layout.safePaneBounds.Max.X,
		screen.Bounds().Min.Y,
		screen.Bounds().Max.X,
		screen.Bounds().Max.Y,
	)))
	assert.Equal(t, "L 8", levelOutput.GetOutputStr(false))
	assert.Equal(t, "Dir: West", directionOutput.GetOutputStr(false))
	assert.Equal(t, "X: 3 Y: 5 Floor: 7 Level: 8 Facing: West", debugOutput.GetOutputStr(false))
	scene.gameState.DebugOptions.DialogBoundingBoxes = false
	screen.Clear()
	scene.drawDungeonHUD(screen, status)
	assert.False(t, dungeonHUDRegionHasVisiblePixel(screen, layout.debugBounds), "ordinary play hides diagnostic coordinates")
	assert.Empty(t, debugOutput.GetOutputStr(false))
	assert.Equal(t, "L 8", levelOutput.GetOutputStr(false))
}

func dungeonHUDRegionHasVisiblePixel(screen *ebiten.Image, bounds image.Rectangle) bool {
	bounds = bounds.Intersect(screen.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := screen.At(x, y).RGBA()
			if alpha != 0 {
				return true
			}
		}
	}

	return false
}
