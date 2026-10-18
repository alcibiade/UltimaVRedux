package references

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInventoryFullDisplayNamesCoverIdentifiedCatalog(t *testing.T) {
	items := NewInventoryItemsReferences()
	// Approved identities in stable family order, independent of raw ItemName.
	equipmentNames := []string{
		"Leather Helm", "Chain Coif", "Iron Helm", "Spiked Helm", "Small Shield", "Large Shield",
		"Spiked Shield", "Magic Shield", "Jewel Shield", "Cloth Armour", "Leather Armour", "Ring Mail",
		"Scale Mail", "Chain Mail", "Plate Mail", "Mystic Armour", "Dagger", "Sling", "Club", "Flaming Oil",
		"Main Gauche", "Spear", "Throwing Axe", "Short Sword", "Mace", "Morning Star", "Bow", "Arrows",
		"Crossbow", "Quarrels", "Long Sword", "Two-Handed Hammer", "Two-Handed Axe", "Two-Handed Sword",
		"Halberd", "Sword of Chaos", "Magic Bow", "Silver Sword", "Magic Axe", "Glass Sword", "Jeweled Sword",
		"Mystic Sword", "Ring of Invisibility", "Ring of Protection", "Ring of Regeneration", "Amulet of Turning",
		"Spiked Collar", "Ankh",
	}
	want := make(map[Item]string)
	for index, name := range equipmentNames {
		equipment := Equipment(index)
		want[equipment] = name
		require.Equal(t, name, equipment.String())
		require.Empty(t, items.Equipment[equipment].ItemDisplayName, "equipment uses the full ItemName fallback")
		require.Equal(t, name, items.Equipment[equipment].ItemName)
	}
	for index, name := range []string{
		"Amulet of Lord British", "Crown of Lord British", "Sceptre of Lord British", "Shard of Falsehood",
		"Shard of Hatred", "Shard of Cowardice", "Magic Carpet", "Grapple", "Spyglass", "H.M.S. Cape Plans",
		"Pocket Watch", "Black Badge", "Sandalwood Box", "Sextant",
	} {
		want[QuestItem(index)] = name
	}
	for index, name := range []string{
		"Sulfurous Ash", "Ginseng", "Garlic", "Spider Silk", "Blood Moss", "Black Pearl", "Nightshade", "Mandrake Root",
	} {
		want[Reagent(index)] = name
	}
	for index, name := range []string{"Blue", "Yellow", "Red", "Green", "Orange", "Purple", "Black", "White"} {
		want[Potion(index)] = name + " Potion"
	}
	for index, name := range []string{"Vas Lor", "Rel Hur", "In Sanct", "In An", "In Quas Wis", "Kal Xen Corp", "In Mani Corp", "An Tym"} {
		want[Scroll(index)] = name + " Scroll"
	}
	for index, name := range []string{
		"New Moon", "Crescent Waxing", "First Quarter", "Gibbous Waxing", "Full Moon", "Gibbous Waning", "Last Quarter", "Crescent Waning",
	} {
		want[Moonstone(index)] = name + " Moonstone"
	}
	for item, name := range map[Provision]string{Food: "Food", Gold: "Gold", Key: "Keys", Gem: "Gems", Torches: "Torches", SkullKeys: "Skull Keys"} {
		want[item] = name
	}
	require.Len(t, want, 100)
	for item, name := range want {
		require.Equal(t, name, items.GetReferenceByItem(item).FullDisplayName(), "%T/%d", item, item.ID())
	}
	require.Equal(t, "Bare Hands", items.Equipment[BareHands].FullDisplayName())
	require.Equal(t, "WoodenBox", items.QuestItem[WoodenBox].ItemName)
	require.Equal(t, "HMSCape", items.QuestItem[HMSCape].ItemName)
	require.Equal(t, "Vas_Lor", items.Scroll[ScrollVasLor].ItemName)
	require.Equal(t, "Blue", items.Potion[Blue].ItemName)
	require.Equal(t, "CrescentWaxing", items.Moonstone[CrescentWaxing].ItemName)
}

func TestInventoryFullDisplayNameOverrideAndChestList(t *testing.T) {
	items := NewInventoryItemsReferences()
	carpet := items.QuestItem[Carpet]
	carpet.ItemDisplayName = "Identified Carpet"
	require.Equal(t, "Identified Carpet", carpet.FullDisplayName())
	require.Equal(t, "Magic Carpet", items.QuestItem[Carpet].FullDisplayName(), "metadata remains detached")
	stack := ItemStack{Items: ItemStacks{
		{Item: Carpet, Quantity: 2}, {Item: WoodenBox, Quantity: 1}, {Item: ScrollVasLor, Quantity: 3},
		{Item: Blue, Quantity: 4}, {Item: ChaosSword, Quantity: 1}, {Item: HMSCape, Quantity: 1},
	}}
	require.Equal(t, strings.Join([]string{
		"Magic Carpet", "Sandalwood Box", "Vas Lor Scroll", "Blue Potion", "Sword of Chaos", "H.M.S. Cape Plans",
	}, "\n"), items.GetListOfItems(&stack))
	require.Equal(t, uint16(3), stack.Items[2].Quantity)
	require.Equal(t, "A sandalwood box!", items.GetQuestItemPickupMessage(WoodenBox))
	require.Equal(t, "A Grappling Hook!", items.GetQuestItemPickupMessage(Grapple))
}

func TestSearchIdentifiedNamesDoNotReplaceMysteryDiscovery(t *testing.T) {
	items := NewInventoryItemsReferences()
	for _, location := range GetAllSearchLocations() {
		decoded := DecodeSearchLocationItem(location)
		if decoded.Item == nil {
			continue
		}
		require.Equal(t, items.GetReferenceByItem(decoded.Item).FullDisplayName(), decoded.ItemName)
		switch decoded.Item.(type) {
		case Potion:
			require.Equal(t, "a potion!", decoded.DiscoveryText)
		case Scroll:
			require.Equal(t, "a scroll!", decoded.DiscoveryText)
		}
		if decoded.Item == HMSCape {
			require.Equal(t, "a scroll!", decoded.DiscoveryText, "finding the plans does not reveal them earlier")
		}
	}
}
