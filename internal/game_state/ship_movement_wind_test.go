package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestShipWindSpeed_FrigateSailing tests wind-based frigate sailing speed
func TestShipWindSpeed_FrigateSailing(t *testing.T) {
	tests := []struct {
		name            string
		wind            environment.WindDirection
		shipDirection   references.Direction
		hasHMSCape      bool
		expectedMinutes int
		description     string
	}{
		{
			name:            "Tailwind without HMS Cape",
			wind:            environment.North, // Wind FROM north (blows south)
			shipDirection:   references.Down,   // Sailing south (with the wind)
			hasHMSCape:      false,
			expectedMinutes: 2, // 1 tick * 2 minutes
			description:     "Sailing with wind (tailwind) - fastest",
		},
		{
			name:            "Tailwind with HMS Cape",
			wind:            environment.North, // Wind FROM north
			shipDirection:   references.Down,   // Sailing south
			hasHMSCape:      true,
			expectedMinutes: 2,
			description:     "HMS Cape does not alter ordinary sailing movement completion",
		},
		{
			name:            "Crosswind without HMS Cape",
			wind:            environment.North, // Wind FROM north {0, 1}
			shipDirection:   references.Up,     // Sailing north {0, -1} (X matches, Y differs)
			hasHMSCape:      false,
			expectedMinutes: 2,
			description:     "Crosswind waits are charged before ordinary movement completion",
		},
		{
			name:            "Crosswind with HMS Cape",
			wind:            environment.North,
			shipDirection:   references.Up,
			hasHMSCape:      true,
			expectedMinutes: 2,
			description:     "HMS Cape wait behavior belongs to the automatic cadence",
		},
		{
			name:            "Headwind without HMS Cape",
			wind:            environment.North, // Wind FROM north {0, 1}
			shipDirection:   references.Right,  // Sailing east {1, 0} (no components match)
			hasHMSCape:      false,
			expectedMinutes: 2,
			description:     "n%3 zero-wait heading receives ordinary movement completion",
		},
		{
			name:            "Headwind with HMS Cape",
			wind:            environment.North,
			shipDirection:   references.Right,
			hasHMSCape:      true,
			expectedMinutes: 2,
			description:     "HMS Cape does not alter ordinary sailing movement completion",
		},
		{
			name:            "Calm wind without HMS Cape",
			wind:            environment.Calm,
			shipDirection:   references.Up,
			hasHMSCape:      false,
			expectedMinutes: 2,
			description:     "calm stalling belongs to the automatic cadence",
		},
		{
			name:            "Calm wind with HMS Cape",
			wind:            environment.Calm,
			shipDirection:   references.Up,
			hasHMSCape:      true,
			expectedMinutes: 2,
			description:     "Cape handling does not alter ordinary movement completion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test game state
			gs := createTestGameState()

			// Set the global HMS Cape preference.
			gs.SetHMSCapeActivated(tt.hasHMSCape)

			// Set wind condition
			gs.windSystem.SetWind(tt.wind, false)

			// Setup frigate with sails raised
			gs.PartyVehicle = createTestFrigate(tt.shipDirection)

			// Create water tile
			waterTile := &references.Tile{
				Index:       indexes.SpriteIndex(0), // Water sprite (exact index doesn't matter for speed test)
				SpeedFactor: 1,                      // Default water speed factor
			}

			// Calculate movement time cost
			timeCost := gs.calculateMovementTimeCost(waterTile)

			// Verify expected time
			assert.Equal(t, tt.expectedMinutes, timeCost, tt.description)
		})
	}
}

// TestShipWindSpeed_FrigateRowing tests rowing (sails furled) is wind-independent
func TestShipWindSpeed_FrigateRowing(t *testing.T) {
	tests := []struct {
		name            string
		wind            environment.WindDirection
		hasHMSCape      bool
		expectedMinutes int
		description     string
	}{
		{
			name:            "Rowing without HMS Cape",
			wind:            environment.North,
			hasHMSCape:      false,
			expectedMinutes: 2,
			description:     "Rowing ignores wind - fixed 2 minutes",
		},
		{
			name:            "Rowing with HMS Cape",
			wind:            environment.North,
			hasHMSCape:      true,
			expectedMinutes: 2,
			description:     "HMS Cape does not alter furled rowing",
		},
		{
			name:            "Rowing in calm",
			wind:            environment.Calm,
			hasHMSCape:      false,
			expectedMinutes: 2,
			description:     "Rowing unaffected by calm conditions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := createTestGameState()

			// Set the global HMS Cape preference.
			gs.SetHMSCapeActivated(tt.hasHMSCape)

			gs.windSystem.SetWind(tt.wind, false)

			// Setup frigate with sails FURLED (rowing)
			gs.PartyVehicle = createTestFrigateRowing(references.Up)

			waterTile := &references.Tile{
				Index:       indexes.SpriteIndex(0),
				SpeedFactor: 1,
			}

			timeCost := gs.calculateMovementTimeCost(waterTile)

			assert.Equal(t, tt.expectedMinutes, timeCost, tt.description)
		})
	}
}

// TestShipWindSpeed_Skiff tests skiff is always fixed speed (no wind effect, no HMS Cape)
func TestShipWindSpeed_Skiff(t *testing.T) {
	tests := []struct {
		name            string
		wind            environment.WindDirection
		expectedMinutes int
		description     string
	}{
		{
			name:            "Skiff with favorable wind",
			wind:            environment.North,
			expectedMinutes: 2,
			description:     "Skiff unaffected by wind - always 2 minutes",
		},
		{
			name:            "Skiff with headwind",
			wind:            environment.South,
			expectedMinutes: 2,
			description:     "Skiff unaffected by headwind - always 2 minutes",
		},
		{
			name:            "Skiff in calm",
			wind:            environment.Calm,
			expectedMinutes: 2,
			description:     "Skiff unaffected by calm - always 2 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := createTestGameState()
			gs.windSystem.SetWind(tt.wind, false)

			// Setup skiff (HMS Cape doesn't exist for skiffs)
			gs.PartyVehicle = createWindTestSkiff(references.Up)

			waterTile := &references.Tile{
				Index:       indexes.SpriteIndex(0),
				SpeedFactor: 1,
			}

			timeCost := gs.calculateMovementTimeCost(waterTile)

			assert.Equal(t, tt.expectedMinutes, timeCost, tt.description)
		})
	}
}

// TestShipWindSpeed_LandMovement tests terrain slowdown terrain costs.
// See ref:SHIP-0001.
func TestShipWindSpeed_LandMovement(t *testing.T) {
	gs := createTestGameState()

	// Setup on foot (no vehicle - use empty NPCFriendly)
	gs.PartyVehicle = map_units.NPCFriendly{}

	// Test with different terrain speeds
	tests := []struct {
		name            string
		tile            indexes.SpriteIndex
		expectedMinutes int
	}{
		{"Grass", indexes.Grass, 2},
		{"Swamp", indexes.Swamp, 4},
		{"Brush", indexes.Forest1, 4},
		{"Forest", indexes.Forest3, 6},
		{"Mountains", indexes.SmallMountains, 6},
		{"Waste", indexes.LeftDesert2, 4},
		{"Road", indexes.Path1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tile := &references.Tile{Index: tt.tile}
			timeCost := gs.calculateMovementTimeCost(tile)
			assert.Equal(t, tt.expectedMinutes, timeCost)
		})
	}
}

func TestHMSCapeDoesNotChangeOtherVehicleOrTerrainMovement(t *testing.T) {
	tests := []struct {
		name        string
		vehicleType references.VehicleType
		want        int
	}{
		{name: "foot", vehicleType: references.NoPartyVehicle, want: 4},
		{name: "horse", vehicleType: references.HorseVehicle, want: 4},
		{name: "carpet", vehicleType: references.CarpetVehicle, want: 4},
		{name: "skiff", vehicleType: references.SkiffVehicle, want: SkiffSpeed},
	}
	tile := &references.Tile{Index: indexes.Swamp}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs := createTestGameState()
			if test.vehicleType == references.NoPartyVehicle {
				gs.PartyVehicle = map_units.NPCFriendly{}
			} else {
				gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
					test.vehicleType,
					references.Position{},
					references.MainFloor,
				)
			}

			gs.SetHMSCapeActivated(false)
			withoutCape := gs.calculateMovementTimeCost(tile)
			gs.SetHMSCapeActivated(true)
			withCape := gs.calculateMovementTimeCost(tile)

			assert.Equal(t, test.want, withoutCape)
			assert.Equal(t, withoutCape, withCape)
		})
	}
}

// Helper functions for wind speed tests

func createTestFrigate(direction references.Direction) map_units.NPCFriendly {
	// Create NPC reference with Vehicle type for proper GetVehicleDetails() behavior
	npcRef := references.NPCReference{}
	npcRef.NpcType = references.Vehicle

	vehicle := map_units.NewNPCFriendlyVehicle(references.FrigateVehicle, npcRef)
	details := vehicle.GetVehicleDetails()
	details.SetSailState(map_units.SailRaised)
	// Note: HMS Cape is now a global flag (gs.hmsCapeActivated), not per-vehicle
	details.SetPartyVehicleDirection(direction)
	return *vehicle
}

func createTestFrigateRowing(direction references.Direction) map_units.NPCFriendly {
	// Create NPC reference with Vehicle type for proper GetVehicleDetails() behavior
	npcRef := references.NPCReference{}
	npcRef.NpcType = references.Vehicle

	vehicle := map_units.NewNPCFriendlyVehicle(references.FrigateVehicle, npcRef)
	details := vehicle.GetVehicleDetails()
	details.SetSailState(map_units.SailFurled) // Rowing
	// Note: HMS Cape is now a global flag (gs.hmsCapeActivated), not per-vehicle
	details.SetPartyVehicleDirection(direction)
	return *vehicle
}

func createWindTestSkiff(direction references.Direction) map_units.NPCFriendly {
	// Create NPC reference with Vehicle type for proper GetVehicleDetails() behavior
	npcRef := references.NPCReference{}
	npcRef.NpcType = references.Vehicle

	vehicle := map_units.NewNPCFriendlyVehicle(references.SkiffVehicle, npcRef)
	details := vehicle.GetVehicleDetails()
	details.SetPartyVehicleDirection(direction)
	return *vehicle
}
