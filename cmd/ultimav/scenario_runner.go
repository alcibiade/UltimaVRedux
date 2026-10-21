package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/scenarios"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	scenarioRunActiveError             = "scenario_run_active"
	scenarioRunLoadError               = "scenario_run_load_failed"
	scenarioRunStaleRevisionError      = "stale_revision"
	scenarioRunStaleSceneError         = "stale_review_scene"
	scenarioRunParametersError         = "invalid_scenario_run_parameters"
	scenarioRunPreflightError          = "scenario_run_preflight_failed"
	scenarioRuntimeParametersError     = "scenario_runtime_parameters_failed"
	scenarioInitLoaderUnavailable      = "scenario_init_loader_unavailable"
	scenarioInitBaselineLoadError      = "scenario_init_baseline_load_failed"
	scenarioBaselineRevisionMissing    = "missing_baseline_revision"
	scenarioBaselineRevisionStale      = "stale_baseline_revision"
	scenarioBaselineRevisionUnexpected = "unexpected_baseline_revision"
	scenarioSnapshotBaselineError      = "scenario_snapshot_baseline_failed"
	scenarioRollbackCaptureError       = "scenario_rollback_capture_failed"
	scenarioRollbackPersistError       = "scenario_rollback_persist_failed"
	scenarioRunIDError                 = "scenario_run_id_failed"
	scenarioRunUnavailableError        = "scenario_run_unavailable"
	scenarioRunCancelStaleError        = "scenario_run_stale"
)

type scenarioRunState = scenarios.RunState

const (
	scenarioRunQueued    = scenarios.RunQueued
	scenarioRunRunning   = scenarios.RunRunning
	scenarioRunSucceeded = scenarios.RunSucceeded
	scenarioRunFailed    = scenarios.RunFailed
	scenarioRunCancelled = scenarios.RunCancelled
)

type scenarioRunStepStatus = scenarios.RunStepStatus
type scenarioRunStatus = scenarios.RunRecord
type scenarioRunRollback = scenarios.RunRollback

type scenarioRunStatusResult struct {
	Run          *scenarioRunStatus  `json:"run"`
	Runs         []scenarioRunStatus `json:"runs"`
	Capacity     int                 `json:"capacity"`
	HistoryError string              `json:"historyError,omitempty"`
}

type scenarioRun struct {
	status              scenarioRunStatus
	steps               []scenarios.BoundStep
	requests            []debugging.Request
	baselinePending     bool
	baselineSnapshot    save_snapshot.SaveSnapshot
	baselinePlaythrough playthroughMetadata
	historyPersisted    bool
}

type scenarioProgressReadiness interface {
	ScenarioCommandsReady() bool
}

type scenarioRunAvailability interface {
	ScenarioRunsAllowed() bool
}

type scenarioRuntimeParameterResolver interface {
	ResolveScenarioRuntimeParameters(request debugging.Request, definition debugging.CommandDefinition) (debugging.Request, error)
}

type scenarioAvatarNamePreserver interface {
	PreserveScenarioAvatarName(name string) error
}

type scenarioRNGProvenance interface {
	ScenarioRNGProvenance() *scenarios.RunRNG
}

// ScenarioRNGProvenance observes the authoritative source without advancing it
// or requiring save eligibility (a current baseline may be waiting on a workflow).
func (g *GameScene) ScenarioRNGProvenance() *scenarios.RunRNG {
	if g.gameState == nil || g.gameState.GetRNG() == nil {
		return nil
	}
	rng := g.gameState.GetRNG()

	return &scenarios.RunRNG{Seed: strconv.FormatUint(rng.Seed(), 10), SourceCalls: strconv.FormatUint(rng.Calls(), 10)}
}

func scenarioSceneRNG(scene Scene) *scenarios.RunRNG {
	if provider, ok := scene.(scenarioRNGProvenance); ok {
		if rng := provider.ScenarioRNGProvenance(); rng != nil {
			copyRNG := *rng

			return &copyRNG
		}
	}

	return nil
}

func (g *Game) executeScenarioRunCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandScenarioRunStart:
		return g.startScenarioRun(request), true
	case debugging.CommandScenarioRunStatus:
		return g.scenarioRunStatusResult(request), true
	case debugging.CommandScenarioRunCancel:
		return g.cancelScenarioRun(request), true
	default:
		return debugging.Result{}, false
	}
}

func (g *Game) startScenarioRun(request debugging.Request) debugging.Result {
	if g.scenarioRepository == nil {
		return debugging.FailureResult(request, scenarioRepositoryUnavailableError, "the scenario repository is unavailable")
	}
	if g.scenarioRun != nil && (g.scenarioRun.status.State == scenarioRunQueued || g.scenarioRun.status.State == scenarioRunRunning) {
		return debugging.FailureResult(request, scenarioRunActiveError, "a scenario run is already active")
	}
	if availability, ok := g.currentScene.(scenarioRunAvailability); ok && !availability.ScenarioRunsAllowed() {
		return debugging.FailureResult(request, scenarioRunUnavailableError, "scenario runs are unavailable during or after the Lord British finale")
	}

	reference := mustDebugStringArgument(request, "reference")
	expectedRevision := mustDebugStringArgument(request, "expectedRevision")
	reviewedSceneGeneration := uint64(mustDebugIntegerArgument(request, "reviewedSceneGeneration"))
	record, err := g.scenarioRepository.LoadRecord(reference)
	if err != nil {
		return debugging.FailureResult(request, scenarioRunLoadError, err.Error())
	}
	if record.Revision != expectedRevision {
		return debugging.FailureResult(request, scenarioRunStaleRevisionError, scenarios.ErrScenarioStale.Error())
	}
	if reviewedSceneGeneration != g.sceneGeneration.Load() {
		return debugging.FailureResult(request, scenarioRunStaleSceneError, "the active scene changed after the scenario was reviewed")
	}
	definition := record.Definition
	if issues := scenarios.ValidateAgainstCatalog(definition, g.debugCatalog, scenarioCatalogValidationPolicy()); len(issues) > 0 {
		return debugging.FailureResult(
			request,
			scenarioRunPreflightError,
			fmt.Sprintf("%s: %s", issues[0].Path, issues[0].Message),
		)
	}
	if definition.Baseline.Type == scenarios.BaselineInit && g.scenarioInitLoader == nil {
		return debugging.FailureResult(request, scenarioInitLoaderUnavailable, "the INIT baseline scene loader is unavailable")
	}
	if definition.Baseline.Type != scenarios.BaselineCurrent && g.snapshotRepository == nil {
		return debugging.FailureResult(request, snapshotRepositoryUnavailableError, "non-current scenario baselines require rollback persistence")
	}
	var baselineSnapshot save_snapshot.SaveSnapshot
	var baselinePlaythrough playthroughMetadata
	baselineRevision := ""
	expectedBaselineRevision := mustDebugStringArgument(request, "expectedBaselineRevision")
	if definition.Baseline.Type == scenarios.BaselineSnapshot {
		if g.snapshotSceneBuilder == nil {
			return debugging.FailureResult(request, scenarioSnapshotBaselineError, "the snapshot baseline scene builder is unavailable")
		}
		if expectedBaselineRevision == "" {
			return debugging.FailureResult(request, scenarioBaselineRevisionMissing, "a reviewed snapshot baseline revision is required")
		}
		baselineRecord, loadErr := g.snapshotRepository.Load(definition.Baseline.Reference)
		if loadErr != nil {
			return debugging.FailureResult(request, scenarioSnapshotBaselineError, loadErr.Error())
		}
		if baselineRecord.Revision != expectedBaselineRevision {
			return debugging.FailureResult(request, scenarioBaselineRevisionStale, save_snapshot.ErrSnapshotStale.Error())
		}
		prepared, migrationErr := g.prepareExcerptMigration(baselineRecord.Envelope)
		if migrationErr != nil {
			return debugging.FailureResult(request, scenarioSnapshotBaselineError, migrationErr.Error())
		}
		baselineSnapshot, err = cloneScenarioBaselineSnapshot(prepared)
		if err != nil {
			return debugging.FailureResult(request, scenarioSnapshotBaselineError, err.Error())
		}
		baselineRevision = baselineRecord.Revision
		baselinePlaythrough = playthroughMetadataFromEnvelope(baselineRecord.Envelope.Metadata)
	} else if expectedBaselineRevision != "" {
		return debugging.FailureResult(request, scenarioBaselineRevisionUnexpected, "only snapshot baselines accept an expected baseline revision")
	}
	parameterValues, err := decodeScenarioParameterValues(mustDebugStringArgument(request, "parameterValues"))
	if err != nil {
		return debugging.FailureResult(request, scenarioRunParametersError, err.Error())
	}
	steps, resolvedValues, err := definition.BindWithValues(parameterValues)
	if err != nil {
		return debugging.FailureResult(request, scenarioRunParametersError, err.Error())
	}
	requests, err := g.preflightScenarioSteps(reference, steps)
	if err != nil {
		return debugging.FailureResult(request, scenarioRunPreflightError, err.Error())
	}
	runID, err := g.scenarioRunID()
	if err != nil {
		return debugging.FailureResult(request, scenarioRunIDError, err.Error())
	}
	startedAt := g.scenarioNow().UTC()
	launch := &scenarios.RunLaunch{ParameterValues: resolvedValues, BaselineApplied: definition.Baseline.Type == scenarios.BaselineCurrent}
	if launch.BaselineApplied {
		launch.RNG = scenarioSceneRNG(g.currentScene)
	}

	stepStatuses := make([]scenarioRunStepStatus, len(steps))
	for index, step := range steps {
		commandID := ""
		if step.Invocation != nil {
			commandID = step.Invocation.CommandID
		}
		stepStatuses[index] = scenarioRunStepStatus{
			Number: index + 1, Type: step.Type, CommandID: commandID, Note: step.Note, State: "pending", Result: nil,
		}
	}
	g.scenarioRun = &scenarioRun{
		status: scenarioRunStatus{
			RunID: runID, Reference: reference, DefinitionID: definition.ID, DefinitionRevision: record.Revision,
			Baseline: definition.Baseline, StartedAt: startedAt, CurrentProcess: true,
			BaselineReference: definition.Baseline.Reference, BaselineRevision: baselineRevision,
			State: scenarioRunQueued, NextStep: 1, TotalSteps: len(steps),
			InitialSceneGeneration: g.sceneGeneration.Load(), SceneGeneration: g.sceneGeneration.Load(),
			Steps: stepStatuses, Launch: launch,
		},
		steps: steps, requests: requests, baselinePending: definition.Baseline.Type != scenarios.BaselineCurrent,
		baselineSnapshot:    baselineSnapshot,
		baselinePlaythrough: baselinePlaythrough,
	}

	return debugSuccessResultWithMode(request, debugging.ExecutionWorkflow, "Scenario run queued.", g.scenarioRunStatusPayload())
}

func cloneScenarioBaselineSnapshot(snapshot save_snapshot.SaveSnapshot) (save_snapshot.SaveSnapshot, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return save_snapshot.SaveSnapshot{}, fmt.Errorf("freeze snapshot baseline: %w", err)
	}
	var frozen save_snapshot.SaveSnapshot
	if err = json.Unmarshal(data, &frozen); err != nil {
		return save_snapshot.SaveSnapshot{}, fmt.Errorf("freeze snapshot baseline: %w", err)
	}
	if snapshot.NeedsMapUnitReconstruction() {
		frozen.MarkMapUnitsForReconstruction()
	}

	return frozen, nil
}

func decodeScenarioParameterValues(value string) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(value))
	decoder.DisallowUnknownFields()
	var values map[string]json.RawMessage
	if err := decoder.Decode(&values); err != nil {
		return nil, fmt.Errorf("parameterValues must be a JSON object: %w", err)
	}
	if values == nil {
		return nil, errors.New("parameterValues must be a JSON object")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("parameterValues must contain one JSON object")
		}

		return nil, fmt.Errorf("parameterValues contains trailing JSON: %w", err)
	}

	return values, nil
}

func (g *Game) preflightScenarioSteps(reference string, steps []scenarios.BoundStep) ([]debugging.Request, error) {
	requests := make([]debugging.Request, len(steps))
	for index, step := range steps {
		if step.Type != scenarios.StepCommand {
			continue
		}
		commandID := step.Invocation.CommandID
		definition, found := g.debugCatalog.Get(commandID)
		if !found {
			return nil, fmt.Errorf("step %d: command %q is not in the debug catalog", index+1, commandID)
		}
		request := debugging.Request{
			SessionID: "scenario:" + reference,
			ID:        fmt.Sprintf("step-%d", index+1),
			CommandID: commandID,
			Arguments: step.Invocation.Arguments,
		}
		if isScenarioManagementCommand(commandID) {
			return nil, fmt.Errorf("step %d: scenario management command %q is not allowed", index+1, commandID)
		}
		if !scenarios.HasAllowedExecutionMode(definition.ExecutionModes, scenarioAllowedExecutionModes()) {
			return nil, fmt.Errorf("step %d: query or presentation command %q is not allowed", index+1, commandID)
		}
		requests[index] = request
	}

	return requests, nil
}

func isScenarioManagementCommand(commandID string) bool {
	switch commandID {
	case debugging.CommandScenarioDefinitionList, debugging.CommandScenarioDefinitionGet,
		debugging.CommandScenarioDefinitionValidate, debugging.CommandScenarioDefinitionCreate,
		debugging.CommandScenarioDefinitionUpdate, debugging.CommandScenarioDefinitionDelete,
		debugging.CommandScenarioDefinitionDuplicate,
		debugging.CommandScenarioRunStart, debugging.CommandScenarioRunStatus, debugging.CommandScenarioRunCancel,
		debugging.CommandSaveSnapshotList, debugging.CommandSaveSnapshotGet, debugging.CommandSaveSnapshotCapture,
		debugging.CommandSaveSnapshotLoad, debugging.CommandSaveSnapshotDelete:
		return true
	default:
		return false
	}
}

func scenarioAllowedExecutionModes() map[debugging.ExecutionMode]struct{} {
	allowed := make(map[debugging.ExecutionMode]struct{})
	for _, mode := range scenarioCatalogValidationPolicy().AllowedExecutionModes {
		allowed[mode] = struct{}{}
	}

	return allowed
}

//nolint:cyclop,nestif // This is the serialized scenario workflow state machine.
func (g *Game) updateScenarioRun() {
	run := g.scenarioRun
	if run == nil || (run.status.State != scenarioRunQueued && run.status.State != scenarioRunRunning) {
		return
	}
	if availability, ok := g.currentScene.(scenarioRunAvailability); ok && !availability.ScenarioRunsAllowed() {
		g.failScenarioRun(
			scenarioRunUnavailableError,
			"scenario run stopped because the Lord British finale took gameplay ownership",
		)

		return
	}
	if run.baselinePending {
		active, ok := g.currentScene.(snapshotCapableScene)
		if !ok {
			g.failScenarioRun(scenarioRollbackCaptureError, "the active scene cannot provide scenario rollback state")

			return
		}
		rollbackSnapshot, err := active.CaptureSaveSnapshot()
		if err != nil {
			g.failScenarioRun(scenarioRollbackCaptureError, err.Error())

			return
		}
		rollback, err := g.createAutomaticRollback("scenario "+run.status.Reference, rollbackSnapshot)
		if err != nil {
			g.failScenarioRun(scenarioRollbackPersistError, err.Error())

			return
		}
		run.status.Rollback = &scenarioRunRollback{Reference: rollback.Reference.String(), Revision: rollback.Revision}

		var factory func() (Scene, error)
		switch run.status.Baseline.Type {
		case scenarios.BaselineInit:
			factory = g.scenarioInitLoader
		case scenarios.BaselineSnapshot:
			configuration := g.currentScene.GetUltimaConfiguration()
			frozen := run.baselineSnapshot
			factory = func() (Scene, error) { return g.snapshotSceneBuilder(configuration, frozen) }
		case scenarios.BaselineCurrent:
			panic("current baseline cannot be pending")
		default:
			panic("unsupported pending scenario baseline: " + run.status.Baseline.Type)
		}
		err = g.ReplaceScene(func() (Scene, error) {
			metadata := run.baselinePlaythrough
			if metadata.id == "" {
				var identityErr error
				metadata, identityErr = freshPlaythroughMetadata(g.playthroughIDProvider)
				if identityErr != nil {
					return nil, identityErr
				}
			}
			candidate, buildErr := factory()
			if buildErr != nil {
				return nil, buildErr
			}
			if transferErr := transferScenePlaythrough(candidate, metadata); transferErr != nil {
				if disposer, ok := candidate.(sceneDisposer); ok {
					disposer.Dispose()
				}

				return nil, transferErr
			}

			return candidate, nil
		})
		if err != nil {
			code := scenarioInitBaselineLoadError
			if run.status.Baseline.Type == scenarios.BaselineSnapshot {
				code = scenarioSnapshotBaselineError
			}
			g.failScenarioRun(code, err.Error())

			return
		}
		// Scene replacement is already committed even if identity preservation fails.
		run.status.Launch.BaselineApplied = true
		run.status.Launch.RNG = scenarioSceneRNG(g.currentScene)
		if run.status.Baseline.Type == scenarios.BaselineInit {
			avatarName := rollbackSnapshot.Party.Roster[0].Name
			if avatarName == "" {
				avatarName = "Avatar"
			}
			preserver, ok := g.currentScene.(scenarioAvatarNamePreserver)
			if !ok {
				g.failScenarioRun(scenarioInitBaselineLoadError, "the INIT baseline scene cannot preserve Avatar identity")

				return
			}
			if err = preserver.PreserveScenarioAvatarName(avatarName); err != nil {
				g.failScenarioRun(scenarioInitBaselineLoadError, err.Error())

				return
			}
		}
		run.baselinePending = false
		run.status.State = scenarioRunRunning
		run.status.SceneGeneration = g.sceneGeneration.Load()

		return
	}
	if readiness, ok := g.currentScene.(scenarioProgressReadiness); ok && !readiness.ScenarioCommandsReady() {
		return
	}
	run.status.State = scenarioRunRunning
	for run.status.NextStep <= len(run.steps) && run.steps[run.status.NextStep-1].Type == scenarios.StepNote {
		index := run.status.NextStep - 1
		run.status.Steps[index].State = "skipped"
		run.status.NextStep++
	}
	if run.status.NextStep > len(run.steps) {
		g.finishScenarioRun(scenarioRunSucceeded, "", "")

		return
	}

	index := run.status.NextStep - 1
	request := run.requests[index]
	definition, exists := g.debugCatalog.Get(request.CommandID)
	if !exists {
		g.failScenarioRun(scenarioRuntimeParametersError, fmt.Sprintf("command %q is no longer in the debug catalog", request.CommandID))

		return
	}
	if resolver, ok := g.currentScene.(scenarioRuntimeParameterResolver); ok {
		var err error
		request, err = resolver.ResolveScenarioRuntimeParameters(request, definition)
		if err != nil {
			g.failScenarioRun(scenarioRuntimeParametersError, err.Error())

			return
		}
	} else if commandHasScenarioValueResolver(definition) {
		g.failScenarioRun(scenarioRuntimeParametersError, "the active scene cannot resolve scenario planning values")

		return
	}
	request, err := g.debugCatalog.NormalizeRequest(request)
	if err != nil {
		g.failScenarioRun(scenarioRuntimeParametersError, err.Error())

		return
	}
	request.ExpectedSceneGeneration = g.sceneGeneration.Load()
	g.refreshDebugDiscoverySnapshot()
	request.ExpectedOptionsRevision = g.debugDiscovery.Load().OptionsRevision
	result := g.ExecuteLocalDebugCommand(request)
	run.status.Steps[index].Result = &result
	run.status.SceneGeneration = g.sceneGeneration.Load()
	if !result.Success {
		run.status.Steps[index].State = "failed"
		g.finishScenarioRun(scenarioRunFailed, result.ErrorCode, result.ErrorMessage)

		return
	}
	run.status.Steps[index].State = "succeeded"
	run.status.NextStep++
	if run.status.NextStep > len(run.steps) {
		g.finishScenarioRun(scenarioRunSucceeded, "", "")
	}
}

func commandHasScenarioValueResolver(definition debugging.CommandDefinition) bool {
	for _, parameter := range definition.Parameters {
		if parameter.ScenarioValueResolver != "" {
			return true
		}
	}

	return false
}

func (g *Game) scenarioRunBlocksRequest(request debugging.Request, definition debugging.CommandDefinition) bool {
	if g.scenarioRun == nil || (g.scenarioRun.status.State != scenarioRunQueued && g.scenarioRun.status.State != scenarioRunRunning) {
		return false
	}
	if request.CommandID == debugging.CommandScenarioRunStart || request.CommandID == debugging.CommandScenarioRunStatus ||
		request.CommandID == debugging.CommandScenarioRunCancel {
		return false
	}
	if request.SessionID == "scenario:"+g.scenarioRun.status.Reference {
		return false
	}
	for _, mode := range definition.ExecutionModes {
		if mode != debugging.ExecutionQuery && mode != debugging.ExecutionPresentation {
			return true
		}
	}

	return false
}

func (g *Game) failScenarioRun(code, message string) {
	g.finishScenarioRun(scenarioRunFailed, code, message)
}

func (g *Game) cancelScenarioRun(request debugging.Request) debugging.Result {
	runID := mustDebugStringArgument(request, "runId")
	if g.scenarioRun == nil || g.scenarioRun.status.RunID != runID {
		return debugging.FailureResult(request, scenarioRunCancelStaleError, "the identified run is not the current process run")
	}
	// Cancellation takes effect at dispatch, after this update's runner advance.
	// Never cancel/reset production workflows or restore a scene here.
	if g.scenarioRun.status.State == scenarioRunQueued || g.scenarioRun.status.State == scenarioRunRunning {
		g.finishScenarioRun(scenarioRunCancelled, "", "")
	}

	return debugSuccessResultWithMode(request, debugging.ExecutionLiveSafe, "No remaining steps will be dispatched. Completed effects and production workflows are retained.", g.scenarioRunStatusPayload())
}

func (g *Game) scenarioRunStatusResult(request debugging.Request) debugging.Result {
	return debugSuccessResult(request, "Scenario run status loaded.", g.scenarioRunStatusPayload())
}

func (g *Game) scenarioRunStatusPayload() scenarioRunStatusResult {
	var status *scenarioRunStatus
	if g.scenarioRun != nil {
		status = &g.scenarioRun.status
	} else if len(g.scenarioHistory) > 0 {
		latest := g.scenarioHistory[0]
		status = &latest
	}
	runs := append([]scenarioRunStatus(nil), g.scenarioHistory...)
	if g.scenarioRun != nil {
		found := false
		for index := range runs {
			if runs[index].RunID == g.scenarioRun.status.RunID {
				runs[index] = g.scenarioRun.status
				found = true

				break
			}
		}
		if !found {
			runs = append([]scenarioRunStatus{g.scenarioRun.status}, runs...)
		}
	}
	capacity := 0
	if g.scenarioRunHistory != nil {
		capacity = g.scenarioRunHistory.Capacity()
	}
	if capacity > 0 && len(runs) > capacity {
		runs = runs[:capacity]
	}

	return scenarioRunStatusResult{
		Run: status, Runs: runs, Capacity: capacity, HistoryError: g.scenarioHistoryError,
	}
}

func (g *Game) finishScenarioRun(state scenarioRunState, code, message string) {
	run := g.scenarioRun
	if run == nil {
		panic("finishScenarioRun called without an active scenario run")
	}
	if run.status.State != scenarioRunQueued && run.status.State != scenarioRunRunning {
		return
	}
	run.status.State = state
	run.status.ErrorCode = code
	run.status.ErrorMessage = message
	run.status.SceneGeneration = g.sceneGeneration.Load()
	run.status.FinishedAt = g.scenarioNow().UTC()
	g.persistScenarioRunHistory()
}

func (g *Game) persistScenarioRunHistory() {
	run := g.scenarioRun
	if run == nil || run.historyPersisted || g.scenarioRunHistory == nil {
		return
	}
	runs, err := g.scenarioRunHistory.Append(run.status)
	if err != nil {
		g.scenarioHistoryError = err.Error()
		logging.Errorf(logging.CategorySystem, "Persisting scenario run history failed: %v", err)

		return
	}
	for index := range runs {
		if runs[index].RunID == run.status.RunID {
			runs[index].CurrentProcess = true
		}
	}
	g.scenarioHistory = runs
	g.scenarioHistoryError = ""
	run.historyPersisted = true
}

func newScenarioRunID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate scenario run ID: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
