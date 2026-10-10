package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatNumberKeysRejectDepartedCharacters(t *testing.T) {
	t.Run("hostiles remain", func(t *testing.T) {
		assertCombatNumberKeysRejectDepartedCharacters(t, false)
	})
	t.Run("after victory", func(t *testing.T) {
		assertCombatNumberKeysRejectDepartedCharacters(t, true)
	})
}

func assertCombatNumberKeysRejectDepartedCharacters(t *testing.T, afterVictory bool) {
	t.Helper()
	scene, actor := setupCombatAutoAttackInputTest(t)
	state := scene.gameState
	if afterVictory {
		state.CombatScene.MonsterManager.Clear()
		state.CombatScene.TurnQueue.RemoveAllMonsters()
		state.CombatScene.Victory = true
	}
	for _, index := range []int{4, 5} {
		require.True(t, state.CombatScene.PartyActive[index])
		require.NotEqual(t, actor.EntityIndex, index)
		// Exercise real individual departure before returning input to the actor.
		state.CombatScene.CurrentTurnIdentity = game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: index}
		state.CombatScene.CurrentTurnIndex = index
		state.CombatScene.SetPartyMemberPosition(index, references.Position{X: 0, Y: 5})
		require.Equal(t, game_state.CombatMoveFled, state.MoveCombatCharacter(references.Left))
	}
	state.CombatScene.CurrentTurnIdentity = actor
	state.CombatScene.CurrentTurnIndex = actor.EntityIndex
	require.True(t, state.SetActivePlayer(actor.EntityIndex))
	forecast := make([]game_state.CombatTurnForecastEntry, 6)
	state.CombatScene.TurnQueue.FillForecast(forecast)
	randomCalls := state.GetRNG().Calls()

	// These are consecutive calls through the production numeric-key router,
	// with no queue update or artificial debounce delay between them.
	require.True(t, scene.handleNumberKey(ebiten.Key5))
	require.True(t, scene.handleNumberKey(ebiten.Key6))

	assert.Equal(t, actor, state.CombatScene.GetCurrentTurnIdentity())
	assert.False(t, state.CombatScene.IsSoloMoveActive())
	assert.False(t, state.CombatScene.HasScheduledCombatStep())
	assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
	assert.Equal(t, randomCalls, state.GetRNG().Calls())
	after := make([]game_state.CombatTurnForecastEntry, len(forecast))
	state.CombatScene.TurnQueue.FillForecast(after)
	assert.Equal(t, forecast, after)
}
