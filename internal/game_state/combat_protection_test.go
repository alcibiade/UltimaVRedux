package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestCombatPartyDefenseUsesCurrentEquipmentAndProtection(t *testing.T) {
	state := &GameState{}
	character := &party_state.PlayerCharacter{
		ArmorClass: 255,
		Helmet:     byte(references.IronHelm),
		Armor:      byte(references.PlateMail),
		Weapon:     byte(references.MainGauche),
		Shield:     byte(references.LargeShield),
		Ring:       byte(references.RingProtection),
		Amulet:     byte(references.SpikedCollar),
	}

	assert.Equal(t, 18, state.combatPartyDefense(character))

	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	assert.Equal(t, 15, state.combatPartyDefense(character))

	state.ActiveDurationEffect = DurationEffect{
		EffectType:     DurationEffectProtection,
		TurnsRemaining: 1,
	}
	assert.Equal(t, 18, state.combatPartyDefense(character))
	assert.Equal(t, byte(255), character.ArmorClass)
}

func TestPlayerWeaponDamageIgnoresPersistedArmorClass(t *testing.T) {
	for _, test := range []struct {
		name       string
		armorClass byte
	}{
		{name: "zero", armorClass: 0},
		{name: "maximum", armorClass: 255},
	} {
		t.Run(test.name, func(t *testing.T) {
			var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
			roster[0] = &party_state.PlayerCharacter{Weapon: byte(references.TwoHSword)}
			roster[1] = &party_state.PlayerCharacter{
				ArmorClass: test.armorClass,
				Helmet:     byte(references.IronHelm),
				Armor:      byte(references.PlateMail),
				Weapon:     byte(references.NoEquipment),
				Shield:     byte(references.LargeShield),
				Ring:       byte(references.NoEquipment),
				Amulet:     byte(references.NoEquipment),
			}
			roster[1].SetMaxHp(100)
			roster[1].SetHp(100)
			state := &GameState{PartyState: *party_state.NewPartyStateFromRoster(roster)}
			state.SetRandomSeed(1)

			_, _, _, diagnostics := state.applyPlayerWeaponHitToPC(0, CombatAttackSource{
				Slot:      references.SlotWeapon,
				Equipment: references.TwoHSword,
			}, 1, CombatDiagnostics{})

			assert.Equal(t, 13, diagnostics.Damage.ArmorValue)
			assert.Equal(t, test.armorClass, state.PartyState.GetCharacter(1).ArmorClass)
		})
	}
}

func TestArmorClassNativeSnapshotRoundTripPreservesCompatibilityValue(t *testing.T) {
	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	roster[0] = &party_state.PlayerCharacter{ArmorClass: 255}
	state := &GameState{PartyState: *party_state.NewPartyStateFromRoster(roster)}
	var snapshot save_snapshot.SaveSnapshot

	state.captureParty(&snapshot)
	assert.Equal(t, 255, snapshot.Party.Roster[0].ArmorClass)

	restored := &GameState{}
	require.NoError(t, restored.restoreParty(snapshot))
	assert.Equal(t, byte(255), restored.PartyState.GetCharacter(0).ArmorClass)
}
