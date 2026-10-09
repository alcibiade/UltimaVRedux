package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/scenarios"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
)

func TestScenarioRunnerPersistsResolvedLaunchBeforeAnyStep(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "launch", Name: "Launch", Description: "Retain all accepted inputs.",
		Baseline: scenarios.CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{
			{ID: "enabled", Name: "Enabled", Kind: debugging.ParameterBoolean, Default: json.RawMessage(`false`)},
			{ID: "count", Name: "Count", Kind: debugging.ParameterInteger, Required: true},
			{ID: "unused", Name: "Unused", Kind: debugging.ParameterString, Default: json.RawMessage(`"default"`)},
		},
		Steps: []scenarios.Step{{Type: scenarios.StepCommand, CommandID: debugging.CommandMovementFreeMoveSet,
			Arguments: map[string]scenarios.CommandArgument{"enabled": {Parameter: "enabled"}}}},
	})
	scene := &scenarioRunnerTestScene{rng: &scenarios.RunRNG{Seed: "18446744073709551615", SourceCalls: "9007199254740993"}}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	history, err := scenarios.NewRunHistoryRepository(t.TempDir(), 3)
	require.NoError(t, err)
	game.ConfigureScenarioRunHistory(history)
	missing := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:launch"))
	require.Equal(t, scenarioRunParametersError, missing.ErrorCode)
	assert.Nil(t, game.scenarioRun)
	result := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequestWithValues(t, game, repository, "project:launch", `{"count":0}`))
	require.True(t, result.Success, result.ErrorMessage)
	var payload scenarioRunStatusResult
	require.NoError(t, json.Unmarshal(result.Data, &payload))
	require.NotNil(t, payload.Run.Launch)
	values, err := json.Marshal(payload.Run.Launch.ParameterValues)
	require.NoError(t, err)
	assert.JSONEq(t, `{"enabled":false,"count":0,"unused":"default"}`, string(values))
	assert.True(t, payload.Run.Launch.BaselineApplied)
	assert.Equal(t, scene.rng, payload.Run.Launch.RNG)
	scene.rng.Seed = "1"
	assert.Equal(t, "18446744073709551615", game.scenarioRun.status.Launch.RNG.Seed)
	assert.Empty(t, scene.executed)
	assert.Zero(t, scene.captures, "RNG observation must not capture a save or require save eligibility")
	require.True(t, game.ExecuteLocalDebugCommand(scenarioCancelRequest(game.scenarioRun.status.RunID)).Success)
	runs, err := history.Load()
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, payload.Run.Launch, runs[0].Launch)
}

func TestScenarioRunnerCancellationAtDispatchRetainsProgressAndWritesHistoryOnce(t *testing.T) {
	for _, test := range []struct {
		name         string
		ready        bool
		priorUpdates int
		executed     int
	}{
		{name: "queued", executed: 0},
		{name: "waiting", priorUpdates: 1, executed: 0},
		{name: "runner advances before bridge", ready: true, executed: 1},
		{name: "waiting after completed effect", ready: true, priorUpdates: 1, executed: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := newScenarioRunnerTestRepository(t)
			saveScenarioRunnerDefinition(t, repository, cancellationTestDefinition())
			scene := &scenarioRunnerTestScene{ready: test.ready}
			game := NewGame(scene)
			game.ConfigureScenarioRunner(repository, nil)
			historyRoot := t.TempDir()
			history, err := scenarios.NewRunHistoryRepository(historyRoot, 3)
			require.NoError(t, err)
			game.ConfigureScenarioRunHistory(history)
			game.scenarioNow = func() time.Time { return time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC) }
			require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:cancel")).Success)
			for range test.priorUpdates {
				require.NoError(t, game.Update())
			}
			if test.priorUpdates > 0 {
				scene.ready = false
			}
			cancel := scenarioCancelRequest(game.scenarioRun.status.RunID)
			// Scene changes do not invalidate run identity; cancellation is application-scoped.
			cancel.ExpectedSceneGeneration = 999
			require.NoError(t, game.SubmitDebugRequest(cancel))
			require.NoError(t, game.Update())
			result, ok := game.NextDebugResult()
			require.True(t, ok)
			require.True(t, result.Success, result.ErrorMessage)
			assert.Equal(t, scenarioRunCancelled, game.scenarioRun.status.State)
			assert.Len(t, scene.executed, test.executed)
			assert.Equal(t, test.executed+1, game.scenarioRun.status.NextStep)
			assert.Equal(t, "pending", game.scenarioRun.status.Steps[1].State)
			if test.executed > 0 {
				assert.Equal(t, "succeeded", game.scenarioRun.status.Steps[0].State)
				require.NotNil(t, game.scenarioRun.status.Steps[0].Result)
			}
			assert.True(t, game.scenarioRun.historyPersisted)
			runs, err := history.Load()
			require.NoError(t, err)
			require.Len(t, runs, 1)
			require.NotNil(t, runs[0].Launch.ParameterValues)
			assert.Empty(t, runs[0].Launch.ParameterValues)
			assert.True(t, runs[0].Launch.BaselineApplied)
			assert.Nil(t, runs[0].Launch.RNG, "unavailable RNG must not look like seed zero")
			path := filepath.Join(historyRoot, "runs.json")
			before, err := os.Stat(path)
			require.NoError(t, err)
			finishedAt := game.scenarioRun.status.FinishedAt
			game.scenarioNow = func() time.Time { t.Fatal("terminal cancellation must not finish twice"); return time.Time{} }
			require.True(t, game.ExecuteLocalDebugCommand(cancel).Success)
			require.NoError(t, game.Update())
			after, err := os.Stat(path)
			require.NoError(t, err)
			assert.True(t, os.SameFile(before, after), "no second atomic history replacement")
			assert.Equal(t, finishedAt, game.scenarioRun.status.FinishedAt)
			assert.Len(t, scene.executed, test.executed)
			mutation := game.ExecuteLocalDebugCommand(debugging.Request{SessionID: "other", ID: "after-cancel",
				CommandID: debugging.CommandMovementFreeMoveSet, Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}})
			require.True(t, mutation.Success, mutation.ErrorMessage)
		})
	}
}

func TestScenarioRunnerCancelCannotAffectReplacementRun(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, cancellationTestDefinition())
	game := NewGame(&scenarioRunnerTestScene{})
	game.ConfigureScenarioRunner(repository, nil)
	history, err := scenarios.NewRunHistoryRepository(t.TempDir(), 3)
	require.NoError(t, err)
	game.ConfigureScenarioRunHistory(history)
	unknown := game.ExecuteLocalDebugCommand(scenarioCancelRequest("missing"))
	assert.Equal(t, scenarioRunCancelStaleError, unknown.ErrorCode)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:cancel")).Success)
	oldCancel := scenarioCancelRequest(game.scenarioRun.status.RunID)
	require.True(t, game.ExecuteLocalDebugCommand(oldCancel).Success)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:cancel")).Success)
	result := game.ExecuteLocalDebugCommand(oldCancel)
	assert.Equal(t, scenarioRunCancelStaleError, result.ErrorCode)
	assert.Equal(t, scenarioRunQueued, game.scenarioRun.status.State)
	runs, err := history.Load()
	require.NoError(t, err)
	require.Len(t, runs, 1, "replacement run remains process-local until terminal")
	assert.Equal(t, scenarioRunCancelled, runs[0].State)
	assert.NotEqual(t, runs[0].RunID, game.scenarioRun.status.RunID)
	restarted := NewGame(&scenarioRunnerTestScene{})
	restarted.ConfigureScenarioRunHistory(history)
	assert.Equal(t, scenarioRunCancelStaleError, restarted.ExecuteLocalDebugCommand(oldCancel).ErrorCode,
		"history does not grant cancellation authority after restart")
}

func TestScenarioRunnerCancelAfterLastStepPreservesSucceededTerminal(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	definition := cancellationTestDefinition()
	definition.Steps = definition.Steps[:1]
	saveScenarioRunnerDefinition(t, repository, definition)
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.ConfigureScenarioRunner(repository, nil)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:cancel")).Success)
	require.NoError(t, game.SubmitDebugRequest(scenarioCancelRequest(game.scenarioRun.status.RunID)))
	require.NoError(t, game.Update())
	result, ok := game.NextDebugResult()
	require.True(t, ok)
	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, scenarioRunSucceeded, game.scenarioRun.status.State)
}

func TestScenarioRunnerCancelledBaselineTruthAndRollbackHandoff(t *testing.T) {
	for _, test := range []struct {
		name         string
		baseline     scenarios.BaselineType
		apply        bool
		preserveFail bool
	}{
		{name: "cancel before INIT baseline", baseline: scenarios.BaselineInit},
		{name: "cancel after INIT baseline", baseline: scenarios.BaselineInit, apply: true},
		{name: "identity failure after committed baseline", baseline: scenarios.BaselineInit, apply: true, preserveFail: true},
		{name: "cancel before snapshot baseline", baseline: scenarios.BaselineSnapshot},
		{name: "cancel after snapshot baseline", baseline: scenarios.BaselineSnapshot, apply: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := newScenarioRunnerTestRepository(t)
			definition := cancellationTestDefinition()
			definition.Baseline = scenarios.Baseline{Type: test.baseline}
			target := snapshotCommandTestRecord(t, "project:cancel-baseline", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			if test.baseline == scenarios.BaselineSnapshot {
				definition.Baseline.Reference = target.Reference.String()
			}
			saveScenarioRunnerDefinition(t, repository, definition)
			old := &scenarioRunnerTestScene{}
			replacement := &scenarioRunnerTestScene{rng: &scenarios.RunRNG{Seed: "0", SourceCalls: "0"}}
			if test.preserveFail {
				replacement.preserveError = errors.New("cannot preserve identity")
			}
			game := NewGame(old)
			snapshots := configureScenarioRunnerSnapshotRepository(game)
			if test.baseline == scenarios.BaselineSnapshot {
				target.Envelope.Snapshot.Random.Seed = ^uint64(0)
				target.Envelope.Snapshot.Random.SourceCalls = 9007199254740993
				snapshots.records[target.Reference.String()] = target
				game.ConfigureSnapshotRepository(snapshots, func(_ *config.UltimaVConfiguration, snapshot save_snapshot.SaveSnapshot) (Scene, error) {
					assert.Equal(t, target.Envelope.Snapshot.Random, snapshot.Random)
					replacement.rng = &scenarios.RunRNG{
						Seed: strconv.FormatUint(snapshot.Random.Seed, 10), SourceCalls: strconv.FormatUint(snapshot.Random.SourceCalls, 10),
					}

					return replacement, nil
				})
			}
			game.ConfigureScenarioRunner(repository, func() (Scene, error) { return replacement, nil })
			history, err := scenarios.NewRunHistoryRepository(t.TempDir(), 3)
			require.NoError(t, err)
			game.ConfigureScenarioRunHistory(history)
			require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:cancel")).Success)
			assert.False(t, game.scenarioRun.status.Launch.BaselineApplied)
			assert.Nil(t, game.scenarioRun.status.Launch.RNG)
			if test.apply {
				require.NoError(t, game.Update())
			}
			require.True(t, game.ExecuteLocalDebugCommand(scenarioCancelRequest(game.scenarioRun.status.RunID)).Success)
			assert.Equal(t, test.apply, game.scenarioRun.status.Launch.BaselineApplied)
			if test.baseline == scenarios.BaselineSnapshot {
				assert.Equal(t, target.Reference.String(), game.scenarioRun.status.BaselineReference)
				assert.Equal(t, target.Revision, game.scenarioRun.status.BaselineRevision)
			}
			if test.apply {
				assert.Same(t, replacement, game.currentScene, "never restore automatically")
				assert.Equal(t, replacement.rng, game.scenarioRun.status.Launch.RNG)
				rollback := game.scenarioRun.status.Rollback
				require.NotNil(t, rollback)
				stored, loadErr := snapshots.Load(rollback.Reference)
				require.NoError(t, loadErr)
				assert.Equal(t, stored.Revision, rollback.Revision)
			} else {
				assert.Same(t, old, game.currentScene)
				assert.Nil(t, game.scenarioRun.status.Rollback)
				assert.Zero(t, old.captures)
			}
			state := scenarioRunCancelled
			if test.preserveFail {
				state = scenarioRunFailed
			}
			assert.Equal(t, state, game.scenarioRun.status.State)
			runs, err := history.Load()
			require.NoError(t, err)
			require.Len(t, runs, 1)
			assert.Equal(t, game.scenarioRun.status.Rollback, runs[0].Rollback)
			assert.Equal(t, game.scenarioRun.status.Launch, runs[0].Launch)
		})
	}
}

func TestScenarioRunnerCancellationPreservesProductionWorkflowGate(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, cancellationTestDefinition())
	game := NewGame(&scenarioRunnerTestScene{})
	game.ConfigureScenarioRunner(repository, nil)
	state := newApprovedToolTestState()
	state.SetRandomSeed(^uint64(0))
	state.GetRNG().Intn(2)
	input := &scenarioWorkflowInput{}
	state.EventQueueManager = event_queue.NewEventQueueManager(func() int64 { return state.ElapsedMs }, input)
	scene := &GameScene{gameState: state}
	game.currentScene = scene
	finished := 0
	state.EventQueueManager.Workflow("production-wait").Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().First(func() {}).Wait(100).
		Finally(func() { finished++ }).Start()
	request := scenarioRunnerStartRequest(t, game, repository, "project:cancel")
	result := game.executeDebugCommand(request)
	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, &scenarios.RunRNG{Seed: "18446744073709551615", SourceCalls: "1"}, game.scenarioRun.status.Launch.RNG)
	game.updateScenarioRun()
	assert.Equal(t, scenarioRunQueued, game.scenarioRun.status.State)
	mutation := debugging.Request{SessionID: "test", ID: "workflow-gated", CommandID: debugging.CommandScenarioWaterfallSetup}
	assert.Equal(t, scenarioRunActiveError, game.executeDebugCommand(mutation).ErrorCode)
	result = game.executeDebugCommand(scenarioCancelRequest(game.scenarioRun.status.RunID))
	require.True(t, result.Success, result.ErrorMessage)
	assert.True(t, state.EventQueueManager.HasActiveWorkflows())
	assert.True(t, input.blocked)
	assert.Zero(t, finished)
	assert.Equal(t, uint64(1), state.GetRNG().Calls(), "cancellation consumes no RNG")
	assert.Equal(t, "gameplay_workflow_active", game.executeDebugCommand(mutation).ErrorCode)
	for range 4 {
		state.EventQueueManager.Update()
		state.ElapsedMs += 100
	}
	assert.False(t, state.EventQueueManager.HasActiveWorkflows())
	assert.Equal(t, 1, finished)
	assert.False(t, input.blocked)
	assert.Equal(t, 1, input.releases)
	assert.True(t, scene.ScenarioCommandsReady())
}

type scenarioWorkflowInput struct {
	blocked  bool
	releases int
}

func (input *scenarioWorkflowInput) BlockAllInput(block bool) {
	input.blocked = block
	if !block {
		input.releases++
	}
}

func (*scenarioWorkflowInput) IsAnyKeyPressed() bool { return false }

func (*scenarioWorkflowInput) GetPressedKey() *semantic_input.Key { return nil }

func scenarioCancelRequest(runID string) debugging.Request {
	return debugging.Request{SessionID: "test", ID: "cancel", CommandID: debugging.CommandScenarioRunCancel,
		Arguments: map[string]json.RawMessage{"runId": json.RawMessage(`"` + runID + `"`)}}
}

func cancellationTestDefinition() scenarios.Definition {
	return scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "cancel", Name: "Cancel", Description: "Stop remaining steps only.",
		Baseline: scenarios.CurrentBaseline(), Steps: []scenarios.Step{
			scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}),
			scenarioRunnerCommandStep(debugging.CommandEnvironmentWindSet, map[string]json.RawMessage{"direction": json.RawMessage(`"south"`)}),
		},
	}
}
