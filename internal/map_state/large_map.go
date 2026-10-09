package map_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// UpdateLargeMap currently doesn't do anything useful. It just checks that the actual map is a large map.
func (m *MapState) UpdateLargeMap() {
	if m.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		panic(fmt.Sprintf("expected large map type, got %d", m.PlayerLocation.Location.GetMapType()))
	}
}

// IsOverworld returns true if the player is in the overworld
func (m *MapState) IsOverworld() bool {
	return m.PlayerLocation.Floor == references.FloorNumber(references.OVERWORLD) &&
		m.PlayerLocation.Location.GetMapType() == references.LargeMapType
}

// IsUnderworld returns true if the player is in the underworld
func (m *MapState) IsUnderworld() bool {
	return m.PlayerLocation.Floor == references.FloorNumber(references.UNDERWORLD) &&
		m.PlayerLocation.Location.GetMapType() == references.LargeMapType
}
