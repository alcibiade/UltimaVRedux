package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/scenarios"
)

const (
	scenarioRepositoryUnavailableError = "scenario_repository_unavailable"
	scenarioDefinitionListError        = "scenario_definition_list_failed"
	scenarioDefinitionReferenceError   = "invalid_scenario_definition_reference"
	scenarioDefinitionGetError         = "scenario_definition_get_failed"
	scenarioDefinitionSourceError      = "invalid_scenario_definition_source"
	scenarioDefinitionInvalidError     = "invalid_scenario_definition"
	scenarioDefinitionValidationError  = "scenario_definition_validation_failed"
	scenarioDefinitionCreateError      = "scenario_definition_create_failed"
	scenarioDefinitionUpdateError      = "scenario_definition_update_failed"
	scenarioDefinitionDeleteError      = "scenario_definition_delete_failed"
	scenarioDefinitionDuplicateError   = "scenario_definition_duplicate_failed"
	scenarioDefinitionStaleError       = "stale_revision"
	scenarioDefinitionExistsError      = "already_exists"
	scenarioDefinitionMigrationError   = "pending_scenario_migration"
)

type scenarioDefinitionSummaryResult struct {
	Reference   string                          `json:"reference"`
	Source      string                          `json:"source"`
	ID          string                          `json:"id"`
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	Baseline    scenarios.Baseline              `json:"baseline"`
	Revision    string                          `json:"revision"`
	Parameters  []debugging.ParameterDefinition `json:"parameters,omitempty"`
	Steps       []scenarios.Step                `json:"steps"`
	Migrations  []scenarios.Migration           `json:"migrations,omitempty"`
}

type scenarioDefinitionListResult struct {
	Definitions []scenarioDefinitionSummaryResult `json:"definitions"`
}

type scenarioDefinitionGetResult struct {
	Reference  string                `json:"reference"`
	Revision   string                `json:"revision"`
	Definition scenarios.Definition  `json:"definition"`
	Migrations []scenarios.Migration `json:"migrations,omitempty"`
}

type scenarioDefinitionValidationResult struct {
	Valid  bool                               `json:"valid"`
	Issues []scenarios.CatalogValidationIssue `json:"issues"`
}

type scenarioDefinitionMutationResultData struct {
	Reference  string                          `json:"reference"`
	Definition scenarioDefinitionSummaryResult `json:"definition"`
}

// executeScenarioDefinitionDebugCommand handles repository-only scenario commands
// without requiring a Game or GameScene integration point.
func executeScenarioDefinitionDebugCommand(
	repository *scenarios.Repository,
	catalog *debugging.Catalog,
	request debugging.Request,
) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandScenarioDefinitionList:
		return executeScenarioDefinitionList(repository, request), true
	case debugging.CommandScenarioDefinitionGet:
		return executeScenarioDefinitionGet(repository, request), true
	case debugging.CommandScenarioDefinitionValidate:
		return executeScenarioDefinitionValidate(catalog, request), true
	case debugging.CommandScenarioDefinitionCreate:
		return executeScenarioDefinitionCreate(repository, catalog, request), true
	case debugging.CommandScenarioDefinitionUpdate:
		return executeScenarioDefinitionUpdate(repository, catalog, request), true
	case debugging.CommandScenarioDefinitionDelete:
		return executeScenarioDefinitionDelete(repository, request), true
	case debugging.CommandScenarioDefinitionDuplicate:
		return executeScenarioDefinitionDuplicate(repository, request), true
	default:
		return debugging.Result{}, false
	}
}

func executeScenarioDefinitionList(repository *scenarios.Repository, request debugging.Request) debugging.Result {
	if repository == nil {
		return debugging.FailureResult(request, scenarioRepositoryUnavailableError, "the scenario repository is unavailable")
	}

	summaries, err := repository.List()
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionListError, err.Error())
	}
	definitions := make([]scenarioDefinitionSummaryResult, 0, len(summaries))
	for _, summary := range summaries {
		definitions = append(definitions, newScenarioDefinitionSummary(
			summary.Reference,
			scenarios.Definition{
				Name: summary.Name, Description: summary.Description, Baseline: summary.Baseline,
				Parameters: summary.Parameters, Steps: summary.Steps,
			},
			summary.Revision,
			summary.Migrations,
		))
	}

	return debugSuccessResult(
		request,
		fmt.Sprintf("Listed %d scenario definitions.", len(definitions)),
		scenarioDefinitionListResult{Definitions: definitions},
	)
}

func executeScenarioDefinitionGet(repository *scenarios.Repository, request debugging.Request) debugging.Result {
	if repository == nil {
		return debugging.FailureResult(request, scenarioRepositoryUnavailableError, "the scenario repository is unavailable")
	}

	reference, err := scenarioDefinitionReferenceArgument(request, "reference")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	qualifiedReference := reference.String()
	record, err := repository.LoadRecord(qualifiedReference)
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionGetError, err.Error())
	}

	return debugSuccessResult(
		request,
		"Loaded scenario definition "+qualifiedReference+".",
		scenarioDefinitionGetResult{
			Reference: qualifiedReference, Revision: record.Revision, Definition: record.Definition, Migrations: record.Migrations,
		},
	)
}

func executeScenarioDefinitionValidate(catalog *debugging.Catalog, request debugging.Request) debugging.Result {
	definition, decodeIssue := decodeScenarioDefinitionIssue(request)
	issues := []scenarios.CatalogValidationIssue{}
	if decodeIssue != nil {
		issues = append(issues, *decodeIssue)
	} else {
		issues = scenarios.ValidateAgainstCatalog(definition, catalog, scenarioCatalogValidationPolicy())
	}

	return debugSuccessResult(
		request,
		fmt.Sprintf("Scenario definition validation found %d issue(s).", len(issues)),
		scenarioDefinitionValidationResult{Valid: len(issues) == 0, Issues: issues},
	)
}

func executeScenarioDefinitionCreate(
	repository *scenarios.Repository,
	catalog *debugging.Catalog,
	request debugging.Request,
) debugging.Result {
	if repository == nil {
		return debugging.FailureResult(request, scenarioRepositoryUnavailableError, "the scenario repository is unavailable")
	}
	source, err := scenarioDefinitionSourceArgument(request, "source")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionSourceError, err.Error())
	}
	definition, result := validatedScenarioDefinitionArgument(catalog, request)
	if result != nil {
		return *result
	}
	reference := scenarios.Reference{Source: source, ID: definition.ID}
	record, err := repository.Create(reference.String(), definition)
	if err != nil {
		return scenarioRepositoryMutationFailure(request, scenarioDefinitionCreateError, err)
	}

	return scenarioDefinitionMutationResult(request, "Created", reference, scenarios.Record{}, record)
}

func executeScenarioDefinitionUpdate(
	repository *scenarios.Repository,
	catalog *debugging.Catalog,
	request debugging.Request,
) debugging.Result {
	if repository == nil {
		return debugging.FailureResult(request, scenarioRepositoryUnavailableError, "the scenario repository is unavailable")
	}
	reference, err := scenarioDefinitionReferenceArgument(request, "reference")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	expectedRevision, err := scenarioStringArgument(request, "expectedRevision")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	definition, result := validatedScenarioDefinitionArgument(catalog, request)
	if result != nil {
		return *result
	}
	if definition.ID != reference.ID {
		return debugging.FailureResult(request, scenarioDefinitionInvalidError, "definition ID must match the update reference; use duplicate for a new destination")
	}
	before, err := repository.LoadRecord(reference.String())
	if err != nil {
		return scenarioRepositoryMutationFailure(request, scenarioDefinitionUpdateError, err)
	}
	record, err := repository.Update(reference.String(), expectedRevision, definition)
	if err != nil {
		return scenarioRepositoryMutationFailure(request, scenarioDefinitionUpdateError, err)
	}

	return scenarioDefinitionMutationResult(request, "Updated", reference, before, record)
}

func executeScenarioDefinitionDelete(repository *scenarios.Repository, request debugging.Request) debugging.Result {
	if repository == nil {
		return debugging.FailureResult(request, scenarioRepositoryUnavailableError, "the scenario repository is unavailable")
	}
	reference, err := scenarioDefinitionReferenceArgument(request, "reference")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	expectedRevision, err := scenarioStringArgument(request, "expectedRevision")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	before, err := repository.LoadRecord(reference.String())
	if err != nil {
		return scenarioRepositoryMutationFailure(request, scenarioDefinitionDeleteError, err)
	}
	if err = repository.Delete(reference.String(), expectedRevision); err != nil {
		return scenarioRepositoryMutationFailure(request, scenarioDefinitionDeleteError, err)
	}

	result := debugSuccessResultWithMode(
		request, debugging.ExecutionAlteration, "Deleted scenario definition "+reference.String()+".",
		map[string]string{"reference": reference.String(), "revision": before.Revision},
	)
	result.Changes = []debugging.StateChange{{
		Path: "scenario.definitions." + reference.String(), Before: mustDebugJSON(before.Definition), After: json.RawMessage("null"),
	}}

	return result
}

func executeScenarioDefinitionDuplicate(repository *scenarios.Repository, request debugging.Request) debugging.Result {
	if repository == nil {
		return debugging.FailureResult(request, scenarioRepositoryUnavailableError, "the scenario repository is unavailable")
	}
	source, err := scenarioDefinitionReferenceArgument(request, "sourceReference")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	sourceRevision, err := scenarioStringArgument(request, "sourceRevision")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	destinationSource, err := scenarioDefinitionSourceArgument(request, "destinationSource")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionSourceError, err.Error())
	}
	destinationID, err := scenarioStringArgument(request, "destinationId")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionReferenceError, err.Error())
	}
	destinationName, err := scenarioStringArgument(request, "destinationName")
	if err != nil {
		return debugging.FailureResult(request, scenarioDefinitionInvalidError, err.Error())
	}
	destination := scenarios.Reference{Source: destinationSource, ID: destinationID}
	created, err := repository.DuplicateAs(source.String(), sourceRevision, destination.String(), destinationName)
	if err != nil {
		return scenarioRepositoryMutationFailure(request, scenarioDefinitionDuplicateError, err)
	}

	return scenarioDefinitionMutationResult(request, "Duplicated", destination, scenarios.Record{}, created)
}

func scenarioDefinitionReferenceArgument(request debugging.Request, argumentID string) (scenarios.Reference, error) {
	qualifiedReference, err := scenarioStringArgument(request, argumentID)
	if err != nil {
		return scenarios.Reference{}, err
	}

	return scenarios.ParseReference(qualifiedReference)
}

func scenarioStringArgument(request debugging.Request, argumentID string) (string, error) {
	raw, exists := request.Arguments[argumentID]
	if !exists {
		return "", fmt.Errorf("%s is required", argumentID)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", argumentID)
	}

	return value, nil
}

func scenarioDefinitionSourceArgument(request debugging.Request, argumentID string) (scenarios.Source, error) {
	value, err := scenarioStringArgument(request, argumentID)
	if err != nil {
		return "", err
	}
	source := scenarios.Source(value)
	if source != scenarios.SourceProject && source != scenarios.SourcePersonal {
		return "", fmt.Errorf("unsupported scenario source %q", source)
	}

	return source, nil
}

func scenarioDefinitionArgument(request debugging.Request) (scenarios.Definition, error) {
	raw, exists := request.Arguments["definition"]
	if !exists {
		return scenarios.Definition{}, errors.New("definition is required")
	}
	var definitionJSON string
	if err := json.Unmarshal(raw, &definitionJSON); err != nil {
		return scenarios.Definition{}, errors.New("definition must be a JSON string")
	}
	definition, err := scenarios.DecodeDefinition([]byte(definitionJSON))
	if err != nil {
		return scenarios.Definition{}, err
	}

	return definition, nil
}

func decodeScenarioDefinitionIssue(request debugging.Request) (scenarios.Definition, *scenarios.CatalogValidationIssue) {
	definition, err := scenarioDefinitionArgument(request)
	if err == nil {
		return definition, nil
	}

	return scenarios.Definition{}, &scenarios.CatalogValidationIssue{
		Path: "$", Code: scenarios.IssueDefinitionInvalid, Message: err.Error(),
	}
}

func validatedScenarioDefinitionArgument(
	catalog *debugging.Catalog,
	request debugging.Request,
) (scenarios.Definition, *debugging.Result) {
	definition, decodeIssue := decodeScenarioDefinitionIssue(request)
	if decodeIssue != nil {
		result := debugging.FailureResult(request, scenarioDefinitionInvalidError, decodeIssue.Message)

		return scenarios.Definition{}, &result
	}
	issues := scenarios.ValidateAgainstCatalog(definition, catalog, scenarioCatalogValidationPolicy())
	if len(issues) > 0 {
		result := debugging.FailureResult(
			request,
			scenarioDefinitionValidationError,
			fmt.Sprintf("%s: %s", issues[0].Path, issues[0].Message),
		)

		return scenarios.Definition{}, &result
	}

	return definition, nil
}

func scenarioCatalogValidationPolicy() scenarios.CatalogValidationPolicy {
	return scenarios.CatalogValidationPolicy{
		AllowedExecutionModes: []debugging.ExecutionMode{
			debugging.ExecutionLiveSafe,
			debugging.ExecutionSynchronized,
			debugging.ExecutionWorkflow,
			debugging.ExecutionAlteration,
			debugging.ExecutionSceneRestart,
		},
		IsManagementCommand: isScenarioManagementCommand,
	}
}

func scenarioRepositoryMutationFailure(request debugging.Request, fallbackCode string, err error) debugging.Result {
	switch {
	case errors.Is(err, scenarios.ErrScenarioStale):
		return debugging.FailureResult(request, scenarioDefinitionStaleError, err.Error())
	case errors.Is(err, scenarios.ErrScenarioExists):
		return debugging.FailureResult(request, scenarioDefinitionExistsError, err.Error())
	case errors.Is(err, scenarios.ErrPendingMigration):
		return debugging.FailureResult(request, scenarioDefinitionMigrationError, err.Error())
	default:
		return debugging.FailureResult(request, fallbackCode, err.Error())
	}
}

func scenarioDefinitionMutationResult(
	request debugging.Request,
	verb string,
	reference scenarios.Reference,
	before scenarios.Record,
	after scenarios.Record,
) debugging.Result {
	summary := newScenarioDefinitionSummary(
		reference,
		after.Definition,
		after.Revision,
		after.Migrations,
	)
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionAlteration,
		verb+" scenario definition "+reference.String()+".",
		scenarioDefinitionMutationResultData{Reference: reference.String(), Definition: summary},
	)
	beforeJSON := json.RawMessage("null")
	if before.Revision != "" {
		beforeJSON = mustDebugJSON(before.Definition)
	}
	afterJSON := mustDebugJSON(after.Definition)
	if !bytes.Equal(beforeJSON, afterJSON) {
		result.Changes = []debugging.StateChange{{
			Path: "scenario.definitions." + reference.String(), Before: beforeJSON, After: afterJSON,
		}}
	}

	return result
}

func newScenarioDefinitionSummary(
	reference scenarios.Reference,
	definition scenarios.Definition,
	revision string,
	migrations []scenarios.Migration,
) scenarioDefinitionSummaryResult {
	return scenarioDefinitionSummaryResult{
		Reference:   reference.String(),
		Source:      string(reference.Source),
		ID:          reference.ID,
		Name:        definition.Name,
		Description: definition.Description,
		Baseline:    definition.Baseline,
		Revision:    revision,
		Parameters:  definition.Parameters,
		Steps:       definition.Steps,
		Migrations:  migrations,
	}
}
