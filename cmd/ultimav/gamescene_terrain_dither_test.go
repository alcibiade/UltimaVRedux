package main

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
)

// No integration builder: all terrain is synthetic and tile metadata is embedded.
func newTerrainDitherTestScene(tb testing.TB, location references.Location, floor references.FloorNumber, position references.Position) *GameScene {
	tb.Helper()
	tileRefs := references.NewTileReferences()
	state := &game_state.GameState{}
	state.SetRandomSeed(522)
	state.PartyState.Inventory = *party_state.NewInventory()
	state.ItemStacksMap = *references.NewItemStacksMap()
	state.GameReferences = &references.GameReferences{
		TileReferences: tileRefs, InventoryItemReferences: references.NewInventoryItemsReferences(),
	}
	state.MapState.PlayerLocation = references.PlayerLocation{Location: location, Floor: floor, Position: position}
	state.MapState.LayeredMaps = *map_state.NewLayeredMaps(
		tileRefs, &references.LargeMapReference{}, &references.LargeMapReference{}, 19, 13,
	)
	if location.GetMapType() == references.SmallMapType {
		small := &references.SmallLocationReference{Location: location, ListOfFloors: []references.FloorNumber{-1, 0, 1}}
		floors := make(map[int]*[references.XSmallMapTiles][references.YSmallMapTiles]byte)
		for _, level := range small.ListOfFloors {
			data := &[references.XSmallMapTiles][references.YSmallMapTiles]byte{}
			for x := range data {
				for y := range data[x] {
					data[x][y] = byte(indexes.Grass)
				}
			}
			floors[int(level)] = data
		}
		// The public small-map loader requires original files. Populate only its
		// private source/catalog fields rather than adding a production test API.
		raw := reflect.ValueOf(small).Elem().FieldByName("rawData")
		reflect.NewAt(raw.Type(), raw.Addr().UnsafePointer()).Elem().Set(reflect.ValueOf(floors))
		state.GameReferences.LocationReferences = &references.LocationReferences{}
		catalog := reflect.ValueOf(state.GameReferences.LocationReferences).Elem().FieldByName("maps")
		reflect.NewAt(catalog.Type(), catalog.Addr().UnsafePointer()).Elem().Set(reflect.ValueOf(
			map[references.Location]*references.SmallLocationReference{location: small},
		))
		state.MapState.LayeredMaps.ResetAndCreateSmallMap(small, tileRefs, 19, 13)
	}
	layer := state.GetLayeredMapByCurrentLocation()
	state.MapState.Lighting = map_state.NewLighting(layer)
	state.DateTime.Hour = 12
	for dx := references.Coordinate(-12); dx <= 12; dx++ {
		for dy := references.Coordinate(-12); dy <= 12; dy++ {
			pos := references.Position{X: position.X + dx, Y: position.Y + dy}
			if layer.IsWrappedMap() {
				pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}
			if !state.IsOutOfBounds(pos) {
				layer.SetTileByLayer(map_state.MapLayer, &pos, indexes.Grass)
			}
		}
	}
	layer.RecalculateVisibleTilesWithMoongates(position, &state.MapState.Lighting, nil, map_state.PlayerLightNone)
	scene := &GameScene{
		gameState: state,
		gameConfig: &config.UltimaVConfiguration{
			GameplayConfiguration: &config.GameplayConfiguration{},
		},
	}
	tb.Cleanup(func() {
		scene.terrainDither.image.Dispose()
		scene.outerTerrainDither.Dispose()
		for _, img := range []*ebiten.Image{scene.mapImage, scene.unscaledMapImage, scene.surfaceTerrainImage} {
			if img != nil {
				img.Deallocate()
			}
		}
	})
	return scene
}

func terrainDitherTestVisibility(scene *GameScene, positions ...references.Position) {
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	flags := layer.GetVisibilityFlags()
	flags.ResetVisibilityCoords(false)
	for _, pos := range positions {
		layer.SetVisible(true, &pos)
	}
}

func TestTerrainMaskReusesUnchangedSamplesAndInvalidatesVisibility(t *testing.T) {
	center := references.Position{X: 50, Y: 50}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, center)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	scene.prepareTerrainDither(layer)
	require.True(t, scene.terrainDither.maskValid)
	builds := scene.terrainDither.maskBuilds
	cells := slices.Clone(scene.terrainDither.cells)
	for range 20 {
		scene.prepareTerrainDither(layer)
	}
	require.Equal(t, builds, scene.terrainDither.maskBuilds)
	require.Equal(t, cells, scene.terrainDither.cells)
	flags := layer.GetVisibilityFlags()
	flags.ResetVisibilityCoords(false)
	layer.SetVisible(true, &center)
	scene.prepareTerrainDither(layer)
	require.Equal(t, builds+1, scene.terrainDither.maskBuilds)
	cached := slices.Clone(scene.terrainDither.cells)
	scene.terrainDither.buildMask()
	require.Equal(t, cached, scene.terrainDither.cells, "cached cells must match forced fresh construction")
	scene.gameState.MapState.PlayerLocation.Position = center.Right()
	scene.prepareTerrainDither(layer)
	require.Equal(t, builds+3, scene.terrainDither.maskBuilds, "changed world-space Bayer origin must invalidate")
}

func terrainDitherTestPixels(tb testing.TB, frame *terrainDitherFrame) []byte {
	tb.Helper()
	// Inspect the uploaded production mask through its real draw boundary, without
	// depending on private storage fields in the graphics resource owner.
	width, height := frame.width*sprites.TileSize, frame.height*sprites.TileSize
	target := ebiten.NewImage(width, height)
	defer target.Deallocate()
	frame.image.Draw(target, ebiten.GeoM{})
	pixels := make([]byte, width*height*4)
	target.ReadPixels(pixels)
	return pixels
}

func TestTerrainDitherModes(t *testing.T) {
	for _, test := range []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
		want     bool
	}{
		{"town", references.Britain, 0, true},
		{"basement", references.Yew, -1, true},
		{"overworld", references.BritanniaUnderworld, 0, true},
		{"underworld", references.BritanniaUnderworld, -1, true},
		{"dungeon", references.Deceit, 0, false},
		{"combat", references.CombatLocation, 0, false},
		{"camping", references.CampingLocation, 0, false},
		{"shrine", references.ShrineLocation, 0, false},
		{"codex", references.CodexLocation, 0, false},
		{"finale", references.FinaleLocation, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &game_state.GameState{}
			state.MapState.PlayerLocation = references.PlayerLocation{Location: test.location, Floor: test.floor}
			scene := &GameScene{gameState: state}
			assert.Equal(t, test.want, scene.shouldDitherTerrain())
		})
	}
}

func TestTerrainDitherPrepareSurfaceFloors(t *testing.T) {
	for _, location := range []references.Location{references.Yew, references.BritanniaUnderworld} {
		for _, floor := range []references.FloorNumber{0, -1} {
			t.Run(fmt.Sprintf("%s/%d", location, floor), func(t *testing.T) {
				player := references.Position{X: 16, Y: 16}
				if location == references.BritanniaUnderworld {
					player = references.Position{X: 100, Y: 100}
				}
				scene := newTerrainDitherTestScene(t, location, floor, player)
				terrainDitherTestVisibility(scene, player)
				layer := scene.gameState.GetLayeredMapByCurrentLocation()
				scene.prepareTerrainDither(layer)
				require.True(t, scene.terrainDither.enabled)
				assert.Equal(t, location == references.BritanniaUnderworld, scene.terrainDither.wrapped)
				cell, found := scene.terrainDither.cellAt(player)
				require.True(t, found)
				assert.Equal(t, terrainDitherCell{visible: true}, cell)
				hidden := references.Position{X: player.X + 1, Y: player.Y}
				cell, found = scene.terrainDither.cellAt(hidden)
				require.True(t, found)
				assert.Equal(t, terrainDitherCell{bonus: true}, cell)
				assert.False(t, scene.isPositionVisible(layer, &hidden), "bonus is never authoritative visibility")
				assert.True(t, scene.ditherAllowsWorldEffect(player))
				assert.False(t, scene.ditherAllowsWorldEffect(hidden))
				assert.False(t, scene.ditherAllowsWorldEffect(references.Position{X: 200, Y: 200}))
			})
		}
	}
}

func TestTerrainDitherExclusionsDisableBonusInSameFrame(t *testing.T) {
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.unscaledMapImage = ebiten.NewImage(sprites.TileSize, sprites.TileSize)
	layer := scene.gameState.GetLayeredMapByCurrentLocation()
	hidden := references.Position{X: 101, Y: 100}
	for _, mode := range []string{"xray", "black fade", "over-fade dissolve", "disposed"} {
		t.Run(mode, func(t *testing.T) {
			terrainDitherTestVisibility(scene, player)
			scene.prepareTerrainDither(layer)
			cell, found := scene.terrainDither.cellAt(hidden)
			require.True(t, found)
			require.True(t, cell.bonus)
			switch mode {
			case "xray":
				scene.xRayVisionActive = true
			case "black fade":
				scene.mapFadeLevel = 1
			case "over-fade dissolve":
				id := scene.StartTileDissolveWithOptions(TileDissolveOptions{
					SpriteIndex: indexes.Apparation1, Mode: visual.DissolveModeReveal, RenderLayer: RenderLayerOverFade,
				})
				t.Cleanup(func() { scene.StopTileDissolve(id) })
			case "disposed":
				scene.disposed = true
			}
			scene.prepareTerrainDither(layer)
			require.False(t, scene.terrainDither.enabled)
			assert.True(t, scene.ditherAllowsWorldEffect(hidden), "disabled dither restores the existing effect path")
			// Leave visibility hidden while checking admission, not X-ray's intentional reveal.
			scene.xRayVisionActive = false
			scene.unscaledMapImage.Clear()
			scene.refreshStaticMapTiles(&hidden, layer, &ebiten.DrawImageOptions{})
			pixels := make([]byte, sprites.TileSize*sprites.TileSize*4)
			scene.unscaledMapImage.ReadPixels(pixels)
			assert.Equal(t, make([]byte, len(pixels)), pixels, "stale bonus cells must not draw terrain")
			scene.disposed, scene.mapFadeLevel = false, 0
		})
	}
	scene.mapFadeLevel = 0.5
	assert.True(t, scene.shouldDitherTerrain(), "partial fades retain the mask")
	id := scene.StartTileDissolveWithOptions(TileDissolveOptions{SpriteIndex: indexes.Apparation1, RenderLayer: RenderLayerDefault})
	assert.True(t, scene.shouldDitherTerrain(), "ordinary world dissolves stay below the mask")
	scene.StopTileDissolve(id)
	scene.activeTileDissolves = []TileDissolveState{{RenderLayer: RenderLayerOverFade}}
	assert.True(t, scene.shouldDitherTerrain(), "inactive over-fade entries do not disable the mask")
}

func TestTerrainDitherUntrustedNeighborhoodFallsBackToHardTiles(t *testing.T) {
	frame := terrainDitherFrame{
		width: 1, height: 1, cells: make([]terrainDitherCell, 1), samples: make([]terrainDitherSample, 9),
		masks: sprites.NewQuarterMaskSet(sprites.EdgeDitherOptions{TileSize: 16, OutwardPixels: 4, InwardPixels: 1, CornerRadius: 2}),
	}
	defer frame.image.Dispose()
	for _, visible := range []bool{false, true} {
		for missing := range frame.samples {
			for i := range frame.samples {
				frame.samples[i] = terrainDitherSample{visible: !visible, trusted: i != missing}
			}
			frame.samples[4].visible = visible
			frame.buildMask()
			assert.Equal(t, terrainDitherCell{visible: visible}, frame.cells[0])
			pixels := terrainDitherTestPixels(t, &frame)
			wantAlpha := byte(255)
			if visible {
				wantAlpha = 0
			}
			for i := 0; i < len(pixels); i += 4 {
				require.Equal(t, []byte{0, 0, 0, wantAlpha}, pixels[i:i+4], "visible=%v missing halo sample=%d", visible, missing)
			}
		}
	}
}

func TestTerrainDitherMissingDistanceAndPhysicalEdge(t *testing.T) {
	for _, location := range []references.Location{references.BritanniaUnderworld, references.Yew} {
		t.Run(location.String(), func(t *testing.T) {
			player := references.Position{X: 100, Y: 100}
			if location == references.Yew {
				player = references.Position{X: 0, Y: 16}
			}
			scene := newTerrainDitherTestScene(t, location, 0, player)
			layer := scene.gameState.GetLayeredMapByCurrentLocation()
			terrainDitherTestVisibility(scene, player)
			scene.prepareTerrainDither(layer)
			hidden := references.Position{X: player.X + 1, Y: player.Y}
			cell, found := scene.terrainDither.cellAt(hidden)
			require.True(t, found)
			require.True(t, cell.bonus)
			if location == references.Yew {
				outside := references.Position{X: -1, Y: player.Y}
				cell, found = scene.terrainDither.cellAt(outside)
				require.True(t, found)
				assert.Equal(t, terrainDitherCell{}, cell, "physical outer terrain must not receive bonus admission")
				layer.SetVisible(true, &outside)
				scene.prepareTerrainDither(layer)
				cell, _ = scene.terrainDither.cellAt(outside)
				assert.Equal(t, terrainDitherCell{visible: true}, cell, "legacy visible outer terrain remains visible")
			}
			missing := references.Position{X: hidden.X + 1, Y: hidden.Y + 1}
			delete(layer.GetPrimaryDistanceMap(), missing)
			scene.prepareTerrainDither(layer)
			cell, _ = scene.terrainDither.cellAt(hidden)
			assert.Equal(t, terrainDitherCell{}, cell, "a missing diagonal distance entry must suppress bonus admission")
		})
	}
}

func TestTerrainDitherWrappedSeamsAndSmoothDisabled(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		for _, player := range []references.Position{{X: 100, Y: 100}, {}, {X: 255, Y: 255}} {
			t.Run(fmt.Sprintf("floor%d/%v", floor, player), func(t *testing.T) {
				scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, floor, player)
				terrainDitherTestVisibility(scene, player)
				scene.prepareTerrainDither(scene.gameState.GetLayeredMapByCurrentLocation())
				require.False(t, scene.gameConfig.GameplayConfiguration.SmoothSurfaceMovement)
				for _, delta := range []references.Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
					pos := references.Position{X: player.X + delta.X, Y: player.Y + delta.Y}
					wrapped := *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
					cell, found := scene.terrainDither.cellAt(wrapped)
					require.True(t, found)
					assert.Equal(t, terrainDitherCell{bonus: true}, cell)
					assert.Equal(t, delta, scene.ditherRelativePosition(wrapped))
					assert.False(t, scene.ditherAllowsWorldEffect(wrapped))
				}
				before := terrainDitherTestPixels(t, &scene.terrainDither)
				scene.gameConfig.GameplayConfiguration.SmoothSurfaceMovement = true
				scene.prepareTerrainDither(scene.gameState.GetLayeredMapByCurrentLocation())
				assert.Equal(t, before, terrainDitherTestPixels(t, &scene.terrainDither), "settled mask cannot depend on smooth movement preference")
			})
		}
	}
}

func TestTerrainDitherPreparePreservesAuthorityAndRNG(t *testing.T) {
	player := references.Position{X: 100, Y: 100}
	scene := newTerrainDitherTestScene(t, references.BritanniaUnderworld, 0, player)
	state := scene.gameState
	layer := state.GetLayeredMapByCurrentLocation()
	terrainDitherTestVisibility(scene, player)
	hidden := references.Position{X: 101, Y: 100}
	require.NoError(t, state.ItemStacksMap.AddOrMerge(&hidden, references.ItemAndQuantity{Item: references.Food, Quantity: 3}))
	state.SurfaceCorpses = []game_state.SurfaceCorpse{{Position: hidden, Class: references.SearchObjectDeadPC}}
	visibility := make(map_state.VisibilityCoords)
	for x, row := range layer.GetVisibilityFlags() {
		visibility[x] = maps.Clone(row)
	}
	distances := maps.Clone(layer.GetPrimaryDistanceMap())
	location, date, elapsed := state.MapState.PlayerLocation, state.DateTime, state.ElapsedMs
	corpses := slices.Clone(state.SurfaceCorpses)
	items, _ := state.ItemStacksMap.Snapshot(&hidden)
	seed, calls := state.GetRNG().Seed(), state.GetRNG().Calls()
	for range 3 {
		scene.prepareTerrainDither(layer)
		state.GetBonusTerrainTile(&hidden)
	}
	assert.Equal(t, visibility, layer.GetVisibilityFlags())
	assert.Equal(t, distances, layer.GetPrimaryDistanceMap())
	assert.Equal(t, location, state.MapState.PlayerLocation)
	assert.Equal(t, date, state.DateTime)
	assert.Equal(t, elapsed, state.ElapsedMs)
	assert.Equal(t, corpses, state.SurfaceCorpses)
	afterItems, _ := state.ItemStacksMap.Snapshot(&hidden)
	assert.Equal(t, items, afterItems)
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), layer.SpriteIndexByLayer(map_state.MapLayer, hidden))
	assert.Equal(t, seed, state.GetRNG().Seed())
	assert.Equal(t, calls, state.GetRNG().Calls())
}

func BenchmarkTerrainDither(b *testing.B) {
	for _, overscan := range []bool{false, true} {
		b.Run(fmt.Sprintf("overscan=%v", overscan), func(b *testing.B) {
			player := references.Position{X: 100, Y: 100}
			scene := newTerrainDitherTestScene(b, references.BritanniaUnderworld, 0, player)
			terrainDitherTestVisibility(scene, player)
			scene.surfaceCamera.active = overscan
			layer := scene.gameState.GetLayeredMapByCurrentLocation()
			scene.prepareTerrainDither(layer)
			b.Run("BuildMask", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					scene.terrainDither.buildMask()
				}
			})
			b.Run("Prepare", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					scene.prepareTerrainDither(layer)
				}
			})
		})
	}
}
