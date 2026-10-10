package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestYellHoist_FrigateFurledToRaised verifies auto-toggle hoists sails from SailFurled to SailRaised
func TestYellHoist_FrigateFurledToRaised(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Setup: Put party on frigate in SailFurled state (default)
	gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.MapState.PlayerLocation.Position,
		gs.MapState.PlayerLocation.Floor,
	)
	// SailFurled is default, but explicitly set for clarity
	gs.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailFurled)

	// Execute auto-toggle (HOIST from Furled)
	success := gs.ActionYellLargeMap()

	// Verify sail state changed to SailRaised
	assert.True(t, success, "Auto-toggle should hoist sails")
	assert.Equal(t, map_units.SailRaised, gs.PartyVehicle.GetVehicleDetails().SailState,
		"Sail state should transition to SailRaised")
	assert.Contains(t, mockCallbacks.Messages[0], "HOIST!",
		"Should display HOIST! message (matches original Ultima V)")
}

// TestYellToggle_FrigateSimpleToggleBehavior verifies the simple 2-state toggle (matches original Ultima V)
func TestYellToggle_FrigateSimpleToggleBehavior(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Setup: Frigate starts with furled sails (default)
	gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.MapState.PlayerLocation.Position,
		gs.MapState.PlayerLocation.Floor,
	)

	// First toggle: Furled → Raised (HOIST)
	success := gs.ActionYellLargeMap()
	assert.True(t, success, "First toggle should succeed")
	assert.Equal(t, map_units.SailRaised, gs.PartyVehicle.GetVehicleDetails().SailState,
		"First toggle: Furled → Raised")
	assert.Contains(t, mockCallbacks.Messages[0], "HOIST!",
		"First toggle should show HOIST!")

	// Second toggle: Raised → Furled (FURL)
	success = gs.ActionYellLargeMap()
	assert.True(t, success, "Second toggle should succeed")
	assert.Equal(t, map_units.SailFurled, gs.PartyVehicle.GetVehicleDetails().SailState,
		"Second toggle: Raised → Furled")
	assert.Contains(t, mockCallbacks.Messages[1], "FURL!",
		"Second toggle should show FURL!")
}

// TestYellFurl_SailRaisedToFurled verifies auto-toggle furls sails from sailing state
func TestYellFurl_SailRaisedToFurled(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Setup: Frigate with sails raised
	gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.MapState.PlayerLocation.Position,
		gs.MapState.PlayerLocation.Floor,
	)
	gs.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailRaised)

	// Execute auto-toggle (SailRaised → SailFurled)
	success := gs.ActionYellLargeMap()

	// Verify sail state changed to SailFurled
	assert.True(t, success, "Auto-toggle should furl sails")
	assert.Equal(t, map_units.SailFurled, gs.PartyVehicle.GetVehicleDetails().SailState,
		"Sail state should transition to SailFurled")
	assert.Contains(t, mockCallbacks.Messages[0], "FURL!",
		"Should display FURL! message (matches original Ultima V)")
}

// TestYellSailCommands_InstantToggle verifies auto-toggle is instant with no time cost (matches original Ultima V)
func TestYellSailCommands_InstantToggle(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Setup: Frigate with default SailFurled state
	gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.MapState.PlayerLocation.Position,
		gs.MapState.PlayerLocation.Floor,
	)

	// Execute auto-toggle and verify instant success
	success := gs.ActionYellLargeMap()
	assert.True(t, success, "Auto-toggle should return true immediately")

	// Verify NO time advancement (original Ultima V has instant toggle with no time cost)
	assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()),
		"Original Ultima V: Sail toggle is instant with no time cost")
}

// TestYellAutoToggle_NotOnFrigate verifies auto-toggle returns false when not on frigate
func TestYellAutoToggle_NotOnFrigate(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Setup: No vehicle (default state is already NoPartyVehicle)
	// No changes needed - party starts on foot

	// Execute auto-toggle while on foot
	success := gs.ActionYellLargeMap()

	// Verify returns false (auto-toggle only works on frigates)
	assert.False(t, success, "Auto-toggle should return false when not on frigate")
}
