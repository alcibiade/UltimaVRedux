package references

// newPotionItemReferences owns player-facing potion metadata.
func newPotionItemReferences() map[Potion]InventoryItem {
	items := map[Potion]InventoryItem{
		Blue:   {ItemName: "Blue", ItemSprite: 259, ItemDescription: "Usually awakens one sleeping party member. Potion labels are unreliable; it may instead cause sleep or another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		Yellow: {ItemName: "Yellow", ItemSprite: 259, ItemDescription: "Usually restores 1-30 health to one living party member, up to their maximum. Potion labels are unreliable; it may instead cause sleep or another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		Red:    {ItemName: "Red", ItemSprite: 259, ItemDescription: "Usually cures one poisoned party member, but restores no lost health. Potion labels are unreliable; it may instead cause sleep or another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		Green:  {ItemName: "Green", ItemSprite: 259, ItemDescription: "Usually poisons one healthy party member. Potion labels are unreliable; it may instead cause sleep or another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		Orange: {ItemName: "Orange", ItemSprite: 259, ItemDescription: "Usually puts one healthy party member to sleep. Potion labels are unreliable; it may instead produce another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		Purple: {ItemName: "Purple", ItemSprite: 259, ItemDescription: "Usually makes the active party member look like a rat for the rest of the battle. It has no noticeable effect outside combat. Potion labels are unreliable; it may instead cause sleep or another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		Black:  {ItemName: "Black", ItemSprite: 259, ItemDescription: "Usually makes the active party member invisible for the battle, preventing most enemies from targeting them. It has no noticeable effect outside combat. Potion labels are unreliable; it may instead cause sleep or another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		White:  {ItemName: "White", ItemSprite: 259, ItemDescription: "Usually reveals everything in the immediate view, even through obstacles. It has no noticeable effect in dungeons or combat. Potion labels are unreliable; it may instead cause sleep or another color's effect.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
	}
	for potion, item := range items {
		item.ItemDisplayName = item.ItemName + " Potion"
		items[potion] = item
	}

	return items
}
