package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestManualGuardPlacement manually places a guard sprite at ladder position
// to verify that the Look command would indeed show "a guard" if one were there
func TestManualGuardPlacement(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(2, 26). // Adjacent to ladder
		WithSystemCallbacks().
		Build()
	t.Run("VerifyGuardSpriteWouldShowAsGuard", func(t *testing.T) {
		gs.DateTime.Hour = byte(15)
		gs.DateTime.Minute = byte(42)

		ladderPos := references.Position{X: 1, Y: 26}

		// STEP 1: Confirm ladder shows as ladder normally
		t.Logf("=== STEP 1: Normal ladder test ===")
		mockCallbacks.Reset()
		gs.ActionLookSmallMap(references.Left) // Look west at ladder

		if len(mockCallbacks.Messages) > 0 {
			t.Logf("Normal ladder message: '%s'", mockCallbacks.Messages[0])
		}

		// STEP 2: Manually place guard sprite in MapUnitLayer at ladder position
		t.Logf("=== STEP 2: Manually placing guard sprite ===")
		currentMap := gs.GetLayeredMapByCurrentLocation()
		currentMap.SetTileByLayer(map_state.MapUnitLayer, &ladderPos, indexes.Guard1) // Sprite 368

		// Verify it was set
		mapUnitTile := currentMap.GetTileByLayer(map_state.MapUnitLayer, &ladderPos)
		if mapUnitTile != nil {
			t.Logf("MapUnitLayer now has: %s (index=%d)", mapUnitTile.Name, mapUnitTile.Index)
		}

		// Check what GetDisplaySpriteAtPosition returns
		displaySprite := gs.GetDisplaySpriteAtPosition(ladderPos)
		t.Logf("GetDisplaySpriteAtPosition returns: %d", displaySprite)

		// Check what the look description would be
		lookDesc := gs.GameReferences.LookReferences.GetTileLookDescription(displaySprite)
		t.Logf("Look description for sprite %d: '%s'", displaySprite, lookDesc)

		// STEP 3: Test Look command with guard sprite present
		t.Logf("=== STEP 3: Look command with guard sprite ===")
		mockCallbacks.Reset()
		gs.ActionLookSmallMap(references.Left) // Look west at ladder

		if len(mockCallbacks.Messages) > 0 {
			message := mockCallbacks.Messages[0]
			t.Logf("Look message with guard sprite: '%s'", message)

			if message == "Thou dost see a guard" {
				t.Logf("✅ CONFIRMED: Guard sprite at ladder DOES show 'a guard'")
			} else {
				t.Logf("❌ Unexpected: Guard sprite shows '%s' instead of 'a guard'", message)
			}
		}

		// STEP 4: Clean up and verify cleanup works
		t.Logf("=== STEP 4: Cleanup test ===")
		currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &ladderPos)

		mockCallbacks.Reset()
		gs.ActionLookSmallMap(references.Left) // Look west at ladder

		if len(mockCallbacks.Messages) > 0 {
			t.Logf("After cleanup message: '%s'", mockCallbacks.Messages[0])
		}
	})

	// Test different guard sprites
	guardSprites := []indexes.SpriteIndex{
		indexes.Guard1, // 368
		indexes.Guard2, // 369
		indexes.Guard3, // 370
		indexes.Guard4, // 371
	}

	for i, guardSprite := range guardSprites {
		t.Run(t.Name()+"_Guard"+string(rune('1'+i)), func(t *testing.T) {
			ladderPos := references.Position{X: 1, Y: 26}
			currentMap := gs.GetLayeredMapByCurrentLocation()

			// Place guard sprite
			currentMap.SetTileByLayer(map_state.MapUnitLayer, &ladderPos, guardSprite)

			// Test look
			mockCallbacks.Reset()
			gs.ActionLookSmallMap(references.Left)

			if len(mockCallbacks.Messages) > 0 {
				t.Logf("Guard sprite %d message: '%s'", guardSprite, mockCallbacks.Messages[0])
			}

			// Cleanup
			currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &ladderPos)
		})
	}
}
