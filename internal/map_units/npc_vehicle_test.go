package map_units

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestNewNPCFriendlyVehiclePreservesAuthoredHorseFacing(t *testing.T) {
	position := references.Position{X: 10, Y: 10}
	for _, test := range []struct {
		name      string
		sprite    indexes.SpriteIndex
		direction references.Direction
	}{
		{name: "right", sprite: indexes.HorseRight, direction: references.Right},
		{name: "left", sprite: indexes.HorseLeft, direction: references.Left},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := references.NPCReference{Schedule: references.CreateNPCScheduledFixedOneLocation(position, 0)}
			reference.SetKeyIndex(test.sprite)

			horse := NewNPCFriendlyVehicle(references.HorseVehicle, reference)

			assert.Equal(t, test.direction, horse.GetVehicleDetails().CurrentDirection())
			assert.Equal(t, test.sprite, horse.GetKeySpriteIndex())
		})
	}
}

// TestVehicleDetails_ShipDefaults verifies that new frigates have correct default values
// for ship-specific fields (HullHP, SailState, HasHMSCape)
func TestVehicleDetails_ShipDefaults(t *testing.T) {
	tests := []struct {
		name         string
		vehicleType  references.VehicleType
		expectedHP   int
		expectedSail SailState
		expectedCape bool
	}{
		{
			name:         "Frigate defaults",
			vehicleType:  references.FrigateVehicle,
			expectedHP:   99,
			expectedSail: SailFurled,
			expectedCape: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			vehicle := NewVehicleDetails(tt.vehicleType)

			// Assert
			assert.Equal(t, tt.expectedHP, vehicle.GetHullHP(),
				"Frigate should default to 99 hull HP")
			assert.Equal(t, tt.expectedSail, vehicle.GetSailState(),
				"Frigate should default to Rowing sail state")
			assert.Equal(t, tt.expectedCape, vehicle.GetHasHMSCape(),
				"Frigate should default to no HMS Cape")
		})
	}
}

// TestVehicleDetails_SkiffNoShipFields verifies that skiffs have ship fields
// but they're not used in gameplay (skiffs have no sails or hull HP)
func TestVehicleDetails_SkiffNoShipFields(t *testing.T) {
	// Arrange & Act
	skiff := NewVehicleDetails(references.SkiffVehicle)

	// Assert - fields should exist with default values
	// but skiff gameplay doesn't use them
	assert.Equal(t, 99, skiff.GetHullHP(),
		"Skiff has HullHP field (but doesn't use it)")
	assert.Equal(t, SailFurled, skiff.GetSailState(),
		"Skiff has SailState field (but doesn't use it)")
	assert.Equal(t, false, skiff.GetHasHMSCape(),
		"Skiff has HasHMSCape field (but doesn't use it)")
}

// TestVehicleDetails_InitialSkiffQuantity verifies correct initial skiff counts
// New frigates come with 2 skiffs; ref:MERCH-0008.
func TestVehicleDetails_InitialSkiffQuantity(t *testing.T) {
	tests := []struct {
		name          string
		vehicleType   references.VehicleType
		expectedSkiff int
	}{
		{
			name:          "Frigate starts with 2 skiffs",
			vehicleType:   references.FrigateVehicle,
			expectedSkiff: 2,
		},
		{
			name:          "Skiff has no skiffs",
			vehicleType:   references.SkiffVehicle,
			expectedSkiff: 0,
		},
		{
			name:          "Horse has no skiffs",
			vehicleType:   references.HorseVehicle,
			expectedSkiff: 0,
		},
		{
			name:          "Carpet has no skiffs",
			vehicleType:   references.CarpetVehicle,
			expectedSkiff: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			vehicle := NewVehicleDetails(tt.vehicleType)

			// Assert
			assert.Equal(t, tt.expectedSkiff, vehicle.GetSkiffQuantity(),
				"Initial skiff quantity should match original Ultima V behavior")
		})
	}
}

// TestSailState_String verifies string representations of sail states
func TestSailState_String(t *testing.T) {
	tests := []struct {
		state    SailState
		expected string
	}{
		{SailRaised, "Sailing"},
		{SailFurled, "Anchored"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

// TestSailState_MovementPrefix verifies movement prefixes for each sail state
func TestSailState_MovementPrefix(t *testing.T) {
	tests := []struct {
		state    SailState
		expected string
	}{
		{SailRaised, "Sail "},
		{SailFurled, ""}, // Can't move when anchored
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.GetMovementPrefix())
		})
	}
}

// TestHullDamage_BasicDamage verifies normal damage application
func TestHullDamage_BasicDamage(t *testing.T) {
	// Arrange
	frigate := NewVehicleDetails(references.FrigateVehicle)
	assert.Equal(t, 99, frigate.GetHullHP(), "Should start at 99 HP")

	// Act - Apply 20 damage
	destroyed := frigate.TakeHullDamage(20)

	// Assert
	assert.False(t, destroyed, "Ship should not be destroyed at 79 HP")
	assert.Equal(t, 79, frigate.GetHullHP(), "Should have 79 HP after 20 damage")

	// Act - Apply another 30 damage
	destroyed = frigate.TakeHullDamage(30)

	// Assert
	assert.False(t, destroyed, "Ship should not be destroyed at 49 HP")
	assert.Equal(t, 49, frigate.GetHullHP(), "Should have 49 HP after 50 total damage")
}

// TestHullDamage_Destruction verifies ship destruction at 0 HP
func TestHullDamage_Destruction(t *testing.T) {
	// Arrange
	frigate := NewVehicleDetails(references.FrigateVehicle)

	// Act - Damage to exactly 0 HP
	destroyed := frigate.TakeHullDamage(99)

	// Assert
	assert.True(t, destroyed, "Ship should be destroyed at 0 HP")
	assert.Equal(t, 0, frigate.GetHullHP(), "Should have 0 HP")
	assert.True(t, frigate.IsShipDestroyed(), "IsShipDestroyed should return true")

	// Test overkill damage (more than current HP)
	frigate2 := NewVehicleDetails(references.FrigateVehicle)
	destroyed = frigate2.TakeHullDamage(150) // Overkill

	assert.True(t, destroyed, "Ship should be destroyed even with overkill damage")
	assert.Equal(t, 0, frigate2.GetHullHP(), "Should clamp to 0 HP, not negative")
}

// TestHullDamage_WarningThresholds verifies warning messages at different HP levels
// Warn at HP < 10; ref:SHIP-0003.
// Destruction displays "Ship sunk!".
func TestHullDamage_WarningThresholds(t *testing.T) {
	tests := []struct {
		name     string
		hp       int
		expected string
	}{
		{"Healthy ship (50 HP)", 50, ""},
		{"Healthy ship (11 HP)", 11, ""},
		{"Healthy ship (10 HP)", 10, ""},
		{"Danger threshold (9 HP)", 9, "DANGER: SHIP BADLY DAMAGED!"},
		{"Danger threshold (5 HP)", 5, "DANGER: SHIP BADLY DAMAGED!"},
		{"Danger threshold (1 HP)", 1, "DANGER: SHIP BADLY DAMAGED!"},
		{"Destroyed (0 HP)", 0, "Ship sunk!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			frigate := NewVehicleDetails(references.FrigateVehicle)
			// Set HP to test value
			damageAmount := 99 - tt.hp
			if damageAmount > 0 {
				frigate.TakeHullDamage(damageAmount)
			}

			// Act
			message := frigate.GetHullWarningMessage()

			// Assert
			assert.Equal(t, tt.expected, message,
				"HP %d should produce message: '%s'", tt.hp, tt.expected)
		})
	}
}

// TestHullDamage_Repair verifies hull repair mechanics
func TestHullDamage_Repair(t *testing.T) {
	// Arrange - Damaged ship
	frigate := NewVehicleDetails(references.FrigateVehicle)
	frigate.TakeHullDamage(50) // Now at 49 HP
	assert.Equal(t, 49, frigate.GetHullHP())

	// Act - Repair 20 HP
	frigate.RepairHull(20)

	// Assert
	assert.Equal(t, 69, frigate.GetHullHP(), "Should have 69 HP after repair")

	// Act - Repair to above maximum
	frigate.RepairHull(50) // Should clamp to 99 max

	// Assert
	assert.Equal(t, 99, frigate.GetHullHP(), "Should clamp to 99 HP maximum")

	// Test repairing destroyed ship
	destroyed := NewVehicleDetails(references.FrigateVehicle)
	destroyed.TakeHullDamage(99) // Destroy it
	assert.True(t, destroyed.IsShipDestroyed())

	destroyed.RepairHull(10)
	assert.Equal(t, 10, destroyed.GetHullHP(), "Destroyed ship can be repaired")
	assert.False(t, destroyed.IsShipDestroyed(), "Ship no longer destroyed after repair")
}

// TestHullDamage_SkiffImmune verifies skiffs have hull damage methods but don't use them
func TestHullDamage_SkiffImmune(t *testing.T) {
	// Arrange
	skiff := NewVehicleDetails(references.SkiffVehicle)

	// Act & Assert - Methods should work without crashing
	// Skiffs start with 99 HP (field exists but not used in gameplay)
	assert.Equal(t, 99, skiff.GetHullHP())

	destroyed := skiff.TakeHullDamage(20)
	assert.False(t, destroyed, "Damage calculation works on skiff")
	assert.Equal(t, 79, skiff.GetHullHP(), "HP reduced on skiff (field exists)")

	// Skiffs don't use hull HP in actual gameplay, but methods still work
	// This allows uniform code without special-casing skiffs everywhere
}
