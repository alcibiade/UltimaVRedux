package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombatTurnQueueFillForecastMatchesAdvancement(t *testing.T) {
	q := NewCombatTurnQueue()
	q.AddCombatant(newMockPC(22), 0)
	q.AddCombatant(newMockPC(18), 1)
	q.AddCombatant(newMockMonster(30), 0)
	q.AddCombatant(newMockMonster(15), 1)

	// Forecast from a non-zero cursor with countdowns already in progress.
	require.NotNil(t, q.AdvanceToNextActor())

	forecast := make([]CombatTurnForecastEntry, 6)
	filled := q.FillForecast(forecast)
	require.Equal(t, len(forecast), filled)

	for i, predicted := range forecast {
		actual := q.AdvanceToNextActor()
		require.NotNil(t, actual, "forecast action %d", i)
		assert.Equal(t, CombatTurnIdentity{
			IsPartyMember: actual.Combatant.IsPC(),
			EntityIndex:   actual.EntityIndex,
		}, predicted.CombatTurnIdentity, "forecast action %d", i)
		assert.Equal(t, actual.Combatant.GetDexterity(), predicted.Dexterity)
		assert.Equal(t, actual.turnCountdown, predicted.Countdown)
		assert.True(t, predicted.CanTakeTurn)
		assert.Equal(t, CombatTurnActive, predicted.Status)
	}
}

func TestCombatTurnQueueFillForecastIncludesHighDexRepeats(t *testing.T) {
	q := NewCombatTurnQueue()
	q.AddCombatant(newMockPC(10), 0)
	q.AddCombatant(newMockMonster(35), 4)

	forecast := make([]CombatTurnForecastEntry, 5)
	require.Equal(t, 5, q.FillForecast(forecast))

	for _, predicted := range forecast {
		assert.Equal(t, CombatTurnIdentity{IsPartyMember: false, EntityIndex: 4}, predicted.CombatTurnIdentity)
		assert.Equal(t, 35, predicted.Dexterity)
		assert.Equal(t, 1, predicted.Countdown)
	}
}

func TestCombatTurnQueueFillForecastSkipsDeadAndFledButIncludesSleeping(t *testing.T) {
	q := NewCombatTurnQueue()
	q.AddCombatant(newMockPC(35), 0)
	q.AddCombatant(newMockPC(34), 1)
	ineligible := newMockMonster(35)
	ineligible.canTakeTurn = false
	q.AddCombatant(ineligible, 0)
	q.AddCombatant(newMockMonster(12), 1)
	q.AddCombatant(newMockMonster(20), 2)
	q.MarkDead(0, true)
	q.MarkFled(1, true)
	q.MarkDead(1, false)

	forecast := make([]CombatTurnForecastEntry, 4)
	require.Equal(t, 4, q.FillForecast(forecast))

	for i, predicted := range forecast {
		assert.Equal(t, CombatTurnIdentity{IsPartyMember: false, EntityIndex: 0}, predicted.CombatTurnIdentity)
		assert.False(t, predicted.CanTakeTurn)
		actual := q.AdvanceToNextActor()
		require.NotNil(t, actual, "forecast action %d", i)
		assert.Equal(t, predicted.EntityIndex, actual.EntityIndex)
		assert.False(t, actual.Combatant.IsPC())
	}
}

func TestCombatTurnQueueFillForecastEmptyAndNoEligibleActors(t *testing.T) {
	q := NewCombatTurnQueue()
	forecast := make([]CombatTurnForecastEntry, 5)
	assert.Zero(t, q.FillForecast(forecast))

	q.AddCombatant(newMockPC(20), 0)
	q.MarkDead(0, true)
	assert.Zero(t, q.FillForecast(forecast))
	assert.Zero(t, q.FillForecast(nil))
}

func TestCombatTurnQueueFillForecastDoesNotMutateQueue(t *testing.T) {
	q := NewCombatTurnQueue()
	q.AddCombatant(newMockPC(20), 0)
	q.AddCombatant(newMockMonster(30), 0)
	require.NotNil(t, q.AdvanceToNextActor())

	countdowns := make([]int, len(q.entries))
	statuses := make([]turnQueueEntryStatus, len(q.entries))
	for i, entry := range q.entries {
		countdowns[i] = entry.turnCountdown
		statuses[i] = entry.status
	}
	cursor := q.currentIndex
	actionCount := q.actionCount

	forecast := make([]CombatTurnForecastEntry, 17)
	assert.Equal(t, len(forecast), q.FillForecast(forecast))
	assert.Equal(t, cursor, q.currentIndex)
	assert.Equal(t, actionCount, q.actionCount)
	for i, entry := range q.entries {
		assert.Equal(t, countdowns[i], entry.turnCountdown, "entry %d countdown", i)
		assert.Equal(t, statuses[i], entry.status, "entry %d status", i)
	}
}
