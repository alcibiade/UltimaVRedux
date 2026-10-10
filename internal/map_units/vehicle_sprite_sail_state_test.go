package map_units

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestVehicleDetails_GetBoardedSpriteIndex_FrigateSailStates verifies that frigates
// return correct sprites based on SailState when GetBoardedSpriteIndex() is called
func TestVehicleDetails_GetBoardedSpriteIndex_FrigateSailStates(t *testing.T) {
	tests := []struct {
		name           string
		sailState      SailState
		direction      references.Direction
		expectedSprite indexes.SpriteIndex
	}{
		// SailRaised state - ShipSails sprites
		{
			name:           "Sailing North",
			sailState:      SailRaised,
			direction:      references.Up,
			expectedSprite: indexes.ShipSailsUp,
		},
		{
			name:           "Sailing East",
			sailState:      SailRaised,
			direction:      references.Right,
			expectedSprite: indexes.ShipSailsRight,
		},
		{
			name:           "Sailing South",
			sailState:      SailRaised,
			direction:      references.Down,
			expectedSprite: indexes.ShipSailsDown,
		},
		{
			name:           "Sailing West",
			sailState:      SailRaised,
			direction:      references.Left,
			expectedSprite: indexes.ShipSailsLeft,
		},

		// SailFurled state - Use ShipNoSails for now (same as Rowing visually)
		// TODO: Find dedicated furled/anchored sprites if they exist in original data
		{
			name:           "Furled North",
			sailState:      SailFurled,
			direction:      references.Up,
			expectedSprite: indexes.ShipNoSailsUp,
		},
		{
			name:           "Furled East",
			sailState:      SailFurled,
			direction:      references.Right,
			expectedSprite: indexes.ShipNoSailsRight,
		},
		{
			name:           "Furled South",
			sailState:      SailFurled,
			direction:      references.Down,
			expectedSprite: indexes.ShipNoSailsDown,
		},
		{
			name:           "Furled West",
			sailState:      SailFurled,
			direction:      references.Left,
			expectedSprite: indexes.ShipNoSailsLeft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create VehicleDetails with specific sail state
			details := &VehicleDetails{
				VehicleType: references.FrigateVehicle,
				SailState:   tt.sailState,
			}
			details.SetPartyVehicleDirection(tt.direction)

			// Call GetBoardedSpriteIndex which should now consider sail state
			sprite := details.GetBoardedSpriteIndex()

			assert.Equal(t, tt.expectedSprite, sprite,
				"Frigate in %s state facing %s should use sprite %d, got %d",
				tt.sailState, tt.direction, tt.expectedSprite, sprite)
		})
	}
}

// TestVehicleDetails_GetBoardedSpriteIndex_NonFrigateVehicles verifies that non-frigate
// vehicles still work correctly and ignore sail state
func TestVehicleDetails_GetBoardedSpriteIndex_NonFrigateVehicles(t *testing.T) {
	tests := []struct {
		name           string
		vehicleType    references.VehicleType
		direction      references.Direction
		prevDirection  references.Direction
		expectedSprite indexes.SpriteIndex
	}{
		{
			name:           "Horse Left",
			vehicleType:    references.HorseVehicle,
			direction:      references.Left,
			prevDirection:  references.Left,
			expectedSprite: indexes.RidingHorseLeft,
		},
		{
			name:           "Horse Right",
			vehicleType:    references.HorseVehicle,
			direction:      references.Right,
			prevDirection:  references.Right,
			expectedSprite: indexes.RidingHorseRight,
		},
		{
			name:           "Carpet Left",
			vehicleType:    references.CarpetVehicle,
			direction:      references.Left,
			prevDirection:  references.Left,
			expectedSprite: indexes.RidingMagicCarpetLeft,
		},
		{
			name:           "Skiff Up",
			vehicleType:    references.SkiffVehicle,
			direction:      references.Up,
			prevDirection:  references.Up,
			expectedSprite: indexes.SkiffUp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create VehicleDetails
			details := &VehicleDetails{
				VehicleType: tt.vehicleType,
			}

			// Set direction(s) - previousDirection matters for horses/carpets
			if tt.vehicleType == references.HorseVehicle || tt.vehicleType == references.CarpetVehicle {
				// Simulate previous direction for horizontal/vertical movement
				if tt.prevDirection != tt.direction {
					// Set previous direction first
					details.SetPartyVehicleDirection(tt.prevDirection)
				}
			}
			// Set current direction
			details.SetPartyVehicleDirection(tt.direction)

			sprite := details.GetBoardedSpriteIndex()

			assert.Equal(t, tt.expectedSprite, sprite,
				"%s facing %s should use sprite %d, got %d",
				tt.vehicleType, tt.direction, tt.expectedSprite, sprite)
		})
	}
}
