package scenarios

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHistoryV1PromotesUnknownProvenanceWithoutRewritingOnLoad(t *testing.T) {
	root := t.TempDir()
	repository, err := NewRunHistoryRepository(root, 3)
	require.NoError(t, err)
	old := []byte(`{"schemaVersion":1,"runs":[{"runId":"old","reference":"project:test","definitionId":"test","definitionRevision":"old-revision","baseline":{"type":"current"},"state":"succeeded","startedAt":"2026-08-07T10:00:00Z","finishedAt":"2026-08-07T10:01:00Z","currentProcess":false,"nextStep":2,"totalSteps":1,"initialSceneGeneration":1,"sceneGeneration":1,"steps":[{"number":1,"type":"command","commandId":"environment.wind.set","state":"succeeded"}]}]}`)
	path := filepath.Join(root, runHistoryFileName)
	require.NoError(t, os.WriteFile(path, old, 0o600))
	runs, err := repository.Load()
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Nil(t, runs[0].Launch, "do not reconstruct launch values from results or today's definition")
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, old, after)
	current := testRunRecord("new", runs[0].FinishedAt.Add(time.Minute))
	current.State = RunCancelled
	current.Launch = &RunLaunch{ParameterValues: map[string]json.RawMessage{}, BaselineApplied: true,
		RNG: &RunRNG{Seed: "18446744073709551615", SourceCalls: "9007199254740993"}}
	_, err = repository.Append(current)
	require.NoError(t, err)
	after, err = os.ReadFile(path)
	require.NoError(t, err)
	var envelope runHistoryEnvelope
	require.NoError(t, json.Unmarshal(after, &envelope))
	assert.Equal(t, 2, envelope.SchemaVersion)
	runs, err = repository.Load()
	require.NoError(t, err)
	require.Len(t, runs, 2)
	assert.Nil(t, runs[1].Launch)
	require.NotNil(t, runs[0].Launch.ParameterValues)
	assert.Empty(t, runs[0].Launch.ParameterValues)
	assert.Equal(t, current.Launch.RNG, runs[0].Launch.RNG)
	assert.Equal(t, RunCancelled, runs[0].State)
}

func TestRunHistoryFrozenV1RejectsNewFieldsAndState(t *testing.T) {
	repository, err := NewRunHistoryRepository(t.TempDir(), 2)
	require.NoError(t, err)
	for _, record := range []RunRecord{
		{State: RunCancelled},
		{State: RunSucceeded, Launch: &RunLaunch{ParameterValues: map[string]json.RawMessage{}}},
	} {
		data, marshalErr := json.Marshal(runHistoryEnvelope{SchemaVersion: 1, Runs: []RunRecord{record}})
		require.NoError(t, marshalErr)
		require.NoError(t, os.WriteFile(filepath.Join(repository.root.Path, runHistoryFileName), data, 0o600))
		_, err = repository.Load()
		require.ErrorIs(t, err, ErrInvalidRunHistory)
	}
}

func TestRunHistoryValidatesKnownLaunchProvenance(t *testing.T) {
	finishedAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	for _, value := range []string{"", "-1", "+1", "01", "1e3", "18446744073709551616"} {
		record := testRunRecord("invalid-rng", finishedAt)
		record.Launch = &RunLaunch{ParameterValues: map[string]json.RawMessage{}, BaselineApplied: true,
			RNG: &RunRNG{Seed: value, SourceCalls: "0"}}
		assert.ErrorContains(t, validateTerminalRun(record), "uint64 decimal")
		record.Launch.RNG = &RunRNG{Seed: "0", SourceCalls: value}
		assert.ErrorContains(t, validateTerminalRun(record), "uint64 decimal")
	}
	record := testRunRecord("invalid-launch", finishedAt)
	record.Launch = &RunLaunch{}
	assert.ErrorContains(t, validateTerminalRun(record), "must be an object")
	record.Launch.ParameterValues = map[string]json.RawMessage{}
	record.Launch.RNG = &RunRNG{Seed: "0", SourceCalls: "0"}
	assert.ErrorContains(t, validateTerminalRun(record), "applied baseline")
}

func TestRunHistoryRepositoryPersistsNewestFirstAndEvictsOldest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "history")
	repository, err := NewRunHistoryRepository(root, 2)
	require.NoError(t, err)

	first := testRunRecord("run-1", time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	second := testRunRecord("run-2", first.FinishedAt.Add(time.Minute))
	third := testRunRecord("run-3", second.FinishedAt.Add(time.Minute))
	_, err = repository.Append(first)
	require.NoError(t, err)
	_, err = repository.Append(third)
	require.NoError(t, err)
	runs, err := repository.Append(second)
	require.NoError(t, err)

	require.Len(t, runs, 2)
	assert.Equal(t, []string{"test-run-3", "test-run-2"}, []string{runs[0].RunID, runs[1].RunID})
	reloaded, err := repository.Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"test-run-3", "test-run-2"}, []string{reloaded[0].RunID, reloaded[1].RunID})
	assert.False(t, reloaded[0].CurrentProcess)
	rootInfo, err := os.Stat(root)
	require.NoError(t, err)
	fileInfo, err := os.Stat(filepath.Join(root, runHistoryFileName))
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o700), rootInfo.Mode().Perm())
		assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	}
}

func TestRunHistoryRepositoryReplacesMatchingRunID(t *testing.T) {
	repository, err := NewRunHistoryRepository(filepath.Join(t.TempDir(), "history"), 2)
	require.NoError(t, err)
	record := testRunRecord("run-1", time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	_, err = repository.Append(record)
	require.NoError(t, err)
	record.State = RunFailed
	record.ErrorCode = "forced"
	runs, err := repository.Append(record)
	require.NoError(t, err)

	require.Len(t, runs, 1)
	assert.Equal(t, RunFailed, runs[0].State)
	assert.Equal(t, "forced", runs[0].ErrorCode)
}

func TestRunHistoryRepositoryRejectsCorruptAndNonTerminalRecords(t *testing.T) {
	root := filepath.Join(t.TempDir(), "history")
	repository, err := NewRunHistoryRepository(root, 2)
	require.NoError(t, err)
	record := testRunRecord("run-1", time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	record.State = RunRunning
	_, err = repository.Append(record)
	assert.ErrorContains(t, err, "not terminal")

	require.NoError(t, os.MkdirAll(root, 0o700))
	for _, data := range []string{
		`{"schemaVersion":99,"runs":[]}`,
		`{"schemaVersion":1,"runs":[],"unknown":true}`,
		`{"schemaVersion":1,"runs":[]} {}`,
		`{"schemaVersion":1,"runs":[]} {`,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(root, runHistoryFileName), []byte(data), 0o600))
		_, err = repository.Load()
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidRunHistory))
	}
}

func TestRunHistoryRepositoryConcurrentInstancesPreserveBothAppends(t *testing.T) {
	root := filepath.Join(t.TempDir(), "history")
	first, err := NewRunHistoryRepository(root, 4)
	require.NoError(t, err)
	second, err := NewRunHistoryRepository(root, 4)
	require.NoError(t, err)

	start := make(chan struct{})
	results := make(chan error, 2)
	for index, repository := range []*RunHistoryRepository{first, second} {
		record := testRunRecord(
			string(rune('a'+index)),
			time.Date(2026, 8, 7, 10, index, 0, 0, time.UTC),
		)
		go func() {
			<-start
			_, appendErr := repository.Append(record)
			results <- appendErr
		}()
	}
	close(start)
	for range 2 {
		require.NoError(t, <-results)
	}

	runs, err := first.Load()
	require.NoError(t, err)
	require.Len(t, runs, 2)
	assert.Equal(t, []string{"test-b", "test-a"}, []string{runs[0].RunID, runs[1].RunID})
}

func TestNewRunHistoryRepositoryRejectsInvalidConfiguration(t *testing.T) {
	_, err := NewRunHistoryRepository("", 2)
	assert.Error(t, err)
	_, err = NewRunHistoryRepository(t.TempDir(), 0)
	assert.Error(t, err)
}

func testRunRecord(id string, finishedAt time.Time) RunRecord {
	return RunRecord{
		RunID: "test-" + id, Reference: "project:test", DefinitionID: "test", DefinitionRevision: "revision",
		Baseline: CurrentBaseline(), State: RunSucceeded, StartedAt: finishedAt.Add(-time.Minute), FinishedAt: finishedAt,
		NextStep: 2, TotalSteps: 1, InitialSceneGeneration: 1, SceneGeneration: 1,
		Steps: []RunStepStatus{{Number: 1, Type: StepCommand, CommandID: "environment.wind.set", State: "succeeded"}},
	}
}
