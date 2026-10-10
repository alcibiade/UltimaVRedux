package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestQuestItemNPC_AlwaysHidden(t *testing.T) {
	// Test that quest item NPCs are ALWAYS hidden (new overlay rendering system)
	// Quest items are now rendered as overlays, not as NPCs

	sceptrePos := &references.Position{X: 15, Y: 15}
	playerSafePos := &references.Position{X: 15, Y: 14} // One tile north of sceptre, safe from trap

	t.Run("NPC_Always_Hidden_Without_Sceptre_In_Inventory", func(t *testing.T) {
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

				// Test the always-hidden logic directly
				isHidden := gs.isQuestItemNPCHidden(*npc)
				t.Logf("isQuestItemNPCHidden result: %t", isHidden)

				assert.True(t, isHidden, "Quest item NPC should ALWAYS be hidden (overlay rendering)")

				// Check if the NPC would be blocked by ObjectPresentAt (should be false - NPC is hidden)
				objectPresent := gs.ObjectPresentAt(sceptrePos)
				t.Logf("ObjectPresentAt(%d,%d): %t", sceptrePos.X, sceptrePos.Y, objectPresent)
				assert.False(t, objectPresent, "Object should NOT be present when quest item NPC is hidden")
			} else {
				t.Logf("❌ NO NPC found at Sceptre position (%d,%d)", sceptrePos.X, sceptrePos.Y)
			}
		} else {
			t.Logf("❌ CurrentNPCAIController or GetNpcs() is nil")
		}
	})

	t.Run("NPC_Always_Hidden_With_Sceptre_In_Inventory", func(t *testing.T) {
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

		// The NPC should still exist in the data structure but be hidden from logic
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npc := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*sceptrePos)

			if npc != nil {
				// NPC exists in data structure - that's correct
				t.Logf("✅ NPC DATA: NPC still exists in data structure at (%d,%d) - this is correct", sceptrePos.X, sceptrePos.Y)

				// Test the always-hidden logic directly
				isHidden := gs.isQuestItemNPCHidden(*npc)
				t.Logf("isQuestItemNPCHidden result: %t", isHidden)

				assert.True(t, isHidden, "Quest item NPC should ALWAYS be hidden (overlay rendering)")

				// Check if the NPC would be blocked by ObjectPresentAt (should be false - NPC is hidden)
				objectPresent := gs.ObjectPresentAt(sceptrePos)
				t.Logf("ObjectPresentAt(%d,%d): %t", sceptrePos.X, sceptrePos.Y, objectPresent)
				assert.False(t, objectPresent, "Object should NOT be present when quest item NPC is hidden")
			} else {
				t.Logf("❌ NPC DATA: No NPC found at (%d,%d) - this shouldn't happen", sceptrePos.X, sceptrePos.Y)
			}
		}
	})

	t.Run("NPC_Sprite_Identification", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// Check if there's an NPC at the Sceptre position and verify its sprite
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npc := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*sceptrePos)

			if npc != nil {
				// Check the sprite index to confirm this is the Sceptre NPC
				if friendly, ok := (*npc).(*map_units.NPCFriendly); ok {
					spriteIndex := friendly.GetKeySpriteIndex()
					t.Logf("NPC sprite index: %d", spriteIndex)
					t.Logf("Sceptre sprite index: %d", indexes.Sceptre)

					assert.Equal(t, int(indexes.Sceptre), int(spriteIndex),
						"NPC sprite should match Sceptre sprite index")
				} else {
					t.Logf("NPC is not NPCFriendly type: %T", *npc)
				}
			}
		}
	})
}

func TestCrownNPC_AlwaysHidden(t *testing.T) {
	// Test that Crown NPCs are ALWAYS hidden (new overlay rendering system)
	// Location: x=15, y=13, floor=3 in Palace of Blackthorne

	crownPos := &references.Position{X: 15, Y: 13}
	playerSafePos := &references.Position{X: 15, Y: 12} // One tile north of crown

	t.Run("Crown_Always_Hidden_Without_Crown_In_Inventory", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.PalaceOfBlackthorn).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithFloor(3). // Crown is on floor 3
			WithSystemCallbacks().
			Build()

		// Verify Crown is NOT in inventory
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
			"Crown should not be in inventory initially")

		// Check if there's an NPC at the Crown position
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npc := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*crownPos)

			if npc != nil {
				t.Logf("✅ FOUND NPC at Crown position (%d,%d) floor 3", crownPos.X, crownPos.Y)
				t.Logf("NPC Type: %T", *npc)

				// Check the sprite index to confirm this is the Crown NPC
				if friendly, ok := (*npc).(*map_units.NPCFriendly); ok {
					spriteIndex := friendly.GetKeySpriteIndex()
					t.Logf("NPC sprite index: %d", spriteIndex)
					t.Logf("Crown sprite index: %d", indexes.Crown)

					assert.Equal(t, int(indexes.Crown), int(spriteIndex),
						"NPC sprite should match Crown sprite index")
				} else {
					t.Logf("NPC is not NPCFriendly type: %T", *npc)
				}

				// Test the always-hidden logic directly
				isHidden := gs.isQuestItemNPCHidden(*npc)
				t.Logf("isQuestItemNPCHidden result: %t", isHidden)

				assert.True(t, isHidden, "Crown NPC should ALWAYS be hidden (overlay rendering)")

				// Check if the NPC would be blocked by ObjectPresentAt (should be false - NPC is hidden)
				objectPresent := gs.ObjectPresentAt(crownPos)
				t.Logf("ObjectPresentAt(%d,%d): %t", crownPos.X, crownPos.Y, objectPresent)
				assert.False(t, objectPresent, "Object should NOT be present when Crown NPC is hidden")
			} else {
				t.Logf("❌ NO NPC found at Crown position (%d,%d) floor 3", crownPos.X, crownPos.Y)
				t.Logf("This might mean:")
				t.Logf("  1. Crown is not at this exact position")
				t.Logf("  2. Crown is on a different floor")
				t.Logf("  3. Crown is not loaded in this location")
			}
		} else {
			t.Logf("❌ CurrentNPCAIController or GetNpcs() is nil")
		}
	})

	t.Run("Crown_Always_Hidden_With_Crown_In_Inventory", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.PalaceOfBlackthorn).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithFloor(3). // Crown is on floor 3
			WithSystemCallbacks().
			Build()

		// Add Crown to inventory first
		crownItem := &references.ItemAndQuantity{
			Item:     references.Crown,
			Quantity: 1,
		}
		err := gs.PartyState.Inventory.PutItemInInventory(crownItem)
		assert.NoError(t, err, "Should be able to add Crown to inventory")

		// Verify Crown is in inventory
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
			"Crown should be in inventory")

		// The NPC should still exist in the data structure but be hidden from logic
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npc := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*crownPos)

			if npc != nil {
				// NPC exists in data structure - that's correct
				t.Logf("✅ NPC DATA: Crown NPC still exists in data structure at (%d,%d) floor 3 - this is correct", crownPos.X, crownPos.Y)

				// Test the always-hidden logic directly
				isHidden := gs.isQuestItemNPCHidden(*npc)
				t.Logf("isQuestItemNPCHidden result: %t", isHidden)

				assert.True(t, isHidden, "Crown NPC should ALWAYS be hidden (overlay rendering)")

				// Check if the NPC would be blocked by ObjectPresentAt (should be false - NPC is hidden)
				objectPresent := gs.ObjectPresentAt(crownPos)
				t.Logf("ObjectPresentAt(%d,%d): %t", crownPos.X, crownPos.Y, objectPresent)
				assert.False(t, objectPresent, "Object should NOT be present when Crown NPC is hidden")
			} else {
				t.Logf("❌ Crown NPC DATA: No NPC found at (%d,%d) floor 3", crownPos.X, crownPos.Y)
			}
		}
	})
}
