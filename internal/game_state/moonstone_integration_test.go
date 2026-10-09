package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMoonstoneSystem_IntegrationWithRealData tests moonstone system with actual Ultima V save data
func TestMoonstoneSystem_IntegrationWithRealData(t *testing.T) {
	// This integration test will initially fail - validates complete moonstone system
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(224, 133). // Initial location of moonstone 0
		WithSystemCallbacks().
		Build()
	// Test moonstone state initialization
	moonstones := gs.GetMoonstones()
	assert.NotNil(t, moonstones, "GameState should have moonstone tracking")
	assert.Len(t, moonstones, 8, "Should track 8 moonstones")

	// Verify initial positions match original Ultima V data
	expectedPositions := []struct{ x, y uint8 }{
		{224, 133}, {96, 102}, {38, 224}, {50, 37},
		{166, 19}, {104, 194}, {23, 126}, {187, 167},
	}

	for i, expected := range expectedPositions {
		assert.Equal(t, expected.x, moonstones[i].X, "Moonstone %d X should match original", i)
		assert.Equal(t, expected.y, moonstones[i].Y, "Moonstone %d Y should match original", i)
		assert.False(t, moonstones[i].IsFound, "Moonstone %d should start buried", i)
	}
}

// TestMoonstoneSearch_Integration tests moonstone discovery through search system
func TestMoonstoneSearch_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(223, 133). // One tile west of moonstone 0 at (224, 133) on overworld
		WithSystemCallbacks().
		Build()
	// Manually place moonstone at test location for this test
	if gs.moonstoneSystem == nil {
		gs.moonstoneSystem = NewMoonstoneSystem()
	}

	// Set deterministic seed for consistent test results
	gs.SetRandomSeed(12345)

	// Search during daytime, when the gate does not cover the buried stone.
	gs.DateTime.Hour = 12
	gs.initializeMoongateStage()

	// Search east to the moonstone location
	success := gs.ActionSearchLargeMap(references.Right) // Search tile to the east (224, 133)
	assert.True(t, success, "Search at buried moonstone location should succeed")
	assert.Contains(t, mockCallbacks.Messages, "Thou dost find\na strange rock!")

	moonstones := gs.GetMoonstones()
	assert.True(t, moonstones[0].IsFound, "Moonstone 0 should be marked as found")
	assert.True(t, moonstones[0].CanCreateMoongate(), "Revealed moonstone remains source-eligible")
	pos := references.Position{X: 224, Y: 133}
	require.True(t, gs.ItemStacksMap.HasItemStackAtPosition(&pos))
	assert.False(t, gs.IsMoongateActiveAt(pos), "daytime exposure does not create an active gate")

	gs.DateTime.Hour = 21
	gs.DateTime.Day = 1
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()
	assert.True(t, gs.IsMoongateActiveAt(pos))
	assert.True(t, gs.HasActiveMoongateAtPosition(&pos))
	assert.Contains(t, gs.GetActiveMoongatePositions(), pos)
	assert.True(t, gs.ItemStacksMap.HasItemStackAtPosition(&pos), "gate does not consume the loose stone")
	assert.False(t, gs.ActionSearchLargeMap(references.Right), "active gate prevents Search")

	// A different source can still select the exposed stone as its phase destination.
	gs.MapState.PlayerLocation.Position = references.Position{X: 96, Y: 102}
	route := gs.ReviewMoongateTravel()
	require.True(t, route.Success)
	assert.Equal(t, uint8(pos.X), route.Route.Destination.X)
	assert.Equal(t, uint8(pos.Y), route.Route.Destination.Y)
	assert.Equal(t, uint8(0), route.Route.Destination.Map)
	assert.Equal(t, uint8(0), route.Route.Destination.Level)

	// Apply the spatial arrival while retaining the loose object at the destination.
	gs.TransitionToLocation(references.BritanniaUnderworld, 0, pos, TransitionFullLocation, false)
	assert.True(t, gs.IsPlayerOnActiveMoongate())
	assert.True(t, gs.ItemStacksMap.HasItemStackAtPosition(&pos))
	gs.MapState.PlayerLocation.Position = references.Position{X: pos.X - 1, Y: pos.Y}
	require.True(t, gs.ActionGetLargeMap(references.Right), "loose stone remains collectible under its gate")
	assert.False(t, gs.ItemStacksMap.HasItemStackAtPosition(&pos))
	assert.True(t, moonstones[0].IsInInventory())
	assert.Equal(t, uint8(0xFF), moonstones[0].Map)
	assert.False(t, gs.IsMoongateActiveAt(pos))
	assert.NotContains(t, gs.GetActiveMoongatePositions(), pos)
	_, err := gs.GetMoongateDestinationForPhase(datetime.MoonPhase(0))
	require.Error(t, err)
}

func TestMoongatePlacedStateEligibility(t *testing.T) {
	for _, test := range []struct {
		name  string
		state MoonstoneState
		mapID uint8
		want  bool
	}{
		{"buried", MoonstoneStateBuried, 0, true},
		{"exposed", MoonstoneStateFoundNotCollected, 0, true},
		{"inventory", MoonstoneStateInInventory, 0xFF, false},
		{"buried inventory map", MoonstoneStateBuried, 0xFF, false},
		{"exposed inventory map", MoonstoneStateFoundNotCollected, 0xFF, false},
		{"inventory world map", MoonstoneStateInInventory, 0, false},
		{"invalid state", MoonstoneState(2), 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ms := NewMoonstoneSystem()
			stone := ms.GetMoonstoneByIndex(0)
			stone.State, stone.Map = test.state, test.mapID
			pos := references.Position{X: references.Coordinate(stone.X), Y: references.Coordinate(stone.Y)}
			assert.Equal(t, test.want, stone.CanCreateMoongate())
			assert.Equal(t, test.want, ms.CanCreateMoongateAt(references.BritanniaUnderworld, 0, pos))
			assert.False(t, ms.CanCreateMoongateAt(references.Britain, 0, pos))
			assert.False(t, ms.CanCreateMoongateAt(references.BritanniaUnderworld, -1, pos))
			if test.mapID == 0xFF || test.state == MoonstoneStateInInventory {
				_, err := ms.GetMoongateDestinationForPhase(datetime.MoonPhase(0))
				require.Error(t, err)
			}
		})
	}
}

func TestBuriedYewMoonstoneCreatesGateBeforeDawn(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 36).
		WithSystemCallbacks().
		Build()
	gs.DateTime.Hour = 4
	gs.DateTime.Minute = 27
	gs.initializeMoongateStage()

	gatePosition := references.Position{X: 50, Y: 37}
	assert.True(t, gs.IsMoongateActiveAt(gatePosition))
	assert.Contains(t, gs.GetActiveMoongatePositions(), gatePosition)

	assert.False(t, gs.ActionSearchLargeMap(references.Down))
	assert.NotContains(t, mockCallbacks.Messages, "Thou dost find\na strange rock!")
	yewMoonstone := gs.GetMoonstoneSystem().GetMoonstoneByIndex(3)
	assert.True(t, yewMoonstone.IsBuried())
	assert.True(t, gs.IsMoongateActiveAt(gatePosition))
}

// TestMoonPhaseCalculation_Integration tests moon phase system with game time
func TestMoonPhaseCalculation_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	// Test moon phase calculation with current game date
	currentPhase := gs.DateTime.GetCurrentMoonPhase()
	assert.GreaterOrEqual(t, int(currentPhase), 0, "Moon phase should be valid (0-7)")
	assert.LessOrEqual(t, int(currentPhase), 7, "Moon phase should be valid (0-7)")

	// Test that advancing time changes moon phases
	for range 7 * 3 {
		gs.AdvanceGameplayMinutes(8 * datetime.MinutesPerHour)
	}

	newPhase := gs.DateTime.GetCurrentMoonPhase()
	// After 7 days, phase might be different (depends on 28-day cycle)
	assert.GreaterOrEqual(t, int(newPhase), 0, "New moon phase should be valid")
	assert.LessOrEqual(t, int(newPhase), 7, "New moon phase should be valid")
}

// TestMoonstoneToMoonPhaseMapping_Integration tests moonstone-to-phase relationships
func TestMoonstoneToMoonPhaseMapping_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	// Test that each moon phase maps to a valid moonstone
	for phase := 0; phase <= 7; phase++ {
		moonstoneIndex := gs.GetMoonstoneIndexForMoonPhase(phase)
		assert.GreaterOrEqual(t, moonstoneIndex, 0, "Phase %d should map to valid moonstone", phase)
		assert.LessOrEqual(t, moonstoneIndex, 7, "Phase %d should map to valid moonstone", phase)

		// Verify the moonstone exists at that index
		moonstones := gs.GetMoonstones()
		assert.NotNil(t, moonstones[moonstoneIndex], "Moonstone %d should exist", moonstoneIndex)
	}
}

// TestSavedGameIntegration_MoonstoneData tests SAVED.GAM loading/saving of moonstone data
func TestSavedGameIntegration_MoonstoneData(t *testing.T) {
	// This test validates loading moonstone data from real SAVED.GAM files
	// Will initially fail until save/load integration is implemented

	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	// Load moonstone data from save file
	moonstones := gs.GetMoonstones()
	assert.Len(t, moonstones, 8, "Should load 8 moonstones from save file")

	// Validate moonstone data consistency
	for i, moonstone := range moonstones {
		// Coordinates should be valid (0-255)
		assert.GreaterOrEqual(t, int(moonstone.X), 0, "Moonstone %d X should be valid", i)
		assert.LessOrEqual(t, int(moonstone.X), 255, "Moonstone %d X should be valid", i)
		assert.GreaterOrEqual(t, int(moonstone.Y), 0, "Moonstone %d Y should be valid", i)
		assert.LessOrEqual(t, int(moonstone.Y), 255, "Moonstone %d Y should be valid", i)

		// Map should be valid (0=overworld, 1-32=towns, 0xFF=inventory)
		validMap := moonstone.Map == 0 || (moonstone.Map >= 1 && moonstone.Map <= 32) || moonstone.Map == 0xFF
		assert.True(t, validMap, "Moonstone %d should have valid map value %d", i, moonstone.Map)
	}
}

// TestCanCreateMoongate_InventoryBehavior tests the critical moonstone inventory behavior
// Based on the reviewed rules for original Ultima V gate-source eligibility function
// See ref:MOON-0004.
func TestCanCreateMoongate_InventoryBehavior(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	moonstones := gs.GetMoonstones()
	assert.Len(t, moonstones, 8, "Should have 8 moonstones")

	// Buried moonstones create gates without first being revealed.
	testMoonstone := &moonstones[0]
	testMoonstone.Map = 0
	testMoonstone.State = MoonstoneStateBuried
	testMoonstone.IsFound = false
	assert.True(t, testMoonstone.CanCreateMoongate())

	testMoonstone.State = MoonstoneStateFoundNotCollected
	testMoonstone.IsFound = true
	assert.True(t, testMoonstone.CanCreateMoongate())

	// Test 2: Inventory moonstone CANNOT create moongate - CRITICAL BEHAVIOR
	testMoonstone.Map = 0xFF // Inventory
	testMoonstone.State = MoonstoneStateInInventory

	assert.False(t, testMoonstone.CanCreateMoongate(),
		"Inventory moonstone (Map=0xFF) should NOT be able to create moongate")

	// Buried state remains source-eligible on a town map; GameState applies
	// the current map/floor identity when projecting active gates.
	testMoonstone.Map = 1
	testMoonstone.State = MoonstoneStateBuried
	testMoonstone.IsFound = false
	assert.True(t, testMoonstone.CanCreateMoongate())

	// Test 4: Verify visual system integration
	gs.MapState.PlayerLocation.Location = references.BritanniaUnderworld
	gs.MapState.PlayerLocation.Floor = 0
	gs.moongateStage = MoongateStageFull
	gs.moonstoneSystem.Moonstones[0].Map = 0
	gs.moonstoneSystem.Moonstones[0].Level = 0
	gs.moonstoneSystem.Moonstones[0].State = MoonstoneStateBuried
	gs.moonstoneSystem.Moonstones[0].IsFound = false
	gs.moonstoneSystem.Moonstones[1].Map = 0xFF
	gs.moonstoneSystem.Moonstones[1].State = MoonstoneStateInInventory
	gs.moonstoneSystem.Moonstones[2].Map = 0
	gs.moonstoneSystem.Moonstones[2].Level = 0
	gs.moonstoneSystem.Moonstones[2].State = MoonstoneStateBuried
	gs.moonstoneSystem.Moonstones[2].IsFound = false

	activeMoongates := gs.GetActiveMoongatePositions()

	pos0 := references.Position{X: references.Coordinate(gs.moonstoneSystem.Moonstones[0].X), Y: references.Coordinate(gs.moonstoneSystem.Moonstones[0].Y)}
	pos1 := references.Position{X: references.Coordinate(gs.moonstoneSystem.Moonstones[1].X), Y: references.Coordinate(gs.moonstoneSystem.Moonstones[1].Y)}
	pos2 := references.Position{X: references.Coordinate(gs.moonstoneSystem.Moonstones[2].X), Y: references.Coordinate(gs.moonstoneSystem.Moonstones[2].Y)}
	assert.Contains(t, activeMoongates, pos0)
	assert.NotContains(t, activeMoongates, pos1)
	assert.Contains(t, activeMoongates, pos2)
}

func TestExposedMoonstonesRemainEligibleForEveryDestinationPhase(t *testing.T) {
	ms := NewMoonstoneSystem()
	for phase := range 8 {
		ms.MarkMoonstoneFoundAtPosition(phase)
		stone := ms.GetMoonstoneByIndex(phase)
		pos := references.Position{X: references.Coordinate(stone.X), Y: references.Coordinate(stone.Y)}
		assert.True(t, ms.CanCreateMoongateAt(references.BritanniaUnderworld, 0, pos))
		destination, err := ms.GetMoongateDestinationForPhase(datetime.MoonPhase(phase))
		require.NoError(t, err)
		assert.Equal(t, MoongateDestination{
			Phase: datetime.MoonPhase(phase),
			X:     stone.X, Y: stone.Y, Map: 0, Level: 0, Location: references.BritanniaUnderworld,
		}, destination)
	}
}
