package semantic

// OriginalItemName returns the semantic name for a verified original talk item ID.
func OriginalItemName(itemID int) (string, bool) {
	armaments := originalArmamentNames()
	if itemID >= 0 && itemID < len(armaments) {
		return armaments[itemID], true //nolint:gosec // Bounds are checked immediately above.
	}
	const firstResourceID = 0x41
	resources := [...]string{
		"food", "gold", "key", "gem", "torch", "grapple", "carpet", "sextant", "spyglass", "black_badge", "skull_key",
	}
	resourceIndex := itemID - firstResourceID
	if resourceIndex < 0 || resourceIndex >= len(resources) {
		return "", false
	}

	return resources[resourceIndex], true //nolint:gosec // Bounds are checked immediately above.
}

// OriginalItemID returns the original talk item ID for a verified semantic name.
func OriginalItemID(name string) (int, bool) {
	for itemID, candidate := range originalArmamentNames() {
		if candidate == name {
			return itemID, true
		}
	}
	const (
		firstResourceID = 0x41
		lastResourceID  = 0x4b
	)
	for itemID := firstResourceID; itemID <= lastResourceID; itemID++ {
		candidate, _ := OriginalItemName(itemID)
		if candidate == name {
			return itemID, true
		}
	}

	return 0, false
}

func originalArmamentNames() [48]string {
	return [...]string{
		"leather_helm", "chain_coif", "iron_helm", "spiked_helm",
		"small_shield", "large_shield", "spiked_shield", "magic_shield", "jewel_shield",
		"cloth_armour", "leather_armour", "ring_mail", "scale_mail", "chain_mail", "plate_mail", "mystic_armour",
		"dagger", "sling", "club", "flaming_oil", "main_gauche", "spear", "throwing_axe", "short_sword",
		"mace", "morning_star", "bow", "arrows", "crossbow", "quarrels", "long_sword", "two_handed_hammer",
		"two_handed_axe", "two_handed_sword", "halberd", "chaos_sword", "magic_bow", "silver_sword",
		"magic_axe", "glass_sword", "jeweled_sword", "mystic_sword", "ring_of_invisibility",
		"ring_of_protection", "ring_of_regeneration", "amulet_of_turning", "spiked_collar", "ankh",
	}
}
