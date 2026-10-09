package scenarios

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
)

const (
	// DefaultRunHistoryLimit bounds private scenario result retention.
	DefaultRunHistoryLimit  = 20
	runHistorySchemaVersion = 2
	runHistoryFileName      = "runs.json"
	maxRunHistoryBytes      = 16 << 20
)

// ErrInvalidRunHistory indicates malformed or inconsistent persisted run history.
var ErrInvalidRunHistory = errors.New("invalid scenario run history")

// RunState identifies the lifecycle state of one accepted scenario run.
type RunState string

const (
	// RunQueued indicates that a run has been accepted but not started.
	RunQueued RunState = "queued"
	// RunRunning indicates that a run is currently executing.
	RunRunning RunState = "running"
	// RunSucceeded identifies a completed scenario run with no execution error.
	RunSucceeded RunState = "succeeded"
	// RunFailed identifies a completed scenario run with an execution error.
	RunFailed RunState = "failed"
	// RunCancelled stops dispatching remaining steps without undoing gameplay.
	RunCancelled RunState = "cancelled"
)

// RunStepStatus reports one bound scenario step and its eventual result.
type RunStepStatus struct {
	Number    int               `json:"number"`
	Type      StepType          `json:"type"`
	CommandID string            `json:"commandId,omitempty"`
	Note      string            `json:"note,omitempty"`
	State     string            `json:"state"`
	Result    *debugging.Result `json:"result,omitempty"`
}

// RunRecord is the durable diagnostic result of one accepted scenario run.
type RunRecord struct {
	RunID                  string          `json:"runId"`
	Reference              string          `json:"reference"`
	DefinitionID           string          `json:"definitionId"`
	DefinitionRevision     string          `json:"definitionRevision"`
	Baseline               Baseline        `json:"baseline"`
	BaselineReference      string          `json:"baselineReference,omitempty"`
	BaselineRevision       string          `json:"baselineRevision,omitempty"`
	Rollback               *RunRollback    `json:"rollback,omitempty"`
	Launch                 *RunLaunch      `json:"launch,omitempty"`
	State                  RunState        `json:"state"`
	StartedAt              time.Time       `json:"startedAt"`
	FinishedAt             time.Time       `json:"finishedAt,omitempty"`
	CurrentProcess         bool            `json:"currentProcess"`
	NextStep               int             `json:"nextStep"`
	TotalSteps             int             `json:"totalSteps"`
	InitialSceneGeneration uint64          `json:"initialSceneGeneration"`
	SceneGeneration        uint64          `json:"sceneGeneration"`
	Steps                  []RunStepStatus `json:"steps"`
	ErrorCode              string          `json:"errorCode,omitempty"`
	ErrorMessage           string          `json:"errorMessage,omitempty"`
}

// RunLaunch records accepted launch inputs, not command results or current defaults.
// Its absence means historical provenance is unknown (promoted v1 history).
type RunLaunch struct {
	ParameterValues map[string]json.RawMessage `json:"parameterValues"`
	BaselineApplied bool                       `json:"baselineApplied"`
	RNG             *RunRNG                    `json:"rng,omitempty"`
}

// RunRNG is the RNG position when the baseline became active, before scenario
// steps. Decimal strings retain exact uint64 values across browser JSON parsing.
// It is diagnostic provenance, not a promise of deterministic session replay.
type RunRNG struct {
	Seed        string `json:"seed"`
	SourceCalls string `json:"sourceCalls"`
}

// RunRollback identifies the automatic recovery snapshot created for a run.
// Hand this reference and revision to the reviewed named-snapshot load flow;
// terminal failure or cancellation never restores it automatically.
type RunRollback struct {
	Reference string `json:"reference"`
	Revision  string `json:"revision"`
}

type runHistoryEnvelope struct {
	SchemaVersion int         `json:"schemaVersion"`
	Runs          []RunRecord `json:"runs"`
}

// RunHistoryRepository atomically retains a bounded private run history.
type RunHistoryRepository struct {
	root  repositoryfs.Root
	limit int
	mu    sync.Mutex
}

// NewRunHistoryRepository constructs a private local run-history repository.
func NewRunHistoryRepository(root string, limit int) (*RunHistoryRepository, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("scenario run history requires a root")
	}
	if limit <= 0 {
		return nil, errors.New("scenario run history requires a positive limit")
	}

	return &RunHistoryRepository{
		root:  repositoryfs.Root{Path: filepath.Clean(root), DirMode: 0o700, FileMode: 0o600},
		limit: limit,
	}, nil
}

// Capacity returns the configured maximum number of retained runs.
func (repository *RunHistoryRepository) Capacity() int {
	return repository.limit
}

// Load returns newest-first detached terminal run records.
func (repository *RunHistoryRepository) Load() ([]RunRecord, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	return repository.load()
}

// Append stores one terminal record and atomically evicts older entries.
func (repository *RunHistoryRepository) Append(record RunRecord) ([]RunRecord, error) {
	if err := validateTerminalRun(record); err != nil {
		return nil, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	unlock, err := repositoryfs.AcquireReferenceLocks([]repositoryfs.Root{repository.root})
	if err != nil {
		return nil, err
	}
	defer unlock()

	runs, err := repository.load()
	if err != nil {
		return nil, err
	}
	next := make([]RunRecord, 0, min(repository.limit, len(runs)+1))
	next = append(next, record)
	for _, existing := range runs {
		if existing.RunID != record.RunID {
			next = append(next, existing)
		}
	}
	sortRuns(next)
	if len(next) > repository.limit {
		next = next[:repository.limit]
	}
	if err = repository.write(next); err != nil {
		return nil, err
	}

	return cloneRuns(next)
}

//nolint:cyclop // Load validates the complete persisted envelope before returning it.
func (repository *RunHistoryRepository) load() ([]RunRecord, error) {
	data, err := repositoryfs.ReadRegularFile(filepath.Join(repository.root.Path, runHistoryFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read scenario run history: %w", err)
	}
	if len(data) > maxRunHistoryBytes {
		return nil, fmt.Errorf("%w: file exceeds %d bytes", ErrInvalidRunHistory, maxRunHistoryBytes)
	}
	envelope, err := decodeRunHistory(data)
	if err != nil {
		return nil, fmt.Errorf("%w: decode: %w", ErrInvalidRunHistory, err)
	}
	if len(envelope.Runs) > repository.limit {
		return nil, fmt.Errorf("%w: contains %d runs above limit %d", ErrInvalidRunHistory, len(envelope.Runs), repository.limit)
	}
	seen := make(map[string]struct{}, len(envelope.Runs))
	for index, run := range envelope.Runs {
		if err = validateTerminalRun(run); err != nil {
			return nil, fmt.Errorf("%w: run %d: %w", ErrInvalidRunHistory, index, err)
		}
		if _, exists := seen[run.RunID]; exists {
			return nil, fmt.Errorf("%w: duplicate run ID %q", ErrInvalidRunHistory, run.RunID)
		}
		seen[run.RunID] = struct{}{}
	}
	sortRuns(envelope.Runs)

	return cloneRuns(envelope.Runs)
}

func (repository *RunHistoryRepository) write(runs []RunRecord) error {
	if err := repositoryfs.EnsureRoot(repository.root); err != nil {
		return fmt.Errorf("create scenario run history root: %w", err)
	}
	persisted := append([]RunRecord(nil), runs...)
	for index := range persisted {
		persisted[index].CurrentProcess = false
	}
	data, err := json.MarshalIndent(runHistoryEnvelope{SchemaVersion: runHistorySchemaVersion, Runs: persisted}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode scenario run history: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxRunHistoryBytes {
		return fmt.Errorf("%w: encoded history exceeds %d bytes", ErrInvalidRunHistory, maxRunHistoryBytes)
	}
	if err = repositoryfs.PublishTemporaryFile(
		repository.root.Path,
		".runs-*.tmp",
		data,
		repository.root.FileMode,
		func(temporaryPath string) error {
			if renameErr := os.Rename(temporaryPath, filepath.Join(repository.root.Path, runHistoryFileName)); renameErr != nil {
				return fmt.Errorf("replace scenario run history: %w", renameErr)
			}

			return nil
		},
	); err != nil {
		return fmt.Errorf("write scenario run history: %w", err)
	}
	if err = repositoryfs.SyncDirectory(repository.root.Path); err != nil {
		return fmt.Errorf("sync scenario run history root: %w", err)
	}

	return nil
}

//nolint:cyclop // Terminal-record validation covers the durable invariants.
func validateTerminalRun(run RunRecord) error {
	if strings.TrimSpace(run.RunID) == "" || strings.TrimSpace(run.Reference) == "" || strings.TrimSpace(run.DefinitionRevision) == "" {
		return errors.New("run ID, reference, and definition revision are required")
	}
	if run.State != RunSucceeded && run.State != RunFailed && run.State != RunCancelled {
		return fmt.Errorf("run state %q is not terminal", run.State)
	}
	if run.StartedAt.IsZero() || run.FinishedAt.IsZero() || run.FinishedAt.Before(run.StartedAt) {
		return errors.New("valid started and finished timestamps are required")
	}
	if run.TotalSteps != len(run.Steps) || run.NextStep < 1 || run.NextStep > run.TotalSteps+1 {
		return errors.New("step totals or next step are inconsistent")
	}
	if run.Launch != nil {
		if run.Launch.ParameterValues == nil {
			return errors.New("known launch parameter values must be an object")
		}
		if rng := run.Launch.RNG; rng != nil {
			if !run.Launch.BaselineApplied {
				return errors.New("RNG provenance requires an applied baseline")
			}
			for _, value := range []string{rng.Seed, rng.SourceCalls} {
				number, err := strconv.ParseUint(value, 10, 64)
				if err != nil || strconv.FormatUint(number, 10) != value {
					return errors.New("RNG provenance requires canonical uint64 decimal strings")
				}
			}
		}
	}

	return nil
}

func sortRuns(runs []RunRecord) {
	sort.Slice(runs, func(i, j int) bool {
		if !runs[i].FinishedAt.Equal(runs[j].FinishedAt) {
			return runs[i].FinishedAt.After(runs[j].FinishedAt)
		}

		return runs[i].RunID > runs[j].RunID
	})
}

func cloneRuns(runs []RunRecord) ([]RunRecord, error) {
	data, err := json.Marshal(runs)
	if err != nil {
		return nil, err
	}
	var cloned []RunRecord
	if err = json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}

	return cloned, nil
}
