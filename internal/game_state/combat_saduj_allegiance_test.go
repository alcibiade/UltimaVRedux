package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatSadujUsesCanonicalIdentityAcrossSourceAndClone(t *testing.T) {
	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	for index := range roster {
		roster[index] = &party_state.PlayerCharacter{PartyMap: party_state.PartyMapUnrecruited}
	}
	activate := func(index int, name string) {
		character := roster[index]
		character.PartyMap = party_state.PartyMapActive
		copy(character.Name[:], name)
		character.SetMaxHp(30)
		character.SetHp(30)
		character.SetStatus(party_state.Good)
	}
	activate(party_state.AvatarCharacterIndex, "Saduj")
	activate(1, "Saduj")
	activate(2, "Abcdj")
	party := party_state.NewPartyStateFromRoster(roster)
	combatMap := references.NewCombatMapReference(0, references.BritanniaCombatTerritory)
	combat := NewCombatState(combatMap, CombatEntryNorth, party)
	state := &GameState{PartyState: *party, CombatScene: combat} //nolint:exhaustruct // Focused allegiance fixture.

	avatar := CombatTurnIdentity{IsPartyMember: true, EntityIndex: party_state.AvatarCharacterIndex}
	saduj := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	control := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 2}
	assert.Equal(t, CombatMonsterLoyaltyParty, state.combatIdentityLoyalty(avatar), "Avatar is exempt even if renamed")
	assert.Equal(t, CombatMonsterLoyaltyMonster, state.combatIdentityLoyalty(saduj))
	assert.Equal(t, CombatMonsterLoyaltyParty, state.combatIdentityLoyalty(control), "fifth-character heuristic is not retained")

	clone := combat.addPCCloneFromSource(saduj, saduj.EntityIndex, references.Position{X: 4, Y: 4}, NewCombatPCProvider(roster[1]))
	require.True(t, clone.IsPartyMember)
	assert.Equal(t, CombatMonsterLoyaltyMonster, state.combatIdentityLoyalty(clone))

	combat.setCombatPCPossessed(control, true)
	assert.Equal(t, CombatMonsterLoyaltyMonster, state.combatIdentityLoyalty(control), "possession remains an independent hostile input")
	partyLoyal, monsterLoyal := state.countActiveCombatLoyalties()
	assert.Equal(t, 1, partyLoyal)
	assert.Equal(t, 3, monsterLoyal)
}
