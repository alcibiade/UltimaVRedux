package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestExecuteScenarioStatusAndStarvation(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	scene := &GameScene{gameState: state} //nolint:exhaustruct // Focused Scenario Lab fixture.

	status, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "status", CommandID: debugging.CommandScenarioStatus,
	})
	require.True(t, handled)
	require.True(t, status.Success)
	var data scenarioStatusResult
	require.NoError(t, json.Unmarshal(status.Data, &data))
	assert.Equal(t, "Britain", data.LocationName)
	assert.NotEmpty(t, data.Party)

	food := state.PartyState.Inventory.Provisions.Food.Get()
	starvation, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "starvation", CommandID: debugging.CommandScenarioStarvationTrigger,
		Arguments: map[string]json.RawMessage{"seed": json.RawMessage(`12345`)},
	})
	require.True(t, handled)
	require.True(t, starvation.Success)
	assert.Equal(t, food, state.PartyState.Inventory.Provisions.Food.Get())
	assert.NotEmpty(t, starvation.Changes)
}

func TestExecuteScenarioGuardActivationReturnsUpdatedStatus(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	scene := &GameScene{gameState: state} //nolint:exhaustruct // Focused Scenario Lab fixture.

	result, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "guards", CommandID: debugging.CommandScenarioGuardsActivate,
	})

	require.True(t, handled)
	require.True(t, result.Success)
	assert.True(t, state.IsCriminal())
	assert.Contains(t, stateChangePaths(result.Changes), "party.criminal")
}

func TestExecuteScenarioWorldSetupsReturnSynchronizedChanges(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	scene := &GameScene{gameState: state} //nolint:exhaustruct // Focused Scenario Lab fixture.

	waterfall, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "waterfall", CommandID: debugging.CommandScenarioWaterfallSetup,
	})
	require.True(t, handled)
	require.True(t, waterfall.Success)
	assert.Equal(t, debugging.ExecutionSynchronized, waterfall.EffectiveExecutionMode)
	assert.Equal(t, references.Position{X: 54, Y: 135}, state.MapState.PlayerLocation.Position)
	assert.Contains(t, stateChangePaths(waterfall.Changes), "world.position.y")

	word, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "word", CommandID: debugging.CommandScenarioWordSetup,
		Arguments: map[string]json.RawMessage{"word": json.RawMessage(`"veramocor"`)},
	})
	require.True(t, handled)
	require.True(t, word.Success)
	var data wordOfPowerSetupDebugResult
	require.NoError(t, json.Unmarshal(word.Data, &data))
	assert.Equal(t, "VERAMOCOR", data.Word)
	assert.Equal(t, references.Doom.String(), data.DungeonName)
	assert.Equal(t, 1, state.MapState.PlayerLocation.Position.HeuristicTileDistance(data.Entrance))
}

func TestExecuteScenarioBridgeTrollStartsProductionSneakAndAttackWorkflow(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Gold.Set(100)
	scene := &GameScene{gameState: state, keyboard: input.NewKeyboard(0)} //nolint:exhaustruct // Focused Scenario Lab fixture.

	result, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "bridge-troll", CommandID: debugging.CommandScenarioBridgeTroll,
		Arguments: map[string]json.RawMessage{"seed": json.RawMessage(`95`)},
	})

	require.True(t, handled)
	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.True(t, state.EventQueueManager.HasWorkflow("bridge_troll_encounter"))
	assert.True(t, state.GetCurrentLayeredMapAvatarTopTile().IsTrollBridge())
	assert.Equal(t, uint16(0), state.PartyState.Inventory.Gold.Get())
	assert.Contains(t, stateChangePaths(result.Changes), "inventory.gold")
	var data bridgeTrollSetupDebugResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.True(t, data.Queued)
	assert.Equal(t, uint64(95), data.Seed)
	assert.Equal(t, byte(0), data.DexterityAfter)
}

func TestExecuteScenarioBlackthornInterrogationStartsProductionWorkflow(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.PalaceOfBlackthorn).
		WithSystemCallbacks().
		Build()
	configuration := finaleMapFixtureConfig(t)
	state.GameConfig = configuration
	state.SetShrineRuined(0, true)
	scene := &GameScene{gameState: state, keyboard: input.NewKeyboard(0)} //nolint:exhaustruct // Focused scripted-scene fixture.

	result, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "blackthorn", CommandID: debugging.CommandScenarioBlackthornInterrogation,
		Arguments: map[string]json.RawMessage{
			"expectedPartyFingerprint": json.RawMessage(`"` + state.CurrentPartyRosterSnapshot().Fingerprint + `"`),
		},
	})

	require.True(t, handled)
	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.True(t, state.EventQueueManager.HasWorkflow("blackthorn_interrogation"))
	assert.True(t, state.HasActiveBlackthornInterrogation())
	assert.Equal(t, references.BlackthornPrison, state.MapState.PlayerLocation.Location)
	assert.False(t, state.IsShrineRuined(0))
	assert.Contains(t, stateChangePaths(result.Changes), "quest.shrines.0.ruined")
}

func TestStartDebugLordBritishFinaleStartsProductionTerminalWorkflow(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	configuration := finaleMapFixtureConfig(t)
	state.GameConfig = configuration
	state.PartyState.Inventory.QuestItems.Set(references.WoodenBox, 3)
	companion := state.PartyState.GetCharacter(1)
	companion.SetStatus('D')
	companion.SetHp(0)
	scene := &GameScene{gameState: state, keyboard: input.NewKeyboard(0)} //nolint:exhaustruct // Focused scripted-scene fixture.
	request := debugging.Request{
		SessionID: "test", ID: "finale", CommandID: debugging.CommandScenarioLordBritishFinale,
		Arguments: map[string]json.RawMessage{
			"expectedPartyFingerprint": json.RawMessage(`"` + state.CurrentPartyRosterSnapshot().Fingerprint + `"`),
			"woodenBox":                json.RawMessage(`false`),
		},
	}

	result := scene.startDebugLordBritishFinale(request, finaleAssetFixture())

	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.True(t, state.EventQueueManager.HasWorkflow(lordBritishFinaleWorkflow))
	assert.Equal(t, references.FinaleLocation, state.MapState.PlayerLocation.Location)
	assert.False(t, state.PartyState.Inventory.QuestItems.HasSome(references.WoodenBox))
	assert.Contains(t, stateChangePaths(result.Changes), "inventory.questItems.woodenBox")
	assert.Contains(t, stateChangePaths(result.Changes), "party.character."+companion.GetNameAsString()+".status")
	var data lordBritishFinaleDebugResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.True(t, data.Queued)
	assert.Contains(t, data.Revived, companion.GetNameAsString())
}

func TestScenarioMutationsRejectStalePartyAndTerminalFinale(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Doom).
		WithFloor(7).
		WithPlayerAt(5, 7).
		WithSystemCallbacks().
		Build()
	configuration := finaleMapFixtureConfig(t)
	state.GameConfig = configuration
	for index := 1; index < party_state.MaxActiveParty; index++ {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	scene := &GameScene{gameState: state, keyboard: input.NewKeyboard(0)} //nolint:exhaustruct // Focused scenario boundary fixture.

	stale, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "stale", CommandID: debugging.CommandScenarioBlackthornInterrogation,
		Arguments: map[string]json.RawMessage{"expectedPartyFingerprint": json.RawMessage(`"stale"`)},
	})
	require.True(t, handled)
	assert.False(t, stale.Success)
	assert.Equal(t, "stale_party", stale.ErrorCode)
	assert.False(t, state.HasActiveBlackthornInterrogation())

	installVerifiedDoomMirrorCombat(t, state)
	state.FinishCombatAction()
	require.True(t, state.HasPendingLordBritishFinale())
	pending, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "pending", CommandID: debugging.CommandScenarioStarvationTrigger,
	})
	require.True(t, handled)
	assert.False(t, pending.Success)
	assert.Equal(t, "finale_active", pending.ErrorCode)
	status := newScenarioStatusResult(scene)
	assert.True(t, status.FinaleActive)
	assert.False(t, status.GameplayTerminal)

	_, err := state.BeginLordBritishFinale()
	require.NoError(t, err)
	require.True(t, state.SubmitLordBritishFinaleAnswer('Y').Accepted)
	require.True(t, state.IsGameplayTerminal())
	blocked, handled := scene.executeScenarioDebugCommand(debugging.Request{
		SessionID: "test", ID: "terminal", CommandID: debugging.CommandScenarioStarvationTrigger,
	})
	require.True(t, handled)
	assert.False(t, blocked.Success)
	assert.Equal(t, "gameplay_terminal", blocked.ErrorCode)
}
