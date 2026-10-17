package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// Use copy is deliberately independent of technical reference descriptions.
// Keep borrowed lore and its attribution together. Original prose below is not
// presented as a quotation; shared catalog metadata and its credits stay intact.
// Browsing an owned item grants no powers, ritual recipes or conversation clues.
func setUseItemLore(info *UseableItemInfo) {
	description, credit := "", ""
	switch item := info.ItemType.(type) {
	case references.Potion:
		description = "A stoppered bottle holds a coloured draught. Its appearance alone cannot tell its nature."
	case references.Scroll:
		description = "Words of magic cover this scroll, waiting to be spoken."
	case references.Moonstone:
		description = "A smooth stone bears the mark of a phase of the moon."
	case references.Provision:
		if item == references.SkullKeys {
			description = "A small key fashioned in the likeness of a skull."
		}
	case references.QuestItem:
		//nolint:exhaustive // Only mystery-bearing descriptions need a Use-specific replacement.
		switch item {
		case references.Amulet:
			description = "It emanates a blue aura and is notably warm to the touch."
			credit = "Excerpt adapted from UltimaCodex.com"
		case references.Crown:
			description = "Lord British's Crown is one of Lord British's Royal Crown Jewels."
			credit = "UltimaCodex.com and The Book of Brad"
		case references.Sceptre:
			description = "The Sceptre of Lord British, one of the three Royal Crown Jewels, is a powerful magical artifact."
			credit = "UltimaCodex.com and The Book of Brad"
		case references.BlackBadge:
			description = "A dark badge, small enough to fasten to a cloak."
		default:
			return // Retained lore keeps its existing complete source attribution.
		}
	default:
		return
	}
	if description != "" {
		info.Description = description
		info.ItemRef.ItemDescription = description
		info.ItemRef.ItemDescriptionAttribution = credit
	}
}
