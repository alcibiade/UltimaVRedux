package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestMapLayerIdentity checks if we're getting the same map instance
func TestMapLayerIdentity(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(1, 26). // At ladder
		WithSystemCallbacks().
		Build()
	ladderPos := references.Position{X: 1, Y: 26}

	t.Run("MapInstanceConsistency", func(t *testing.T) {
		// Get map before UpdateSmallMap
		map1 := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
		t.Logf("Map before UpdateSmallMap: %p", map1)

		// Set a sprite
		map1.SetTileByLayer(map_state.MapUnitLayer, &ladderPos, indexes.AvatarKeyIndex)
		tile1 := map1.GetTileByLayer(map_state.MapUnitLayer, &ladderPos)
		t.Logf("Set sprite: %v (index=%d)", tile1.Name, tile1.Index)

		// Call UpdateSmallMap
		gs.UpdateSmallMap(gs.GameReferences.TileReferences, gs.GameReferences.LocationReferences)

		// Get map after UpdateSmallMap
		map2 := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
		t.Logf("Map after UpdateSmallMap: %p", map2)

		// Check if sprite is still there
		tile2 := map2.GetTileByLayer(map_state.MapUnitLayer, &ladderPos)
		if tile2 != nil {
			t.Logf("After UpdateSmallMap: %v (index=%d)", tile2.Name, tile2.Index)
		} else {
			t.Logf("After UpdateSmallMap: nil")
		}

		// Are they the same map?
		if map1 == map2 {
			t.Logf("Same map instance")
		} else {
			t.Logf("DIFFERENT map instances!")
		}
	})
}
