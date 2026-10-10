package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestShipTurning_TimeCost verifies that FinishTurn advances game time by 1 minute
// Based on original Ultima V: turning direction costs 1 minute of game time
// Note: The actual time cost is implemented in gamescene_movement.go when turning
func TestShipTurning_TimeCost(t *testing.T) {
	// Create test game state with real data
	gs, _, _ := createTestGameStateWithRealData(t, references.BritanniaUnderworld)
	if gs == nil {
		t.Skip("Converting to use real game data - see TESTING.md")
		return
	}

	// Setup: Create a frigate and board it
	frigatePos := references.Position{X: 100, Y: 100}
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		frigatePos,
		gs.MapState.PlayerLocation.Floor,
	)
	gs.PartyVehicle = *frigate

	// Record initial time
	initialMinute := gs.DateTime.Minute

	// Call FinishTurn (simulates what happens when turning in UI layer)
	gs.FinishTurn()

	// Verify time advanced by 2 minutes (frigate rowing speed, see turn.go:85)
	// When sails are furled (default), frigate uses RowingSpeed = 2 minutes
	expectedMinute := (initialMinute + 2) % 60
	assert.Equal(t, expectedMinute, gs.DateTime.Minute, "FinishTurn should advance time by 2 minutes (rowing speed)")
}

// TestShipTurning_NoMovement verifies that turning doesn't change position
// Based on original Ultima V: turning in place only changes direction, not position
func TestShipTurning_NoMovement(t *testing.T) {
	// Create test game state with real data
	gs, _, _ := createTestGameStateWithRealData(t, references.BritanniaUnderworld)
	if gs == nil {
		t.Skip("Converting to use real game data - see TESTING.md")
		return
	}

	// Setup: Create a frigate and board it
	frigatePos := references.Position{X: 100, Y: 100}
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		frigatePos,
		gs.MapState.PlayerLocation.Floor,
	)
	gs.PartyVehicle = *frigate
	gs.MapState.PlayerLocation.Position = frigatePos

	// Set initial direction (North)
	gs.PartyVehicle.GetVehicleDetails().SetPartyVehicleDirection(references.Up)

	// Record initial position
	initialPos := gs.MapState.PlayerLocation.Position

	// Turn West (should not move)
	gs.PartyVehicle.GetVehicleDetails().SetPartyVehicleDirection(references.Left)

	// Verify position unchanged
	assert.Equal(t, initialPos, gs.MapState.PlayerLocation.Position, "Turning should not change position")
	assert.Equal(t, references.Left, gs.PartyVehicle.GetVehicleDetails().CurrentDirection(), "Direction should change to West")
}
