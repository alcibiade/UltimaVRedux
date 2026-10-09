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
	assert.Equal(t, "Bury the moonstone at your current position.", info.NextInteraction)
}
