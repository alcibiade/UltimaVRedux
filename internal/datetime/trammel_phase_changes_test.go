package datetime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrammelPhaseChangesCountsEachDayTrammelChanges(t *testing.T) {
	brute := 0
	for day := 0; day <= DaysInMonth*3+5; day++ {
		if day > 0 && moonPhases[day%DaysInMonth][0] != moonPhases[(day-1)%DaysInMonth][0] {
			brute++
		}
		assert.Equal(t, brute, TrammelPhaseChanges(day), "day %d", day)
	}
	assert.Equal(t, trammelPhaseChangesPerMonth, TrammelPhaseChanges(DaysInMonth), "one full month")
	assert.Zero(t, TrammelPhaseChanges(-4))
}

func TestTrammelPhaseChangesFollowTheCalendar(t *testing.T) {
	date := UltimaDate{Year: 139, Month: 4, Day: 1}
	start := TrammelPhaseChanges(date.GetTotalDays())
	date.Month = 5
	assert.Equal(t, start+trammelPhaseChangesPerMonth, TrammelPhaseChanges(date.GetTotalDays()))
}
