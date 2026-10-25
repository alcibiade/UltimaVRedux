package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const deepWaterImpactPauseMs = 25

// Test helper functions

func createTestSkiff(x, y references.Coordinate) *map_units.NPCFriendly {
	pos := references.Position{X: x, Y: y}
	return map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, pos, 0)
}

func createTestVehicle(vehicleType references.VehicleType, x, y references.Coordinate) *map_units.NPCFriendly {
	pos := references.Position{X: x, Y: y}
	return map_units.NewNPCFriendlyVehiceNewRef(vehicleType, pos, 0)
}

func TestSkiffDeepWaterWorkflowImmediatelyOwnsMovementInput(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	gs.PartyVehicle = *createTestSkiff(10, 10)
	position := gs.MapState.PlayerLocation.Position
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)

	gs.ProcessSkiffDeepWaterDamage(&position)

	require.True(t, gs.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, 1, callbacks.BlockInputCalls)
}

func TestSkiffDeepWater_DamageTriggered(t *testing.T) {
	const seed = uint64(12345)
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	gs.SetRandomSeed(seed)
	gs.PartyVehicle = *createTestSkiff(15, 15)
	setDeepWaterActiveParty(gs, 3)
	expectedRNG := beradrng.New(seed)
	initialHP := make(map[int]int)
	expectedDamage := make(map[int]int)
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap != party_state.PartyMapActive || char.IsDead() {
			return
		}
		char.SetMaxHp(100)
		char.SetHp(100)
		char.SetStatus(party_state.Good)
		initialHP[i] = int(char.GetCurrentHp())
		expectedDamage[i] = expectedRNG.Intn(8) + 1
	})
	require.NotEmpty(t, expectedDamage)
	pos := references.Position{X: 10, Y: 10}
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &pos, indexes.Water1)
	timeBefore := gs.DateTime

	gs.ProcessSkiffDeepWaterDamage(&pos)

	require.True(t, gs.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, timeBefore, gs.DateTime)
	drainSkiffDeepWaterWorkflow(t, gs, callbacks)

	for index, wantDamage := range expectedDamage {
		gotDamage := initialHP[index] - int(gs.PartyState.GetCharacter(index).GetCurrentHp())
		assert.Equal(t, wantDamage, gotDamage, "party member %d", index)
		assert.GreaterOrEqual(t, gotDamage, 1)
		assert.LessOrEqual(t, gotDamage, 8)
	}
	assert.Equal(t, expectedRNG.Calls(), gs.GetRNG().Calls())
	assert.Equal(t, timeBefore, gs.DateTime)
	assert.Equal(t, 1, callbacks.BlockInputCalls)
	assert.Equal(t, 1, callbacks.UnblockInputCalls)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, gs.EventQueueManager.HasActiveWorkflows())
	assert.Equal(t, 1, countEqual(callbacks.Messages, "Rough seas!"))
	require.NotEmpty(t, callbacks.ScreenFlashIntensities)
	assert.Zero(t, callbacks.ScreenFlashIntensities[len(callbacks.ScreenFlashIntensities)-1])
	require.NotEmpty(t, callbacks.ScreenShakeOffsets)
	assert.Equal(t, ScreenShakeOffsetCall{}, callbacks.ScreenShakeOffsets[len(callbacks.ScreenShakeOffsets)-1])

	gs.EventQueueManager.Update()
	assert.Equal(t, timeBefore, gs.DateTime)
	assert.Equal(t, 1, callbacks.UnblockInputCalls)
}

func TestSkiffDeepWaterStartsPartyWipeAfterDamagePresentation(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	for index := range gs.PartyState.GetCharacterCount() {
		gs.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapAtInn(references.Britain)
	}
	member := gs.PartyState.GetCharacter(0)
	member.PartyMap = party_state.PartyMapActive
	member.SetStatus(party_state.Good)
	member.SetMaxHp(1)
	member.SetHp(1)
	gs.PartyVehicle = *createTestSkiff(10, 10)
	position := gs.MapState.PlayerLocation.Position
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)
	deathSequences := 0
	gs.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		deathSequences++
		assert.NotEmpty(t, callbacks.ScreenFlashIntensities)
		assert.Zero(t, callbacks.ScreenFlashIntensities[len(callbacks.ScreenFlashIntensities)-1])
	}

	gs.ProcessSkiffDeepWaterDamage(&position)
	drainSkiffDeepWaterWorkflow(t, gs, callbacks)

	assert.True(t, gs.PartyState.IsAllPartyDead())
	assert.Equal(t, 1, deathSequences)
}

func TestSkiffDeepWaterShutdownClearsVisualState(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	gs.PartyVehicle = *createTestSkiff(10, 10)
	position := gs.MapState.PlayerLocation.Position
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)

	gs.ProcessSkiffDeepWaterDamage(&position)
	gs.EventQueueManager.Update()
	gs.ElapsedMs += 100
	gs.EventQueueManager.Update()
	gs.EventQueueManager.Shutdown()

	require.NotEmpty(t, callbacks.ScreenFlashIntensities)
	assert.Zero(t, callbacks.ScreenFlashIntensities[len(callbacks.ScreenFlashIntensities)-1])
	require.NotEmpty(t, callbacks.ScreenShakeOffsets)
	assert.Equal(t, ScreenShakeOffsetCall{}, callbacks.ScreenShakeOffsets[len(callbacks.ScreenShakeOffsets)-1])
	assert.False(t, callbacks.InputBlocked)
}

// TestSkiffDeepWater_NoDeepWater verifies no damage in SEA or SHOALS water
func TestSkiffDeepWater_NoDeepWater(t *testing.T) {
	tests := []struct {
		name      string
		tileIndex indexes.SpriteIndex
	}{
		{"SEA water (no damage)", indexes.Water2},
		{"SHOALS water (no damage)", indexes.WaterCoast},
		{"Grass (no damage)", indexes.Grass},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()
			gs.PartyVehicle = *createTestSkiff(15, 15)
			initialHP := make([]int, gs.PartyState.GetCharacterCount())
			for i := 0; i < gs.PartyState.GetCharacterCount(); i++ {
				initialHP[i] = int(gs.PartyState.GetCharacter(i).GetCurrentHp())
			}
			pos := references.Position{X: 10, Y: 10}
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &pos, tt.tileIndex)
			timeBefore := gs.DateTime

			assertDeepWaterNoOp(t, gs, callbacks, func() {
				gs.ProcessSkiffDeepWaterDamage(&pos)
			})
			assert.Equal(t, timeBefore, gs.DateTime)
			for i := 0; i < gs.PartyState.GetCharacterCount(); i++ {
				character := gs.PartyState.GetCharacter(i)
				currentHP := int(character.GetCurrentHp())
				assert.Equal(t, initialHP[i], currentHP,
					"Party member %d should not take damage on tile %v", i, tt.tileIndex)
			}
		})
	}
}

func TestSkiffDeepWater_NilPosition(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	gs.PartyVehicle = *createTestSkiff(15, 15)
	timeBefore := gs.DateTime

	assertDeepWaterNoOp(t, gs, callbacks, func() {
		gs.ProcessSkiffDeepWaterDamage(nil)
	})
	assert.Equal(t, timeBefore, gs.DateTime)
}

// TestSkiffDeepWater_NotOnSkiff verifies no damage when not on skiff or carpet
func TestSkiffDeepWater_NotOnSkiff(t *testing.T) {
	tests := []struct {
		name        string
		vehicleType references.VehicleType
	}{
		{"On foot (no damage)", references.NoPartyVehicle},
		{"On horse (no damage)", references.HorseVehicle},
		{"On frigate (no damage)", references.FrigateVehicle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()
			if tt.vehicleType != references.NoPartyVehicle {
				vehicle := createTestVehicle(tt.vehicleType, 15, 15)
				gs.PartyVehicle = *vehicle
			}
			initialHP := make([]int, gs.PartyState.GetCharacterCount())
			for i := 0; i < gs.PartyState.GetCharacterCount(); i++ {
				initialHP[i] = int(gs.PartyState.GetCharacter(i).GetCurrentHp())
			}
			pos := references.Position{X: 10, Y: 10}
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &pos, indexes.Water1)
			timeBefore := gs.DateTime

			assertDeepWaterNoOp(t, gs, callbacks, func() {
				gs.ProcessSkiffDeepWaterDamage(&pos)
			})
			assert.Equal(t, timeBefore, gs.DateTime)
			for i := 0; i < gs.PartyState.GetCharacterCount(); i++ {
				character := gs.PartyState.GetCharacter(i)
				currentHP := int(character.GetCurrentHp())
				assert.Equal(t, initialHP[i], currentHP,
					"Party member %d should not take damage on %v", i, tt.vehicleType)
			}
		})
	}
}

func TestSkiffDeepWater_CarpetSame(t *testing.T) {
	const seed = uint64(12345)
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	gs.SetRandomSeed(seed)
	gs.PartyVehicle = *createTestVehicle(references.CarpetVehicle, 15, 15)
	setDeepWaterActiveParty(gs, 3)
	expectedRNG := beradrng.New(seed)
	expectedDamage := make(map[int]int)
	gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if char.PartyMap == party_state.PartyMapActive && !char.IsDead() {
			expectedDamage[i] = expectedRNG.Intn(8) + 1
		}
	})
	require.NotEmpty(t, expectedDamage)
	pos := references.Position{X: 10, Y: 10}
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &pos, indexes.Water1)
	timeBefore := gs.DateTime

	gs.ProcessSkiffDeepWaterDamage(&pos)
	drainSkiffDeepWaterWorkflow(t, gs, callbacks)

	for index, wantDamage := range expectedDamage {
		gotDamage := 100 - int(gs.PartyState.GetCharacter(index).GetCurrentHp())
		assert.Equal(t, wantDamage, gotDamage, "party member %d", index)
	}
	assert.Equal(t, expectedRNG.Calls(), gs.GetRNG().Calls())
	assert.Equal(t, timeBefore, gs.DateTime)
	assert.Equal(t, 1, callbacks.BlockInputCalls)
	assert.Equal(t, 1, callbacks.UnblockInputCalls)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, gs.EventQueueManager.HasActiveWorkflows())
	assert.Equal(t, 1, countEqual(callbacks.Messages, "Rough seas!"))
}

// TestSkiffDeepWater_DeadMembersSkipped verifies dead party members are not damaged
func TestSkiffDeepWater_DeadMembersSkipped(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	gs.PartyVehicle = *createTestSkiff(15, 15)
	setDeepWaterActiveParty(gs, 3)
	const seed = uint64(12345)
	gs.SetRandomSeed(seed)
	expectedRNG := beradrng.New(seed)
	living := gs.PartyState.GetCharacter(0)
	wantLivingDamage := expectedRNG.Intn(8) + 1
	dead := gs.PartyState.GetCharacter(1)
	dead.SubtractHp(100)
	laterLiving := gs.PartyState.GetCharacter(2)
	wantLaterLivingDamage := expectedRNG.Intn(8) + 1
	inactive := gs.PartyState.GetCharacter(3)
	inactive.SetMaxHp(100)
	inactive.SetHp(100)
	inactive.SetStatus(party_state.Good)
	pos := references.Position{X: 10, Y: 10}
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &pos, indexes.Water1)

	gs.ProcessSkiffDeepWaterDamage(&pos)
	drainSkiffDeepWaterWorkflow(t, gs, callbacks)

	assert.Equal(t, uint16(100-wantLivingDamage), living.GetCurrentHp())
	assert.Equal(t, uint16(0), dead.GetCurrentHp())
	assert.True(t, dead.IsDead())
	assert.Equal(t, uint16(100), inactive.GetCurrentHp())
	assert.Equal(t, uint16(100-wantLaterLivingDamage), laterLiving.GetCurrentHp())
	assert.Equal(t, expectedRNG.Calls(), gs.GetRNG().Calls())
}

func TestSkiffDeepWater_DamageRange(t *testing.T) {
	for seed := uint64(1); seed <= 10; seed++ {
		gs, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()
		gs.SetRandomSeed(seed)
		gs.PartyVehicle = *createTestSkiff(15, 15)
		setDeepWaterActiveParty(gs, 3)
		initialHP := make(map[int]int)
		gs.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
			if char.PartyMap == party_state.PartyMapActive && !char.IsDead() {
				char.SetMaxHp(100)
				char.SetHp(100)
				initialHP[i] = int(char.GetCurrentHp())
			}
		})
		require.NotEmpty(t, initialHP)
		pos := references.Position{X: 10, Y: 10}
		gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &pos, indexes.Water1)

		gs.ProcessSkiffDeepWaterDamage(&pos)
		drainSkiffDeepWaterWorkflow(t, gs, callbacks)

		for index, hp := range initialHP {
			damage := hp - int(gs.PartyState.GetCharacter(index).GetCurrentHp())
			assert.GreaterOrEqual(t, damage, 1, "seed %d, party member %d", seed, index)
			assert.LessOrEqual(t, damage, 8, "seed %d, party member %d", seed, index)
		}
	}
}

func setDeepWaterActiveParty(state *GameState, activeCount int) {
	state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		character.PartyMap = party_state.PartyMapUnrecruited
	})
	for index := range activeCount {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetMaxHp(100)
		character.SetHp(100)
		character.SetStatus(party_state.Good)
	}
}

func assertDeepWaterNoOp(
	t *testing.T,
	state *GameState,
	callbacks *MockSystemCallbacks,
	action func(),
) {
	t.Helper()
	timeBefore := state.DateTime
	rngCallsBefore := state.GetRNG().Calls()
	messageCallsBefore := callbacks.MessageCallCount
	audioCallsBefore := callbacks.AudioCallCount
	visualCallsBefore := callbacks.VisualCallCount
	blockCallsBefore := callbacks.BlockInputCalls
	unblockCallsBefore := callbacks.UnblockInputCalls

	action()

	assert.False(t, state.EventQueueManager.HasActiveWorkflows())
	assert.False(t, callbacks.InputBlocked)
	assert.Equal(t, timeBefore, state.DateTime)
	assert.Equal(t, rngCallsBefore, state.GetRNG().Calls())
	assert.Equal(t, messageCallsBefore, callbacks.MessageCallCount)
	assert.Equal(t, audioCallsBefore, callbacks.AudioCallCount)
	assert.Equal(t, visualCallsBefore, callbacks.VisualCallCount)
	assert.Equal(t, blockCallsBefore, callbacks.BlockInputCalls)
	assert.Equal(t, unblockCallsBefore, callbacks.UnblockInputCalls)
}

func drainSkiffDeepWaterWorkflow(t *testing.T, state *GameState, callbacks *MockSystemCallbacks) {
	t.Helper()
	require.True(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	require.True(t, callbacks.InputBlocked)
	timeBefore := state.DateTime

	state.EventQueueManager.Update()
	require.True(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, timeBefore, state.DateTime)

	state.ElapsedMs += WaterDurationMs - 1
	state.EventQueueManager.Update()
	require.True(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.Equal(t, timeBefore, state.DateTime)

	state.ElapsedMs++
	state.EventQueueManager.Update()
	require.True(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.Equal(t, timeBefore, state.DateTime)

	state.EventQueueManager.Update()
	require.True(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.Equal(t, timeBefore, state.DateTime)

	state.ElapsedMs += deepWaterImpactPauseMs - 1
	state.EventQueueManager.Update()
	require.True(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.Equal(t, timeBefore, state.DateTime)

	state.ElapsedMs++
	state.EventQueueManager.Update()
	require.True(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, timeBefore, state.DateTime)

	state.EventQueueManager.Update()
	require.False(t, state.EventQueueManager.HasWorkflow("skiff_deep_water"))
	assert.False(t, callbacks.InputBlocked)
}
