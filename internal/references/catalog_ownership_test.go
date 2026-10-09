package references

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTileDefinitionsAreCompleteAndDetached(t *testing.T) {
	first, second := NewTileReferences(), NewTileReferences()
	require.Len(t, *first, 530)
	for index := indexes.SpriteIndex(-1); index <= 528; index++ {
		tile := first.GetTile(index)
		require.NotNil(t, tile, "tile %d", index)
		assert.Equal(t, index, tile.Index)
		assert.NotEmpty(t, tile.Name, "tile %d", index)
	}
	first.GetTile(indexes.RightSconce).LightEmission = 0
	assert.Positive(t, second.GetTile(indexes.RightSconce).LightEmission,
		"mutable map light sources must not corrupt shared reference definitions")
}

func TestTileDefinitionsPreserveMovementFeedbackAndVisibility(t *testing.T) {
	tiles := NewTileReferences()
	assert.Equal(t, "Slow Progress!", tiles.GetTile(indexes.Forest1).GetExtraMovementString())
	assert.Equal(t, "Very Slow!", tiles.GetTile(indexes.Forest4).GetExtraMovementString())
	assert.Empty(t, tiles.GetTile(indexes.Grass).GetExtraMovementString())
	assert.True(t, tiles.GetTile(indexes.StoneBrickWall).BlocksLight)
	assert.True(t, tiles.GetTile(indexes.StoneCrossWindow).IsWindow())
	assert.False(t, tiles.GetTile(indexes.Grass).BlocksLight)
}

func TestItemDefinitionsCoverRuntimeIdentities(t *testing.T) {
	refs := NewInventoryItemsReferences()
	require.Len(t, refs.Equipment, 49) // Equipment 0..47 and the real BareHands identity.
	for equipment := LeatherHelm; equipment <= Ankh; equipment++ {
		item, exists := refs.Equipment[equipment]
		require.True(t, exists, "equipment %d", equipment)
		assert.Equal(t, equipment.String(), item.ItemName)
		assert.Positive(t, item.ItemSprite)
	}
	assert.Equal(t, "Bare Hands", refs.Equipment[BareHands].ItemName)
	require.Len(t, refs.Spell, TotalSpellCount)
	for spell := Spell(0); spell < Spell(TotalSpellCount); spell++ {
		assert.Equal(t, indexes.SpriteIndex(indexes.ItemScroll), refs.Spell[spell].ItemSprite)
	}
	check := func(items []Item, count int) {
		t.Helper()
		require.Len(t, items, count)
		for _, identity := range items {
			item := refs.GetReferenceByItem(identity)
			assert.NotEmpty(t, item.ItemName, "%v", identity)
			assert.NotEmpty(t, item.ItemDescription, "%v", identity)
			assert.Positive(t, item.ItemSprite, "%v", identity)
		}
	}
	var reagents, scrolls, potions, quests, moonstones, provisions []Item
	for reagent := SulfurAsh; reagent <= MandrakeRoot; reagent++ {
		reagents = append(reagents, reagent)
	}
	for _, scroll := range ScrollCatalog() {
		scrolls = append(scrolls, scroll)
	}
	for _, potion := range PotionCatalog() {
		potions = append(potions, potion)
	}
	for _, quest := range QuestItemCatalog() {
		quests = append(quests, quest)
		assert.NotEmpty(t, refs.GetQuestItemDisplayName(quest))
		assert.NotEmpty(t, refs.GetQuestItemPickupMessage(quest))
	}
	for _, moonstone := range MoonstoneCatalog() {
		moonstones = append(moonstones, moonstone)
	}
	for provision := Torches; provision <= Gold; provision++ {
		provisions = append(provisions, provision)
	}
	check(reagents, 8)
	check(scrolls, 8)
	check(potions, 8)
	check(quests, 14)
	check(moonstones, 8)
	check(provisions, 6)
	require.Len(t, refs.Reagent, 8)
	require.Len(t, refs.Scroll, 8)
	require.Len(t, refs.Potion, 8)
	require.Len(t, refs.QuestItem, 14)
	require.Len(t, refs.Moonstone, 8)
	require.Len(t, refs.Provision, 6)
}

func TestScrollDefinitionsRetainDurations(t *testing.T) {
	refs := NewInventoryItemsReferences()
	for scroll, duration := range map[Scroll]int{
		ScrollVasLor: 240, ScrollInSanct: 100, ScrollInAn: 20, ScrollAnTym: 20,
		ScrollRelHur: 0, ScrollInQuasWis: 0, ScrollKalXenCorp: 0, ScrollInManiCorp: 0,
	} {
		assert.Equal(t, duration, refs.GetScrollEffectDuration(scroll), "scroll %d", scroll)
	}
}

func TestItemDefinitionsReturnDetachedMaps(t *testing.T) {
	first, second := NewInventoryItemsReferences(), NewInventoryItemsReferences()
	delete(first.QuestItem, Amulet)
	delete(first.Scroll, ScrollVasLor)
	delete(first.Equipment, LeatherHelm)
	assert.Equal(t, "Amulet of Lord British", second.GetQuestItemDisplayName(Amulet))
	assert.Equal(t, 240, second.GetScrollEffectDuration(ScrollVasLor))
	assert.Equal(t, LeatherHelm.String(), second.Equipment[LeatherHelm].ItemName)
}
