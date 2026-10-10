package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestCollisionDamage_WarningMessages verifies correct warning messages
// Based on original Ultima V: original command rules (HP < 10), original world-creature rules (destruction)
// This test uses VehicleDetails directly to isolate hull damage behavior.
// See ref:SHIP-0003.
func TestCollisionDamage_WarningMessages(t *testing.T) {
	// Test warning messages using VehicleDetails directly
	frigate := map_units.NewVehicleDetails(references.FrigateVehicle)

	tests := []struct {
		name          string
		hp            int
		expectedMsg   string
		expectWarning bool
	}{
		{"Healthy ship (50 HP)", 50, "", false},
		{"Healthy ship (11 HP)", 11, "", false},
		{"Healthy ship (10 HP)", 10, "", false},
		{"Danger threshold (9 HP)", 9, "DANGER: SHIP BADLY DAMAGED!", true},
		{"Danger threshold (5 HP)", 5, "DANGER: SHIP BADLY DAMAGED!", true},
		{"Danger threshold (1 HP)", 1, "DANGER: SHIP BADLY DAMAGED!", true},
		{"Destroyed (0 HP)", 0, "Ship sunk!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set ship HP to test value
			frigate.RepairHull(99) // Reset to max
			damage := 99 - tt.hp
			if damage > 0 {
				frigate.TakeHullDamage(damage)
			}

			// Get warning message
			warningMsg := frigate.GetHullWarningMessage()

			if tt.expectWarning {
				assert.Equal(t, tt.expectedMsg, warningMsg, "Should show correct warning message")
			} else {
				assert.Empty(t, warningMsg, "Should not show warning for healthy ship")
			}
		})
	}
}
