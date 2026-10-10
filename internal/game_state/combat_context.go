package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

func (combat *CombatState) isDoomInteriorCombat() bool {
	if combat == nil {
		return false
	}
	if combat.DungeonRoomOrigin != nil && combat.DungeonRoomOrigin.Location == references.Doom {
		return true
	}
	if combat.DungeonWandererOrigin != nil && combat.DungeonWandererOrigin.Location == references.Doom {
		return true
	}
	if combat.CampingOrigin != nil && combat.CampingOrigin.ReturnLocation.Location == references.Doom {
		return true
	}

	return false
}

func (combat *CombatState) invisibilityProtectsTargets() bool {
	return !combat.isDoomInteriorCombat()
}
