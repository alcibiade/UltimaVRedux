package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestScenarioCommandsReadyRejectsActiveCombat(t *testing.T) {
	state := newApprovedToolTestState()
	state.EventQueueManager = event_queue.NewEventQueueManager(func() int64 { return 0 }, nil)
	scene := &GameScene{gameState: state}
	assert.True(t, scene.ScenarioCommandsReady())

	state.CombatScene = &game_state.CombatState{}
	state.MapState.PlayerLocation.Location = references.CombatLocation

	assert.False(t, scene.ScenarioCommandsReady())
}
