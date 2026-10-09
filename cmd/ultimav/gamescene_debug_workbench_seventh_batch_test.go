package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestDebugSetAllShrineQuestStateReturnsExactChangesAndSemanticPaths(t *testing.T) {
	scene := newDeathDebugTestScene(t)
	scene.gameState.SetShrineQuestActive(0, true)
	review := scene.gameState.ReviewAllShrineQuestState()

	result := scene.ExecuteDebugCommand(allShrineQuestStateRequest(game_state.AllShrineQuestCompleted, review.Fingerprint))

	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	var data allShrineQuestStateDebugResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.Equal(t, string(game_state.AllShrineQuestCompleted), data.TargetState)
	assert.Equal(t, review.Fingerprint, data.BeforeFingerprint)
	assert.NotEqual(t, data.BeforeFingerprint, data.AfterFingerprint)
	require.Len(t, data.Changes, game_state.TotalShrines)
	assert.Equal(t, "Honesty", data.Changes[0].VirtueName)
	assert.Equal(t, "hone", data.Changes[0].VirtueKey)
	assert.Equal(t, game_state.ShrineProgressAssigned, game_state.ShrineProgressState(data.Changes[0].Before.QuestState))
	assert.Equal(t, game_state.ShrineProgressOffering, game_state.ShrineProgressState(data.Changes[0].After.QuestState))
	assert.Equal(t, "shrines.hone.questState", result.Changes[0].Path)
	assert.Equal(t, "shrines.hone.completed", result.Changes[1].Path)
	assert.Equal(t, "shrines.hone.active", result.Changes[2].Path)
	assert.Len(t, result.Changes, 17)

	for index := range game_state.TotalShrines {
		progress := scene.gameState.ShrineProgress(index)
		assert.True(t, progress.Completed)
		assert.False(t, progress.Active)
		assert.Equal(t, game_state.ShrineProgressOffering, progress.QuestState)
	}
}

func TestDebugSetAllShrineQuestStateRejectsStaleReviewWithoutPartialMutation(t *testing.T) {
	scene := newDeathDebugTestScene(t)
	review := scene.gameState.ReviewAllShrineQuestState()
	scene.gameState.SetShrineQuestActive(3, true)
	before := scene.gameState.ReviewAllShrineQuestState()

	result := scene.ExecuteDebugCommand(allShrineQuestStateRequest(game_state.AllShrineQuestCompleted, review.Fingerprint))

	assert.False(t, result.Success)
	assert.Equal(t, "stale_shrine_quest_state", result.ErrorCode)
	assert.Equal(t, before.Shrines, scene.gameState.ReviewAllShrineQuestState().Shrines)
}

func TestDebugSetAllShrineQuestStateSupportsNoOp(t *testing.T) {
	scene := newDeathDebugTestScene(t)
	review := scene.gameState.ReviewAllShrineQuestState()

	result := scene.ExecuteDebugCommand(allShrineQuestStateRequest(game_state.AllShrineQuestNotAssigned, review.Fingerprint))

	require.True(t, result.Success, result.ErrorMessage)
	var data allShrineQuestStateDebugResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.Empty(t, data.Changes)
	assert.Empty(t, result.Changes)
	assert.Equal(t, data.BeforeFingerprint, data.AfterFingerprint)
}

func allShrineQuestStateRequest(target game_state.AllShrineQuestState, fingerprint string) debugging.Request {
	return debugging.Request{CommandID: debugging.CommandCodexQuestStateSetAll, Arguments: map[string]json.RawMessage{
		"targetState":         json.RawMessage(`"` + string(target) + `"`),
		"expectedFingerprint": json.RawMessage(`"` + fingerprint + `"`),
	}}
}
