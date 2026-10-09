package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHorseMounting_UnownedWishingWell tests mounting a wishing well horse (unowned)
// This simulates the wishing well flow where a new horse is spawned for the player
func TestHorseMounting_UnownedWishingWell(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Paws).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Spawn unowned wishing well horse at player position
	// This replicates the wishing well spawning logic
	playerPos := gs.MapState.PlayerLocation.Position
	playerFloor := gs.MapState.PlayerLocation.Floor

	horseRef := *references.NewNPCReferenceForVehicle(
		references.HorseVehicle,
		playerPos,
		playerFloor,
	)
	horseRef.Schedule.OverrideAllPositions(byte(playerPos.X), byte(playerPos.Y))

	horse := *map_units.NewNPCFriendlyVehicle(references.HorseVehicle, horseRef)

	// Verify default is unowned
	assert.False(t, horse.GetVehicleDetails().IsOwnedByNPC,
		"Wishing well horse should be unowned by default")

	// Add horse to map
	bAddedVehicle := gs.CurrentNPCAIController.GetNpcs().AddVehicle(horse)
	assert.True(t, bAddedVehicle, "Should successfully add horse to map")

	// Attempt to mount the horse
	success := gs.ActionBoard()

	// Assertions
	assert.True(t, success, "Should successfully mount unowned wishing well horse")
	assert.Equal(t, []string{"Board "}, mockCallbacks.CommandPrompts)
	assert.Equal(t, "horse", mockCallbacks.CurrentRowContent)
	assert.Equal(t, 1, len(mockCallbacks.TimeAdvanced()),
		"Should have one time advancement call")
	assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0],
		"Time should advance 1 minute on successful boarding")
}

// TestHorseMounting_OwnedByNPC tests attempting to mount an NPC-owned horse
// This simulates trying to steal an NPC's horse (should fail with "Nay!")
func TestHorseMounting_OwnedByNPC(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Create an owned NPC horse at player position
	playerPos := gs.MapState.PlayerLocation.Position
	playerFloor := gs.MapState.PlayerLocation.Floor

	horseRef := *references.NewNPCReferenceForVehicle(
		references.HorseVehicle,
		playerPos,
		playerFloor,
	)
	horseRef.Schedule.OverrideAllPositions(byte(playerPos.X), byte(playerPos.Y))

	horse := *map_units.NewNPCFriendlyVehicle(references.HorseVehicle, horseRef)

	// Mark as owned by NPC (simulates SAVED.GAM loading)
	horse.GetVehicleDetails().IsOwnedByNPC = true

	// Add horse to map
	bAddedVehicle := gs.CurrentNPCAIController.GetNpcs().AddVehicle(horse)
	assert.True(t, bAddedVehicle, "Should successfully add horse to map")
	storedHorse := gs.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(playerPos, playerFloor)
	assert.NotNil(t, storedHorse)
	storedBefore := storedHorse.GetVehicleDetails().Snapshot()

	// Attempt to mount the owned horse
	success := gs.ActionBoard()

	// Assertions
	assert.True(t, success, "Refusal should complete the ordinary command")
	assert.False(t, mockCallbacks.HorseRefusalShown,
		"Horse refusal should not open a modal workflow")
	assert.Equal(t, []string{"Board "}, mockCallbacks.CommandPrompts)
	assert.Equal(t, "\"Nay!\"", mockCallbacks.CurrentRowContent)
	assert.Equal(t, []int{1}, mockCallbacks.TimeAdvanced())
	assert.Equal(t, references.NoPartyVehicle, gs.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Same(t, storedHorse, gs.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(playerPos, playerFloor))
	assert.Equal(t, storedBefore, storedHorse.GetVehicleDetails().Snapshot())
}

// TestHorseMounting_OwnedVsUnowned_SameMap tests both owned and unowned horses on same map
// Verifies that ownership check correctly distinguishes between different horses
func TestHorseMounting_OwnedVsUnowned_SameMap(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	playerFloor := gs.MapState.PlayerLocation.Floor

	// Create owned horse at position (16, 15)
	ownedPos := references.Position{X: 16, Y: 15}
	ownedHorseRef := *references.NewNPCReferenceForVehicle(
		references.HorseVehicle,
		ownedPos,
		playerFloor,
	)
	ownedHorseRef.Schedule.OverrideAllPositions(byte(ownedPos.X), byte(ownedPos.Y))
	ownedHorse := *map_units.NewNPCFriendlyVehicle(references.HorseVehicle, ownedHorseRef)
	ownedHorse.GetVehicleDetails().IsOwnedByNPC = true
	gs.CurrentNPCAIController.GetNpcs().AddVehicle(ownedHorse)

	// Create unowned horse at position (14, 15)
	unownedPos := references.Position{X: 14, Y: 15}
	unownedHorseRef := *references.NewNPCReferenceForVehicle(
		references.HorseVehicle,
		unownedPos,
		playerFloor,
	)
	unownedHorseRef.Schedule.OverrideAllPositions(byte(unownedPos.X), byte(unownedPos.Y))
	unownedHorse := *map_units.NewNPCFriendlyVehicle(references.HorseVehicle, unownedHorseRef)
	// Leave IsOwnedByNPC as false (default)
	gs.CurrentNPCAIController.GetNpcs().AddVehicle(unownedHorse)

	// Test 1: Try to mount owned horse
	gs.MapState.PlayerLocation.Position = ownedPos
	mockCallbacks.Reset()
	success1 := gs.ActionBoard()
	assert.True(t, success1, "Horse refusal should complete the command")
	assert.False(t, mockCallbacks.HorseRefusalShown,
		"Horse refusal should not show a modal dialog")

	// Test 2: Try to mount unowned horse
	gs.MapState.PlayerLocation.Position = unownedPos
	mockCallbacks.Reset()
	success2 := gs.ActionBoard()
	assert.True(t, success2, "Should successfully mount unowned horse")
	assert.False(t, mockCallbacks.HorseRefusalShown,
		"Horse refusal dialog should NOT be shown for unowned horse")
	assert.Equal(t, []string{"Board "}, mockCallbacks.CommandPrompts)
	assert.Equal(t, "horse", mockCallbacks.CurrentRowContent)
}

func TestHorseRefusalPrecedesOnFootCheck(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	position := gs.MapState.PlayerLocation.Position
	floor := gs.MapState.PlayerLocation.Floor
	owned := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, floor)
	owned.GetVehicleDetails().IsOwnedByNPC = true
	stored := gs.CurrentNPCAIController.GetNpcs().InsertVehicle(*owned)
	assert.NotNil(t, stored)
	gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, position, floor)
	partyBefore := gs.PartyVehicle.GetVehicleDetails().Snapshot()
	mockCallbacks.Reset()

	success := gs.ActionBoard()

	assert.True(t, success)
	assert.Equal(t, "\"Nay!\"", mockCallbacks.CurrentRowContent)
	assert.Equal(t, []int{1}, mockCallbacks.TimeAdvanced())
	assert.Equal(t, partyBefore, gs.PartyVehicle.GetVehicleDetails().Snapshot())
	assert.Same(t, stored, gs.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(position, floor))
}

// TestHorseMounting_OtherVehiclesNotAffected tests that ownership check only affects horses
// Carpets, skiffs, and frigates should not be affected by ownership checks
func TestHorseMounting_OtherVehiclesNotAffected(t *testing.T) {
	testCases := []struct {
		vehicleType references.VehicleType
		name        string
		result      string
	}{
		{references.CarpetVehicle, "Carpet", "carpet"},
		{references.SkiffVehicle, "Skiff", "skiff"},
		{references.FrigateVehicle, "Frigate", "Ship"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs, mockCallbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()

			playerPos := gs.MapState.PlayerLocation.Position
			playerFloor := gs.MapState.PlayerLocation.Floor

			// Create vehicle with IsOwnedByNPC = true
			vehicleRef := *references.NewNPCReferenceForVehicle(
				tc.vehicleType,
				playerPos,
				playerFloor,
			)
			vehicleRef.Schedule.OverrideAllPositions(byte(playerPos.X), byte(playerPos.Y))
			vehicle := *map_units.NewNPCFriendlyVehicle(tc.vehicleType, vehicleRef)
			vehicle.GetVehicleDetails().IsOwnedByNPC = true // Set to true
			gs.CurrentNPCAIController.GetNpcs().AddVehicle(vehicle)

			// Attempt to board
			success := gs.ActionBoard()

			// Non-horse vehicles should board successfully even with IsOwnedByNPC = true
			assert.True(t, success, "%s should board successfully even with IsOwnedByNPC = true", tc.name)
			assert.Equal(t, []string{"Board "}, mockCallbacks.CommandPrompts)
			assert.Equal(t, tc.result, mockCallbacks.CurrentRowContent)
		})
	}
}

// TestHorseMounting_EdgeCases tests edge cases for horse ownership
func TestHorseMounting_EdgeCases(t *testing.T) {
	t.Run("NoHorseAtPosition", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// No vehicle at player position
		success := gs.ActionBoard()

		assert.False(t, success, "Should fail when no vehicle present")
		assert.Equal(t, []string{"Board "}, mockCallbacks.CommandPrompts)
		assert.Equal(t, "What?", mockCallbacks.CurrentRowContent)
	})

	t.Run("AlreadyMounted", func(t *testing.T) {
		gs, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()
		position := gs.MapState.PlayerLocation.Position
		floor := gs.MapState.PlayerLocation.Floor
		vehicles := gs.CurrentNPCAIController.GetNpcs()
		first := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, floor)
		first.GetVehicleDetails().SetPartyVehicleDirection(references.Right)
		require.True(t, vehicles.AddVehicle(*first))
		require.True(t, gs.ActionBoard())
		require.Equal(t, references.HorseVehicle, gs.PartyVehicle.GetVehicleDetails().VehicleType)
		require.Nil(t, vehicles.GetVehicleAtPositionAndFloorOrNil(position, floor))

		second := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, floor)
		second.GetVehicleDetails().SetPartyVehicleDirection(references.Left)
		stored := vehicles.InsertVehicle(*second)
		require.NotNil(t, stored)
		partyBefore := gs.PartyVehicle.GetVehicleDetails().Snapshot()
		storedBefore := stored.GetVehicleDetails().Snapshot()
		timeBefore := gs.DateTime
		callbacks.Reset()

		assert.False(t, gs.ActionBoard())
		assert.Equal(t, []string{"Board "}, callbacks.CommandPrompts)
		assert.Equal(t, []string{"On foot"}, callbacks.Messages)
		assert.Empty(t, callbacks.TimeAdvanced())
		assert.Equal(t, timeBefore, gs.DateTime)
		assert.Equal(t, partyBefore, gs.PartyVehicle.GetVehicleDetails().Snapshot())
		assert.Same(t, stored, vehicles.GetVehicleAtPositionAndFloorOrNil(position, floor))
		assert.Equal(t, storedBefore, stored.GetVehicleDetails().Snapshot())
	})
}
