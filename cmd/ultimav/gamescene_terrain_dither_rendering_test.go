package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestTerrainDitherRenderingBonusExcludesHiddenItemsAndCorpses(t *testing.T) {
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.unscaledMapImage = ebiten.NewImage(sprites.TileSize, sprites.TileSize)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	hidden := references.Position{X: 101, Y: 100}
	layer.SetTileByLayer(map_state.MapLayer, &hidden, indexes.WaterCoast)
	terrainDitherTestVisibility(scene, player)
	scene.prepareTerrainDither(layer)
	cell, found := scene.terrainDither.cellAt(hidden)
	require.True(t, found)
	require.Equal(t, terrainDitherCell{bonus: true}, cell)
	render := func() []byte {
		scene.unscaledMapImage.Clear()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-1, -1)
		scene.refreshStaticMapTiles(&hidden, layer, op)
		pixels := make([]byte, sprites.TileSize*sprites.TileSize*4)
		scene.unscaledMapImage.ReadPixels(pixels)
		return pixels
	}
	baseline := render()
	require.False(t, bytes.Equal(make([]byte, len(baseline)), baseline), "bonus fixture must actually render terrain")
	for _, corpse := range []references.SearchObjectClass{references.SearchObjectDeadPC, references.SearchObjectDeadMonster} {
		state.SurfaceCorpses = []game_state.SurfaceCorpse{{
			Class: corpse, Position: hidden, Location: references.BritanniaUnderworld, Floor: 0,
		}}
		require.NotEqual(t, indexes.SpriteIndex(indexes.WaterCoast), state.GetEffectiveTile(&hidden))
		assert.Equal(t, baseline, render(), "hidden corpses must not enter bonus terrain pixels")
	}
	require.NoError(t, state.ItemStacksMap.AddOrMerge(&hidden, references.ItemAndQuantity{Item: references.Food, Quantity: 3}))
	layer.SetTileByLayer(map_state.MapUnitLayer, &hidden, indexes.Chest)
	require.Equal(t, indexes.SpriteIndex(indexes.ItemFood), state.GetEffectiveTile(&hidden))
	assert.Equal(t, baseline, render(), "hidden inventory and map units must not enter bonus terrain pixels")

	state.ElapsedMs = 500
	animated := render()
	assert.NotEqual(t, baseline, animated, "bonus water must use the normal deterministic animated terrain path")
	seed, calls := state.GetRNG().Seed(), state.GetRNG().Calls()
	assert.Equal(t, animated, render(), "fixed gameplay time produces identical terrain pixels")
	assert.Equal(t, seed, state.GetRNG().Seed())
	assert.Equal(t, calls, state.GetRNG().Calls())

	layer.SetVisible(true, &hidden)
	scene.prepareTerrainDither(layer)
	assert.NotEqual(t, animated, render(), "normally visible terrain must retain the effective item overlay")
}

func TestTerrainDitherRenderingBonusCropsExpandedTerrain(t *testing.T) {
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.unscaledMapImage = ebiten.NewImage(sprites.ExpandedTileSize, sprites.ExpandedTileSize)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	hidden := references.Position{X: 101, Y: 100}
	layer.SetTileByLayer(map_state.MapLayer, &hidden, indexes.WaterCoast)
	terrainDitherTestVisibility(scene, player)
	scene.prepareTerrainDither(layer)
	source := scene.spriteSheet.GetAnimatedSprite(indexes.WaterCoast, hidden.AnimationHash(), scene.getElapsedMs())
	require.Equal(t, sprites.ExpandedTileSize, source.Bounds().Dx(), "fixture must exercise bordered artwork")
	scene.refreshStaticMapTiles(&hidden, layer, &ebiten.DrawImageOptions{})
	pixels := make([]byte, sprites.ExpandedTileSize*sprites.ExpandedTileSize*4)
	scene.unscaledMapImage.ReadPixels(pixels)
	for y := range sprites.ExpandedTileSize {
		for x := range sprites.ExpandedTileSize {
			if x == 0 || y == 0 || x == sprites.ExpandedTileSize-1 || y == sprites.ExpandedTileSize-1 {
				require.Zero(t, pixels[(y*sprites.ExpandedTileSize+x)*4+3], "bonus artwork spilled beyond its logical tile at %d,%d", x, y)
			}
		}
	}
	require.False(t, bytes.Equal(make([]byte, len(pixels)), pixels))
}

func TestTerrainDitherRenderingPhysicalOuterTerrainParity(t *testing.T) {
	player := references.Position{X: 0, Y: 16}
	scene := newTerrainDitherTestScene(t, references.Yew, 0, player)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.unscaledMapImage = ebiten.NewImage(sprites.ExpandedTileSize, sprites.ExpandedTileSize)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	outside := references.Position{X: -1, Y: 16}
	terrainDitherTestVisibility(scene, player, outside)
	scene.prepareTerrainDither(layer)
	scene.refreshStaticMapTiles(&outside, layer, &ebiten.DrawImageOptions{})
	want := make([]byte, sprites.ExpandedTileSize*sprites.ExpandedTileSize*4)
	scene.unscaledMapImage.ReadPixels(want)
	require.False(t, bytes.Equal(make([]byte, len(want)), want))
	scene.terrainDither.enabled = false
	scene.unscaledMapImage.Clear()
	scene.refreshStaticMapTiles(&outside, layer, &ebiten.DrawImageOptions{})
	got := make([]byte, len(want))
	scene.unscaledMapImage.ReadPixels(got)
	assert.Equal(t, want, got, "physical outer tile and darkness gradient must retain their legacy pixels")
}

func TestTerrainDitherRenderingClearsRepeatedFrames(t *testing.T) {
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	terrainDitherTestVisibility(scene, player)
	scene.prepareTerrainDither(layer)
	frame := &scene.terrainDither
	first := terrainDitherTestPixels(t, frame)
	transparent, opaque := 0, 0
	for i := 0; i < len(first); i += 4 {
		require.Equal(t, []byte{0, 0, 0}, first[i:i+3])
		switch first[i+3] {
		case 0:
			transparent++
		case 255:
			opaque++
		default:
			t.Fatalf("mask alpha must be binary, got %d", first[i+3])
		}
	}
	require.Positive(t, transparent)
	require.Positive(t, opaque)
	for i := range frame.samples {
		frame.samples[i] = terrainDitherSample{visible: true, trusted: true}
	}
	frame.buildMask()
	assert.Equal(t, make([]byte, len(first)), terrainDitherTestPixels(t, frame), "same-size rebuild must erase old rejection pixels")
	for _, cell := range frame.cells {
		require.Equal(t, terrainDitherCell{visible: true}, cell, "same-size rebuild must erase old bonus admission")
	}
	for _, overscan := range []bool{true, false} {
		scene.surfaceCamera.active = overscan
		scene.prepareTerrainDither(layer)
		if !overscan {
			assert.Equal(t, first, terrainDitherTestPixels(t, frame), "resizing back must restore the same world-anchored mask")
		}
	}
}

func TestTerrainDitherRenderingApertureCameraShakeAndUI(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	scene.mapImage = ebiten.NewImage(xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	terrainDitherTestVisibility(scene, player)
	for _, size := range []image.Point{{X: 400, Y: 208}, {X: 600, Y: 312}, {X: 800, Y: 416}, {X: 1001, Y: 701}} {
		for _, motion := range []struct {
			name                 string
			active, allVisible   bool
			x, y, shakeX, shakeY float64
		}{
			{name: "settled"},
			{name: "shake strips", shakeX: 19.25, shakeY: -13.5},
			{name: "overscan", active: true, x: 7.25, y: -3.5, shakeX: -9.5, shakeY: 8.25},
			{name: "overscan exceeded", active: true, x: -15.5, y: 8, shakeX: -140.25, shakeY: 150.5},
			{name: "fully uncovered", shakeX: 10000},
			{name: "clear top left half", allVisible: true, shakeX: 19.5, shakeY: 13.5},
			{name: "clear bottom right half", allVisible: true, shakeX: -19.5, shakeY: -13.5},
			{name: "clear top left below half", allVisible: true, shakeX: 19.49, shakeY: 13.49},
			{name: "clear bottom right above half", allVisible: true, shakeX: -19.49, shakeY: -13.49},
			{name: "clear top left float32 tie", allVisible: true, shakeX: 19.49999999, shakeY: 13.49999999},
			{name: "clear bottom right float32 tie", allVisible: true, shakeX: -19.50000001, shakeY: -13.50000001},
		} {
			t.Run(fmt.Sprintf("%v/%s", size, motion.name), func(t *testing.T) {
				manager.PublishLogicalSize(size.X, size.Y)
				_, scene.borders.surfaceAperture = surfaceMapBorderGeometry()
				scene.surfaceCamera = surfaceCameraProjection{active: motion.active, x: motion.x, y: motion.y}
				scene.screenShakeOffsetX, scene.screenShakeOffsetY = motion.shakeX, motion.shakeY
				scene.prepareTerrainDither(layer)
				frame := &scene.terrainDither
				require.Equal(t, xTilesVisibleOnGameScreen+2*frame.margin, frame.width)
				if motion.allVisible {
					// Transparent edges expose strips that would otherwise be hidden by black mask pixels.
					for i := range frame.samples {
						frame.samples[i].visible = true
					}
					frame.buildMask()
				}
				mask := terrainDitherTestPixels(t, frame)
				// Match real screen storage so target atlas placement cannot perturb nearest ties.
				screen := ebiten.NewImageWithOptions(image.Rectangle{Max: size}, &ebiten.NewImageOptions{Unmanaged: true})
				defer screen.Deallocate()
				chrome := color.RGBA{R: 173, B: 219, A: 255}
				screen.Fill(chrome)
				target := scene.worldRenderTarget(screen)
				target.Fill(color.RGBA{R: 80, G: 120, B: 160, A: 255})
				glow := ebiten.NewImage(1, 1)
				defer glow.Deallocate()
				glow.Fill(color.RGBA{R: 30, G: 20, B: 10, A: 80})
				op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
				op.GeoM.Scale(float64(size.X), float64(size.Y))
				target.DrawImage(glow, op)
				world := make([]byte, size.X*size.Y*4)
				screen.ReadPixels(world)
				scene.drawTerrainDither(target)
				got := make([]byte, len(world))
				screen.ReadPixels(got)
				// Build an opaque reference composition and project it independently.
				// A black backing rejects uncovered strips without using production's strip code.
				logicalWidth, logicalHeight := display.GetManager().GetCurrentSize()
				sx := float64(logicalWidth) * gameScreenPercents.EndPercentX / float64(scene.mapImage.Bounds().Dx())
				sy := float64(logicalHeight) / float64(scene.mapImage.Bounds().Dy())
				tx := (motion.x-float64(frame.margin*sprites.TileSize))*sx + motion.shakeX
				ty := (motion.y-float64(frame.margin*sprites.TileSize))*sy + motion.shakeY
				for i := 0; i < len(mask); i += 4 {
					if mask[i+3] == 0 {
						copy(mask[i:i+3], []byte{110, 140, 170})
					}
					mask[i+3] = 255
				}
				// Match the mask's standalone texture coordinates, not an arbitrary atlas origin.
				source := ebiten.NewImageWithOptions(image.Rect(0, 0, frame.width*sprites.TileSize, frame.height*sprites.TileSize),
					&ebiten.NewImageOptions{Unmanaged: true})
				defer source.Deallocate()
				source.WritePixels(mask)
				expected := ebiten.NewImageWithOptions(image.Rectangle{Max: size}, &ebiten.NewImageOptions{Unmanaged: true})
				defer expected.Deallocate()
				expected.Fill(chrome)
				expectedTarget, ok := expected.SubImage(target.Bounds()).(*ebiten.Image)
				require.True(t, ok)
				expectedTarget.Fill(color.Black)
				expectedOp := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
				expectedOp.GeoM.Scale(sx, sy)
				expectedOp.GeoM.Translate(tx, ty)
				expectedTarget.DrawImage(source, expectedOp)
				wantPixels := make([]byte, len(got))
				expected.ReadPixels(wantPixels)
				blackPoint := image.Pt(-1, -1)
				for y := range size.Y {
					for x := range size.X {
						i := (y*size.X + x) * 4
						want := wantPixels[i : i+4]
						if bytes.Equal(want, []byte{0, 0, 0, 255}) && image.Pt(x, y).In(target.Bounds()) {
							blackPoint = image.Pt(x, y)
						}
						if !bytes.Equal(want, got[i:i+4]) {
							t.Fatalf("mask/world/aperture mismatch at %d,%d: want %v, got %v", x, y, want, got[i:i+4])
						}
					}
				}
				require.NotEqual(t, image.Pt(-1, -1), blackPoint)
				// A later UI pass must remain above rejection, even inside the aperture.
				ui := ebiten.NewImage(1, 1)
				defer ui.Deallocate()
				ui.Fill(color.White)
				uiOp := &ebiten.DrawImageOptions{}
				uiOp.GeoM.Translate(float64(blackPoint.X), float64(blackPoint.Y))
				screen.DrawImage(ui, uiOp)
				assert.Equal(t, color.RGBA{R: 255, G: 255, B: 255, A: 255}, screen.At(blackPoint.X, blackPoint.Y))
			})
		}
	}
}

func TestTerrainDitherRenderingMoongateSeamsWithSmoothOff(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	const width, height = 400, 208
	manager.PublishLogicalSize(width, height)
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.initMoongateSprite()
	scene.mapImage = ebiten.NewImage(xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize)
	screen := newPresentationRaster(width, height)
	defer screen.Deallocate()
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	for _, delta := range []references.Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		var baseline []byte
		for _, origin := range []references.Position{player, {}, {X: 255, Y: 255}} {
			scene.gameState.MapState.PlayerLocation.Position = origin
			pos := references.Position{X: origin.X + delta.X, Y: origin.Y + delta.Y}
			pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			layer.RecalculateVisibleTilesWithMoongates(origin, &scene.gameState.MapState.Lighting, nil, map_state.PlayerLightNone)
			terrainDitherTestVisibility(scene, origin, pos)
			scene.prepareTerrainDither(layer)
			screen.Clear()
			scene.drawAnimatedMoongate(screen, pos, 1, sprites.TileSize)
			pixels := make([]byte, width*height*4)
			screen.ReadPixels(pixels)
			require.False(t, bytes.Equal(make([]byte, len(pixels)), pixels), "fixture must draw a visible gate and glow")
			if baseline == nil {
				baseline = bytes.Clone(pixels)
			} else {
				assert.True(t, bytes.Equal(baseline, pixels), "gate and glow must use the same world copy at seams with smoothing disabled")
			}
			terrainDitherTestVisibility(scene, origin)
			scene.prepareTerrainDither(layer)
			screen.Clear()
			scene.drawAnimatedMoongate(screen, pos, 1, sprites.TileSize)
			screen.ReadPixels(pixels)
			assert.True(t, bytes.Equal(make([]byte, len(pixels)), pixels), "hidden gates and glows must not exploit terrain-only bonus admission")
		}
	}
}

func TestTerrainDitherProjectileAdmissionUsesDrawnCenter(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	const width, height = 400, 208
	manager.PublishLogicalSize(width, height)
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	scene.mapImage = ebiten.NewImage(xTilesVisibleOnGameScreen*sprites.TileSize, yTilesVisibleOnGameScreen*sprites.TileSize)
	screen := newPresentationRaster(width, height)
	defer screen.Deallocate()
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	for _, axis := range []string{"x", "y"} {
		for _, direction := range []float64{-1, 1} {
			for _, fraction := range []float64{0.25, 0.75} {
				for _, visibleSide := range []int{0, 1} {
					t.Run(fmt.Sprintf("%s/%v/%v/visible=%d", axis, direction, fraction, visibleSide), func(t *testing.T) {
						visible := player
						projectile := &ActiveProjectile{
							active: true, projectileType: references.MissileArrow,
							currentTileX: float64(xCenter), currentTileY: float64(yCenter),
						}
						if axis == "x" {
							visible.X += references.Coordinate(visibleSide)
							projectile.currentTileX += fraction
							projectile.toTileX = direction
						} else {
							visible.Y += references.Coordinate(visibleSide)
							projectile.currentTileY += fraction
							projectile.toTileY = direction
						}
						terrainDitherTestVisibility(scene, visible)
						scene.prepareTerrainDither(layer)
						scene.activeProjectile = projectile
						before := *projectile
						screen.Clear()
						scene.drawActiveProjectile(screen)
						pixels := make([]byte, width*height*4)
						screen.ReadPixels(pixels)
						drawn := !bytes.Equal(make([]byte, len(pixels)), pixels)
						want := (fraction < 0.5 && visibleSide == 0) || (fraction > 0.5 && visibleSide == 1)
						assert.Equal(t, want, drawn, "the center's cell controls rendering in either travel direction")
						assert.Equal(t, before, *projectile, "render admission must not change the simulation")
					})
				}
			}
		}
	}
}
