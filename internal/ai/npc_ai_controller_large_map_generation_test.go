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

type scriptedGenerationRNG struct {
	t      *testing.T
	values []int
	calls  []int
}

func (r *scriptedGenerationRNG) Intn(n int) int {
	r.t.Helper()
	r.calls = append(r.calls, n)
	if len(r.values) == 0 {
		r.t.Fatalf("unexpected Intn(%d) call", n)
	}
	value := r.values[0]
	r.values = r.values[1:]
	if value < 0 || value >= n {
		r.t.Fatalf("scripted value %d is outside Intn(%d)", value, n)
	}

	return value
}

func TestLargeMapMonsterGenerationUsesOriginalThresholdRoll(t *testing.T) {
	tests := []struct {
		name      string
		tile      indexes.SpriteIndex
		hour      uint8
		rollValue int
		want      bool
	}{
		{name: "hut daytime cannot generate", tile: indexes.Hut, hour: 8, rollValue: 0, want: false},
		{name: "grass daytime cannot generate", tile: indexes.Grass, hour: 8, rollValue: 0, want: false},
		{name: "deep sea daytime cannot generate", tile: indexes.Water1, hour: 8, rollValue: 0, want: false},
		{name: "deep sea dawn cannot generate", tile: indexes.Water1, hour: 5, rollValue: 0, want: false},
		{name: "deep sea before dawn can generate", tile: indexes.Water1, hour: 4, rollValue: 0, want: true},
		{name: "forest one daytime remains standard", tile: indexes.Forest1, hour: 8, rollValue: 0, want: false},
		{name: "swamp daytime succeeds on one", tile: indexes.Swamp, hour: 8, rollValue: 0, want: true},
		{name: "swamp daytime fails on two", tile: indexes.Swamp, hour: 8, rollValue: 1, want: false},
		{name: "road daytime cannot generate", tile: indexes.Path1, hour: 8, rollValue: 0, want: false},
		{name: "road during evening does not generate", tile: indexes.Path1, hour: 23, rollValue: 0, want: false},
		{name: "road before dawn succeeds on two", tile: indexes.Path1, hour: 4, rollValue: 1, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t, values: []int{test.rollValue}}
			controller, layeredMap := newLargeMapGenerationTestController(
				t,
				references.FloorNumber(references.OVERWORLD),
				test.hour,
				rng,
			)
			layeredMap.SetTileByLayer(map_state.MapLayer, &controller.mapState.PlayerLocation.Position, test.tile)

			assert.Equal(t, test.want, controller.shouldGenerateTileBasedMonster())
			assert.Equal(t, []int{references.DefaultMonsterGenerationRollMaximum}, rng.calls)
		})
	}
}

func TestLargeMapMonsterGenerationExhaustivelyMatchesOriginalSourceThreshold(t *testing.T) {
	controller, _ := newLargeMapGenerationTestController(
		t,
		references.FloorNumber(references.OVERWORLD),
		0,
		&scriptedGenerationRNG{t: t},
	)
	playerPosition := &controller.mapState.PlayerLocation.Position
	tiles := []indexes.SpriteIndex{indexes.Path1, indexes.Grass, indexes.Swamp, indexes.Forest3}

	for _, floor := range []references.FloorNumber{
		references.FloorNumber(references.OVERWORLD),
		references.FloorNumber(references.UNDERWORLD),
	} {
		controller.mapState.PlayerLocation.Floor = floor
		layeredMap := controller.mapState.GetLayeredMapByCurrentLocation()
		for hour := range 24 {
			controller.dateTime.Hour = uint8(hour)
			for _, tile := range tiles {
				layeredMap.SetTileByLayer(map_state.MapLayer, playerPosition, tile)
				for roll := 1; roll <= references.DefaultMonsterGenerationRollMaximum; roll++ {
					rng := &scriptedGenerationRNG{t: t, values: []int{roll - 1}}
					controller.rng = rng

					probability := originalSourceGenerationProbability(tile, uint8(hour), floor)
					assert.Equal(t, probability > roll, controller.shouldGenerateTileBasedMonster(),
						"floor=%d hour=%d tile=%d roll=%d", floor, hour, tile, roll)
					assert.Equal(t, []int{references.DefaultMonsterGenerationRollMaximum}, rng.calls)
				}
			}
		}
	}
}

//nolint:mnd // Independent terrain/hour attempt thresholds; ref:ENEMY-0004.
func originalSourceGenerationProbability(
	tile indexes.SpriteIndex,
	hour uint8,
	floor references.FloorNumber,
) int {
	if floor == references.FloorNumber(references.UNDERWORLD) {
		return 3
	}

	probability := 1
	switch {
	case tile >= 0x20 && tile <= 0x26:
		probability = 0
	case tile == 0x04 || (tile >= 0x09 && tile <= 0x0f):
		probability = 2
	}
	if hour >= 0x20 || hour < 5 { // 0x20 is intentionally unreachable for a 0-23 clock.
		probability += 3
	}

	return probability
}

func TestLargeMapMonsterGenerationUsesFixedUnderworldThreshold(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t, values: []int{1}}
	controller, layeredMap := newLargeMapGenerationTestController(
		t,
		references.FloorNumber(references.UNDERWORLD),
		23,
		rng,
	)
	layeredMap.SetTileByLayer(map_state.MapLayer, &controller.mapState.PlayerLocation.Position, indexes.Grass)

	assert.True(t, controller.shouldGenerateTileBasedMonster(), "underworld p=3 should exceed a roll of two")
	assert.Equal(t, []int{references.DefaultMonsterGenerationRollMaximum}, rng.calls)
}

func TestDisabledLargeMapMonsterGenerationConsumesNoRandomness(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t}
	controller, _ := newLargeMapGenerationTestController(t, references.FloorNumber(references.OVERWORLD), 8, rng)
	controller.debugOptions.MonsterGen = false

	assert.False(t, controller.shouldGenerateTileBasedMonster())
	assert.Empty(t, rng.calls)
}

func TestLargeMapGenerationProbabilityIgnoresAbandonedVehicleOverlay(t *testing.T) {
	for _, test := range []struct {
		name  string
		floor references.FloorNumber
		hour  uint8
		tile  indexes.SpriteIndex
		roll  int
		want  bool
	}{
		{"daytime swamp", 0, 12, indexes.Swamp, 0, true},
		{"predawn road", 0, 4, indexes.Path1, 2, false},
		{"underworld", -1, 12, indexes.Swamp, 1, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t, values: []int{test.roll}}
			controller, layer := newLargeMapGenerationTestController(t, test.floor, test.hour, rng)
			position := controller.mapState.PlayerLocation.Position
			layer.SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
			layer.SetTileByLayer(map_state.MapOverrideLayer, &position, test.tile)
			assert.Equal(t, test.want, controller.shouldGenerateTileBasedMonster())
			assert.Equal(t, []int{references.DefaultMonsterGenerationRollMaximum}, rng.calls)

			// Exit leaves the unboarded vehicle at the party's current square.
			// Project a real vehicle, rather than replacing the terrain fixture.
			vehicle := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, test.floor)
			vehicle.SetPos(position)
			controller.mapUnits = append(controller.mapUnits, vehicle)
			controller.FreshenExistingNPCsOnMap()
			require.NotEqual(t, test.tile, layer.GetTopTile(&position).Index)
			require.Equal(t, test.tile, layer.GetTileTopMapOnlyTile(&position).Index)
			overlaidRNG := &scriptedGenerationRNG{t: t, values: []int{test.roll}}
			controller.rng = overlaidRNG

			assert.Equal(t, test.want, controller.shouldGenerateTileBasedMonster(), "generation reads map terrain, not the object overlay (ref:ENEMY-0004)")
			assert.Equal(t, rng.calls, overlaidRNG.calls, "the overlay cannot change the generation draw")
		})
	}
}

func TestWaterGenerationPrecedesContactAndUsesOnlyCanonicalDraws(t *testing.T) {
	for _, test := range []struct {
		name   string
		floor  references.FloorNumber
		hour   byte
		values []int
		bounds []int
		want   references.MonsterType
	}{
		{name: "daylight surface", floor: 0, hour: 12, values: []int{0}, bounds: []int{30}, want: references.MonsterNone},
		{name: "night surface whirlpool", floor: 0, hour: 4, values: []int{0, 0, 0, 0, 0, 0, 7}, bounds: []int{30, 2, 2, 5, 2, 65, 8}, want: references.MonsterWhirlpool},
		{name: "underworld cannot roll whirlpool", floor: -1, hour: 12, values: []int{0, 0, 0, 0, 0, 0, 0}, bounds: []int{30, 2, 2, 5, 2, 65, 256}, want: references.UnderworldWaterMonsters[0]},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t, values: test.values}
			controller, layeredMap := newLargeMapGenerationTestController(t, test.floor, test.hour, rng)
			candidate := references.Position{X: 107, Y: 108}
			layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, indexes.Water1)
			layeredMap.SetTileByLayer(map_state.MapLayer, &controller.mapState.PlayerLocation.Position, indexes.Water1)
			contacts := 0
			controller.AdvanceNextTurnWithActorCheck(func(*map_units.NPCEnemy) OutdoorActorAction {
				contacts++
				assert.Equal(t, test.bounds, rng.calls)
				if test.want == references.MonsterNone {
					assert.Empty(t, controller.mapUnits)
				} else {
					require.Len(t, controller.mapUnits, 1)
					enemy, ok := controller.mapUnits[0].(*map_units.NPCEnemy)
					require.True(t, ok)
					assert.Equal(t, test.want, enemy.EnemyReference.MonsterType)
					assert.Equal(t, candidate, enemy.Pos())
					assert.False(t, enemy.MapUnitDetails().SkipNextTurn, "movement has not run yet")
				}
				return OutdoorActorAttacked
			})
			if test.want == references.MonsterNone {
				assert.Zero(t, contacts)
			} else {
				assert.Equal(t, 1, contacts)
			}
			assert.Equal(t, test.bounds, rng.calls, "contact suppresses movement draws")
		})
	}
}

func TestGeneratedLargeMapMonsterStartsBeyondViewportWithVacantToken(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t, values: []int{0, 0, 0, 0, 0}}
	controller, layeredMap := newLargeMapGenerationTestController(
		t,
		references.FloorNumber(references.OVERWORLD),
		8,
		rng,
	)
	candidate := references.Position{X: 107, Y: 108}
	layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, indexes.Grass)

	friendly := map_units.NewNPCFriendly(references.NPCReference{}, 1) //nolint:exhaustruct
	controller.mapUnits = map_units.MapUnits{friendly}
	controller.generateTileBasedMonster()

	require.Len(t, controller.mapUnits, 2)
	enemy, ok := controller.mapUnits[1].(*map_units.NPCEnemy)
	require.True(t, ok)
	assert.Equal(t, candidate, enemy.Pos())
	assert.Equal(t, 0, enemy.MapUnitDetails().NPCNum)
	assert.Greater(t, int(enemy.Pos().Y-controller.mapState.PlayerLocation.Position.Y),
		controller.mapState.YTilesVisibleOnGameScreen/2)
	assert.GreaterOrEqual(t, absCoordinateDifference(enemy.Pos().X, controller.mapState.PlayerLocation.Position.X), 7)
	assert.GreaterOrEqual(t, absCoordinateDifference(enemy.Pos().Y, controller.mapState.PlayerLocation.Position.Y), 7)
}

func absCoordinateDifference(first, second references.Coordinate) int {
	difference := int(first - second)
	if difference < 0 {
		return -difference
	}

	return difference
}

func newLargeMapGenerationTestController(
	t *testing.T,
	floor references.FloorNumber,
	hour uint8,
	rng references.RandomProvider,
) (*NPCAIControllerLargeMap, *map_state.LayeredMap) {
	t.Helper()
	tileRefs := references.NewTileReferences()
	emptyMap := &references.LargeMapReference{}
	layeredMaps := map_state.NewLayeredMaps(tileRefs, emptyMap, emptyMap, 19, 13)
	state := &map_state.MapState{ //nolint:exhaustruct // Focused generation fixture.
		PlayerLocation: references.PlayerLocation{
			Location: references.BritanniaUnderworld,
			Position: references.Position{X: 100, Y: 100},
			Floor:    floor,
		},
		LayeredMaps:               *layeredMaps,
		XTilesVisibleOnGameScreen: 19,
		YTilesVisibleOnGameScreen: 13,
	}
	odds := references.NewDefaultTheOdds()
	enemies := references.NewAllEnemyReferences(tileRefs)
	controller := &NPCAIControllerLargeMap{ //nolint:exhaustruct // Focused generation fixture.
		tileRefs:        tileRefs,
		mapState:        state,
		debugOptions:    &references.DebugOptions{MonsterGen: true},
		enemyReferences: enemies,
		theOdds:         &odds,
		dateTime:        &datetime.UltimaDate{Hour: hour},
		rng:             rng,
		mapUnits:        make(map_units.MapUnits, 0, map_units.MaximumNpcsPerMap),
	}

	return controller, state.GetLayeredMapByCurrentLocation()
}
