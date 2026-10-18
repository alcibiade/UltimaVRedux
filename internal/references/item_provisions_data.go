package references

// newProvisionItemReferences owns player-facing provisions metadata.
// Torch and gem commitment: ref:PROV-0003 and ref:PROV-0005.
// Keys, food, and gold: ref:PROV-0004, ref:PROV-0001, and ref:INV-0004.
//
//nolint:exhaustruct // Only Skull Keys expose attribution in the Use dialog.
func newProvisionItemReferences() map[Provision]InventoryItem {
	items := map[Provision]InventoryItem{
		Torches:   {ItemName: "Torches", ItemDescription: "Ignite a torch to light dark areas. One torch is consumed; light lasts 240 game minutes outside dungeons, while dungeon light gains 112-127 minutes up to a 255-minute cap."},
		Gem:       {ItemName: "Gems", ItemDescription: "View through a gem to reveal a 32x32 local area or a symbolic view of the reachable area on the current dungeon level. Each View consumes one gem and is unavailable in combat."},
		Key:       {ItemName: "Keys", ItemDescription: "Use Jimmy with a regular key on ordinary locked doors, chests, stocks, or manacles. A successful physical unlock keeps the key; failure breaks one. Magical locks always break it."},
		SkullKeys: {ItemName: "SkullKeys", ItemDisplayName: "Skull Keys", ItemDescription: "Use a Skull Key to invoke In Ex Por and open magical doors on surface, town, or combat maps. One is consumed before targeting and is not refunded if ineffective.", ItemDescriptionAttribution: "Ultima V behavior: ref:PROV-0006"},
		Food:      {ItemName: "Food", ItemDescription: "Shared rations eaten at 06:00, 12:00, and 18:00 by each living, awake party member. When none remains at an hourly update, every living member suffers starvation damage."},
		Gold:      {ItemName: "Gold", ItemDescription: "Shared coin used for goods, lodging, healing, ships, offerings, tolls, and other payments. Gold is spent when a transaction or payment is accepted."},
	}
	for provision, item := range items {
		item.ItemSprite = provision.GetSpriteIndex()
		items[provision] = item
	}

	return items
}
