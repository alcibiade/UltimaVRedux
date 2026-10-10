package game_state

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// generateCombatChestContents reproduces miscellaneous chest-loot selection and chest equipment selection from original Search/Get/Open/Jimmy rules.
// See ref:CMD-0001.
func (g *GameState) generateCombatChestContents(position references.Position, maximumValue int) []string {
	revealLines := g.generateCombatMiscellaneous(position, maximumValue)

	return append(revealLines, g.generateCombatArmaments(position, maximumValue)...)
}

// The tables and loop bounds are original U5 data rather than tunable values.
func (g *GameState) generateCombatMiscellaneous(position references.Position, maximumValue int) []string {
	miscFrequencies := [...]int{25, 3, 17, 17, 9, 15, 7, 7}
	miscMaximums := [...]int{10, 90, 8, 8, 2, 2, 2, 2}
	var revealLines []string

	for index := len(miscFrequencies) - 1; index >= 0; index-- {
		frequency := miscFrequencies[index]
		if frequency > maximumValue || frequency > g.RollDice(combatTreasureDiceSides) {
			continue
		}

		number := g.RandomIntInRange(1, miscMaximums[index])
		if reveal := g.generateCombatMiscellaneousItem(position, maximumValue, index, number); reveal != "" {
			revealLines = append(revealLines, reveal)
		}
	}

	return revealLines
}

//nolint:mnd // Switch indexes and gold multiplier are original chkmisc/additem data.
func (g *GameState) generateCombatMiscellaneousItem(
	position references.Position,
	maximumValue, index, number int,
) string {
	switch index {
	case 0:
		treasureValue := g.RandomIntInRange(1, maximumValue)
		if !g.CombatScene.addLootObject(NewCombatChest(
			position,
			treasureValue,
			false,
		)) {
			return ""
		}

		return "a chest!"
	case 1:
		quantity := combatQuantity(g.RandomIntInRange(1, maximumValue*3))
		if !g.addCombatChestLooseItem(position, references.Gold, quantity) {
			return ""
		}

		return "a sack of gold!"
	case 2:
		if !g.addCombatChestLooseItem(position, references.Potion(number-1), 1) {
			return ""
		}

		return "a potion!"
	case 3:
		if !g.addCombatChestLooseItem(position, references.Scroll(number-1), 1) {
			return ""
		}

		return "a scroll!"
	case 4:
		if !g.addCombatChestLooseItem(position, references.Key, combatQuantity(number)) {
			return ""
		}

		return "a ring of keys!"
	case 5:
		if !g.addCombatChestLooseItem(position, references.Gem, combatQuantity(number)) {
			return ""
		}

		return "a gem!"
	case 6:
		if !g.addCombatChestLooseItem(position, references.Torches, combatQuantity(number)) {
			return ""
		}

		return "some torches!"
	case 7:
		if !g.addCombatChestLooseItem(position, references.Food, combatQuantity(number)) {
			return ""
		}

		return "some food!"
	default:
		panic("combat chest selected an unknown miscellaneous item")
	}
}

func (g *GameState) generateCombatArmaments(position references.Position, maximumValue int) []string {
	armamentFrequencies := [...]int{
		10, 10, 15, 20,
		10, 15, 20, 28, 0xff,
		15, 15, 20, 20, 20, 24, 0xff,
		5, 10, 10, 10, 10, 10, 10,
		10, 15, 15, 15, 10, 15, 10,
		20, 20, 20, 20, 20, 0xff, 23,
		23, 23, 0xff, 0xff, 0xff,
		23, 23, 23,
		23, 15, 0xff,
	}
	var revealLines []string
	for range maximumValue/2 + 1 {
		index := g.RandomIntInRange(0, len(armamentFrequencies)-1)
		frequency := armamentFrequencies[index]
		if frequency > maximumValue || frequency > g.RollDice(combatTreasureDiceSides) {
			continue
		}
		equipment := references.Equipment(index)
		if g.addCombatChestLooseItem(position, equipment, 1) {
			revealLines = append(revealLines, combatArmamentRevealLine(equipment))
		}
	}

	return revealLines
}

func combatArmamentRevealLine(equipment references.Equipment) string {
	switch equipment.GetCategory() {
	case references.CategoryHelm:
		return "a helm!"
	case references.CategoryShield:
		return "a shield!"
	case references.CategoryArmour:
		return "some armour!"
	case references.CategoryWeapon, references.CategoryAmmo:
		return "a weapon!"
	case references.CategoryRing:
		return "a ring!"
	case references.CategoryAmulet:
		return "an amulet!"
	case references.CategoryNone:
		panic("combat chest generated uncategorized equipment")
	default:
		panic("combat chest generated equipment with unknown category")
	}
}

func combatQuantity(value int) uint16 {
	if value < 1 || value > math.MaxUint16 {
		panic("combat item quantity is outside uint16 range")
	}

	return uint16(value)
}

func (g *GameState) addCombatChestLooseItem(position references.Position, item references.Item, quantity uint16) bool {
	itemReference := g.GameReferences.InventoryItemReferences.GetReferenceByItem(item)

	return g.CombatScene.addLootObject(NewCombatLooseItem(position, item, quantity, itemReference.ItemSprite))
}

func (g *GameState) triggerCombatChestTrap(pcIndex int) CombatChestTrapResult {
	trapType := TrapType(g.RandomIntInRange(int(TrapAcid), int(TrapPoison)))
	character := g.PartyState.GetCharacter(pcIndex)
	beforeHP := character.GetCurrentHp()

	g.SystemCallbacks.Message.AddRowStr("Trapped!")
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundTrapTrigger)
	switch trapType {
	case TrapAcid:
		g.SystemCallbacks.Message.AddRowStr("ACID!")
		damage := g.RollOriginalD30() // Failed-experiment trap selection calls transformed d30 for acid damage.
		// See ref:RNG-0001.
		character.SubtractHp(uint16(damage)) //nolint:gosec // Original acid damage is bounded to 1-30.
		if character.IsDead() {
			g.markCombatPCDead(pcIndex)
		}
	case TrapPoison:
		g.SystemCallbacks.Message.AddRowStr("POISON!")
		statusBefore := character.GetStatus()
		character.SetStatus(party_state.Poisoned)
		changes := transientStatusChanges{}
		changes.observePoison(statusBefore, character)
		g.publishTransientStatusChanges(changes)
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundPoison)
	case TrapNone, TrapBomb, TrapGas:
		panic("combat chest selected an invalid trap type")
	default:
		panic("combat chest selected an unknown trap type")
	}
	g.SystemCallbacks.Screen.UpdateStatsDisplay()
	result := CombatChestTrapResult{
		Type: trapType, ActorName: character.GetNameAsString(),
		HPLost: beforeHP - character.GetCurrentHp(), Fatal: character.IsDead(),
	}
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{Kind: CombatEventChestTrap, ChestTrap: result})

	return result
}
