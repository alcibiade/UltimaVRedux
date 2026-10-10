package party_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestLeaveCharacterAtInnPreservesCompleteRecordOrder(t *testing.T) {
	party := innRosterTestParty()
	selected := party.GetCharacter(2)
	selected.MonthsAtInn = 9
	before := innRosterNames(party)

	party.LeaveCharacterAtInn(2, references.Britain)

	want := append(append([]string(nil), before[:2]...), before[3:]...)
	want = append(want, before[2])
	assert.Equal(t, want, innRosterNames(party))
	assert.Same(t, selected, party.GetCharacter(NRosterSlots-1))
	assert.Equal(t, PartyMapAtInn(references.Britain), selected.PartyMap)
	assert.Zero(t, selected.MonthsAtInn)
}

func TestPickUpCharacterFromInnPreservesCompleteRecordOrder(t *testing.T) {
	party := innRosterTestParty()
	for index := 3; index < NRosterSlots; index++ {
		party.GetCharacter(index).PartyMap = PartyMapUnrecruited
	}
	guest := party.GetCharacter(9)
	guest.PartyMap = PartyMapAtInn(references.Britain)
	before := innRosterNames(party)

	inserted := party.PickUpCharacterFromInn(9)

	want := append([]string(nil), before[:3]...)
	want = append(want, before[9])
	want = append(want, before[3:9]...)
	want = append(want, before[10:]...)
	assert.Equal(t, 3, inserted)
	assert.Equal(t, want, innRosterNames(party))
	assert.Same(t, guest, party.GetCharacter(3))
	assert.Equal(t, PartyMapActive, guest.PartyMap)
}

func innRosterTestParty() *PartyState {
	var roster [NRosterSlots]*PlayerCharacter
	for index := range roster {
		character := &PlayerCharacter{PartyMap: PartyMapActive, Strength: byte(index + 1)} //nolint:gosec // Fixed test range.
		copy(character.Name[:], fmt.Sprintf("C%02d", index))
		roster[index] = character
	}

	return NewPartyStateFromRoster(roster)
}

func innRosterNames(party *PartyState) []string {
	names := make([]string, NRosterSlots)
	for index := range names {
		names[index] = party.GetCharacter(index).GetNameAsString()
	}

	return names
}
