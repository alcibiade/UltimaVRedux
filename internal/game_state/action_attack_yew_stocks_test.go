package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

// TestYewStocksMurder_Integration tests the exact scenario reported by user:
// Avatar at Yew (20,17), person in stocks at (20,16)
// This reproduces the bug where Look works but Attack says "Nothing to attack!"
func TestYewStocksMurder_Integration(t *testing.T) {
	// Real Yew map, avatar at (20,17), NPC in stocks at (20,16)
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Yew).
		WithPlayerAt(20, 17).
		WithSystemCallbacks().
		Build()

	t.Run("LookAtStocks", func(t *testing.T) {
		// Look north at person in stocks (20,16)
		lookResult := gs.ActionLookSmallMap(references.Up)

		// Should succeed and show description
		assert.True(t, lookResult, "Look should succeed")
		assert.True(t, len(mockCallbacks.Messages) > 0, "Should have message")
		t.Logf("Look message: %s", mockCallbacks.Messages[0])

		// Should NOT say "nothing to see"
		if len(mockCallbacks.Messages) > 0 {
			assert.NotContains(t, mockCallbacks.Messages[0], "nothing",
				"Look should see the NPC in stocks")
		}
	})

	t.Run("AttackStocks", func(t *testing.T) {
		// Clear previous messages
		mockCallbacks.Messages = []string{}

		// Attack north at person in stocks (20,16)
		attackResult := gs.ActionAttackSmallMap(references.Up)

		// Log what happened
		t.Logf("Attack result: %v", attackResult)
		if len(mockCallbacks.Messages) > 0 {
			t.Logf("Attack message: %s", mockCallbacks.Messages[0])
		} else {
			t.Logf("Attack message: <no message>")
		}

		// Should succeed and show "Murdered!"
		assert.True(t, attackResult, "Attack should succeed")
		assert.True(t, len(mockCallbacks.Messages) > 0, "Should have message")

		if len(mockCallbacks.Messages) > 0 {
			assert.Contains(t, mockCallbacks.Messages[0], "Murdered!",
				"Should show murder message, got: %s", mockCallbacks.Messages[0])
		}
	})

	t.Run("DebugMapAccess", func(t *testing.T) {
		// Debug: Check what GetTopTile returns
		targetPos := references.Up.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)

		t.Logf("Avatar position: (%d, %d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)
		t.Logf("Target position: (%d, %d)", targetPos.X, targetPos.Y)
		t.Logf("Current location: %s", gs.MapState.PlayerLocation.Location)
		t.Logf("Current floor: %d", gs.MapState.PlayerLocation.Floor)
		t.Logf("Map type: %v", gs.MapState.PlayerLocation.Location.GetMapType())

		// Method: GetLayeredMapByCurrentLocation (used by Look and fixed Attack)
		currentMap := gs.GetLayeredMapByCurrentLocation()
		tile := currentMap.GetTopTile(targetPos)

		t.Logf("GetLayeredMapByCurrentLocation: map=%v", currentMap)
		t.Logf("GetTopTile result: tile=%v", tile)

		if tile != nil {
			t.Logf("  Tile index: %d (0x%x)", tile.Index, tile.Index)

			// Check if tile is defenseless base tile
			if gs.isDefenselessBaseTile(tile.Index) {
				t.Logf("  ✅ IS DEFENSELESS BASE TILE")
			} else {
				t.Logf("  ❌ NOT DEFENSELESS BASE TILE")
				t.Logf("  Stocks tile: %d, Manacles tile: %d, LeftBed tile: %d", indexes.Stocks, indexes.Manacles, indexes.LeftBed)
			}
		} else {
			t.Logf("  ❌ TILE IS NIL - This is the bug!")
		}
	})

	t.Run("CompareWithLookImplementation", func(t *testing.T) {
		// Show exactly what Look does vs what Attack does
		targetPos := references.Up.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)

		t.Logf("=== LOOK COMMAND PATH ===")
		lookMap := gs.GetLayeredMapByCurrentLocation()
		lookTile := lookMap.GetTopTile(targetPos)
		t.Logf("Look: map=%p, tile=%v", lookMap, lookTile)

		t.Logf("=== ATTACK COMMAND PATH ===")
		attackMap := gs.GetLayeredMapByCurrentLocation()
		attackTile := attackMap.GetTopTile(targetPos)
		t.Logf("Attack: map=%p, tile=%v", attackMap, attackTile)

		// They should be identical
		assert.Equal(t, lookMap, attackMap, "Maps should be same instance")
		assert.Equal(t, lookTile, attackTile, "Tiles should be identical")
	})
}
