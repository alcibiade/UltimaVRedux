package ai

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// SetAttackAvatar marks appropriate NPCs to attack the Avatar when hostile actions occur.
// This is triggered by actions like stealing or attacking in towns, causing guards and
// certain NPCs to become hostile.
func (n *NPCAIControllerSmallMap) SetAttackAvatar() {
	for _, mapUnit := range n.mapUnits {
		if npc, ok := mapUnit.(*map_units.NPCFriendly); ok {
			if npc.NPCReference.WantsToAttackAvatarWhenBadStuffGoesDown() {
				// Guard activation selects action 7 for NPC tile bytes >= 47; ref:CRIME-0002.
				mapUnit.MapUnitDetails().SetLawAIOverride(references.DrudgeWorthThing)
				mapUnit.MapUnitDetails().SetCurrentPath(nil)
				mapUnit.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionNone
			}
		}
	}
}
