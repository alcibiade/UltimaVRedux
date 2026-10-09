package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/scenarios"
)

const defaultScenarioProjectDirectory = "debug/scenarios"

type scenarioParameterFlags map[string]json.RawMessage

func (values scenarioParameterFlags) String() string {
	return ""
}

//nolint:err113 // CLI validation errors identify the invalid parameter.
func (values scenarioParameterFlags) Set(value string) error {
	key, rawValue, found := strings.Cut(value, "=")
	if !found || strings.TrimSpace(key) == "" || rawValue == "" {
		return errors.New("scenario parameter must use key=value")
	}
	if _, exists := values[key]; exists {
		return fmt.Errorf("scenario parameter %q was supplied more than once", key)
	}

	encoded := json.RawMessage(rawValue)
	if !json.Valid(encoded) {
		stringValue, err := json.Marshal(rawValue)
		if err != nil {
			return fmt.Errorf("encoding scenario parameter %q: %w", key, err)
		}
		encoded = stringValue
	}
	values[key] = append(json.RawMessage(nil), encoded...)

	return nil
}

type scenarioStartupOptions struct {
	reference       string
	parameterValues scenarioParameterFlags
	projectRoot     string
	historyRoot     string
	initPath        string
}

func addScenarioFlags(options *scenarioStartupOptions) {
	options.parameterValues = make(scenarioParameterFlags)
	flag.StringVar(&options.reference, "scenario", "", "skip the title menu and run a project:<id> or personal:<id> scenario")
	flag.Var(options.parameterValues, "scenario-param", "scenario launch parameter as key=JSON-value; may be repeated")
	flag.StringVar(&options.projectRoot, "scenario-project-dir", defaultScenarioProjectDirectory, "project scripted-scenario directory")
	flag.StringVar(&options.historyRoot, "scenario-history-dir", "", "personal scenario run-history directory (default ~/.ultima_v_redux/scenario-runs)")
	flag.StringVar(&options.initPath, "scenario-init", "data/init/INIT.GAM", "clean scripted-scenario INIT.GAM baseline")
}

func configureScenarioRunner(game *Game, options scenarioStartupOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locating personal scenario directory: %w", err)
	}
	repository, err := scenarios.NewRepository(
		options.projectRoot,
		filepath.Join(home, ".ultima_v_redux", "scenarios"),
	)
	if err != nil {
		return fmt.Errorf("configuring scenario repository: %w", err)
	}
	historyRoot := options.historyRoot
	if historyRoot == "" {
		historyRoot = filepath.Join(home, ".ultima_v_redux", "scenario-runs")
	}
	history, err := scenarios.NewRunHistoryRepository(historyRoot, scenarios.DefaultRunHistoryLimit)
	if err != nil {
		return fmt.Errorf("configuring scenario run history: %w", err)
	}
	game.ConfigureScenarioRunner(repository, func() (Scene, error) {
		return newGameSceneFromInitTemplateFile(game.currentScene.GetUltimaConfiguration(), options.initPath)
	})
	game.ConfigureScenarioRunHistory(history)

	return nil
}

//nolint:err113 // Startup errors identify unavailable injected repositories.
func submitStartupScenario(game *Game, options scenarioStartupOptions) error {
	if options.reference == "" {
		return nil
	}
	parameterValues, err := json.Marshal(options.parameterValues)
	if err != nil {
		return fmt.Errorf("encoding scenario launch parameters: %w", err)
	}
	if game.scenarioRepository == nil {
		return errors.New("starting scenario: the scenario repository is unavailable")
	}
	record, err := game.scenarioRepository.LoadRecord(options.reference)
	if err != nil {
		return fmt.Errorf("loading startup scenario: %w", err)
	}
	baselineRevision := ""
	if record.Definition.Baseline.Type == scenarios.BaselineSnapshot {
		if game.snapshotRepository == nil {
			return errors.New("starting scenario: the snapshot repository is unavailable")
		}
		baseline, loadErr := game.snapshotRepository.Load(record.Definition.Baseline.Reference)
		if loadErr != nil {
			return fmt.Errorf("loading startup scenario baseline: %w", loadErr)
		}
		baselineRevision = baseline.Revision
	}
	sceneGeneration := game.DebugSceneGeneration()
	request := debugging.Request{
		SessionID:               "cli",
		ID:                      "cli-scenario-start",
		CommandID:               debugging.CommandScenarioRunStart,
		ExpectedSceneGeneration: sceneGeneration,
		Arguments: map[string]json.RawMessage{
			"reference":                json.RawMessage(fmt.Sprintf("%q", options.reference)),
			"expectedRevision":         json.RawMessage(fmt.Sprintf("%q", record.Revision)),
			"expectedBaselineRevision": json.RawMessage(fmt.Sprintf("%q", baselineRevision)),
			"reviewedSceneGeneration":  json.RawMessage(strconv.FormatUint(sceneGeneration, 10)),
			"parameterValues":          mustMarshalJSONString(parameterValues),
		},
	}
	if err = game.SubmitDebugRequest(request); err != nil {
		return fmt.Errorf("queueing startup scenario: %w", err)
	}

	return nil
}

func mustMarshalJSONString(value []byte) json.RawMessage {
	encoded, err := json.Marshal(string(value))
	if err != nil {
		panic("marshal scenario JSON string: " + err.Error())
	}

	return encoded
}
