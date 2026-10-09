package references

// newScrollItemReferences owns player-facing scroll metadata.
//
//nolint:exhaustruct // Only metadata meaningful to this item family is populated.
func newScrollItemReferences() map[Scroll]InventoryItem {
	return map[Scroll]InventoryItem{
		ScrollVasLor:     {ItemName: "Vas_Lor", ItemSprite: 260, ItemDescription: "Creates powerful magical light for 240 in-game minutes.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)", EffectDuration: 240},
		ScrollRelHur:     {ItemName: "Rel_Hur", ItemSprite: 260, ItemDescription: "Outside dungeons and combat, sets the wind north, south, east, west, or calm.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		ScrollInSanct:    {ItemName: "In_Sanct", ItemSprite: 260, ItemDescription: "Grants the whole party +3 armour for a limited time. It can be used before combat, but its duration begins immediately.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)", EffectDuration: 100},
		ScrollInAn:       {ItemName: "In_An", ItemSprite: 260, ItemDescription: "For a limited time, suppresses magic for both sides in combat and prevents enemies from using magical abilities.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)", EffectDuration: 20},
		ScrollInQuasWis:  {ItemName: "In_Quas_Wis", ItemSprite: 260, ItemDescription: "Shows an overhead view of the surrounding surface or town, or a map of the surrounding dungeon level. It has no effect in combat.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		ScrollKalXenCorp: {ItemName: "Kal_Xen_Corp", ItemSprite: 260, ItemDescription: "In combat, attempts to summon a loyal daemon to fight for the party. It requires open space and has no effect outside combat.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		ScrollInManiCorp: {ItemName: "In_Mani_Corp", ItemSprite: 260, ItemDescription: "Outside combat, resurrects one dead party member with 1 health. If karma is below 98, the character also loses experience and may lose levels.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)"},
		ScrollAnTym:      {ItemName: "An_Tym", ItemSprite: 260, ItemDescription: "Briefly freezes the clock and other creatures while the party remains active. It fails in Stonegate and Doom.", ItemDescriptionAttribution: "Ultima V (verified DOS behavior)", EffectDuration: 20},
	}
}
