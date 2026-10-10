package scenarios

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
)

// Frozen v1 history shape. New fields and terminal states belong only to v2.
type runRecordV1 struct {
	RunID                  string          `json:"runId"`
	Reference              string          `json:"reference"`
	DefinitionID           string          `json:"definitionId"`
	DefinitionRevision     string          `json:"definitionRevision"`
	Baseline               Baseline        `json:"baseline"`
	BaselineReference      string          `json:"baselineReference,omitempty"`
	BaselineRevision       string          `json:"baselineRevision,omitempty"`
	Rollback               *RunRollback    `json:"rollback,omitempty"`
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

func decodeRunHistory(data []byte) (runHistoryEnvelope, error) {
	var header struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return runHistoryEnvelope{}, err
	}
	switch header.SchemaVersion {
	case 1:
		var old struct {
			SchemaVersion int           `json:"schemaVersion"`
			Runs          []runRecordV1 `json:"runs"`
		}
		if err := repositoryfs.DecodeStrictJSON(data, &old); err != nil {
			return runHistoryEnvelope{}, err
		}
		envelope := runHistoryEnvelope{SchemaVersion: runHistorySchemaVersion, Runs: make([]RunRecord, len(old.Runs))}
		for index, run := range old.Runs {
			if run.State != RunSucceeded && run.State != RunFailed {
				return runHistoryEnvelope{}, fmt.Errorf("v1 run state %q is not terminal", run.State)
			}
			envelope.Runs[index] = RunRecord{
				RunID: run.RunID, Reference: run.Reference, DefinitionID: run.DefinitionID, DefinitionRevision: run.DefinitionRevision,
				Baseline: run.Baseline, BaselineReference: run.BaselineReference, BaselineRevision: run.BaselineRevision,
				Rollback: run.Rollback, State: run.State, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt,
				CurrentProcess: run.CurrentProcess, NextStep: run.NextStep, TotalSteps: run.TotalSteps,
				InitialSceneGeneration: run.InitialSceneGeneration, SceneGeneration: run.SceneGeneration,
				Steps: run.Steps, ErrorCode: run.ErrorCode, ErrorMessage: run.ErrorMessage,
			}
		}

		return envelope, nil
	case runHistorySchemaVersion:
		var envelope runHistoryEnvelope
		if err := repositoryfs.DecodeStrictJSON(data, &envelope); err != nil {
			return runHistoryEnvelope{}, err
		}

		return envelope, nil
	default:
		return runHistoryEnvelope{}, fmt.Errorf("unsupported schema version %d", header.SchemaVersion)
	}
}
