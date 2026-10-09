package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDungeonIgniteTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	attachDungeonStateForTest(t, state, references.DungeonCells{})
	state.PartyState.Inventory.Provisions.Torches.Set(0)

	return state, callbacks
}

func TestDungeonIgniteAddsOriginalFuelWithoutDuplicateTime(t *testing.T) {
	state, _ := newDungeonIgniteTestState(t)
	state.PartyState.Inventory.Provisions.Torches.IncrementBy(2)
	state.MapState.Lighting.RestoreTorchMinutes(10)
	state.SetRandomSeed(1234)
	expectedAddition := state.RandomIntInRange(dungeonTorchMinimumMinutes, dungeonTorchMaximumMinutes)
	state.SetRandomSeed(1234)
	rngCallsBefore := state.rng.Calls()
	minuteBefore := state.DateTime.Minute

	token, start := state.BeginDungeonCommandCycle()
	require.True(t, state.ActionIgnite())
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.Equal(t, 1, start.ElapsedMinutes)
	assert.Equal(t, byte((int(minuteBefore)+1)%60), state.DateTime.Minute)
	assert.Equal(t, 10-1+expectedAddition, state.MapState.Lighting.TorchMinutesRemaining(),
		"precharge burns existing light before Ignite adds new fuel")
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Provisions.Torches.Get())
	assert.Equal(t, rngCallsBefore+1, state.rng.Calls())
}

func TestDungeonIgniteSaturatesAndStillConsumesTorch(t *testing.T) {
	state, _ := newDungeonIgniteTestState(t)
	state.PartyState.Inventory.Provisions.Torches.IncrementByOne()
	state.MapState.Lighting.RestoreTorchMinutes(250)
	token, _ := state.BeginDungeonCommandCycle()

	require.True(t, state.ActionIgnite())
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.Equal(t, 255, state.MapState.Lighting.TorchMinutesRemaining())
	assert.Equal(t, uint16(0), state.PartyState.Inventory.Provisions.Torches.Get())
}

func TestDungeonIgniteWithoutTorchRetainsPrechargedCycle(t *testing.T) {
	state, callbacks := newDungeonIgniteTestState(t)
	rngCallsBefore := state.rng.Calls()
	minuteBefore := state.DateTime.Minute
	token, start := state.BeginDungeonCommandCycle()

	assert.False(t, state.ActionIgnite())
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.Equal(t, 1, start.ElapsedMinutes)
	assert.Equal(t, byte((int(minuteBefore)+1)%60), state.DateTime.Minute)
	assert.Contains(t, callbacks.Messages, "None owned!")
	assert.Equal(t, rngCallsBefore, state.rng.Calls())
}

func TestDungeonIgniteCannotResolveOneCycleTwice(t *testing.T) {
	state, _ := newDungeonIgniteTestState(t)
	state.PartyState.Inventory.Provisions.Torches.Set(2)
	token, _ := state.BeginDungeonCommandCycle()
	require.True(t, state.ActionIgnite())

	assert.PanicsWithValue(t, "dungeon command cycle already resolved an action", func() {
		state.ActionIgnite()
	})
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Provisions.Torches.Get())
}

func TestDungeonIgniteIncludesBothOriginalFuelEndpoints(t *testing.T) {
	for _, target := range []int{dungeonTorchMinimumMinutes, dungeonTorchMaximumMinutes} {
		t.Run(fmt.Sprintf("%d_minutes", target), func(t *testing.T) {
			state, _ := newDungeonIgniteTestState(t)
			var seed uint64
			for candidate := uint64(1); candidate < 10000; candidate++ {
				state.SetRandomSeed(candidate)
				if state.RandomIntInRange(dungeonTorchMinimumMinutes, dungeonTorchMaximumMinutes) == target {
					seed = candidate
					break
				}
			}
			require.NotZero(t, seed)
			state.SetRandomSeed(seed)
			state.PartyState.Inventory.Provisions.Torches.Set(1)
			token, _ := state.BeginDungeonCommandCycle()

			require.True(t, state.ActionIgnite())
			state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

			assert.Equal(t, target, state.MapState.Lighting.TorchMinutesRemaining())
		})
	}
}

func TestDungeonIgniteRequiresCommandCycle(t *testing.T) {
	state, _ := newDungeonIgniteTestState(t)
	state.PartyState.Inventory.Provisions.Torches.IncrementByOne()

	assert.PanicsWithValue(t, "dungeon action requires an active command cycle", func() {
		state.ActionIgnite()
	})
}

func TestSurfaceIgniteAdvancesAuthoritativeLightAndTime(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Provisions.Torches.Set(1)

	require.True(t, state.ActionIgnite())

	assert.Equal(t, 239, state.MapState.Lighting.TorchMinutesRemaining())
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
}

func TestSurfaceIgniteSetsFuelTo240BeforeCommandCompletion(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Provisions.Torches.Set(1)
	callbacks.Reset()

	state.BeginSurfaceCommandRouting()
	require.True(t, state.ActionIgnite())
	state.EndSurfaceCommandRouting()

	assert.Equal(t, 240, state.MapState.Lighting.TorchMinutesRemaining())
	assert.Empty(t, callbacks.TimeAdvanced())
}
