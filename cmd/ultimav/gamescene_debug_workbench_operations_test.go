package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestOperationalStatusQueriesReturnAuthoritativeStructuredState(t *testing.T) {
	state := newSpellWorkbenchTestState(t)
	state.PartyState.Karma.Value = 60
	state.ShadowlordState.Kill(references.Faulinei)
	state.CurrentNPCAIController = nil
	scene := &GameScene{gameState: state}

	guards, handled := scene.executeOperationalDebugCommand(debugging.Request{
		SessionID: "test", ID: "guards", CommandID: debugging.CommandGuardsStatus,
	})
	require.True(t, handled)
	require.True(t, guards.Success)
	var guardData guardStatusResult
	require.NoError(t, json.Unmarshal(guards.Data, &guardData))
	assert.False(t, guardData.Supported)
	assert.NotEmpty(t, guardData.UnavailableReason)

	shadowlords, _ := scene.executeOperationalDebugCommand(debugging.Request{
		SessionID: "test", ID: "shadowlords", CommandID: debugging.CommandShadowlordStatus,
	})
	var shadowlordData shadowlordStatusResult
	require.NoError(t, json.Unmarshal(shadowlords.Data, &shadowlordData))
	assert.Equal(t, 2, shadowlordData.AliveCount)
	assert.Len(t, shadowlordData.Shadowlords, references.ShadowlordCount)
	assert.False(t, shadowlordData.Shadowlords[references.Faulinei].Alive)
	assert.True(t, shadowlordData.Shadowlords[references.Astaroth].Alive)
	assert.True(t, shadowlordData.Shadowlords[references.Nosfentor].Alive)

	death, _ := scene.executeOperationalDebugCommand(debugging.Request{
		SessionID: "test", ID: "death", CommandID: debugging.CommandDeathStatus,
	})
	var deathData deathStatusResult
	require.NoError(t, json.Unmarshal(death.Data, &deathData))
	assert.Equal(t, 60, deathData.ExperienceRetainedPercent)
	assert.Equal(t, 40, deathData.ExperienceLostPercent)
	assert.Equal(t, game_state.ResurrectionLocationX, deathData.Resurrection.X)
	assert.Equal(t, game_state.ResurrectionLocationY, deathData.Resurrection.Y)
}

func TestDeathStatusReportsXPPreservationAtTopKarmaWithoutWireChanges(t *testing.T) {
	state := newSpellWorkbenchTestState(t)
	state.PartyState.Karma.Value = 98

	result := newDeathStatusResult(state)

	assert.Equal(t, 100, result.ExperienceRetainedPercent)
	assert.Zero(t, result.ExperienceLostPercent)
}

func TestDebugOptionCommandsUseExplicitIdempotentState(t *testing.T) {
	state := newSpellWorkbenchTestState(t)
	scene := &GameScene{gameState: state}
	request := debugging.Request{
		SessionID: "test", ID: "free-move", CommandID: debugging.CommandMovementFreeMoveSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	}
	first, handled := scene.executeOperationalDebugCommand(request)
	require.True(t, handled)
	require.True(t, first.Success)
	require.Len(t, first.Changes, 1)
	assert.True(t, state.DebugOptions.FreeMove)

	second, _ := scene.executeOperationalDebugCommand(request)
	require.True(t, second.Success)
	assert.Empty(t, second.Changes)
	assert.True(t, state.DebugOptions.FreeMove)

	unlimitedRequest := debugging.Request{
		SessionID: "test", ID: "unlimited-casting", CommandID: debugging.CommandUnlimitedCastingResourcesSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	}
	unlimited, handled := scene.executeOperationalDebugCommand(unlimitedRequest)
	require.True(t, handled)
	require.True(t, unlimited.Success)
	assert.True(t, state.DebugOptions.UnlimitedCastingResources)

	levelRequest := debugging.Request{
		SessionID: "test", ID: "spell-level-bypass", CommandID: debugging.CommandSpellLevelRestrictionsBypassSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	}
	level, handled := scene.executeOperationalDebugCommand(levelRequest)
	require.True(t, handled)
	require.True(t, level.Success)
	assert.True(t, state.DebugOptions.BypassSpellLevelRestrictions)

	status, _ := scene.executeOperationalDebugCommand(debugging.Request{
		SessionID: "test", ID: "debug-status", CommandID: debugging.CommandDebugOptionsStatus,
	})
	var statusData debugOptionsStatusResult
	require.NoError(t, json.Unmarshal(status.Data, &statusData))
	assert.True(t, statusData.UnlimitedCastingResources)
	assert.True(t, statusData.BypassSpellLevelRestrictions)
}
