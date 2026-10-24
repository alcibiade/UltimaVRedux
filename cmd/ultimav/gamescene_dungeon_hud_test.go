package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	gameText "github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
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
			for floor := range 8 {
				status.floor = references.FloorNumber(floor)
				assert.Equal(t, fmt.Sprintf("L %d", floor+1), status.levelText())
			}
		})
	}
}

func TestDungeonHUDLayoutStaysInsideMapPaneAtSupportedResolutions(t *testing.T) {
	t.Parallel()

	resolutions := []image.Point{
		{X: 640, Y: 360},
		{X: 1001, Y: 701},
		{X: 1280, Y: 720},
		{X: 1366, Y: 768},
		{X: 1367, Y: 769},
		{X: 1920, Y: 1080},
		{X: 1921, Y: 1081},
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
	scene := newDungeonBorderHUDTestScene(t, image.Pt(1280, 720))
	scene.gameState.DebugOptions.DialogBoundingBoxes = true
	levelOutput, directionOutput, debugOutput := scene.dungeonHUD.level, scene.dungeonHUD.direction, scene.dungeonHUD.debug
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Deallocate)
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
	require.True(t, dungeonHUDRegionHasVisiblePixel(screen, scene.dungeonHUD.levelLabel.textBounds))
	require.True(t, dungeonHUDRegionHasVisiblePixel(screen, scene.dungeonHUD.directionLabel.textBounds))
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

func newDungeonBorderHUDTestScene(t *testing.T, size image.Point) *GameScene {
	t.Helper()
	manager := display.GetManager()
	saved := *manager
	// Restore the pending-change state as well as dimensions. Republishing the
	// old size during cleanup would leave a resize for the next Game.Update.
	// Register this first so fixture resources retire before global restoration.
	t.Cleanup(func() { *manager = saved })
	manager.PublishLogicalSize(size.X, size.Y)
	scene := &GameScene{gameState: &game_state.GameState{}}
	t.Cleanup(scene.releaseDungeonHUD)
	setDungeonBorderHUDTestSize(scene)

	return scene
}

func setDungeonBorderHUDTestSize(scene *GameScene) {
	scene.borders.outsideBorder, scene.borders.outsideBorderOp = sprites.NewBorderSprites().
		VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(borderWidthScaling, gameScreenPercents)
	_, scene.borders.surfaceAperture = surfaceMapBorderGeometry()
	scene.initializeDungeonHUDOutputs()
}

func TestDungeonHUDDecoratedBandsAtSupportedResolutions(t *testing.T) {
	for _, size := range []image.Point{
		{X: 640, Y: 360}, {X: 1001, Y: 701}, {X: 1280, Y: 720},
		{X: 1366, Y: 768}, {X: 1367, Y: 769}, {X: 1920, Y: 1080},
		{X: 1921, Y: 1081}, {X: 2560, Y: 1440}, {X: 3840, Y: 2160},
	} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			scene := newDungeonBorderHUDTestScene(t, size)
			outer, aperture := scene.borders.outsideBorder.Bounds(), scene.borders.surfaceAperture
			facings := []references.DungeonFacing{
				references.DungeonFacingNorth, references.DungeonFacingEast,
				references.DungeonFacingSouth, references.DungeonFacingWest,
			}
			for _, top := range []bool{true, false} {
				band := image.Rect(outer.Min.X, aperture.Max.Y, outer.Max.X, outer.Max.Y)
				region := image.Rect(0, aperture.Max.Y-2, size.X, size.Y)
				if top {
					band = image.Rect(outer.Min.X, outer.Min.Y, outer.Max.X, aperture.Min.Y)
					region = image.Rect(0, 0, size.X, aperture.Min.Y+2)
				}
				// Narrow full-width rasters include the rail and two aperture rows,
				// without retaining full-screen 4K targets for a border-only oracle.
				screen := ebiten.NewImageWithOptions(region, &ebiten.NewImageOptions{Unmanaged: true})
				defer screen.Deallocate()
				baseline := ebiten.NewImageWithOptions(region, &ebiten.NewImageOptions{Unmanaged: true})
				defer baseline.Deallocate()
				baseline.Fill(color.RGBA{R: 91, G: 37, B: 83, A: 255})
				baseline.DrawImage(scene.borders.outsideBorder, scene.borders.outsideBorderOp)
				want := dissolveMaskPixels(baseline)
				for floor := range 8 {
					status := dungeonHUDStatus{floor: references.FloorNumber(floor), facing: facings[floor%len(facings)]}
					screen.WritePixels(want) // Production redraws the frame before the labels.
					scene.drawDungeonHUD(screen, status)
					label := &scene.dungeonHUD.directionLabel
					value := status.directionText()
					if top {
						label, value = &scene.dungeonHUD.levelLabel, status.levelText()
					}
					assertDungeonBorderLabelLayout(t, label, band, size.X)
					assert.Equal(t, value, label.key.output.GetOutputStr(false))
					// Geometry at every supported size; detailed GPU oracles at the
					// compact, fractional and odd/even border-regression sizes.
					if size.X <= 1367 {
						assertDungeonBorderLabelPixels(t, screen, scene.borders.outsideBorder, label, band)
						assertDungeonLabelOutsideUnchanged(t, screen, want, label.backingBounds)
					}
				}
			}
		})
	}
}

func assertDungeonBorderLabelLayout(t *testing.T, label *dungeonBorderLabel, band image.Rectangle, screenWidth int) {
	t.Helper()
	require.True(t, label.backingBounds.In(band), "backing must remain inside actual border band: %v in %v", label.backingBounds, band)
	require.True(t, label.textBounds.In(label.backingBounds))
	assert.Equal(t, band.Dy(), label.backingBounds.Dy(), "mask every frame rule in the label area")
	assert.InDelta(t, float64(band.Min.X+band.Max.X)/2, float64(label.backingBounds.Min.X+label.backingBounds.Max.X)/2, 0.5)
	assert.Equal(t, label.backingBounds.Min.X+label.backingBounds.Max.X, label.textBounds.Min.X+label.textBounds.Max.X)
	assert.InDelta(t, float64(band.Min.Y+band.Max.Y)/2, float64(label.textBounds.Min.Y+label.textBounds.Max.Y)/2, 0.5)
	capWidth := max(1, band.Dy()/2)
	padding := max(1, int(math.Round(gameText.GetScaledNumberToWidth(screenWidth, windTextPadding))))
	assert.Equal(t, padding, label.textBounds.Min.X-label.backingBounds.Min.X-capWidth)
	assert.Equal(t, padding, label.backingBounds.Max.X-label.textBounds.Max.X-capWidth)
	width, _ := label.key.font.MeasureString(label.key.value)
	assert.Equal(t, int(math.Ceil(width)), label.textBounds.Dx())
}

func assertDungeonBorderLabelPixels(t *testing.T, screen, frame *ebiten.Image, label *dungeonBorderLabel, band image.Rectangle) {
	t.Helper()
	// Independently read the band's real blue and longest opaque white rule.
	stroke, run := 0, 0
	var blue color.Color
	for y := band.Min.Y; y < band.Max.Y; y++ {
		pixel := frame.At(band.Min.X+band.Dx()/2, y)
		if pixel == ucolor.EGAWhite.ToRGBA() {
			run++
			stroke = max(stroke, run)
		} else {
			run = 0
		}
		red, green, b, alpha := pixel.RGBA()
		if b > red && b > green && alpha == 65535 {
			blue = pixel
		}
	}
	require.Positive(t, stroke, "oracle must exercise a real white frame rule")
	require.NotNil(t, blue, "oracle must exercise the frame's actual blue")
	assert.Equal(t, stroke, label.borderStroke)
	assert.Equal(t, blue, label.borderBlue)
	backing := label.backingBounds
	capWidth := max(1, backing.Dy()/2)
	wantWhite := max(1, int(math.Round(float64(stroke)*math.Hypot(1, float64(2*(capWidth-1))/float64(max(1, backing.Dy()-1))))))
	previousReach := 0
	for y := range backing.Dy() {
		reach, white := 0, 0
		for x := range capWidth {
			left := screen.At(backing.Min.X+x, backing.Min.Y+y)
			right := screen.At(backing.Max.X-x-1, backing.Min.Y+y)
			require.Equal(t, left, right, "caps mirror at %d,%d", x, y)
			require.Equal(t, label.backing.At(x, y), left, "text cannot paint into a cap")
			require.Contains(t, []color.Color{ucolor.EGABlack.ToRGBA(), ucolor.EGAWhite.ToRGBA(), blue}, left)
			if left != ucolor.EGABlack.ToRGBA() {
				reach++
			}
			if left == ucolor.EGAWhite.ToRGBA() {
				white++
			}
		}
		assert.Equal(t, min(reach, wantWhite), white, "stroke perpendicular to diagonal matches frame")
		if y == 0 || y == backing.Dy()-1 {
			assert.Equal(t, 1, reach, "wedge endpoints connect to frame")
		}
		if y <= (backing.Dy()-1)/2 {
			assert.GreaterOrEqual(t, reach, previousReach, "left wedge points inward")
		} else {
			assert.LessOrEqual(t, reach, previousReach)
		}
		if y == backing.Dy()/2 {
			assert.GreaterOrEqual(t, reach, capWidth-1, "a vertical rule is not a wedge")
			require.Equal(t, blue, screen.At(backing.Min.X, backing.Min.Y+y), "cap interior uses frame blue")
		}
		previousReach = reach
	}
	textPixels := 0
	for y := backing.Min.Y; y < backing.Max.Y; y++ {
		for x := backing.Min.X + capWidth; x < backing.Max.X-capWidth; x++ {
			pixel := screen.At(x, y)
			if !image.Pt(x, y).In(label.textBounds) {
				require.Equal(t, ucolor.EGABlack.ToRGBA(), pixel, "opaque backing and cap clearance at %d,%d", x, y)
			} else if pixel != ucolor.EGABlack.ToRGBA() {
				textPixels++
			}
		}
	}
	assert.Positive(t, textPixels, "the label contains visible glyphs, not just decoration")
}

func assertDungeonLabelOutsideUnchanged(t *testing.T, screen *ebiten.Image, before []byte, changed image.Rectangle) {
	t.Helper()
	after := dissolveMaskPixels(screen)
	bounds := screen.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		start, end := (y-bounds.Min.Y)*bounds.Dx()*4, (y-bounds.Min.Y+1)*bounds.Dx()*4
		if y < changed.Min.Y || y >= changed.Max.Y {
			require.Equal(t, before[start:end], after[start:end], "outside label changed in row %d", y)
			continue
		}
		left := start + (changed.Min.X-bounds.Min.X)*4
		right := start + (changed.Max.X-bounds.Min.X)*4
		require.Equal(t, before[start:left], after[start:left], "left of label changed in row %d", y)
		require.Equal(t, before[right:end], after[right:end], "right of label changed in row %d", y)
	}
}

func TestDungeonHUDCacheTextFontResizeAndRelease(t *testing.T) {
	scene := newDungeonBorderHUDTestScene(t, image.Pt(1280, 720))
	screen := newPresentationRaster(1280, 720)
	defer screen.Deallocate()
	status := dungeonHUDStatus{floor: 0, facing: references.DungeonFacingSouth}
	scene.drawDungeonHUD(screen, status)
	level, direction := scene.dungeonHUD.levelLabel.backing, scene.dungeonHUD.directionLabel.backing
	levelOutput, directionOutput := scene.dungeonHUD.level, scene.dungeonHUD.direction
	font := levelOutput.Font
	for range 3 {
		scene.drawDungeonHUD(screen, status)
		assert.Same(t, level, scene.dungeonHUD.levelLabel.backing)
		assert.Same(t, direction, scene.dungeonHUD.directionLabel.backing)
		assert.Same(t, levelOutput, scene.dungeonHUD.level)
		assert.Same(t, font, scene.dungeonHUD.level.Font)
	}
	status.floor, status.facing = 7, references.DungeonFacingEast
	scene.drawDungeonHUD(screen, status)
	assert.NotSame(t, level, scene.dungeonHUD.levelLabel.backing)
	assert.NotSame(t, direction, scene.dungeonHUD.directionLabel.backing)
	assert.Equal(t, "L 8", levelOutput.GetOutputStr(false))
	assert.Equal(t, "Dir: East", directionOutput.GetOutputStr(false))
	assertDungeonBackingReleased(t, level)
	assertDungeonBackingReleased(t, direction)

	// Both replacement and in-place font-size changes invalidate geometry even
	// when the label string and screen dimensions remain unchanged.
	for _, replace := range []bool{true, false} {
		previous := scene.dungeonHUD.levelLabel.backing
		if replace {
			levelOutput.SetFont(gameText.NewUltimaFont(10.5), 10.5)
		} else {
			levelOutput.Font.TextFace.Size = 9.25
		}
		scene.drawDungeonHUD(screen, status)
		assert.NotSame(t, previous, scene.dungeonHUD.levelLabel.backing)
		assertDungeonBackingReleased(t, previous)
		assertDungeonBorderLabelLayout(t, &scene.dungeonHUD.levelLabel, scene.dungeonHUD.levelLabel.key.band, 1280)
	}
	for _, size := range []image.Point{{X: 1001, Y: 701}, {X: 1367, Y: 769}, {X: 640, Y: 360}, {X: 1280, Y: 720}} {
		level, direction = scene.dungeonHUD.levelLabel.backing, scene.dungeonHUD.directionLabel.backing
		oldFont := scene.dungeonHUD.level.Font
		display.GetManager().PublishLogicalSize(size.X, size.Y)
		setDungeonBorderHUDTestSize(scene)
		assert.Nil(t, scene.dungeonHUD.levelLabel.backing, "resize releases old labels before drawing")
		assert.Nil(t, scene.dungeonHUD.directionLabel.backing)
		assertDungeonBackingReleased(t, level)
		assertDungeonBackingReleased(t, direction)
		assert.NotSame(t, oldFont, scene.dungeonHUD.level.Font)
		assert.InDelta(t, gameText.GetScaledNumberToWidth(size.X, float64(widgets.FontSizeHUD)), scene.dungeonHUD.level.Font.TextFace.Size, 1e-9)
		resized := newPresentationRaster(size.X, size.Y)
		t.Cleanup(resized.Deallocate)
		scene.drawDungeonHUD(resized, status)
		assert.Equal(t, "L 8", scene.dungeonHUD.level.GetOutputStr(false))
		assert.Equal(t, "Dir: East", scene.dungeonHUD.direction.GetOutputStr(false))
		assertDungeonBorderLabelLayout(t, &scene.dungeonHUD.levelLabel, scene.dungeonHUD.levelLabel.key.band, size.X)
		assertDungeonBorderLabelLayout(t, &scene.dungeonHUD.directionLabel, scene.dungeonHUD.directionLabel.key.band, size.X)
		resized.Deallocate()
	}
	level, direction = scene.dungeonHUD.levelLabel.backing, scene.dungeonHUD.directionLabel.backing
	scene.releaseDungeonHUD()
	assert.Zero(t, scene.dungeonHUD)
	assertDungeonBackingReleased(t, level)
	assertDungeonBackingReleased(t, direction)
	assert.NotPanics(t, scene.releaseDungeonHUD)
}

func assertDungeonBackingReleased(t *testing.T, backing *ebiten.Image) {
	t.Helper()
	// Reading a deallocated image may recreate its now-empty storage. Retire
	// that observation too, including when a later assertion aborts the test.
	t.Cleanup(backing.Deallocate)
	_, _, _, alpha := backing.At(0, 0).RGBA()
	assert.Zero(t, alpha, "replaced backing storage was released")
}

func TestDungeonBorderLabelInvalidatesChangedFrameAndBand(t *testing.T) {
	scene := newDungeonBorderHUDTestScene(t, image.Pt(1001, 701))
	screen := newPresentationRaster(1001, 701)
	defer screen.Deallocate()
	status := dungeonHUDStatus{floor: 0, facing: references.DungeonFacingNorth}
	scene.drawDungeonHUD(screen, status)
	label := &scene.dungeonHUD.levelLabel
	band := label.key.band
	oldBacking, oldBounds := label.backing, label.backingBounds
	// Replacement frame identity matters even at identical dimensions, since
	// its sampled colour/stroke belong to the new immutable raster.
	replacement := ebiten.NewImageWithOptions(band, &ebiten.NewImageOptions{Unmanaged: true})
	defer replacement.Deallocate()
	replacement.DrawImage(scene.borders.outsideBorder, scene.borders.outsideBorderOp)
	label.draw(screen, scene.dungeonHUD.level, status.levelText(), replacement, band)
	assert.NotSame(t, oldBacking, label.backing)
	assert.Equal(t, oldBounds, label.backingBounds)
	assertDungeonBackingReleased(t, oldBacking)
	oldBacking = label.backing
	// Reposition within the same frame without a font or text change.
	band.Max.X -= 2
	label.draw(screen, scene.dungeonHUD.level, status.levelText(), replacement, band)
	assert.NotSame(t, oldBacking, label.backing)
	assert.Equal(t, oldBounds.Add(image.Pt(-1, 0)), label.backingBounds)
	assertDungeonBackingReleased(t, oldBacking)
}

func TestDungeonHUDIgnoresWorldTransforms(t *testing.T) {
	scene := newDungeonBorderHUDTestScene(t, image.Pt(1001, 701))
	screen := newPresentationRaster(1001, 701)
	defer screen.Deallocate()
	status := dungeonHUDStatus{floor: 3, facing: references.DungeonFacingNorth}
	scene.drawDungeonHUD(screen, status)
	want := dissolveMaskPixels(screen)
	scene.surfaceCamera = surfaceCameraProjection{active: true, x: 7.25, y: -3.5}
	scene.screenShakeOffsetX, scene.screenShakeOffsetY = -9.5, 8.25
	screen.Clear()
	scene.drawDungeonHUD(screen, status)
	assert.Equal(t, want, dissolveMaskPixels(screen), "both border labels stay static during world movement and shake")
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
