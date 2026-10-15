package game_state

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) newUseableItemInfo(
	itemRef references.InventoryItem,
	category UseItemCategory,
	usage UsageContext,
	quantity int,
	item references.Item,
) UseableItemInfo {
	name := formatItemName(itemRef.ItemName)
	fullName := strings.TrimSpace(itemRef.ItemDisplayName)
	if fullName == "" {
		fullName = name
	}
	info := UseableItemInfo{
		Name:              name,
		FullName:          fullName,
		Description:       itemRef.ItemDescription,
		EffectSummary:     "Uses this item.",
		NextInteraction:   "The effect resolves immediately.",
		Requirement:       "Ready to use.",
		Consumption:       "Not consumed.",
		Warning:           "",
		UnavailableReason: "",
		AvailableNow:      true,
		Category:          category,
		UsageContext:      usage,
		Quantity:          quantity,
		SpriteIndex:       itemRef.ItemSprite,
		ItemRef:           itemRef,
		ItemType:          item,
	}

	return g.withUseItemGuidance(info, UsageField)
}

func (g *GameState) withUseItemGuidance(info UseableItemInfo, context UsageContext) UseableItemInfo {
	info.AvailableNow = true
	info.Warning = ""
	info.UnavailableReason = ""

	switch item := info.ItemType.(type) {
	case references.Potion:
		g.addPotionUseGuidance(&info, item, context)
	case references.Scroll:
		g.addScrollUseGuidance(&info, item, context)
	case references.QuestItem:
		g.addQuestItemUseGuidance(&info, item, context)
	case references.Moonstone:
		g.addMoonstoneUseGuidance(&info, context)
	case references.Provision:
		g.addProvisionUseGuidance(&info, item, context)
	default:
		panic("use item projection received an unsupported item type")
	}
	if !useItemSupportsContext(info.ItemType, info.UsageContext, context) {
		markUseItemUnavailable(&info, useItemContextRestriction(info.ItemType, context))
	}

	return info
}

func (g *GameState) addPotionUseGuidance(info *UseableItemInfo, potion references.Potion, context UsageContext) {
	review := g.ReviewPotionUse(potion, context)
	info.NextInteraction = "Choose a party member."
	if context == UsageCombat {
		info.NextInteraction = "Affects the active combatant immediately."
	}
	info.Consumption = "One potion is consumed before its effect is known."
	info.Warning = "Potion labels are unreliable; another colour's effect may occur."

	switch potion {
	case references.Blue:
		info.EffectSummary = "Usually awakens a sleeping party member."
		info.Requirement = potionRequirement(
			context,
			"Most useful when the chosen member is asleep.",
			"Most useful when the active combatant is asleep.",
		)
	case references.Yellow:
		info.EffectSummary = "Usually restores 1-30 health to a living party member."
		info.Requirement = potionRequirement(
			context,
			"Choose a living, wounded party member.",
			"Most useful when the active combatant is living and wounded.",
		)
	case references.Red:
		info.EffectSummary = "Usually cures poison without restoring health."
		info.Requirement = potionRequirement(
			context,
			"Choose a poisoned party member.",
			"Most useful when the active combatant is poisoned.",
		)
	case references.Green:
		info.EffectSummary = "Usually poisons a healthy party member."
		info.Requirement = potionRequirement(
			context,
			"Choose a living party member.",
			"Requires a living active combatant.",
		)
	case references.Orange:
		info.EffectSummary = "Usually puts a healthy party member to sleep."
		info.Requirement = potionRequirement(
			context,
			"Choose a living party member.",
			"Requires a living active combatant.",
		)
	case references.Purple:
		info.EffectSummary = "Usually gives the active combatant a rat's appearance for the battle, without changing abilities."
		info.NextInteraction = usePotionTargetInteraction(context)
		info.Requirement = "Its intended effect requires combat."
		if !review.EffectAvailable {
			warnUseItemIneffective(info, "No noticeable effect outside combat; it is still consumed.")
		}
	case references.Black:
		info.EffectSummary = "Usually makes the active combatant invisible for the battle."
		info.NextInteraction = usePotionTargetInteraction(context)
		info.Requirement = "Its intended effect requires combat."
		if !review.EffectAvailable {
			warnUseItemIneffective(info, "No noticeable effect outside combat; it is still consumed.")
		}
	case references.White:
		info.EffectSummary = "Usually reveals the immediate surface or town view."
		info.NextInteraction = usePotionTargetInteraction(context)
		info.Requirement = "Its intended effect requires exploration outside dungeons."
		if !review.EffectAvailable {
			warning := "No noticeable effect in combat; it is still consumed."
			if context == UsageDungeon {
				warning = "No noticeable effect in a dungeon; it is still consumed."
			}
			warnUseItemIneffective(info, warning)
		}
	default:
		panic("use item projection received an unknown potion")
	}
}

func potionRequirement(context UsageContext, fieldRequirement string, combatRequirement string) string {
	if context == UsageCombat {
		return combatRequirement
	}

	return fieldRequirement
}

func usePotionTargetInteraction(context UsageContext) string {
	if context == UsageCombat {
		return "Affects the active combatant immediately."
	}

	return "Choose a party member."
}

func (g *GameState) addScrollUseGuidance(info *UseableItemInfo, scroll references.Scroll, context UsageContext) {
	info.Consumption = "One scroll is consumed when its effect commits."
	info.Requirement = "Ready to use."

	switch scroll {
	case references.ScrollVasLor:
		info.EffectSummary = "Creates powerful magical light for 240 game minutes."
		info.NextInteraction = "The light activates after a brief magical flash."
	case references.ScrollRelHur:
		info.EffectSummary = "Changes the overworld wind direction."
		info.NextInteraction = "Choose north, south, east, west, or calm."
		info.Requirement = "Usable in towns and outdoors."
		if !g.isScrollLocationValid(scroll) {
			markUseItemUnavailable(info, "Wind cannot be changed from this map.")
		}
	case references.ScrollInSanct:
		info.EffectSummary = "Grants the party +3 armour for a limited time."
		info.NextInteraction = "Protection begins immediately."
		info.Requirement = "Useful before or during combat; duration starts now."
	case references.ScrollInAn:
		info.EffectSummary = "Suppresses combat magic and magical enemy abilities for both sides."
		info.NextInteraction = "Negate Magic begins immediately."
		info.Requirement = "The party's magic is suppressed too."
	case references.ScrollInQuasWis:
		info.EffectSummary = "Shows an overhead view of the surrounding area."
		info.NextInteraction = "Open the dungeon map view."
		if context != UsageDungeon {
			info.NextInteraction = "Open the surface or town overhead view."
		}
		info.Requirement = "No effect in combat."
	case references.ScrollKalXenCorp:
		info.EffectSummary = "Attempts to summon a loyal daemon in combat."
		info.NextInteraction = "The daemon appears in an available combat square."
		info.Requirement = "Requires combat and open space."
	case references.ScrollInManiCorp:
		info.EffectSummary = "Resurrects one dead party member with 1 health."
		info.NextInteraction = "Choose a dead party member."
		info.Requirement = "Requires a dead party member outside combat."
		info.Consumption = "Consumed when a target is confirmed, even if the target is living."
		if context == UsageDungeon {
			info.Consumption = "Consumed on target confirmation, even with no effect; cancellation is free."
		}
		if !g.PartyState.HasDeadMembers() {
			warnUseItemIneffective(info, "No party member is currently dead; a confirmed attempt still consumes the scroll.")
		}
	case references.ScrollAnTym:
		info.EffectSummary = "Briefly freezes time while the party remains active."
		info.NextInteraction = "Time Stop begins after a brief magical flash."
		info.Requirement = "Outside combat, fails in Stonegate and Doom."
		location := g.MapState.PlayerLocation.Location
		if location == references.Stonegate || location == references.Doom {
			info.NextInteraction = "Reports 'No effect!' after consumption."
			warnUseItemIneffective(info, "No effect here; this scroll is still consumed.")
		}
	default:
		panic("use item projection received an unknown scroll")
	}
}

func (g *GameState) addQuestItemUseGuidance(
	info *UseableItemInfo,
	item references.QuestItem,
	context UsageContext,
) {
	info.Consumption = "Not consumed."

	switch item {
	case references.Amulet:
		info.EffectSummary = "Wear or remove the Amulet."
		info.NextInteraction = "The Amulet toggles immediately."
		info.Requirement = toggleRequirement(g.IsAmuletActive())
	case references.Crown:
		info.EffectSummary = "Dons or removes the Crown's magic-suppression state."
		info.NextInteraction = "The Crown toggles immediately."
		info.Requirement = toggleRequirement(g.IsCrownActive())
	case references.Sceptre:
		info.EffectSummary = "Dispels magical energy fields."
		info.NextInteraction = "Clears nearby fields or asks for a direction."
		if context == UsageDungeon {
			info.NextInteraction = "Dispels the field in the current dungeon cell."
		}
		info.Requirement = "A magical field must be within reach."
	case references.ShardOfFalsehood, references.ShardOfHatred, references.ShardOfCowardice:
		info.EffectSummary = "Begins the ritual to destroy this Shadowlord shard."
		info.NextInteraction = "Begin the ritual."
		info.Requirement = "The ritual requires the right conditions."
		info.Consumption = "Consumed on successful destruction."
		info.Warning = "Destruction is permanent. The shard is retained if there is no effect."
		if context == UsageDungeon {
			info.EffectSummary = "Has no effect when used in a dungeon."
			info.NextInteraction = "Reports 'No effect!' immediately."
			info.Requirement = "The destruction ritual cannot begin in a dungeon."
			markUseItemUnavailable(info, "No effect in a dungeon; the shard is retained.")
		}
	case references.Carpet:
		info.EffectSummary = "Places and boards the Magic Carpet."
		info.NextInteraction = "The party boards the carpet immediately."
		info.Requirement = "Requires an Overworld, Underworld, or town tile, on foot, away from mountains."
		//nolint:exhaustive // Ready and inventory presence need no warning; commit-only results are impossible here.
		switch g.ReviewCarpetBoarding().Result {
		case CarpetBoardingNotHere:
			markUseItemUnavailable(info, "The carpet cannot be placed on this map.")
		case CarpetBoardingExitShipFirst, CarpetBoardingOnlyOnFoot:
			markUseItemUnavailable(info, "Exit the current vehicle before using the carpet.")
		case CarpetBoardingOccupied:
			markUseItemUnavailable(info, "The current tile is occupied by another map object.")
		case CarpetBoardingNoMapCapacity:
			markUseItemUnavailable(info, "There is no room to place the carpet on this map.")
		}
	case references.Grapple:
		info.EffectSummary = "Enables mountain routes that require a grappling hook."
		info.NextInteraction = "Use the Klimb command near an applicable route."
		info.Requirement = "The Use command does not activate the grapple directly."
		markUseItemUnavailable(info, "Use 'K'limb instead; selecting this item only reports that guidance.")
	case references.Spyglass:
		info.EffectSummary = "Studies the night sky from Britannia or an above-ground small map."
		info.NextInteraction = "Opens the astronomy view at night."
		info.Requirement = "Requires Britannia or an above-ground small map at night."
		if !g.IsSpyglassLocationValid() {
			markUseItemUnavailable(info, "The spyglass cannot be used from this location.")
		} else if !g.IsUseAstronomyNight() {
			markUseItemUnavailable(info, "No stars are visible during daylight.")
		}
	case references.HMSCape:
		info.EffectSummary = "Toggles faster automatic sailing for all frigates."
		info.NextInteraction = "Review and confirm the global H.M.S. Cape setting."
		info.Requirement = fmt.Sprintf("The global setting is currently %s.", activeState(g.IsHMSCapeActivated()))
	case references.PocketWatch:
		info.EffectSummary = "Shows the current Britannian time."
		info.NextInteraction = "Dismiss the time display when finished."
		info.Requirement = "Ready to use in exploration or combat."
		if context == UsageCombat {
			info.Consumption = "Not consumed; reading it spends one combat action after dismissal."
		}
	case references.BlackBadge:
		info.EffectSummary = "Wear or remove the Black Badge."
		info.NextInteraction = "The Badge toggles immediately."
		info.Requirement = toggleRequirement(g.IsBadgeActive())
	case references.WoodenBox:
		info.EffectSummary = "Carries a curious object reserved for the endgame."
		info.NextInteraction = "No direct Use interaction is available here."
		info.Requirement = "Requires its specific endgame context."
		warning := "Selecting it here reports 'Not here!'."
		if context == UsageDungeon {
			warning = "Selecting it in a dungeon reports 'How?'."
		}
		markUseItemUnavailable(info, warning)
	case references.Sextant:
		info.EffectSummary = "Reports the party's overworld coordinates."
		info.NextInteraction = "Coordinates appear in the message log."
		info.Requirement = "Requires the Britannian overworld at night."
		if g.GameConfig.GameplayConfiguration.FlexibleSextantUse {
			info.Requirement = "Requires Britannia's surface or a surface settlement."
		}
		switch g.CurrentSextantEligibility() {
		case SextantMissing, SextantNotOverworld, SextantDaylight:
			markUseItemUnavailable(info, g.sextantUnavailableReason())
		case SextantReady:
			// Ready without additional requirements.
		}
	default:
		panic("use item projection received an unknown quest item")
	}
}

func (g *GameState) addMoonstoneUseGuidance(info *UseableItemInfo, context UsageContext) {
	info.EffectSummary = "Buries this moonstone to establish its moongate location."
	info.NextInteraction = "Bury the moonstone at your current position."
	info.Requirement = "Requires suitable surface or town terrain."
	info.Consumption = "Removed from inventory while buried; it can later be recovered."
	if context != UsageField {
		markUseItemUnavailable(info, "Moonstones cannot be buried here.")

		return
	}
	mapType := g.MapState.PlayerLocation.Location.GetMapType()
	if mapType != references.LargeMapType && mapType != references.SmallMapType {
		markUseItemUnavailable(info, "Moonstones cannot be buried on this map.")

		return
	}
	position := g.MapState.PlayerLocation.Position
	tile := g.GetLayeredMapByCurrentLocation().GetTopTile(&position)
	if tile == nil || !tile.IsValidForMoonstoneBurial() {
		markUseItemUnavailable(info, "The current terrain cannot hold a buried moonstone.")
	}
}

func (g *GameState) addProvisionUseGuidance(
	info *UseableItemInfo,
	item references.Provision,
	context UsageContext,
) {
	if item != references.SkullKeys {
		panic("use item projection received an unsupported provision")
	}
	info.EffectSummary = "Attempts to unlock a magically locked door."
	info.NextInteraction = "Choose the adjacent door direction."
	info.Requirement = "Unlocks an adjacent magical door on the Britannian surface, a small map, or in combat."
	info.Consumption = "One Skull Key is consumed on direction confirmation; cancellation is free."
	info.Warning = "A confirmed direction consumes the key even if no door is unlocked."
	if context == UsageCombat {
		info.NextInteraction = "Choose a direction adjacent to the acting combatant."
		info.Consumption = "One key and one combat action on direction confirmation, even with no effect; cancellation is free."
	}
	if (context != UsageField && context != UsageCombat) || g.ReviewSkullKeyUse().Result != SkullKeyUseReady {
		markUseItemUnavailable(info, "Skull Keys cannot be used on this map.")
	}
}

func toggleRequirement(active bool) string {
	if active {
		return "Currently active; using it removes the effect."
	}

	return "Currently inactive; using it activates the effect."
}

func activeState(active bool) string {
	if active {
		return "active"
	}

	return "inactive"
}

func markUseItemUnavailable(info *UseableItemInfo, warning string) {
	info.UnavailableReason = warning
	info.NextInteraction = "Unavailable: inspect this restriction or choose another item."
	warnUseItemIneffective(info, warning)
}

func warnUseItemIneffective(info *UseableItemInfo, warning string) {
	info.AvailableNow = false
	info.Warning = warning
}

func useItemContextRestriction(item references.Item, context UsageContext) string {
	if scroll, ok := item.(references.Scroll); ok {
		//nolint:exhaustive // Only context-restricted scrolls need special wording.
		switch scroll {
		case references.ScrollInManiCorp:
			return "Cannot resurrect during combat."
		case references.ScrollRelHur:
			return "Wind can only be changed in towns or outdoors."
		case references.ScrollInQuasWis:
			return "The map view cannot be opened during combat."
		case references.ScrollKalXenCorp:
			return "Summoning a daemon requires combat."
		}
	}

	return fmt.Sprintf("This item's Use interaction is not supported in %s context.", strings.ToLower(context.String()))
}
