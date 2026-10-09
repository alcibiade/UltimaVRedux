package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type horseRNGCall struct {
	min int
	max int
}

type scriptedHorseRNG struct {
	t      *testing.T
	values []int
	calls  []horseRNGCall
}

func (r *scriptedHorseRNG) RandomIntInRange(minimum, maximum int) int {
	r.t.Helper()
	r.calls = append(r.calls, horseRNGCall{min: minimum, max: maximum})
	require.NotEmpty(r.t, r.values, "unexpected horse RNG call")
	value := r.values[0]
	r.values = r.values[1:]
	require.GreaterOrEqual(r.t, value, minimum)
	require.LessOrEqual(r.t, value, maximum)

	return value
}

func (r *scriptedHorseRNG) OneInXOdds(int) bool {
	r.t.Fatal("horse movement must not use OneInXOdds")

	return false
}

func (r *scriptedHorseRNG) ShuffleDirections([]references.Position) {
	r.t.Fatal("horse movement must not shuffle directions")
}

func TestHorseUpdatePreservesOriginalRNGOrderAndDirection(t *testing.T) {
	tests := []struct {
		name          string
		rolls         []int
		hitchTile     indexes.SpriteIndex
		startFacing   references.Direction
		wantPosition  references.Position
		wantFacing    references.Direction
		wantCallCount int
	}{
		{name: "gate skips update", rolls: []int{1}, startFacing: references.Right, wantPosition: references.Position{X: 10, Y: 10}, wantFacing: references.Right, wantCallCount: 1},
		{name: "hitch short circuits direction", rolls: []int{0}, hitchTile: indexes.HitchingPost, startFacing: references.Right, wantPosition: references.Position{X: 10, Y: 10}, wantFacing: references.Right, wantCallCount: 1},
		{name: "west faces left", rolls: []int{0, 1, 0}, startFacing: references.Right, wantPosition: references.Position{X: 9, Y: 10}, wantFacing: references.Left, wantCallCount: 3},
		{name: "east faces right", rolls: []int{0, 1, 1}, startFacing: references.Left, wantPosition: references.Position{X: 11, Y: 10}, wantFacing: references.Right, wantCallCount: 3},
		{name: "north preserves facing", rolls: []int{0, 0, 0}, startFacing: references.Left, wantPosition: references.Position{X: 10, Y: 9}, wantFacing: references.Left, wantCallCount: 3},
		{name: "south preserves facing", rolls: []int{0, 0, 1}, startFacing: references.Right, wantPosition: references.Position{X: 10, Y: 11}, wantFacing: references.Right, wantCallCount: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedHorseRNG{t: t, values: append([]int(nil), test.rolls...)}
			controller, horse, layeredMap := newHorseAITestController(t, rng, references.Position{X: 10, Y: 10}, 0)
			horse.SetDirection(test.startFacing)
			for _, destination := range []references.Position{
				{X: 9, Y: 10}, {X: 11, Y: 10}, {X: 10, Y: 9}, {X: 10, Y: 11},
			} {
				layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.BrickFloor)
			}
			if test.hitchTile != indexes.NoSprites {
				hitch := references.Position{X: 10, Y: 11}
				layeredMap.SetTileByLayer(map_state.MapLayer, &hitch, test.hitchTile)
			}

			controller.calculateNextNPCPosition(horse)

			assert.Equal(t, test.wantPosition, horse.Pos())
			assert.Equal(t, test.wantFacing, horse.GetVehicleDetails().CurrentDirection())
			assert.Len(t, rng.calls, test.wantCallCount)
			assert.Empty(t, rng.values)
			for _, call := range rng.calls {
				assert.Equal(t, horseRNGCall{min: 0, max: 1}, call)
			}
		})
	}
}

func TestHorseHitchingUsesCardinalTerrainBeneathOverlays(t *testing.T) {
	position := references.Position{X: 10, Y: 10}
	probes := []struct {
		name string
		pos  references.Position
		tile indexes.SpriteIndex
	}{
		{name: "south post", pos: position.Down(), tile: indexes.HitchingPost},
		{name: "east rail", pos: position.Right(), tile: indexes.WoodFloorShipEdge},
		{name: "north post", pos: position.Up(), tile: indexes.HitchingPost},
		{name: "west rail", pos: position.Left(), tile: indexes.WoodFloorShipEdge},
	}
	for _, test := range probes {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedHorseRNG{t: t, values: []int{0}}
			controller, horse, layeredMap := newHorseAITestController(t, rng, position, 0)
			layeredMap.SetTileByLayer(map_state.MapLayer, &test.pos, test.tile)
			layeredMap.SetTileByLayer(map_state.MapUnitLayer, &test.pos, indexes.Guard1)

			controller.calculateNextNPCPosition(horse)

			assert.Equal(t, position, horse.Pos())
			assert.Len(t, rng.calls, 1)
		})
	}
}

func TestHorseHitchingRejectsNonCardinalAndOffFloorTerrain(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*map_state.MapState, *map_state.LayeredMap, references.Position)
	}{
		{
			name: "same tile",
			setup: func(_ *map_state.MapState, layeredMap *map_state.LayeredMap, position references.Position) {
				layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.HitchingPost)
			},
		},
		{
			name: "diagonal tile",
			setup: func(_ *map_state.MapState, layeredMap *map_state.LayeredMap, position references.Position) {
				diagonal := position.Down().Right()
				layeredMap.SetTileByLayer(map_state.MapLayer, &diagonal, indexes.HitchingPost)
			},
		},
		{
			name: "other floor",
			setup: func(state *map_state.MapState, _ *map_state.LayeredMap, position references.Position) {
				offFloor := state.LayeredMaps.GetLayeredMap(references.LargeMapType, -1)
				south := position.Down()
				offFloor.SetTileByLayer(map_state.MapLayer, &south, indexes.HitchingPost)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedHorseRNG{t: t, values: []int{0, 1, 1}}
			controller, horse, layeredMap := newHorseAITestController(t, rng, references.Position{X: 10, Y: 10}, 0)
			destination := references.Position{X: 11, Y: 10}
			layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.BrickFloor)
			test.setup(controller.mapState, layeredMap, horse.Pos())

			controller.calculateNextNPCPosition(horse)

			assert.Equal(t, destination, horse.Pos())
			assert.Len(t, rng.calls, 3)
		})
	}
}

func TestHorseAtMapEdgeIgnoresSoutheastHitchAlias(t *testing.T) {
	rng := &scriptedHorseRNG{t: t, values: []int{0, 1, 0}}
	controller, horse, layeredMap := newHorseAITestController(t, rng, references.Position{}, 0)
	southeast := references.Position{X: 31, Y: 31}
	layeredMap.SetTileByLayer(map_state.MapLayer, &southeast, indexes.HitchingPost)

	controller.calculateNextNPCPosition(horse)

	assert.Equal(t, references.Position{}, horse.Pos())
	assert.Len(t, rng.calls, 3, "off-map hitch probes must not short circuit direction rolls")
}

func TestHorseMoveRejectsBlockedDestinationWithoutFallback(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*NPCAIControllerSmallMap, *map_units.NPCFriendly, *map_state.LayeredMap, references.Position)
	}{
		{
			name: "impassable terrain",
			setup: func(_ *NPCAIControllerSmallMap, _ *map_units.NPCFriendly, layeredMap *map_state.LayeredMap, destination references.Position) {
				layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.Lava)
			},
		},
		{
			name: "party occupancy",
			setup: func(controller *NPCAIControllerSmallMap, _ *map_units.NPCFriendly, layeredMap *map_state.LayeredMap, destination references.Position) {
				layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.BrickFloor)
				controller.mapState.PlayerLocation.Position = destination
			},
		},
		{
			name: "same floor map unit",
			setup: func(controller *NPCAIControllerSmallMap, _ *map_units.NPCFriendly, layeredMap *map_state.LayeredMap, destination references.Position) {
				layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.BrickFloor)
				blocker := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, destination, 0)
				blocker.SetPos(destination)
				controller.mapUnits = append(controller.mapUnits, blocker)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedHorseRNG{t: t, values: []int{0, 1, 1}}
			start := references.Position{X: 10, Y: 10}
			destination := start.Right()
			controller, horse, layeredMap := newHorseAITestController(t, rng, start, 0)
			north := start.Up()
			layeredMap.SetTileByLayer(map_state.MapLayer, &north, indexes.BrickFloor)
			test.setup(controller, horse, layeredMap, destination)

			controller.calculateNextNPCPosition(horse)

			assert.Equal(t, start, horse.Pos(), "a blocked choice must not retry the open north tile")
			assert.Len(t, rng.calls, 3)
		})
	}
}

func TestHorseMoveIgnoresOtherFloorOccupancy(t *testing.T) {
	rng := &scriptedHorseRNG{t: t, values: []int{0, 1, 1}}
	start := references.Position{X: 10, Y: 10}
	destination := start.Right()
	controller, horse, layeredMap := newHorseAITestController(t, rng, start, 0)
	layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.BrickFloor)
	blocker := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, destination, -1)
	blocker.SetPos(destination)
	controller.mapUnits = append(controller.mapUnits, blocker)

	controller.calculateNextNPCPosition(horse)

	assert.Equal(t, destination, horse.Pos())
}

func TestHorseUpdatesUseMapUnitOrderWithoutScheduleRadius(t *testing.T) {
	rng := &scriptedHorseRNG{t: t, values: []int{0, 1, 0, 0, 1, 1}}
	controller, first, layeredMap := newHorseAITestController(t, rng, references.Position{X: 2, Y: 2}, 0)
	second := newHorseAITestUnit(references.Position{X: 20, Y: 20}, 0, indexes.HorseRight)
	controller.mapUnits = append(controller.mapUnits, second)
	for _, destination := range []references.Position{{X: 1, Y: 2}, {X: 21, Y: 20}} {
		layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.BrickFloor)
	}

	controller.AdvanceNextTurnCalcAndMoveNPCs()

	assert.Equal(t, references.Position{X: 1, Y: 2}, first.Pos())
	assert.Equal(t, references.Position{X: 21, Y: 20}, second.Pos())
	assert.Len(t, rng.calls, 6)
}

func TestHorseOnOtherFloorConsumesNoRNG(t *testing.T) {
	rng := &scriptedHorseRNG{t: t}
	controller, horse, _ := newHorseAITestController(t, rng, references.Position{X: 10, Y: 10}, -1)

	controller.calculateNextNPCPosition(horse)

	assert.Empty(t, rng.calls)
}

func newHorseAITestController(
	t *testing.T,
	rng RNGProvider,
	position references.Position,
	floor references.FloorNumber,
) (*NPCAIControllerSmallMap, *map_units.NPCFriendly, *map_state.LayeredMap) {
	t.Helper()
	tileRefs := references.NewTileReferences()
	emptyMap := &references.LargeMapReference{}
	layeredMaps := map_state.NewLayeredMaps(tileRefs, emptyMap, emptyMap, 11, 11)
	state := &map_state.MapState{ //nolint:exhaustruct // Horse movement only needs location and loaded map layers.
		PlayerLocation: references.PlayerLocation{
			Location: references.BritanniaUnderworld,
			Position: references.Position{X: 31, Y: 31},
			Floor:    0,
		},
		LayeredMaps: *layeredMaps,
	}
	horse := newHorseAITestUnit(position, floor, indexes.HorseRight)
	controller := &NPCAIControllerSmallMap{ //nolint:exhaustruct // Focused deterministic horse fixture.
		tileRefs:    tileRefs,
		dateTime:    &datetime.UltimaDate{},
		mapState:    state,
		mapUnits:    map_units.MapUnits{horse},
		rngProvider: rng,
	}

	return controller, horse, state.GetLayeredMapByCurrentLocation()
}

func newHorseAITestUnit(
	position references.Position,
	floor references.FloorNumber,
	sprite indexes.SpriteIndex,
) *map_units.NPCFriendly {
	reference := references.NPCReference{
		Location: references.Britain,
		Schedule: references.CreateNPCScheduledFixedOneLocation(position, floor),
	}
	reference.SetKeyIndex(sprite)
	horse := map_units.NewNPCFriendlyVehicle(references.HorseVehicle, reference)
	horse.SetPos(position)
	horse.SetFloor(floor)

	return horse
}
