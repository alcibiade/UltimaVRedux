package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// GetShadowlordPlanetIndices returns the planet indices (0-7) where Shadowlords are currently located.
// Planet index = Town location - 1 (Town 1-8 maps to Planet 0-7).
// This is used by the astronomy display to show red comets near planets.
func (g *GameState) GetShadowlordPlanetIndices() []int {
	var indices []int

	// Check each of the 8 Shadowlord towns
	for loc := ShadowlordTownMinLocation; loc <= ShadowlordTownMaxLocation; loc++ {
		lord := g.GetShadowlordForLocation(references.Location(loc))
		if lord != references.NoShadowlord {
			// Planet index = town location - 1
			planetIndex := loc - ShadowlordTownMinLocation
			indices = append(indices, planetIndex)
		}
	}

	return indices
}
