package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// newReagentItemReferences owns the names used by inventory and reagent shops.
// Descriptions are from The Book of Lore (Ultima V).
func newReagentItemReferences() map[Reagent]InventoryItem {
	definitions := map[Reagent]struct{ name, description string }{
		SulfurAsh:    {"Sulfurous Ash", "Common material left by volcanic action, ash is an excellent source of energy in magical mixtures."},
		Ginseng:      {"Ginseng", "Ancient reagent used extensively in healing."},
		Garlic:       {"Garlic", "Ubiquitous and strong-scented reagent, used effectively in warding off evil spirits."},
		SpiderSilk:   {"Spider Silk", "The magical product of the garden spider and its relatives has no peer in its ability to bind. As a reagent mixed for magic, spider silk magnifies its binding power many times over."},
		BloodMoss:    {"Blood Moss", "Fungal growth usually found in deep forests and warm, damp areas; especially favored as a reagent for its ability to enhance mobility."},
		BlackPearl:   {"Black Pearl", "A rare version of the standard white pearl, a black pearl is a forceful reagent in the creation of kinesthetic magic, that is, projecting objects."},
		NightShade:   {"Nightshade", "Rare, poisonous plant that appears only when the moons are in a certain conjunction. Those who learn its whereabouts and manage to be there in the dead of night when the moons are full can pick nightshade without danger and benefit from its powerful ability as a reagent to create illusions."},
		MandrakeRoot: {"Mandrake Root", "Very rare and usually expensive magical herb growing only in swampy areas, mandrake root is said to bring great power to magic that uses it."},
	}
	items := make(map[Reagent]InventoryItem, len(definitions))
	for reagent, definition := range definitions {
		items[reagent] = InventoryItem{ItemName: definition.name, ItemSprite: indexes.Water1,
			ItemDescription: definition.description} // Retain the current placeholder sprite.
	}

	return items
}
