package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDoomInteriorCombatUsesAuthoritativeOrigin(t *testing.T) {
	finalRoom := &CombatState{
		DungeonRoomOrigin: &DungeonRoomEncounterOrigin{
			Location: references.Doom,
			Room:     doomFinalRoomID,
		},
	}
	assert.True(t, finalRoom.isDoomInteriorCombat())

	ordinaryRoom := &CombatState{
		DungeonRoomOrigin: &DungeonRoomEncounterOrigin{Location: references.Doom, Room: 2},
	}
	assert.True(t, ordinaryRoom.isDoomInteriorCombat())

	wanderer := &CombatState{
		DungeonWandererOrigin: &DungeonWandererEncounterOrigin{Location: references.Doom},
	}
	assert.True(t, wanderer.isDoomInteriorCombat())

	camping := &CombatState{
		CampingOrigin: &CampingCombatOrigin{
			ReturnLocation: references.PlayerLocation{Location: references.Doom},
		},
	}
	assert.True(t, camping.isDoomInteriorCombat())

	nonDoom := &CombatState{
		DungeonRoomOrigin: &DungeonRoomEncounterOrigin{Location: references.Deceit},
	}
	assert.False(t, nonDoom.isDoomInteriorCombat())

	underworldEncounter := &CombatState{
		PreCombatPlayerLocation: references.PlayerLocation{Location: references.Doom},
		CombatMapRef: references.NewCombatMapReference(
			references.CalculateDungeonCombatMapIndex(references.DoomDungeon, 2),
			references.DungeonCombatTerritory,
		),
	}
	assert.False(t, underworldEncounter.isDoomInteriorCombat(),
		"location restoration data and a Doom combat-map selection are not encounter origins")
}

func TestDoomInteriorPartyDefenseUsesMysticArmourGateAndSeparateProtection(t *testing.T) {
	state, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 279)
	if state == nil {
		return
	}
	character := state.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotHelm, references.IronHelm)
	character.SetEquippedItem(references.SlotArmour, references.PlateMail)
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)
	character.SetEquippedItem(references.SlotShield, references.LargeShield)
	character.SetEquippedItem(references.SlotRing, references.RingProtection)
	character.SetEquippedItem(references.SlotAmulet, references.AmuletOfTurning)
	normalDefense := character.GetTotalDefense()
	require.Positive(t, normalDefense)

	state.CombatScene.DungeonRoomOrigin = &DungeonRoomEncounterOrigin{Location: references.Deceit}
	assert.Equal(t, normalDefense, state.combatPartyDefense(character), "ordinary dungeon control")

	state.CombatScene.DungeonRoomOrigin = nil
	state.CombatScene.PreCombatPlayerLocation.Location = references.Doom
	assert.Equal(t, normalDefense, state.combatPartyDefense(character), "rejected-entry control")

	state.CombatScene.DungeonRoomOrigin = &DungeonRoomEncounterOrigin{Location: references.Doom, Room: 2}
	assert.Zero(t, state.combatPartyDefense(character), "non-Mystic equipment contributes no defense in Doom")

	state.ActivateProtectionEffect(5)
	assert.Equal(t, combatInSanctDefenseBonus, state.combatPartyDefense(character),
		"Protection remains independent of equipment defense")

	character.SetEquippedItem(references.SlotArmour, references.MysticArmour)
	assert.Equal(t, character.GetTotalDefense()+combatInSanctDefenseBonus, state.combatPartyDefense(character),
		"Mystic Armour preserves the complete equipment sum")
}
