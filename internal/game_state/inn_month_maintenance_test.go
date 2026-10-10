package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

func TestAdvanceGameplayMinutesMaintainsAllInnMonthCountersAtYearRollover(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.DateTime.Year, state.DateTime.Month, state.DateTime.Day = 7, 13, 28
	state.DateTime.Hour, state.DateTime.Minute = 23, 59
	values := []byte{0, 1, 24, 25, 0xff}
	for index := range party_state.NRosterSlots {
		state.PartyState.GetCharacter(index).MonthsAtInn = values[index%len(values)]
	}

	elapsed := state.AdvanceGameplayMinutes(1)

	assert.Equal(t, 1, elapsed)
	assert.Equal(t, uint16(8), state.DateTime.Year)
	assert.Equal(t, byte(1), state.DateTime.Month)
	assert.Equal(t, byte(1), state.DateTime.Day)
	for index := range party_state.NRosterSlots {
		before := values[index%len(values)]
		want := before
		if want < 25 {
			want++
		}
		assert.Equal(t, want, state.PartyState.GetCharacter(index).MonthsAtInn, "slot %d", index)
	}
}

func TestAdvanceGameplayMinutesDoesNotMaintainInnMonthsBeforeBoundary(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.DateTime.Year, state.DateTime.Month, state.DateTime.Day = 7, 13, 28
	state.DateTime.Hour, state.DateTime.Minute = 23, 58
	for index := range party_state.NRosterSlots {
		state.PartyState.GetCharacter(index).MonthsAtInn = byte(index)
	}

	state.AdvanceGameplayMinutes(1)

	for index := range party_state.NRosterSlots {
		assert.Equal(t, byte(index), state.PartyState.GetCharacter(index).MonthsAtInn, "slot %d", index)
	}
}

func TestMaintainInnMonthsAppliesEveryCrossedBoundaryWithSaturation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	values := []byte{0, 24, 25, 0xff}
	for index := range party_state.NRosterSlots {
		state.PartyState.GetCharacter(index).MonthsAtInn = values[index%len(values)]
	}

	state.maintainInnMonths(
		datetime.UltimaDate{Year: 1, Month: 12, Day: 28},
		datetime.UltimaDate{Year: 2, Month: 1, Day: 1},
	)

	want := []byte{2, 25, 25, 0xff}
	for index := range party_state.NRosterSlots {
		assert.Equal(t, want[index%len(want)], state.PartyState.GetCharacter(index).MonthsAtInn, "slot %d", index)
	}
}
