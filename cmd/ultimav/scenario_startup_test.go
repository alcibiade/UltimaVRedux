package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/scenarios"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestScenarioParameterFlagsPreserveJSONTypesAndQuotePlainText(t *testing.T) {
	values := make(scenarioParameterFlags)

	require.NoError(t, values.Set("count=3"))
	require.NoError(t, values.Set("enabled=true"))
	require.NoError(t, values.Set("direction=north"))

	assert.JSONEq(t, `3`, string(values["count"]))
	assert.JSONEq(t, `true`, string(values["enabled"]))
	assert.JSONEq(t, `"north"`, string(values["direction"]))
	assert.Error(t, values.Set("count=4"))
	assert.Error(t, values.Set("missing-separator"))
}

func TestConfigureScenarioRunnerUsesConfiguredHistoryRoot(t *testing.T) {
	root := t.TempDir()
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	require.NoError(t, configureScenarioRunner(game, scenarioStartupOptions{
		projectRoot: filepath.Join(root, "project"),
		historyRoot: filepath.Join(root, "history"),
		initPath:    filepath.Join(root, "INIT.GAM"),
	}))

	require.NotNil(t, game.scenarioRunHistory)
	assert.Equal(t, scenarios.DefaultRunHistoryLimit, game.scenarioRunHistory.Capacity())
}

func TestSubmitStartupScenarioResolvesSnapshotBaselineRevision(t *testing.T) {
	root := t.TempDir()
	repository, err := scenarios.NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	require.NoError(t, err)
	target := snapshotCommandTestRecord(t, "project:cli-baseline", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	_, err = repository.Create("project:cli-snapshot", scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion, ID: "cli-snapshot", Name: "CLI Snapshot",
		Description: "Pins its baseline without browser review.", Baseline: scenarios.SnapshotBaseline(target.Reference.String()),
		Steps: []scenarios.Step{{Type: scenarios.StepNote, Note: "Ready."}},
	})
	require.NoError(t, err)
	snapshotRepository := &snapshotCommandTestRepository{records: map[string]save_snapshot.Record{target.Reference.String(): target}}
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.ConfigureSnapshotRepository(snapshotRepository, func(_ *config.UltimaVConfiguration, _ save_snapshot.SaveSnapshot) (Scene, error) {
		return &scenarioRunnerTestScene{ready: true}, nil
	})
	game.ConfigureScenarioRunner(repository, nil)

	require.NoError(t, submitStartupScenario(game, scenarioStartupOptions{reference: "project:cli-snapshot", parameterValues: scenarioParameterFlags{}}))
	require.NoError(t, game.Update())

	require.NotNil(t, game.scenarioRun)
	assert.Equal(t, target.Revision, game.scenarioRun.status.BaselineRevision)
}

func TestSubmitStartupScenarioQueuesBoundRun(t *testing.T) {
	root := t.TempDir()
	repository, err := scenarios.NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	require.NoError(t, err)
	_, err = repository.Create("project:cli-run", scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion,
		ID:            "cli-run",
		Name:          "CLI Run",
		Description:   "Verifies startup scenario parameter binding.",
		Baseline:      scenarios.CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{{
			ID: "enabled", Name: "Enabled", Kind: debugging.ParameterBoolean, Required: true,
		}},
		Steps: []scenarios.Step{{
			Type: scenarios.StepCommand, CommandID: debugging.CommandMovementFreeMoveSet,
			Arguments: map[string]scenarios.CommandArgument{"enabled": {Parameter: "enabled"}},
		}},
	})
	require.NoError(t, err)
	scene := &scenarioRunnerTestScene{ready: true}
	game := NewGame(scene)
	game.ConfigureScenarioRunner(repository, nil)
	options := scenarioStartupOptions{
		reference:       "project:cli-run",
		parameterValues: scenarioParameterFlags{"enabled": json.RawMessage(`true`)},
	}

	require.NoError(t, submitStartupScenario(game, options))
	require.NoError(t, game.Update())
	assert.Empty(t, scene.executed, "the accepted start command must not execute a scenario step in the same update")
	require.NoError(t, game.Update())
	require.Len(t, scene.executed, 1)
	assert.JSONEq(t, `true`, string(scene.executed[0].Arguments["enabled"]))
}

func TestSubmitStartupScenarioUsesMigratedCharacterTargetWithoutWritingSource(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))
	repository, err := scenarios.NewRepository(projectRoot, filepath.Join(root, "personal"))
	require.NoError(t, err)
	stored := []byte(`{"schemaVersion":2,"id":"cli-legacy","name":"CLI Legacy","description":"Uses a migrated character target.","baseline":{"type":"current"},"steps":[{"type":"command","commandId":"party.character.update","arguments":{"character":"0.QXZhdGFy","strength":20}}]}`)
	path := filepath.Join(projectRoot, "cli-legacy.json")
	require.NoError(t, os.WriteFile(path, stored, 0o644))
	game := NewGame(&scenarioRunnerTestScene{ready: true})
	game.ConfigureScenarioRunner(repository, nil)

	require.NoError(t, submitStartupScenario(game, scenarioStartupOptions{
		reference: "project:cli-legacy", parameterValues: scenarioParameterFlags{},
	}))
	require.NoError(t, game.Update())

	require.NotNil(t, game.scenarioRun)
	assert.JSONEq(t, `"1"`, string(game.scenarioRun.requests[0].Arguments["character"]))
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, stored, after)
}
