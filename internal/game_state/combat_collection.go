package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// CombatCollectedItem is an encounter total of actual inventory additions, not
// treasure revealed or a before/after inventory balance. Order is first acquired.
type CombatCollectedItem struct {
	ItemType references.ItemType
	ItemID   int
	Name     string
	Sprite   indexes.SpriteIndex
	Quantity uint64
}

// CombatPickupEvent keeps identity and wording stable after the acting turn ends.
type CombatPickupEvent struct {
	ActorName string
	Position  references.Position
	Result    CombatPickupResult
}

// CombatChestTrapResult is detached from the victim, including fatal acid hits.
type CombatChestTrapResult struct {
	Type      TrapType
	ActorName string
	HPLost    uint16
	Fatal     bool
}

// GetCombatItem publishes exactly one result and records only committed gains.
func (g *GameState) GetCombatItem(direction references.Direction) CombatPickupResult {
	result := g.getCombatItem(direction)
	if !g.IsInCombat() || g.CombatScene == nil {
		return result
	}
	combat := g.CombatScene
	position := g.getActiveCombatantPosition()
	if target := direction.GetNewPositionInDirection(&position); target != nil {
		position = *target
	}
	combat.recordCollection(result)
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind: CombatEventPickup,
		Pickup: CombatPickupEvent{
			ActorName: g.combatIdentityName(combat.GetCurrentTurnIdentity()),
			Position:  position, Result: result,
		},
	})

	return result
}

func (s *CombatState) recordCollection(result CombatPickupResult) {
	if result.Acquired == 0 {
		return
	}
	for index := range s.collectedItems {
		item := &s.collectedItems[index]
		if item.ItemType == result.ItemType && item.ItemID == result.ItemID {
			item.Quantity += uint64(result.Acquired)

			return
		}
	}
	s.collectedItems = append(s.collectedItems, CombatCollectedItem{
		ItemType: result.ItemType, ItemID: result.ItemID, Name: result.ItemName,
		Sprite: result.Sprite, Quantity: uint64(result.Acquired),
	})
}
