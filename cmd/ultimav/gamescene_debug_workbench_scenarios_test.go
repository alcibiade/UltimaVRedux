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

func TestExecuteScenarioStatusReportsActiveContext(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	scene := &GameScene{gameState: state}

	status, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "status", CommandID: debugging.CommandScenarioStatus,
	})
	require.True(t, handled)
	require.True(t, status.Success)
	var data scenarioStatusResult
	require.NoError(t, json.Unmarshal(status.Data, &data))
	assert.Equal(t, "Britain", data.LocationName)
	assert.NotEmpty(t, data.Party)
	assert.Equal(t, state.CurrentPartyRosterSnapshot().Fingerprint, data.PartyFingerprint)
	assert.False(t, data.FinaleActive)
	assert.False(t, data.GameplayTerminal)
}
