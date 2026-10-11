package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestAutomaticSailing_RequiresFrigateWithSails tests that automatic sailing only works for frigates with sails raised
func TestAutomaticSailing_RequiresFrigateWithSails(t *testing.T) {
	tests := []struct {
		name          string
		vehicleType   references.VehicleType
		sailState     map_units.SailState
		shouldSucceed bool
		description   string
	}{
		{
			name:          "Frigate with sails raised - should succeed",
			vehicleType:   references.FrigateVehicle,
			sailState:     map_units.SailRaised,
			shouldSucceed: true,
			description:   "Automatic sailing works with raised sails",
		},
		{
			name:          "Frigate with sails furled - should fail",
			vehicleType:   references.FrigateVehicle,
			sailState:     map_units.SailFurled,
			shouldSucceed: false,
			description:   "Cannot auto-sail when rowing",
		},
		{
			name:          "Skiff - should fail",
			vehicleType:   references.SkiffVehicle,
			sailState:     map_units.SailFurled, // Skiffs don't have sails
			shouldSucceed: false,
			description:   "Skiffs don't support automatic sailing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := createTestGameState()

			// Setup vehicle
			npcRef := references.NPCReference{}
			npcRef.NpcType = references.Vehicle
			vehicle := map_units.NewNPCFriendlyVehicle(tt.vehicleType, npcRef)
			details := vehicle.GetVehicleDetails()
			details.SetSailState(tt.sailState)
			details.SetPartyVehicleDirection(references.Up)
			gs.PartyVehicle = *vehicle

			// Attempt to start automatic sailing
			success := gs.StartAutomaticSailing(references.Up)

			if tt.shouldSucceed {
				assert.True(t, success, tt.description)
				assert.True(t, gs.IsSailing(), "Should be in sailing state")
			} else {
				assert.False(t, success, tt.description)
				assert.False(t, gs.IsSailing(), "Should not be in sailing state")
			}
		})
	}
}

// TestAutomaticSailing_StopOnKeyPress tests that sailing stops when any key is pressed
func TestAutomaticSailing_StopOnKeyPress(t *testing.T) {
	gs := createTestGameState()

	// Setup frigate with sails
	npcRef := references.NPCReference{}
	npcRef.NpcType = references.Vehicle
	vehicle := map_units.NewNPCFriendlyVehicle(references.FrigateVehicle, npcRef)
	details := vehicle.GetVehicleDetails()
	details.SetSailState(map_units.SailRaised)
	details.SetPartyVehicleDirection(references.Up)
	gs.PartyVehicle = *vehicle

	// Start sailing
	success := gs.StartAutomaticSailing(references.Up)
	assert.True(t, success, "Should start sailing successfully")
	assert.True(t, gs.IsSailing(), "Should be in sailing state")

	// Stop sailing
	gs.StopAutomaticSailing()
	assert.False(t, gs.IsSailing(), "Should no longer be in sailing state")
	assert.Equal(t, map_units.SailRaised, details.SailState, "stopping must not furl sails")
}

// TestAutomaticSailing_StateTransitions tests sailing state transitions
func TestAutomaticSailing_StateTransitions(t *testing.T) {
	gs := createTestGameState()

	// Setup frigate
	npcRef := references.NPCReference{}
	npcRef.NpcType = references.Vehicle
	vehicle := map_units.NewNPCFriendlyVehicle(references.FrigateVehicle, npcRef)
	details := vehicle.GetVehicleDetails()
	details.SetSailState(map_units.SailRaised)
	details.SetPartyVehicleDirection(references.Up)
	gs.PartyVehicle = *vehicle

	// Initial state - not sailing
	assert.False(t, gs.IsSailing(), "Should not be sailing initially")

	// Start sailing
	success := gs.StartAutomaticSailing(references.Up)
	assert.True(t, success, "Should start sailing")
	assert.True(t, gs.IsSailing(), "Should be sailing after start")

	// Try to start again - should fail (already sailing)
	success = gs.StartAutomaticSailing(references.Up)
	assert.False(t, success, "Should not start sailing when already sailing")
	assert.True(t, gs.IsSailing(), "Should still be sailing")

	// Stop sailing
	gs.StopAutomaticSailing()
	assert.False(t, gs.IsSailing(), "Should not be sailing after stop")

	// Stop again - should be idempotent
	gs.StopAutomaticSailing()
	assert.False(t, gs.IsSailing(), "Should still not be sailing")
}

func TestAutomaticSailingWaitPhases(t *testing.T) {
	tests := []struct {
		name          string
		wind          environment.WindDirection
		shipDirection references.Direction
		expectedWaits int
		canMove       bool
	}{
		{
			name:          "n one waits once",
			wind:          environment.North, // Wind FROM north (blows south)
			shipDirection: references.Down,   // Sailing south (with wind)
			expectedWaits: 1,
			canMove:       true,
		},
		{
			name:          "n two waits twice",
			wind:          environment.North, // Wind FROM north
			shipDirection: references.Up,     // Sailing north (partial match)
			expectedWaits: 2,
			canMove:       true,
		},
		{
			name:          "n three moves immediately",
			wind:          environment.North, // Wind FROM north
			shipDirection: references.Right,  // Sailing east (against wind)
			expectedWaits: 0,
			canMove:       true,
		},
		{
			name:          "calm has no movement threshold",
			wind:          environment.Calm,
			shipDirection: references.Up,
			expectedWaits: 0,
			canMove:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := createTestGameState()

			// Set wind condition
			gs.windSystem.SetWind(tt.wind, false)

			waits, canMove := gs.AutomaticSailingWaitPhases(tt.shipDirection)

			assert.Equal(t, tt.expectedWaits, waits)
			assert.Equal(t, tt.canMove, canMove)
		})
	}
}

func TestCompleteAutomaticSailingWaitAdvancesTimeAndWorldActorsOnly(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 50).
		WithRandomSeed(458).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.MapState.PlayerLocation.Position,
		gs.MapState.PlayerLocation.Floor,
	)
	frigate.GetVehicleDetails().SetSailState(map_units.SailRaised)
	gs.PartyVehicle = *frigate
	addOverworldTestEnemy(
		t,
		gs,
		references.MonsterDragon,
		*references.Right.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position),
	)
	gs.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}
	require.True(t, gs.StartAutomaticSailing(references.Up))
	callbacks.Reset()

	gs.CompleteAutomaticSailingWait()

	callbacks.AssertTimeAdvanced(DefaultLargeMapMinutesPerTurn)
	assert.True(t, gs.HasPendingOverworldContact(), "the wait must run one overworld actor phase")
	assert.Equal(t, 5, gs.ActiveDurationEffect.TurnsRemaining,
		"source waits do not run ordinary player-turn effects")
}

func TestCompleteAutomaticSailingInterruptionLeavesObservationForNextCommand(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()
	state.isSailing = true
	state.DateTime.Hour, state.DateTime.Minute = 5, 59
	state.AdvanceGameplayMinutes(1)
	foodBefore := state.PartyState.Inventory.Provisions.Food.Get()

	state.CompleteAutomaticSailingInterruption()

	assert.False(t, state.IsSailing())
	assert.Equal(t, []int{6}, state.pendingProvisionHours)
	assert.Equal(t, foodBefore, state.PartyState.Inventory.Provisions.Food.Get())
	state.AdvanceGameplayMinutes(2)
	state.checkHourlyFoodConsumption()
	assert.Empty(t, state.pendingProvisionHours)
	assert.Equal(t, foodBefore, state.PartyState.Inventory.Provisions.Food.Get(),
		"a following same-hour command replaces the interrupted crossing")
}

func TestCompleteAutomaticSailingWaitWithHMSCapeUsesOneMinuteAndAlternatingActors(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 50).
		WithRandomSeed(462).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.MapState.PlayerLocation.Position,
		gs.MapState.PlayerLocation.Floor,
	)
	frigate.GetVehicleDetails().SetSailState(map_units.SailRaised)
	gs.PartyVehicle = *frigate
	addOverworldTestEnemy(
		t,
		gs,
		references.MonsterDragon,
		*references.Right.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position),
	)
	gs.SetHMSCapeActivated(true)
	require.True(t, gs.StartAutomaticSailing(references.Up))
	callbacks.Reset()

	gs.CompleteAutomaticSailingWait()

	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	assert.False(t, gs.HasPendingOverworldContact(), "the first deterministic Cape wait skips actors")

	gs.CompleteAutomaticSailingWait()

	assert.Equal(t, []int{1, 1}, callbacks.TimeAdvanced())
	assert.True(t, gs.HasPendingOverworldContact(), "the second deterministic Cape wait runs actors")
}
