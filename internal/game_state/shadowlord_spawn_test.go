package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type nilMapUnitsController struct{}

func (nilMapUnitsController) PopulateMapFirstLoad()           {}
func (nilMapUnitsController) AdvanceNextTurnCalcAndMoveNPCs() {}
func (nilMapUnitsController) FreshenExistingNPCsOnMap()       {}
func (nilMapUnitsController) GetNpcs() *map_units.MapUnits    { return nil }
func (nilMapUnitsController) RemoveAllEnemies()               {}
func (nilMapUnitsController) HardResetNPCs(_ bool)            {}

func TestTrySummonShadowlordCommitsOnePhysicalIdentity(t *testing.T) {
	state, callbacks := newShadowlordSpawnTestState(t)

	require.True(t, state.TrySummonShadowlord("FAULINEI"))

	assert.True(t, state.ShadowlordState.Summoned)
	assert.Equal(t, references.Faulinei, state.ShadowlordState.LastCalled)
	assert.Equal(t, 1, countMatchingShadowlords(
		state.CurrentNPCAIController.GetNpcs(),
		references.Faulinei,
		state.MapState.PlayerLocation.Floor,
	))
	assert.Equal(t, []references.Shadowlord{references.Faulinei}, callbacks.ShadowlordSummonedTriggered)
}

func TestTrySummonShadowlordInsertionFailuresLeaveStateAndPresentationUnchanged(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GameState, references.Position)
	}{
		{
			name: "missing controller",
			mutate: func(state *GameState, _ references.Position) {
				state.CurrentNPCAIController = nil
			},
		},
		{
			name: "controller without map units",
			mutate: func(state *GameState, _ references.Position) {
				state.CurrentNPCAIController = nilMapUnitsController{}
			},
		},
		{
			name: "missing enemy reference",
			mutate: func(state *GameState, _ references.Position) {
				removeShadowlordEnemyReference(state)
			},
		},
		{
			name: "missing key frame reference",
			mutate: func(state *GameState, _ references.Position) {
				clearShadowlordKeyFrameReference(state)
			},
		},
		{
			name: "missing location references",
			mutate: func(state *GameState, _ references.Position) {
				gameReferences := *state.GameReferences
				gameReferences.LocationReferences = nil
				state.GameReferences = &gameReferences
			},
		},
		{
			name: "invalid floor",
			mutate: func(state *GameState, _ references.Position) {
				state.MapState.PlayerLocation.Floor = 99
			},
		},
		{
			name: "out of bounds",
			mutate: func(state *GameState, _ references.Position) {
				state.MapState.PlayerLocation.Position.X = references.XSmallMapTiles
			},
		},
		{
			name: "occupied position",
			mutate: func(state *GameState, target references.Position) {
				addSpawnBlocker(t, state, target)
			},
		},
		{
			name: "full capacity",
			mutate: func(state *GameState, _ references.Position) {
				state.CurrentNPCAIController.GetNpcs().ReplaceWith(
					make([]map_units.MapUnit, map_units.MaximumNpcsPerMap),
				)
			},
		},
		{
			name: "duplicate physical Shadowlord",
			mutate: func(state *GameState, _ references.Position) {
				addPhysicalShadowlord(t, state, references.Nosfentor, references.Position{X: 1, Y: 1})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newShadowlordSpawnTestState(t)
			state.ShadowlordState.LastCalled = references.Astaroth
			target := state.MapState.PlayerLocation.Position.Up().Up()
			test.mutate(state, target)
			rngCalls := state.GetRNG().Calls()

			assert.False(t, state.TrySummonShadowlord("FAULINEI"))

			assert.False(t, state.ShadowlordState.Summoned)
			assert.Equal(t, references.Astaroth, state.ShadowlordState.LastCalled)
			assert.Empty(t, callbacks.ShadowlordSummonedTriggered)
			assert.Equal(t, rngCalls, state.GetRNG().Calls())
		})
	}
}

func TestTrySummonShadowlordRejectsMissingProgressionState(t *testing.T) {
	state, callbacks := newShadowlordSpawnTestState(t)
	state.ShadowlordState = nil

	assert.NotPanics(t, func() {
		assert.False(t, state.TrySummonShadowlord("FAULINEI"))
	})
	assert.Empty(t, callbacks.ShadowlordSummonedTriggered)
	assert.Empty(t, *state.CurrentNPCAIController.GetNpcs())
}

func TestSpawnShadowlordNPCAtPositionRejectsArbitraryImpassableAndPartyOccupiedTiles(t *testing.T) {
	state, _ := newShadowlordSpawnTestState(t)
	wallPosition := references.Position{X: 1, Y: 1}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &wallPosition, indexes.StoneBrickWall)

	assert.False(t, state.SpawnShadowlordNPCAtPosition(references.Faulinei, wallPosition))
	assert.False(t, state.SpawnShadowlordNPCAtPosition(
		references.Faulinei,
		state.MapState.PlayerLocation.Position,
	))
	assert.Empty(t, *state.CurrentNPCAIController.GetNpcs())
	assert.False(t, state.ShadowlordState.Summoned)
	assert.Equal(t, references.NoShadowlord, state.ShadowlordState.LastCalled)
}

func TestAutomaticShadowlordInsertionFailureHasNoPartialEffects(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(references.MainFloor).
		WithPlayerAt(15, 10).
		WithRandomSeed(309).
		WithSystemCallbacks().
		Build()
	state.ShadowlordState.Reset()
	state.ShadowlordState.TownAssignments[references.Astaroth] = references.Britain
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	require.True(t, ok)
	overriddenBefore := countNPCsWithOverriddenAI(controller)
	rngCalls := state.GetRNG().Calls()
	removeShadowlordEnemyReference(state)

	lords := state.collectShadowlordPresenceData(
		references.Britain,
		state.MapState.PlayerLocation.Position,
	)

	assert.Empty(t, lords)
	assert.False(t, state.ShadowlordState.HasAutomaticPresence())
	assert.Zero(t, countShadowlordNPCs(controller))
	assert.Equal(t, overriddenBefore, countNPCsWithOverriddenAI(controller))
	assert.Equal(t, rngCalls, state.GetRNG().Calls())
	assert.Empty(t, callbacks.ShadowlordSummonedTriggered)
}

func TestEnsureSingleAutomaticShadowlordNPCRollsBackReplacementWhenCanonicalSpawnIsOccupied(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(references.MainFloor).
		WithPlayerAt(15, 10).
		WithSystemCallbacks().
		Build()
	state.ShadowlordState.Reset()
	units := state.CurrentNPCAIController.GetNpcs()
	units.ReplaceWith(nil)
	addSpawnBlocker(t, state, GetShadowlordSpawnPosition(references.Britain))
	addPhysicalShadowlord(t, state, references.Nosfentor, references.Position{X: 1, Y: 1})
	unitsBefore := append([]map_units.MapUnit(nil), (*units)...)
	visibilityBefore := make([]bool, len(unitsBefore))
	for index, unit := range unitsBefore {
		visibilityBefore[index] = unit.IsVisible()
	}

	assert.False(t, state.ensureSingleAutomaticShadowlordNPC(references.Astaroth, references.Britain))

	require.Len(t, *units, len(unitsBefore))
	for index, unit := range *units {
		assert.Same(t, unitsBefore[index], unit, "map-unit order and identity must be restored")
		assert.Equal(t, visibilityBefore[index], unit.IsVisible())
	}
	assert.Equal(t, 1, countMatchingShadowlords(units, references.Nosfentor, references.MainFloor))
	assert.Zero(t, countMatchingShadowlords(units, references.Astaroth, references.MainFloor))
}

func TestAutomaticShadowlordPresenceCommitsExactPhysicalIdentity(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(references.MainFloor).
		WithPlayerAt(15, 10).
		WithRandomSeed(309).
		WithSystemCallbacks().
		Build()
	state.ShadowlordState.Reset()
	state.ShadowlordState.TownAssignments[references.Astaroth] = references.Britain

	lords := state.collectShadowlordPresenceData(
		references.Britain,
		state.MapState.PlayerLocation.Position,
	)

	assert.Equal(t, []references.Shadowlord{references.Astaroth}, lords)
	assert.Equal(t, references.Astaroth, state.ShadowlordState.AutomaticPresence)
	assert.Equal(t, 1, countMatchingShadowlords(
		state.CurrentNPCAIController.GetNpcs(),
		references.Astaroth,
		references.MainFloor,
	))
	assert.NotNil(t, matchingShadowlordAt(
		state.CurrentNPCAIController.GetNpcs(),
		GetShadowlordSpawnPosition(references.Britain),
		references.MainFloor,
		references.Astaroth,
	))
}

func newShadowlordSpawnTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Lycaeum).
		WithFloor(2).
		WithPlayerAt(15, 9).
		WithRandomSeed(309).
		WithSystemCallbacks().
		Build()
	state.ShadowlordState.Reset()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)

	return state, callbacks
}

func addSpawnBlocker(t *testing.T, state *GameState, position references.Position) {
	t.Helper()
	reference := state.enemyReference(references.MonsterOrc)
	require.NotNil(t, reference)
	blocker := map_units.NewEnemyNPC(*reference, references.DynamicSpawnIndex)
	blocker.SetPos(position)
	blocker.SetFloor(state.MapState.PlayerLocation.Floor)
	blocker.SetVisible(true)
	require.True(t, state.CurrentNPCAIController.GetNpcs().AddEnemy(blocker))
}

func addPhysicalShadowlord(
	t *testing.T,
	state *GameState,
	lord references.Shadowlord,
	position references.Position,
) {
	t.Helper()
	reference := state.enemyReference(references.MonsterShadowlord)
	require.NotNil(t, reference)
	enemy := map_units.NewEnemyNPC(*reference, references.DynamicSpawnIndex)
	enemy.SetShadowlordIdentity(lord)
	enemy.SetPos(position)
	enemy.SetFloor(state.MapState.PlayerLocation.Floor)
	enemy.SetVisible(true)
	require.True(t, state.CurrentNPCAIController.GetNpcs().AddEnemy(enemy))
}

func removeShadowlordEnemyReference(state *GameState) {
	gameReferences := *state.GameReferences
	enemyReferences := append(references.EnemyReferences(nil), (*state.GameReferences.EnemyReferences)...)
	for index := range enemyReferences {
		if enemyReferences[index].MonsterType == references.MonsterShadowlord {
			enemyReferences[index].MonsterType = references.MonsterRat
		}
	}
	gameReferences.EnemyReferences = &enemyReferences
	state.GameReferences = &gameReferences
}

func clearShadowlordKeyFrameReference(state *GameState) {
	gameReferences := *state.GameReferences
	enemyReferences := append(references.EnemyReferences(nil), (*state.GameReferences.EnemyReferences)...)
	for index := range enemyReferences {
		if enemyReferences[index].MonsterType == references.MonsterShadowlord {
			enemyReferences[index].KeyFrameTile = nil
		}
	}
	gameReferences.EnemyReferences = &enemyReferences
	state.GameReferences = &gameReferences
}

func countMatchingShadowlords(
	units *map_units.MapUnits,
	lord references.Shadowlord,
	floor references.FloorNumber,
) int {
	count := 0
	if units == nil {
		return count
	}
	for _, unit := range *units {
		enemy, ok := unit.(*map_units.NPCEnemy)
		if !ok || !enemy.IsVisible() || enemy.Floor() != floor {
			continue
		}
		identity, hasIdentity := enemy.ShadowlordIdentity()
		if hasIdentity && identity == lord {
			count++
		}
	}

	return count
}
