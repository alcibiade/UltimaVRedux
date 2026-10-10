package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlterShrineProgressNormalizesQuestFlagsWithoutRewards(t *testing.T) {
	gs := &GameState{SystemCallbacks: journalTestCallbacks()}
	ready := ShrineProgressReadyToComplete
	ruined := true

	result, err := gs.AlterShrineProgress(ShrineProgressAlteration{
		Index:      3,
		QuestState: &ready,
		Ruined:     &ruined,
	})

	require.NoError(t, err)
	assert.Equal(t, ShrineProgressNotAssigned, result.Before.QuestState)
	assert.Equal(t, ShrineProgressReadyToComplete, result.After.QuestState)
	assert.True(t, result.After.Completed)
	assert.True(t, result.After.Active)
	assert.True(t, result.After.Ruined)
}

func TestAlterShrineProgressOfferingClearsActiveFlag(t *testing.T) {
	gs := &GameState{SystemCallbacks: journalTestCallbacks()}
	gs.SetShrineQuestCompleted(1, true)
	gs.SetShrineQuestActive(1, true)
	offering := ShrineProgressOffering

	result, err := gs.AlterShrineProgress(ShrineProgressAlteration{Index: 1, QuestState: &offering})

	require.NoError(t, err)
	assert.True(t, result.After.Completed)
	assert.False(t, result.After.Active)
	assert.Equal(t, ShrineProgressOffering, result.After.QuestState)
}

func TestAlterShrineProgressPreservesAssignedSeekingCodexState(t *testing.T) {
	gs := &GameState{SystemCallbacks: journalTestCallbacks()}
	assigned := ShrineProgressAssigned

	result, err := gs.AlterShrineProgress(ShrineProgressAlteration{Index: 4, QuestState: &assigned})

	require.NoError(t, err)
	assert.False(t, result.After.Completed)
	assert.True(t, result.After.Active)
	assert.Equal(t, ShrineProgressAssigned, result.After.QuestState)
}

func TestAlterShrineProgressRejectsEmptyAlteration(t *testing.T) {
	gs := &GameState{}

	_, err := gs.AlterShrineProgress(ShrineProgressAlteration{Index: 0})

	require.Error(t, err)
}
