package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/scenarios"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

type scenarioRunnerTestScene struct {
	playthrough         playthroughMetadata
	executed            []debugging.Request
	ready               bool
	scenarioRunsBlocked bool
	fail                bool
	updates             int
	captures            int
	prepared            bool
	runtimeFingerprints []string
	runtimeDefinitions  []debugging.CommandDefinition
	runtimeResolutions  int
	snapshot            *save_snapshot.SaveSnapshot
	preservedAvatarName string
	preserveError       error
	rng                 *scenarios.RunRNG
}

func (s *scenarioRunnerTestScene) setPlaythroughMetadata(metadata playthroughMetadata) {
	s.playthrough = metadata
}
func (s *scenarioRunnerTestScene) playthroughMetadata() playthroughMetadata { return s.playthrough }

func TestCloneScenarioBaselineSnapshotPreservesActiveDungeon(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	snapshot.Presentation = journalSnapshotTestPresentation()
	snapshot.World.Position = save_snapshot.PositionSnapshot{Location: 33, X: 2, Y: 3, Floor: 4}
	snapshot.Dungeon = &save_snapshot.DungeonSnapshot{
		Location: 33, Cells: make([]int, save_snapshot.DungeonCellCount), Facing: 1, LastDirection: 5,
		Wanderer: &save_snapshot.DungeonWandererSnapshot{
			Monster: 39, Floor: 4, X: 4, Y: 5, PreviousX: 3, PreviousY: 5, OnCeiling: true,
		},
		QuicknessSkipNext: true,
	}

	clone, err := cloneScenarioBaselineSnapshot(snapshot)
	require.NoError(t, err)
	assert.Equal(t, snapshot, clone)
	clone.Presentation.Visits.Changes[0].Revision++
	assert.NotEqual(t, snapshot.Presentation, clone.Presentation, "frozen baseline owns detached marker storage")
}

func TestCloneScenarioBaselineSnapshotPreservesTransientReconstruction(t *testing.T) {
	snapshot := validCommandTestSnapshot()
	snapshot.MapUnits = nil
	snapshot.MarkMapUnitsForReconstruction()

	clone, err := cloneScenarioBaselineSnapshot(snapshot)
	require.NoError(t, err)
	assert.True(t, clone.NeedsMapUnitReconstruction())
	assert.Nil(t, clone.MapUnits)
}

func (s *scenarioRunnerTestScene) Update(_ *Game) error {
	s.updates++

	return nil
}

func (s *scenarioRunnerTestScene) Draw(_ *ebiten.Image) {}

func (s *scenarioRunnerTestScene) GetUltimaConfiguration() *config.UltimaVConfiguration {
	return nil
}

func (s *scenarioRunnerTestScene) InvalidateResolution() {}

func (s *scenarioRunnerTestScene) PrepareForSceneReplacement() {
	s.prepared = true
}

func (s *scenarioRunnerTestScene) ScenarioCommandsReady() bool {
	return s.ready
}

func (s *scenarioRunnerTestScene) ScenarioRunsAllowed() bool {
	return !s.scenarioRunsBlocked
}

func (s *scenarioRunnerTestScene) ValidateSaveSnapshotEligibility() error {
	return nil
}

func (s *scenarioRunnerTestScene) CaptureSaveSnapshot() (save_snapshot.SaveSnapshot, error) {
	s.captures++
	if s.snapshot != nil {
		return *s.snapshot, nil
	}
	return validCommandTestSnapshot(), nil
}

func (s *scenarioRunnerTestScene) PreserveScenarioAvatarName(name string) error {
	s.preservedAvatarName = name

	return s.preserveError
}

func (s *scenarioRunnerTestScene) ScenarioRNGProvenance() *scenarios.RunRNG {
	return s.rng
}

func (s *scenarioRunnerTestScene) ExecuteDebugCommand(request debugging.Request) debugging.Result {
	s.executed = append(s.executed, request)
	if s.fail {
		return debugging.FailureResult(request, "forced_failure", "forced scenario step failure")
	}

	return debugging.SuccessResult(request, debugging.ExecutionLiveSafe, "executed", nil)
}

func (s *scenarioRunnerTestScene) ResolveScenarioRuntimeParameters(
	request debugging.Request,
	definition debugging.CommandDefinition,
) (debugging.Request, error) {
	s.runtimeResolutions++
	s.runtimeDefinitions = append(s.runtimeDefinitions, definition)
	if len(s.runtimeFingerprints) < s.runtimeResolutions {
		return request, nil
	}
	if request.Arguments == nil {
		request.Arguments = make(map[string]json.RawMessage)
	}
	request.Arguments["expectedFingerprint"] = json.RawMessage(`"` + s.runtimeFingerprints[s.runtimeResolutions-1] + `"`)

	return request, nil
}

func TestScenarioRunnerCurrentBaselineExecutesOneNormalizedCommandPerUpdate(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion,
		ID:            "current-success",
		Name:          "Current Success",
		Description:   "Executes two current-scene commands.",
		Baseline:      scenarios.CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{{
			ID: "wind", Name: "Wind", Kind: debugging.ParameterEnum, Required: true,
			Choices: []debugging.Choice{
				{Value: "north", Label: "North"},
				{Value: "south", Label: "South"},
			},
		}},
		Steps: []scenarios.Step{
			{
				Type: scenarios.StepCommand, CommandID: debugging.CommandEnvironmentWindSet,
				Arguments: map[string]scenarios.CommandArgument{"direction": {Parameter: "wind"}},
			},
			scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}),
		},
	})
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)

	queued := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequestWithValues(t, game, repository, "project:current-success", `{"wind":"north"}`))
	require.True(t, queued.Success, queued.ErrorMessage)
	assertScenarioRunState(t, queued.Data, scenarioRunQueued)
	var queuedData scenarioRunStatusResult
	require.NoError(t, json.Unmarshal(queued.Data, &queuedData))
	require.NotNil(t, queuedData.Run)
	require.Len(t, queuedData.Run.Steps, 2)
	assert.Equal(t, debugging.CommandEnvironmentWindSet, queuedData.Run.Steps[0].CommandID)
	assert.Equal(t, debugging.CommandMovementFreeMoveSet, queuedData.Run.Steps[1].CommandID)
	assert.Empty(t, scene.executed)

	require.NoError(t, game.Update())
	require.Len(t, scene.executed, 1)
	assert.Equal(t, debugging.CommandEnvironmentWindSet, scene.executed[0].CommandID)
	assert.Equal(t, game.DebugSceneGeneration(), scene.executed[0].ExpectedSceneGeneration)
	assert.NotZero(t, scene.executed[0].ExpectedOptionsRevision)

	require.NoError(t, game.Update())
	require.Len(t, scene.executed, 2)
	assert.Equal(t, debugging.CommandMovementFreeMoveSet, scene.executed[1].CommandID)
	assert.Equal(t, scenarioRunSucceeded, game.scenarioRun.status.State)
	require.NotNil(t, game.scenarioRun.status.Steps[0].Result.Invocation)
	assert.JSONEq(t, `"north"`, string(game.scenarioRun.status.Steps[0].Result.Invocation.Arguments["direction"]))
}

type dynamicScenarioTestScene struct {
	*scenarioRunnerTestScene
	metadata int
}

func (s *dynamicScenarioTestScene) DebugOptionSets() []debugging.OptionSet {
	return []debugging.OptionSet{{
		Provider: debugging.OptionProviderActivePartyCharacters,
		Choices: []debugging.Choice{{Value: "0.QXZhdGFy", Label: "Avatar",
			Metadata: json.RawMessage(fmt.Sprintf(`{"hp":%d}`, s.metadata))}},
	}}
}

func (*dynamicScenarioTestScene) ResolveScenarioRuntimeParameters(
	request debugging.Request, _ debugging.CommandDefinition,
) (debugging.Request, error) {
	request.Arguments["character"] = json.RawMessage(`"0.QXZhdGFy"`)
	return request, nil
}

func TestScenarioRefreshesDynamicReviewsWithoutWorkbench(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	step := scenarioRunnerCommandStep(debugging.CommandPartyCharacterUpdate,
		map[string]json.RawMessage{"character": json.RawMessage(`"1"`), "strength": json.RawMessage(`20`)})
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "dynamic-current", Name: "Dynamic current",
		Description: "Refreshes current actor choices before each step.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{step, step},
	})
	scene := &dynamicScenarioTestScene{scenarioRunnerTestScene: &scenarioRunnerTestScene{ready: true}, metadata: 100}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	queued := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequestWithValues(t, game, repository, "project:dynamic-current", `{}`))
	require.True(t, queued.Success, queued.ErrorMessage)
	scene.metadata = 90
	require.NoError(t, game.Update())
	require.Len(t, scene.executed, 1)
	firstRevision := scene.executed[0].ExpectedOptionsRevision
	scene.metadata = 80
	require.NoError(t, game.Update())
	require.Len(t, scene.executed, 2)
	assert.Greater(t, scene.executed[1].ExpectedOptionsRevision, firstRevision)
	assert.Equal(t, scenarioRunSucceeded, game.scenarioRun.status.State)
	assert.False(t, game.debugDiscoveryPolling.Load())
}

func TestScenarioRunnerRejectsRunThatCannotProgressDuringFinale(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion,
		ID:            "blocked-finale",
		Name:          "Blocked Finale",
		Description:   "Must not queue after finale ownership begins.",
		Baseline:      scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{
			scenarioRunnerCommandStep(
				debugging.CommandUnlimitedCastingResourcesSet,
				map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
			),
		},
	})
	game := NewGame(&scenarioRunnerTestScene{ready: false, scenarioRunsBlocked: true})
	game.ConfigureScenarioRunner(repository, nil)

	result := game.ExecuteLocalDebugCommand(
		scenarioRunnerStartRequest(t, game, repository, "project:blocked-finale"),
	)

	require.False(t, result.Success)
	assert.Equal(t, scenarioRunUnavailableError, result.ErrorCode)
	assert.Nil(t, game.scenarioRun)
}

func TestScenarioRunnerFailsQueuedRunWhenFinaleTakesOwnership(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion,
		ID:            "interrupted-by-finale",
		Name:          "Interrupted By Finale",
		Description:   "Fails instead of waiting forever after finale ownership begins.",
		Baseline:      scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{
			scenarioRunnerCommandStep(
				debugging.CommandUnlimitedCastingResourcesSet,
				map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
			),
		},
	})
	scene := &scenarioRunnerTestScene{ready: false}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	queued := game.ExecuteLocalDebugCommand(
		scenarioRunnerStartRequest(t, game, repository, "project:interrupted-by-finale"),
	)
	require.True(t, queued.Success, queued.ErrorMessage)
	scene.scenarioRunsBlocked = true

	require.NoError(t, game.Update())

	require.NotNil(t, game.scenarioRun)
	assert.Equal(t, scenarioRunFailed, game.scenarioRun.status.State)
	assert.Equal(t, scenarioRunUnavailableError, game.scenarioRun.status.ErrorCode)
	assert.Empty(t, scene.executed)
}

func TestScenarioRunnerQueuesMigratedLegacyCharacterTargetWithoutRewritingSource(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))
	repository, err := scenarios.NewRepository(projectRoot, filepath.Join(root, "personal"))
	require.NoError(t, err)
	stored := []byte(`{"schemaVersion":2,"id":"legacy-runner","name":"Legacy Runner","description":"Runs an in-memory migrated target.","baseline":{"type":"current"},"steps":[{"type":"command","commandId":"party.character.update","arguments":{"character":"0.QXZhdGFy","strength":20}}]}`)
	path := filepath.Join(projectRoot, "legacy-runner.json")
	require.NoError(t, os.WriteFile(path, stored, 0o644))
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.ConfigureScenarioRunner(repository, nil)

	queued := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:legacy-runner"))

	require.True(t, queued.Success, queued.ErrorMessage)
	require.NotNil(t, game.scenarioRun)
	assert.JSONEq(t, `"1"`, string(game.scenarioRun.requests[0].Arguments["character"]))
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, stored, after)
}

func TestScenarioRunnerResolvesTransientFingerprintImmediatelyBeforeExecution(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion,
		ID:            "runtime-fingerprint",
		Name:          "Runtime Fingerprint",
		Description:   "Resolves reviewed state after scenario setup.",
		Baseline:      scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{
			{Type: scenarios.StepCommand, CommandID: debugging.CommandPartyFill},
			{Type: scenarios.StepCommand, CommandID: debugging.CommandPartyFill},
		},
	})
	scene := &scenarioRunnerTestScene{ready: true, runtimeFingerprints: []string{"initial-roster", "roster-after-step-one"}}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)

	queued := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:runtime-fingerprint"))
	require.True(t, queued.Success, queued.ErrorMessage)
	require.NoError(t, game.Update())
	require.NoError(t, game.Update())

	require.Len(t, scene.executed, 2)
	assert.Equal(t, 2, scene.runtimeResolutions)
	require.Len(t, scene.runtimeDefinitions, 2)
	assert.Equal(t, debugging.CommandPartyFill, scene.runtimeDefinitions[0].ID)
	assert.JSONEq(t, `"initial-roster"`, string(scene.executed[0].Arguments["expectedFingerprint"]))
	assert.JSONEq(t, `"roster-after-step-one"`, string(scene.executed[1].Arguments["expectedFingerprint"]))
	require.NotNil(t, game.scenarioRun.status.Steps[0].Result.Invocation)
	assert.JSONEq(t, `"initial-roster"`, string(game.scenarioRun.status.Steps[0].Result.Invocation.Arguments["expectedFingerprint"]))
	require.NotNil(t, game.scenarioRun.status.Steps[1].Result.Invocation)
	assert.JSONEq(t, `"roster-after-step-one"`, string(game.scenarioRun.status.Steps[1].Result.Invocation.Arguments["expectedFingerprint"]))
}

func TestScenarioRunnerInitBaselineReplacesSceneBeforeLaterCommandUpdate(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "init-success", Name: "INIT Success",
		Description: "Loads INIT before execution.", Baseline: scenarios.InitBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	})
	rollbackSnapshot := validCommandTestSnapshot()
	rollbackSnapshot.Party.Roster[0] = save_snapshot.CharacterSnapshot{Occupied: true, Name: "Custom", Level: 1}
	oldScene := &scenarioRunnerTestScene{ready: true, snapshot: &rollbackSnapshot}
	initScene := &scenarioRunnerTestScene{ready: true}
	loaderCalls := 0
	game := NewGame(oldScene)
	configureScenarioRunnerSnapshotRepository(game)
	initialGeneration := game.DebugSceneGeneration()
	game.ConfigureScenarioRunner(repository, func() (Scene, error) {
		loaderCalls++
		return initScene, nil
	})
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:init-success")).Success)

	require.NoError(t, game.Update())
	assert.Equal(t, 1, loaderCalls)
	assert.Same(t, initScene, game.currentScene)
	assert.True(t, oldScene.prepared)
	assert.Equal(t, initialGeneration+1, game.DebugSceneGeneration())
	assert.Empty(t, oldScene.executed)
	assert.Empty(t, initScene.executed, "scene replacement and command dispatch must occur on separate updates")
	assert.Equal(t, "Custom", initScene.preservedAvatarName, "INIT baselines must retain player identity")

	require.NoError(t, game.Update())
	require.Len(t, initScene.executed, 1)
	assert.Equal(t, scenarioRunSucceeded, game.scenarioRun.status.State)
	assert.Equal(t, game.DebugSceneGeneration(), game.scenarioRun.status.SceneGeneration)
}

func TestScenarioRunnerRejectsStartPreparedForReplacedScene(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "stale-start", Name: "Stale Start",
		Description: "Must not run against a replacement scene.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	})
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.ConfigureScenarioRunner(repository, nil)
	request := scenarioRunnerStartRequest(t, game, repository, "project:stale-start")
	game.replaceScene(&scenarioRunnerTestScene{ready: true})

	result := game.ExecuteLocalDebugCommand(request)

	assert.False(t, result.Success)
	assert.Equal(t, "stale_scene", result.ErrorCode)
	assert.Nil(t, game.scenarioRun)
}

func TestScenarioRunnerRejectsStaleReviewedRevisionWithoutMutation(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	definition := scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "stale-revision", Name: "Stale Revision",
		Description: "Must run only after its reviewed revision is submitted.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	}
	record := saveScenarioRunnerDefinition(t, repository, definition)
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	request := scenarioRunnerStartRequest(t, game, repository, "project:stale-revision")
	definition.Name = "Changed After Review"
	_, err := repository.Update("project:stale-revision", record.Revision, definition)
	require.NoError(t, err)

	result := game.ExecuteLocalDebugCommand(request)

	assert.False(t, result.Success)
	assert.Equal(t, scenarioRunStaleRevisionError, result.ErrorCode)
	assert.Nil(t, game.scenarioRun)
	assert.Empty(t, scene.executed)
}

func TestScenarioRunnerRejectsBaselineRevisionForCurrentBaseline(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "unexpected-baseline-revision", Name: "Unexpected Baseline Revision",
		Description: "Current baselines do not accept snapshot revisions.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{{Type: scenarios.StepNote, Note: "Never reached."}},
	})
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	request := scenarioRunnerStartRequest(t, game, repository, "project:unexpected-baseline-revision")
	request.Arguments["expectedBaselineRevision"] = json.RawMessage(`"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`)

	result := game.ExecuteLocalDebugCommand(request)

	assert.Equal(t, scenarioBaselineRevisionUnexpected, result.ErrorCode)
	assert.Nil(t, game.scenarioRun)
	assert.Empty(t, scene.executed)
}

func TestScenarioRunnerRejectsStaleReviewedSceneWhenTransportCheckIsBypassed(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "stale-review-scene", Name: "Stale Review Scene",
		Description: "Must not run after the reviewed scene is replaced.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	})
	oldScene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(oldScene)
	game.ConfigureScenarioRunner(repository, nil)
	request := scenarioRunnerStartRequest(t, game, repository, "project:stale-review-scene")
	request.ExpectedSceneGeneration = 0
	newScene := &scenarioRunnerTestScene{ready: true}
	game.replaceScene(newScene)

	result := game.ExecuteLocalDebugCommand(request)

	assert.False(t, result.Success)
	assert.Equal(t, scenarioRunStaleSceneError, result.ErrorCode)
	assert.Nil(t, game.scenarioRun)
	assert.Empty(t, oldScene.executed)
	assert.Empty(t, newScene.executed)
}

func TestScenarioRunnerSkipsNotesWithoutConsumingCommandBudget(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "notes", Name: "Notes",
		Description: "Skips notes before a command.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{
			{Type: scenarios.StepNote, Note: "First note."},
			{Type: scenarios.StepNote, Note: "Second note."},
			scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}),
		},
	})
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:notes")).Success)

	require.NoError(t, game.Update())
	require.Len(t, scene.executed, 1)
	assert.Equal(t, "skipped", game.scenarioRun.status.Steps[0].State)
	assert.Equal(t, "skipped", game.scenarioRun.status.Steps[1].State)
	assert.Equal(t, "succeeded", game.scenarioRun.status.Steps[2].State)
}

func TestScenarioRunnerStopsOnFirstFailureAndDoesNotExecuteLaterStep(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "failure", Name: "Failure",
		Description: "Stops after a failed command.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{
			scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}),
			scenarioRunnerCommandStep(debugging.CommandEnvironmentWindSet, map[string]json.RawMessage{"direction": json.RawMessage(`"south"`)}),
		},
	})
	scene := &scenarioRunnerTestScene{ready: true, fail: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:failure")).Success)

	require.NoError(t, game.Update())
	require.NoError(t, game.Update())
	assert.Len(t, scene.executed, 1)
	assert.Equal(t, scenarioRunFailed, game.scenarioRun.status.State)
	assert.Equal(t, "forced_failure", game.scenarioRun.status.ErrorCode)
	assert.Equal(t, "pending", game.scenarioRun.status.Steps[1].State)
}

func TestScenarioRunnerWaitsForSceneReadinessBeforeNextCommand(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "readiness", Name: "Readiness",
		Description: "Waits for active gameplay work.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	})
	scene := &scenarioRunnerTestScene{ready: false}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:readiness")).Success)

	require.NoError(t, game.Update())
	assert.Empty(t, scene.executed)
	assert.Equal(t, scenarioRunQueued, game.scenarioRun.status.State)
	scene.ready = true
	require.NoError(t, game.Update())
	assert.Len(t, scene.executed, 1)
}

func TestScenarioRunnerRejectsRecursiveCommandDuringPreflightWithoutMutation(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "recursive", Name: "Recursive",
		Description: "Must fail before loading INIT.", Baseline: scenarios.InitBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandScenarioRunStatus, nil)},
	})
	scene := &scenarioRunnerTestScene{ready: true}
	loaderCalls := 0
	game := NewGame(scene)
	configureScenarioRunnerSnapshotRepository(game)
	game.ConfigureScenarioRunner(repository, func() (Scene, error) {
		loaderCalls++
		return nil, errors.New("must not be called")
	})

	result := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:recursive"))

	assert.False(t, result.Success)
	assert.Equal(t, scenarioRunPreflightError, result.ErrorCode)
	assert.Nil(t, game.scenarioRun)
	assert.Zero(t, loaderCalls)
	assert.Empty(t, scene.executed)
}

func TestScenarioRunnerTreatsEveryScenarioManagementIDAsRecursive(t *testing.T) {
	for _, commandID := range []string{
		debugging.CommandScenarioDefinitionList,
		debugging.CommandScenarioDefinitionGet,
		debugging.CommandScenarioDefinitionValidate,
		debugging.CommandScenarioDefinitionCreate,
		debugging.CommandScenarioDefinitionUpdate,
		debugging.CommandScenarioDefinitionDelete,
		debugging.CommandScenarioDefinitionDuplicate,
		debugging.CommandScenarioRunStart,
		debugging.CommandScenarioRunStatus,
		debugging.CommandScenarioRunCancel,
	} {
		assert.True(t, isScenarioManagementCommand(commandID), commandID)
	}
}

func TestScenarioRunnerPreflightUsesSameAnyAllowedModeRuleAsValidator(t *testing.T) {
	catalog, err := debugging.NewCatalog([]debugging.CommandDefinition{{
		ID: "test.multi-mode", Category: "Test", Name: "Multi-mode", Description: "Has one scenario-eligible mode.",
		ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionQuery, debugging.ExecutionSynchronized},
	}})
	require.NoError(t, err)
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.debugCatalog = catalog
	steps := []scenarios.BoundStep{{
		Type:       scenarios.StepCommand,
		Invocation: &scenarios.Invocation{CommandID: "test.multi-mode"},
	}}

	requests, err := game.preflightScenarioSteps("project:multi-mode", steps)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, "test.multi-mode", requests[0].CommandID)
}

func TestScenarioRunnerInitBaselineRequiresInjectedLoader(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "missing-loader", Name: "Missing Loader",
		Description: "Requires the INIT loader.", Baseline: scenarios.InitBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	})
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.ConfigureScenarioRunner(repository, nil)

	result := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:missing-loader"))

	assert.False(t, result.Success)
	assert.Equal(t, scenarioInitLoaderUnavailable, result.ErrorCode)
	assert.Nil(t, game.scenarioRun)
}

func TestScenarioRunnerStatusReturnsLatestStructuredResults(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "status", Name: "Status",
		Description: "Exposes completed results.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`false`),
		})},
	})
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:status")).Success)
	require.NoError(t, game.Update())

	result := game.ExecuteLocalDebugCommand(debugging.Request{
		SessionID: "test", ID: "status", CommandID: debugging.CommandScenarioRunStatus,
	})

	require.True(t, result.Success, result.ErrorMessage)
	var data scenarioRunStatusResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	require.NotNil(t, data.Run)
	assert.Equal(t, "project:status", data.Run.Reference)
	assert.Equal(t, scenarioRunSucceeded, data.Run.State)
	require.Len(t, data.Run.Steps, 1)
	require.NotNil(t, data.Run.Steps[0].Result)
	require.NotNil(t, data.Run.Steps[0].Result.Invocation)
	assert.JSONEq(t, `false`, string(data.Run.Steps[0].Result.Invocation.Arguments["enabled"]))
}

func TestScenarioRunnerPersistsTerminalHistoryAndRestoresLatestRun(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "durable", Name: "Durable",
		Description: "Persists completed runs.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	})
	historyRoot := filepath.Join(t.TempDir(), "run-history")
	history, err := scenarios.NewRunHistoryRepository(historyRoot, scenarios.DefaultRunHistoryLimit)
	require.NoError(t, err)
	startedAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Second)
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	game.ConfigureScenarioRunHistory(history)
	game.scenarioRunID = func() (string, error) { return "durable-run", nil }
	game.scenarioNow = func() time.Time { return startedAt }
	queued := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:durable"))
	require.True(t, queued.Success, queued.ErrorMessage)
	runs, err := history.Load()
	require.NoError(t, err)
	assert.Empty(t, runs, "non-terminal runs must remain process-local")

	game.scenarioNow = func() time.Time { return finishedAt }
	require.NoError(t, game.Update())
	runs, err = history.Load()
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, "durable-run", runs[0].RunID)
	assert.Equal(t, scenarios.RunSucceeded, runs[0].State)
	assert.Equal(t, startedAt, runs[0].StartedAt)
	assert.Equal(t, finishedAt, runs[0].FinishedAt)
	assert.Equal(t, debugging.CommandMovementFreeMoveSet, runs[0].Steps[0].CommandID)

	restarted := NewGame(&scenarioRunnerTestScene{ready: true})
	restarted.ConfigureScenarioRunHistory(history)
	result := restarted.ExecuteLocalDebugCommand(debugging.Request{
		SessionID: "test", ID: "status", CommandID: debugging.CommandScenarioRunStatus,
	})
	require.True(t, result.Success, result.ErrorMessage)
	var data scenarioRunStatusResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	require.NotNil(t, data.Run)
	assert.Equal(t, "durable-run", data.Run.RunID)
	assert.False(t, data.Run.CurrentProcess)
	require.Len(t, data.Runs, 1)
	assert.Equal(t, scenarios.DefaultRunHistoryLimit, data.Capacity)
}

func TestScenarioRunnerPersistsFailedRunWithoutChangingFailureResult(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "durable-failure", Name: "Durable Failure",
		Description: "Persists failed runs.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	})
	history, err := scenarios.NewRunHistoryRepository(filepath.Join(t.TempDir(), "run-history"), scenarios.DefaultRunHistoryLimit)
	require.NoError(t, err)
	scene := &scenarioRunnerTestScene{ready: true, fail: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	game.ConfigureScenarioRunHistory(history)
	game.scenarioRunID = func() (string, error) { return "failed-run", nil }
	game.scenarioNow = func() time.Time { return time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC) }
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:durable-failure")).Success)
	require.NoError(t, game.Update())

	assert.Equal(t, scenarioRunFailed, game.scenarioRun.status.State)
	assert.Equal(t, "forced_failure", game.scenarioRun.status.ErrorCode)
	runs, err := history.Load()
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, scenarios.RunFailed, runs[0].State)
	assert.Equal(t, "forced_failure", runs[0].ErrorCode)
}

func TestScenarioRunnerSnapshotBaselinePinsPayloadPersistsRollbackAndDefersCommand(t *testing.T) {
	scenarioRepository := newScenarioRunnerTestRepository(t)
	target := snapshotCommandTestRecord(t, "project:baseline", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	snapshotRepository := &snapshotCommandTestRepository{records: map[string]save_snapshot.Record{target.Reference.String(): target}}
	definition := scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "snapshot-run", Name: "Snapshot Run",
		Description: "Runs from a reviewed named snapshot.", Baseline: scenarios.SnapshotBaseline(target.Reference.String()),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{
			"enabled": json.RawMessage(`true`),
		})},
	}
	saveScenarioRunnerDefinition(t, scenarioRepository, definition)
	oldScene := &scenarioRunnerTestScene{ready: true}
	newScene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(oldScene)
	var builtSnapshot save_snapshot.SaveSnapshot
	game.ConfigureSnapshotRepository(snapshotRepository, func(_ *config.UltimaVConfiguration, snapshot save_snapshot.SaveSnapshot) (Scene, error) {
		builtSnapshot = snapshot
		return newScene, nil
	})
	game.ConfigureScenarioRunner(scenarioRepository, nil)
	request := scenarioRunnerStartRequest(t, game, scenarioRepository, "project:snapshot-run")

	queued := game.ExecuteLocalDebugCommand(request)
	require.True(t, queued.Success, queued.ErrorMessage)
	changed := snapshotRepository.records[target.Reference.String()]
	changed.Envelope.Snapshot.World.Position.X = 99
	snapshotRepository.records[target.Reference.String()] = changed

	require.NoError(t, game.Update())
	assert.Same(t, newScene, game.currentScene)
	assert.Equal(t, target.Envelope.Snapshot.World.Position.X, builtSnapshot.World.Position.X, "accepted baseline payload must be frozen")
	assert.Empty(t, newScene.executed, "baseline replacement and command dispatch must occur on different updates")
	assert.Equal(t, 1, oldScene.captures)
	require.NotNil(t, game.scenarioRun.status.Rollback)
	assert.Contains(t, game.scenarioRun.status.Rollback.Reference, "rollback:")
	assert.Equal(t, target.Reference.String(), game.scenarioRun.status.BaselineReference)
	assert.Equal(t, target.Revision, game.scenarioRun.status.BaselineRevision)
	assert.True(t, game.scenarioRun.status.Launch.BaselineApplied)

	require.NoError(t, game.Update())
	require.Len(t, newScene.executed, 1)
	assert.Equal(t, scenarioRunSucceeded, game.scenarioRun.status.State)
}

func TestScenarioRunnerSnapshotBaselineRejectsMissingAndStaleRevisionBeforeMutation(t *testing.T) {
	for _, test := range []struct {
		name     string
		revision string
		code     string
	}{
		{name: "missing", revision: "", code: scenarioBaselineRevisionMissing},
		{name: "stale", revision: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", code: scenarioBaselineRevisionStale},
	} {
		t.Run(test.name, func(t *testing.T) {
			scenarioRepository := newScenarioRunnerTestRepository(t)
			target := snapshotCommandTestRecord(t, "project:baseline", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			saveScenarioRunnerDefinition(t, scenarioRepository, scenarios.Definition{
				SchemaVersion: scenarios.CurrentSchemaVersion, ID: "snapshot-review", Name: "Snapshot Review",
				Description: "Requires the exact reviewed baseline.", Baseline: scenarios.SnapshotBaseline(target.Reference.String()),
				Steps: []scenarios.Step{{Type: scenarios.StepNote, Note: "Never reached."}},
			})
			scene := &scenarioRunnerTestScene{ready: true}
			snapshotRepository := &snapshotCommandTestRepository{records: map[string]save_snapshot.Record{target.Reference.String(): target}}
			game := NewGame(scene)
			game.ConfigureSnapshotRepository(snapshotRepository, func(_ *config.UltimaVConfiguration, _ save_snapshot.SaveSnapshot) (Scene, error) {
				t.Fatal("snapshot builder called before revision validation")
				return nil, nil
			})
			game.ConfigureScenarioRunner(scenarioRepository, nil)
			request := scenarioRunnerStartRequest(t, game, scenarioRepository, "project:snapshot-review")
			request.Arguments["expectedBaselineRevision"] = json.RawMessage(`"` + test.revision + `"`)

			result := game.ExecuteLocalDebugCommand(request)

			assert.Equal(t, test.code, result.ErrorCode)
			assert.Nil(t, game.scenarioRun)
			assert.Zero(t, scene.captures)
			assert.Len(t, snapshotRepository.records, 1)
		})
	}
}

func TestScenarioRunnerBaselineFailurePreservesCurrentSceneAndExposesRollback(t *testing.T) {
	scenarioRepository := newScenarioRunnerTestRepository(t)
	target := snapshotCommandTestRecord(t, "project:baseline", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	saveScenarioRunnerDefinition(t, scenarioRepository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "failed-baseline", Name: "Failed Baseline",
		Description: "Keeps the current scene when candidate construction fails.", Baseline: scenarios.SnapshotBaseline(target.Reference.String()),
		Steps: []scenarios.Step{{Type: scenarios.StepNote, Note: "Never reached."}},
	})
	oldScene := &scenarioRunnerTestScene{ready: true}
	snapshotRepository := &snapshotCommandTestRepository{records: map[string]save_snapshot.Record{target.Reference.String(): target}}
	game := NewGame(oldScene)
	game.ConfigureSnapshotRepository(snapshotRepository, func(_ *config.UltimaVConfiguration, _ save_snapshot.SaveSnapshot) (Scene, error) {
		return nil, errors.New("candidate rejected")
	})
	game.ConfigureScenarioRunner(scenarioRepository, nil)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, scenarioRepository, "project:failed-baseline")).Success)
	generation := game.DebugSceneGeneration()

	require.NoError(t, game.Update())

	assert.Same(t, oldScene, game.currentScene)
	assert.Equal(t, generation, game.DebugSceneGeneration())
	assert.Equal(t, scenarioRunFailed, game.scenarioRun.status.State)
	assert.Equal(t, scenarioSnapshotBaselineError, game.scenarioRun.status.ErrorCode)
	require.NotNil(t, game.scenarioRun.status.Rollback)
	assert.False(t, game.scenarioRun.status.Launch.BaselineApplied)
	assert.Nil(t, game.scenarioRun.status.Launch.RNG)
}

func TestScenarioRunnerGatesUnrelatedMutationsButAllowsQueriesAndOwnedExecution(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "gated", Name: "Gated",
		Description: "Waits while the mutation gate is tested.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{scenarioRunnerCommandStep(debugging.CommandMovementFreeMoveSet, map[string]json.RawMessage{"enabled": json.RawMessage(`true`)})},
	})
	scene := &scenarioRunnerTestScene{ready: false}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	require.True(t, game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:gated")).Success)

	mutation := game.ExecuteLocalDebugCommand(debugging.Request{
		SessionID: "other", ID: "mutation", CommandID: debugging.CommandMovementFreeMoveSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	})
	assert.Equal(t, scenarioRunActiveError, mutation.ErrorCode)
	query := game.ExecuteLocalDebugCommand(debugging.Request{SessionID: "other", ID: "query", CommandID: debugging.CommandEnvironmentWindStatus})
	assert.True(t, query.Success, query.ErrorMessage)
	owned := game.ExecuteLocalDebugCommand(debugging.Request{
		SessionID: "scenario:project:gated", ID: "owned", CommandID: debugging.CommandMovementFreeMoveSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	})
	assert.True(t, owned.Success, owned.ErrorMessage)
	assert.Len(t, scene.executed, 2)
}

func TestScenarioRunnerRejectsStepsAfterCombatEncounterStart(t *testing.T) {
	repository := newScenarioRunnerTestRepository(t)
	saveScenarioRunnerDefinition(t, repository, scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "combat-terminal", Name: "Combat Terminal",
		Description: "Invalid setup after combat starts.", Baseline: scenarios.CurrentBaseline(),
		Steps: []scenarios.Step{
			scenarioRunnerCommandStep(debugging.CommandCombatEncounterStart, map[string]json.RawMessage{
				"territory": json.RawMessage(`"britannia"`), "britanniaMap": json.RawMessage(`2`),
				"entryDirection": json.RawMessage(`"north"`), "monsterType": json.RawMessage(`"32"`),
				"countMode": json.RawMessage(`"exact"`), "exactCount": json.RawMessage(`4`),
				"friends": json.RawMessage(`true`), "seed": json.RawMessage(`1`),
			}),
			scenarioRunnerCommandStep(debugging.CommandUnlimitedCastingResourcesSet, map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}),
		},
	})
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.ConfigureScenarioRunner(repository, nil)

	result := game.ExecuteLocalDebugCommand(scenarioRunnerStartRequest(t, game, repository, "project:combat-terminal"))

	require.False(t, result.Success)
	assert.Equal(t, scenarioRunPreflightError, result.ErrorCode)
	assert.Contains(t, result.ErrorMessage, "steps[1]")
	assert.Contains(t, result.ErrorMessage, debugging.CommandCombatEncounterStart)
}

func newScenarioRunnerTestRepository(t *testing.T) *scenarios.Repository {
	t.Helper()
	root := t.TempDir()
	repository, err := scenarios.NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	require.NoError(t, err)

	return repository
}

func configureScenarioRunnerSnapshotRepository(game *Game) *snapshotCommandTestRepository {
	repository := &snapshotCommandTestRepository{records: make(map[string]save_snapshot.Record)}
	game.ConfigureSnapshotRepository(repository, nil)

	return repository
}

func saveScenarioRunnerDefinition(t *testing.T, repository *scenarios.Repository, definition scenarios.Definition) scenarios.Record {
	t.Helper()
	record, err := repository.Create("project:"+definition.ID, definition)
	require.NoError(t, err)

	return record
}

func scenarioRunnerCommandStep(commandID string, arguments map[string]json.RawMessage) scenarios.Step {
	converted := make(map[string]scenarios.CommandArgument, len(arguments))
	for id, literal := range arguments {
		converted[id] = scenarios.CommandArgument{Literal: literal}
	}

	return scenarios.Step{Type: scenarios.StepCommand, CommandID: commandID, Arguments: converted}
}

func scenarioRunnerStartRequest(
	t *testing.T,
	game *Game,
	repository *scenarios.Repository,
	reference string,
) debugging.Request {
	t.Helper()

	return scenarioRunnerStartRequestWithValues(t, game, repository, reference, "")
}

func scenarioRunnerStartRequestWithValues(
	t *testing.T,
	game *Game,
	repository *scenarios.Repository,
	reference string,
	values string,
) debugging.Request {
	t.Helper()
	record, err := repository.LoadRecord(reference)
	require.NoError(t, err)
	sceneGeneration := game.DebugSceneGeneration()
	arguments := map[string]json.RawMessage{
		"reference":                json.RawMessage(`"` + reference + `"`),
		"expectedRevision":         json.RawMessage(`"` + record.Revision + `"`),
		"reviewedSceneGeneration":  json.RawMessage(fmt.Sprintf("%d", sceneGeneration)),
		"expectedBaselineRevision": json.RawMessage(`""`),
	}
	if record.Definition.Baseline.Type == scenarios.BaselineSnapshot {
		baseline, loadErr := game.snapshotRepository.Load(record.Definition.Baseline.Reference)
		require.NoError(t, loadErr)
		arguments["expectedBaselineRevision"] = json.RawMessage(`"` + baseline.Revision + `"`)
	}
	if values != "" {
		encoded, err := json.Marshal(values)
		if err != nil {
			panic(err)
		}
		arguments["parameterValues"] = encoded
	}

	return debugging.Request{
		SessionID: "test", ID: "start", CommandID: debugging.CommandScenarioRunStart,
		ExpectedSceneGeneration: sceneGeneration, Arguments: arguments,
	}
}

func assertScenarioRunState(t *testing.T, raw json.RawMessage, expected scenarioRunState) {
	t.Helper()
	var data scenarioRunStatusResult
	require.NoError(t, json.Unmarshal(raw, &data))
	require.NotNil(t, data.Run)
	assert.Equal(t, expected, data.Run.State)
}
