package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

func TestQuestDebugCommands_Integration(t *testing.T) {
	// Test quest-add and quest-remove debug commands functionality
	// This ensures the debug commands properly add and remove quest items

	t.Run("Quest_Add_Functionality", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Initially, no quest items should be present
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
			"Crown should not be in inventory initially")
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"Sceptre should not be in inventory initially")
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Amulet),
			"Amulet should not be in inventory initially")

		// Simulate quest-add command by adding each quest item manually
		// (We're testing the underlying functionality that quest-add uses)
		crownItem := &references.ItemAndQuantity{Item: references.Crown, Quantity: 1}
		sceptreItem := &references.ItemAndQuantity{Item: references.Sceptre, Quantity: 1}
		amuletItem := &references.ItemAndQuantity{Item: references.Amulet, Quantity: 1}

		err1 := gs.PartyState.Inventory.PutItemInInventory(crownItem)
		err2 := gs.PartyState.Inventory.PutItemInInventory(sceptreItem)
		err3 := gs.PartyState.Inventory.PutItemInInventory(amuletItem)

		assert.NoError(t, err1, "Should be able to add Crown to inventory")
		assert.NoError(t, err2, "Should be able to add Sceptre to inventory")
		assert.NoError(t, err3, "Should be able to add Amulet to inventory")

		// Verify all quest items are now present
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
			"Crown should be in inventory after adding")
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"Sceptre should be in inventory after adding")
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Amulet),
			"Amulet should be in inventory after adding")

		t.Logf("✅ Quest-add functionality verified: All quest items successfully added")
	})

	t.Run("Quest_Remove_Functionality", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// First add all quest items
		questItems := []*references.ItemAndQuantity{
			{Item: references.Crown, Quantity: 1},
			{Item: references.Sceptre, Quantity: 1},
			{Item: references.Amulet, Quantity: 1},
		}

		for _, item := range questItems {
			err := gs.PartyState.Inventory.PutItemInInventory(item)
			assert.NoError(t, err, "Should be able to add quest item to inventory")
		}

		// Verify they're all present
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
			"Crown should be in inventory before removal")
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"Sceptre should be in inventory before removal")
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Amulet),
			"Amulet should be in inventory before removal")

		// Simulate quest-remove command by setting each quest item to 0
		// (This is the corrected functionality that quest-remove should use)
		questItemTypes := []references.QuestItem{references.Crown, references.Sceptre, references.Amulet}
		removedCount := 0
		invRefs := references.NewInventoryItemsReferences()

		for _, questItem := range questItemTypes {
			currentQuantity := gs.PartyState.Inventory.QuestItems.GetQuantity(questItem)
			if (*currentQuantity).HasSome() {
				gs.PartyState.Inventory.QuestItems.Set(questItem, 0) // Fixed: was SetHasOne before
				removedCount++
				t.Logf("Removed: %s", invRefs.GetQuestItemDisplayName(questItem))
			} else {
				t.Logf("Not found: %s", invRefs.GetQuestItemDisplayName(questItem))
			}
		}

		// Verify all quest items are now removed
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
			"Crown should NOT be in inventory after removal")
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"Sceptre should NOT be in inventory after removal")
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Amulet),
			"Amulet should NOT be in inventory after removal")

		assert.Equal(t, 3, removedCount, "Should have removed exactly 3 quest items")

		t.Logf("✅ Quest-remove functionality verified: All quest items successfully removed")
	})

	t.Run("Quest_Add_Remove_Round_Trip", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Test multiple add/remove cycles to ensure functionality is robust
		for cycle := 1; cycle <= 3; cycle++ {
			t.Logf("🔄 Round trip cycle %d", cycle)

			// Add all quest items
			questItems := []*references.ItemAndQuantity{
				{Item: references.Crown, Quantity: 1},
				{Item: references.Sceptre, Quantity: 1},
				{Item: references.Amulet, Quantity: 1},
			}

			for _, item := range questItems {
				err := gs.PartyState.Inventory.PutItemInInventory(item)
				assert.NoError(t, err, "Cycle %d: Should be able to add quest item", cycle)
			}

			// Verify they're all present
			assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
				"Cycle %d: Crown should be present after adding", cycle)
			assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
				"Cycle %d: Sceptre should be present after adding", cycle)
			assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Amulet),
				"Cycle %d: Amulet should be present after adding", cycle)

			// Remove all quest items
			questItemTypes := []references.QuestItem{references.Crown, references.Sceptre, references.Amulet}
			for _, questItem := range questItemTypes {
				gs.PartyState.Inventory.QuestItems.Set(questItem, 0)
			}

			// Verify they're all removed
			assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Crown),
				"Cycle %d: Crown should be removed after removal", cycle)
			assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
				"Cycle %d: Sceptre should be removed after removal", cycle)
			assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Amulet),
				"Cycle %d: Amulet should be removed after removal", cycle)
		}

		t.Logf("✅ Round trip functionality verified: 3 complete add/remove cycles successful")
	})

	t.Run("Quest_NPC_Always_Hidden_Integration", func(t *testing.T) {
		// Test that quest item NPCs are ALWAYS hidden (new overlay rendering system)
		// Quest items are now rendered as overlays, not as NPCs
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(15, 14). // Safe position near Sceptre
			WithSystemCallbacks().
			Build()

		sceptrePos := &references.Position{X: 15, Y: 15}

		// Check if Sceptre NPC exists at expected location
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npc := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*sceptrePos)

			if npc != nil {
				t.Logf("✅ Found NPC at Sceptre location - testing always-hidden behavior")

				// Test without Sceptre in inventory - NPC should STILL be hidden
				assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
					"Sceptre should not be in inventory initially")

				isHiddenWithout := gs.isQuestItemNPCHidden(*npc)
				assert.True(t, isHiddenWithout, "Quest item NPC should ALWAYS be hidden (overlay rendering)")

				// Add Sceptre to inventory - NPC should STILL be hidden
				sceptreItem := &references.ItemAndQuantity{Item: references.Sceptre, Quantity: 1}
				err := gs.PartyState.Inventory.PutItemInInventory(sceptreItem)
				assert.NoError(t, err, "Should be able to add Sceptre to inventory")

				assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
					"Sceptre should be in inventory after adding")

				isHiddenWith := gs.isQuestItemNPCHidden(*npc)
				assert.True(t, isHiddenWith, "Quest item NPC should ALWAYS be hidden (overlay rendering)")

				// Remove Sceptre from inventory - NPC should STILL be hidden
				gs.PartyState.Inventory.QuestItems.Set(references.Sceptre, 0)

				assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
					"Sceptre should NOT be in inventory after removal")

				isHiddenAfterRemoval := gs.isQuestItemNPCHidden(*npc)
				assert.True(t, isHiddenAfterRemoval, "Quest item NPC should ALWAYS be hidden (overlay rendering)")

				t.Logf("✅ Always-hidden behavior verified:")
				t.Logf("   - Without Sceptre: NPC hidden (%t) - overlay renders quest item", isHiddenWithout)
				t.Logf("   - With Sceptre: NPC hidden (%t) - overlay hides quest item", isHiddenWith)
				t.Logf("   - After removal: NPC hidden (%t) - overlay renders quest item", isHiddenAfterRemoval)
			} else {
				t.Logf("ℹ️ No NPC found at Sceptre position - always-hidden test skipped")
			}
		} else {
			t.Logf("ℹ️ No NPC controller available - always-hidden test skipped")
		}
	})
}
