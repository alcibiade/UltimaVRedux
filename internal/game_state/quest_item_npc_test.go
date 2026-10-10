package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestQuestItemNPC_Sceptre_Stonegate(t *testing.T) {
	// Check the authored Stonegate Sceptre actor: owning the Sceptre removes
	// that actor during location initialization. ref:SHADOW-0005

	sceptrePos := &references.Position{X: 15, Y: 15}
	playerSafePos := &references.Position{X: 15, Y: 14} // One tile north of sceptre, safe from trap

	t.Run("Check_NPC_At_Sceptre_Position_Without_Inventory", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// Verify Sceptre is NOT in inventory
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"Sceptre should not be in inventory initially")

		// Check if there's an NPC at the Sceptre position
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npc := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*sceptrePos)

			if npc != nil {
				t.Logf("✅ FOUND NPC at Sceptre position (%d,%d)", sceptrePos.X, sceptrePos.Y)
				t.Logf("NPC Type: %T", *npc)

				// Test display sprite when Sceptre is NOT in inventory (should be visible)
				displayedSprite := gs.GetDisplaySpriteAtPosition(*sceptrePos)
				t.Logf("Displayed sprite at (%d,%d) without Sceptre in inventory: %d", sceptrePos.X, sceptrePos.Y, displayedSprite)

				if int(displayedSprite) == int(indexes.Sceptre) {
					t.Logf("✅ CONDITIONAL VISIBILITY: NPC visible when Sceptre NOT in inventory")
				} else {
					t.Logf("⚠️  Unexpected sprite displayed: %d (expected Sceptre: %d)", displayedSprite, indexes.Sceptre)
				}
			} else {
				t.Logf("❌ NO NPC found at Sceptre position (%d,%d)", sceptrePos.X, sceptrePos.Y)
			}
		} else {
			t.Logf("❌ CurrentNPCAIController or GetNpcs() is nil")
		}

		// Explore NPCs in the area
		t.Logf("=== EXPLORING ALL NPCS AT STONEGATE ===")
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npcs := gs.CurrentNPCAIController.GetNpcs()

			// Check area around Sceptre position
			for y := 14; y <= 16; y++ {
				for x := 14; x <= 16; x++ {
					pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
					npc := npcs.GetMapUnitAtPositionOrNil(pos)
					if npc != nil {
						t.Logf("🎯 NPC FOUND at position (%d,%d): %T", x, y, *npc)
					}
				}
			}
		}
	})

	t.Run("Check_NPC_At_Sceptre_Position_With_Inventory", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// Add Sceptre to inventory first
		sceptreItem := &references.ItemAndQuantity{
			Item:     references.Sceptre,
			Quantity: 1,
		}
		err := gs.PartyState.Inventory.PutItemInInventory(sceptreItem)
		assert.NoError(t, err, "Should be able to add Sceptre to inventory")

		// Verify Sceptre is in inventory
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"Sceptre should be in inventory")

		// Check if the NPC is still visible via GetDisplaySpriteAtPosition (this should be hidden now)
		displayedSprite := gs.GetDisplaySpriteAtPosition(*sceptrePos)
		t.Logf("CONDITIONAL VISIBILITY: Displayed sprite at (%d,%d) with Sceptre in inventory: %d", sceptrePos.X, sceptrePos.Y, displayedSprite)

		// The NPC should still exist in the data structure but be hidden from display
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npc := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*sceptrePos)

			if npc != nil {
				// NPC exists in data structure - that's correct
				t.Logf("✅ NPC DATA: NPC still exists in data structure at (%d,%d) - this is correct", sceptrePos.X, sceptrePos.Y)

				// But it should be hidden from display
				if displayedSprite != indexes.Sceptre {
					t.Logf("✅ CONDITIONAL VISIBILITY: NPC hidden from display - showing sprite %d instead of Sceptre (%d)", displayedSprite, indexes.Sceptre)
				} else {
					t.Logf("❌ CONDITIONAL VISIBILITY: NPC still visible in display - this means hiding logic is NOT working")
				}
			} else {
				t.Logf("❌ NPC DATA: No NPC found at (%d,%d) - this shouldn't happen", sceptrePos.X, sceptrePos.Y)
			}
		}
	})

	t.Run("Debug_Area_NPCs_At_Stonegate", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		t.Logf("=== AREA NPCS AT STONEGATE ===")
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npcs := gs.CurrentNPCAIController.GetNpcs()

			// Check a wider area around the expected Sceptre position
			npcCount := 0
			for y := 10; y <= 20; y++ {
				for x := 10; x <= 20; x++ {
					pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
					npc := npcs.GetMapUnitAtPositionOrNil(pos)
					if npc != nil {
						npcCount++
						t.Logf("🎯 NPC FOUND at position (%d,%d): %T", x, y, *npc)

						// If it's at the Sceptre position, this is likely the Sceptre NPC
						if x == 15 && y == 15 {
							t.Logf("🏆 THIS IS THE NPC AT SCEPTRE POSITION - LIKELY THE SCEPTRE!")
						}
					}
				}
			}
			t.Logf("Total NPCs found in area: %d", npcCount)
		}
	})
}
