package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestUseProjectionCarpetAvailableInUnderworld(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithFloor(-1).WithPlayerAt(100, 100).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer,
		&state.MapState.PlayerLocation.Position, indexes.Grass)
	for _, item := range state.GetUsableItems() {
		if item.ItemType == references.Carpet {
			require.True(t, item.AvailableNow)
			assert.Contains(t, item.Requirement, "Underworld")
			assert.Empty(t, item.Warning)
			return
		}
	}
	t.Fatal("inventory carpet missing from Use menu")
}

func TestUseProjectionApprovedGuidance(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	for _, shard := range []references.QuestItem{
		references.ShardOfFalsehood, references.ShardOfHatred, references.ShardOfCowardice,
	} {
		info := state.withUseItemGuidance(UseableItemInfo{ItemType: shard}, UsageField)
		assert.Equal(t, "Consumed on successful destruction.", info.Consumption)
		assert.Equal(t, "Begin the ritual.", info.NextInteraction)
		assert.Equal(t, "The ritual requires the right conditions.", info.Requirement)
		assert.Equal(t, "Destruction is permanent. The shard is retained if there is no effect.", info.Warning)
		info = state.withUseItemGuidance(info, UsageDungeon)
		assert.False(t, info.AvailableNow)
		assert.Equal(t, "No effect in a dungeon; the shard is retained.", info.Warning)
	}
	for _, test := range []struct {
		item references.QuestItem
		want string
	}{
		{references.Amulet, "Wear or remove the Amulet."},
		{references.BlackBadge, "Wear or remove the Black Badge."},
	} {
		info := state.withUseItemGuidance(UseableItemInfo{ItemType: test.item}, UsageField)
		assert.Equal(t, test.want, info.EffectSummary)
	}
	info := state.withUseItemGuidance(UseableItemInfo{ItemType: references.Sceptre}, UsageField)
	assert.Equal(t, "A magical field must be within reach.", info.Requirement)
	state.addMoonstoneUseGuidance(&info, UsageField)
	assert.Equal(t, "The current terrain cannot hold a buried moonstone.", info.UnavailableReason)
	assert.Equal(t, "Unavailable: inspect this restriction or choose another item.", info.NextInteraction)
}

func TestUseOwnershipAndCategoryMatrixAcrossContexts(t *testing.T) {
	for _, test := range []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
		context  UsageContext
	}{
		{"town", references.Britain, 0, UsageField},
		{"overworld", references.BritanniaUnderworld, 0, UsageField},
		{"underworld", references.BritanniaUnderworld, -1, UsageField},
		{"dungeon", references.Deceit, 0, UsageDungeon},
		{"combat", references.Britain, 0, UsageCombat},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(test.location).
				WithFloor(test.floor).WithSystemCallbacks().Build()
			if test.context == UsageCombat {
				require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
			}
			expected := make(map[references.Item]int)
			for item := references.Amulet; item <= references.Sextant; item++ {
				quantity := uint16(item) + 1
				state.PartyState.Inventory.QuestItems.Set(item, quantity)
				expected[item] = int(quantity)
			}
			for item := references.Blue; item <= references.White; item++ {
				quantity := uint16(item) + 2
				state.PartyState.Inventory.Potions.Set(item, quantity)
				expected[item] = int(quantity)
			}
			for item := references.ScrollVasLor; item <= references.ScrollAnTym; item++ {
				quantity := uint16(item) + 3
				state.PartyState.Inventory.Scrolls.Set(item, quantity)
				expected[item] = int(quantity)
			}
			for item := references.NewMoon; item <= references.CrescentWaning; item++ {
				quantity := uint16(item) + 4
				state.PartyState.Inventory.Moonstones.Set(item, quantity)
				expected[item] = int(quantity)
			}
			state.PartyState.Inventory.Provisions.SkullKeys.Set(9)
			expected[references.SkullKeys] = 9
			inventory := state.PartyState.Inventory.LockerSnapshot()
			rngCalls := state.GetRNG().Calls()
			sequence := state.CastUseCommitSequence()
			items := state.GetUsableItemsForContext(test.context)
			require.Len(t, items, len(expected))
			seen := make(map[references.Item]int)
			categories := make(map[UseItemCategory]int)
			for _, item := range items {
				require.Contains(t, expected, item.ItemType)
				require.NotContains(t, seen, item.ItemType, "each owned identity appears once")
				seen[item.ItemType] = item.Quantity
				categories[item.Category]++
				require.NotEmpty(t, item.Consumption)
				if item.UnavailableReason != "" {
					require.False(t, item.AvailableNow)
				}
			}
			require.Equal(t, expected, seen)
			require.Equal(t, map[UseItemCategory]int{
				UseItemCategoryQuest: 11, UseItemCategoryTools: 3, UseItemCategoryPotion: 8,
				UseItemCategoryScroll: 8, UseItemCategoryMoonstone: 8, UseItemCategoryKeys: 1,
			}, categories)
			require.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
			require.Equal(t, rngCalls, state.GetRNG().Calls())
			require.Equal(t, sequence, state.CastUseCommitSequence())
		})
	}
}

func TestUseWarningsDoNotBecomeActivationRestrictions(t *testing.T) {
	for _, context := range []UsageContext{UsageField, UsageDungeon, UsageCombat} {
		location := references.Britain
		if context == UsageDungeon {
			location = references.Deceit
		}
		state, _ := NewIntegrationTestBuilder(t).WithLocation(location).WithSystemCallbacks().Build()
		if context == UsageCombat {
			require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
		}
		for _, potion := range []references.Potion{references.Purple, references.Black, references.White} {
			state.PartyState.Inventory.Potions.Set(potion, 2)
		}
		items := state.GetUsableItemsForContext(context)
		for _, potion := range []references.Potion{references.Purple, references.Black, references.White} {
			item := requireUsableItem(t, items, potion)
			require.Empty(t, item.UnavailableReason, "potion substitution remains possible in every context")
			require.Contains(t, item.Consumption, "consumed")
			ineffective := (potion == references.White && context != UsageField) ||
				(potion != references.White && context != UsageCombat)
			require.Equal(t, !ineffective, item.AvailableNow)
			if ineffective {
				require.Contains(t, item.Warning, "still consumed")
			}
		}
	}
}
