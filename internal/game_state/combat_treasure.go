package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const combatTreasureDiceSides = 30

func combatTreasureRoll(treasureValue, dropRoll, trapRoll int) (bool, bool) {
	if treasureValue <= 0 || dropRoll > treasureValue {
		return false, false
	}

	return true, trapRoll < treasureValue
}

func rollCombatTreasure(treasureValue int, rollD30 func() int) (bool, bool) {
	dropRoll := rollD30()
	if dropRoll > treasureValue {
		return false, false
	}

	return combatTreasureRoll(treasureValue, dropRoll, rollD30())
}

func combatDeathObjectEligible(monster *CombatMonster, terrain *references.Tile) bool {
	if monster == nil || terrain == nil || monster.EnemyRef() == nil {
		return false
	}
	if monster.HasAbility(references.NoCorpse) {
		return false
	}

	// Original combat damage rules remove the corpse object instead of replacing it when the
	// underlying tile is ARCH or a low-index water tile.
	// See ref:COMBAT-0009.
	return terrain.Index != indexes.BrickWallArchway && terrain.Index >= indexes.Swamp
}

func (g *GameState) placeCombatDeathObject(monster *CombatMonster, monsterIdx int) bool {
	position := monster.Position()
	terrain := g.GetCurrentLayeredMap().GetTopTile(&position)
	if !combatDeathObjectEligible(monster, terrain) {
		return false
	}
	treasureValue := monster.EnemyRef().TreasureNumber
	// Original: damage application calls transformed d30 independently for the drop and trap checks.
	// See ref:RNG-0001.
	dropChest, lockedAndTrapped := rollCombatTreasure(treasureValue, g.RollOriginalD30)
	source := CombatTurnIdentity{IsPartyMember: false, EntityIndex: monsterIdx}
	if !dropChest {
		if !g.CombatScene.transferCombatantObjectTo(source, NewCombatCorpse(position)) {
			panic("placeCombatDeathObject: dead monster has no original object slot")
		}

		return true
	}

	if !g.CombatScene.transferCombatantObjectTo(
		source,
		NewCombatChest(position, treasureValue, lockedAndTrapped),
	) {
		panic("placeCombatDeathObject: dead monster has no original object slot")
	}

	return true
}
