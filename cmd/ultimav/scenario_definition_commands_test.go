package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/scenarios"
)

func TestExecuteScenarioDefinitionCreateAndList(t *testing.T) {
	repository := newScenarioDefinitionTestRepository(t)
	definitionJSON := `{"schemaVersion":2,"id":"starvation-check","name":"Starvation Check","description":"Runs a deterministic starvation check.","baseline":{"type":"current"},"steps":[{"type":"note","note":"Prepare starvation."}]}`
	createRequest := scenarioDefinitionTestRequest(
		"create",
		debugging.CommandScenarioDefinitionCreate,
		map[string]any{"source": "personal", "definition": definitionJSON},
	)

	saved, handled := executeScenarioDefinitionDebugCommand(repository, debugging.NewInitialCatalog(), createRequest)

	require.True(t, handled)
	require.True(t, saved.Success, saved.ErrorMessage)
	assert.Equal(t, debugging.ExecutionAlteration, saved.EffectiveExecutionMode)
	assert.Equal(t, "scenario.definitions.personal:starvation-check", saved.Changes[0].Path)
	assert.JSONEq(t, "null", string(saved.Changes[0].Before))
	var saveData scenarioDefinitionMutationResultData
	require.NoError(t, json.Unmarshal(saved.Data, &saveData))
	assert.Equal(t, "personal:starvation-check", saveData.Reference)
	assert.Equal(t, "Starvation Check", saveData.Definition.Name)
	assert.NotEmpty(t, saveData.Definition.Revision)

	listed, handled := executeScenarioDefinitionDebugCommand(repository, debugging.NewInitialCatalog(), debugging.Request{
		SessionID: "test", ID: "list", CommandID: debugging.CommandScenarioDefinitionList,
	})
	require.True(t, handled)
	require.True(t, listed.Success, listed.ErrorMessage)
	var listData scenarioDefinitionListResult
	require.NoError(t, json.Unmarshal(listed.Data, &listData))
	require.Len(t, listData.Definitions, 1)
	assert.Equal(t, saveData.Definition, listData.Definitions[0])
	require.Len(t, listData.Definitions[0].Steps, 1)
	assert.Equal(t, scenarios.StepNote, listData.Definitions[0].Steps[0].Type)

}

func TestExecuteScenarioDefinitionGetReturnsFullVersionedDefinition(t *testing.T) {
	repository := newScenarioDefinitionTestRepository(t)
	minimum, maximum := 1, 99
	definition := scenarios.Definition{
		SchemaVersion: scenarios.CurrentSchemaVersion,
		ID:            "full-definition",
		Name:          "Full Definition",
		Description:   "Preserves the complete versioned schema.",
		Baseline:      scenarios.CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{{
			ID: "amount", Name: "Amount", Kind: debugging.ParameterInteger, Required: true,
			Minimum: &minimum, Maximum: &maximum,
		}},
		Steps: []scenarios.Step{
			{Type: scenarios.StepNote, Note: "Prepare the scenario."},
			{
				Type: scenarios.StepCommand, CommandID: debugging.CommandInventoryProvisionSet,
				Arguments: map[string]scenarios.CommandArgument{
					"provision": {Literal: json.RawMessage(`"food"`)},
					"quantity":  {Parameter: "amount"},
				},
			},
		},
	}
	_, err := repository.Create("project:full-definition", definition)
	require.NoError(t, err)
	request := scenarioDefinitionTestRequest(
		"get",
		debugging.CommandScenarioDefinitionGet,
		map[string]any{"reference": "project:full-definition"},
	)

	result, handled := executeScenarioDefinitionDebugCommand(repository, debugging.NewInitialCatalog(), request)

	require.True(t, handled)
	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, debugging.ExecutionQuery, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	var data scenarioDefinitionGetResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.Equal(t, "project:full-definition", data.Reference)
	assert.NotEmpty(t, data.Revision)
	assert.Equal(t, definition, data.Definition)
}

func TestExecuteScenarioDefinitionListAndGetExposePendingMigration(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))
	repository, err := scenarios.NewRepository(projectRoot, filepath.Join(root, "personal"))
	require.NoError(t, err)
	stored := []byte(`{"schemaVersion":2,"id":"legacy-character","name":"Legacy Character","description":"Uses an old character identity token.","baseline":{"type":"current"},"steps":[{"type":"command","commandId":"party.character.update","arguments":{"character":"0.QXZhdGFy","strength":20}}]}`)
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "legacy-character.json"), stored, 0o644))

	listed := executeScenarioDefinitionList(repository, debugging.Request{SessionID: "test", ID: "list", CommandID: debugging.CommandScenarioDefinitionList})
	require.True(t, listed.Success, listed.ErrorMessage)
	var listData scenarioDefinitionListResult
	require.NoError(t, json.Unmarshal(listed.Data, &listData))
	require.Len(t, listData.Definitions, 1)
	require.Len(t, listData.Definitions[0].Migrations, 1)
	assert.Equal(t, scenarios.MigrationLegacyCharacterToken, listData.Definitions[0].Migrations[0].Code)
	assert.JSONEq(t, `"1"`, string(listData.Definitions[0].Steps[0].Arguments["character"].Literal))

	result := executeScenarioDefinitionGet(repository, scenarioDefinitionTestRequest(
		"get", debugging.CommandScenarioDefinitionGet, map[string]any{"reference": "project:legacy-character"},
	))
	require.True(t, result.Success, result.ErrorMessage)
	var getData scenarioDefinitionGetResult
	require.NoError(t, json.Unmarshal(result.Data, &getData))
	assert.Equal(t, listData.Definitions[0].Migrations, getData.Migrations)

	duplicate := executeScenarioDefinitionDuplicate(repository, scenarioDefinitionTestRequest(
		"duplicate", debugging.CommandScenarioDefinitionDuplicate, map[string]any{
			"sourceReference": "project:legacy-character", "sourceRevision": getData.Revision,
			"destinationSource": "personal", "destinationId": "legacy-copy", "destinationName": "Legacy Copy",
		},
	))
	assert.False(t, duplicate.Success)
	assert.Equal(t, scenarioDefinitionMigrationError, duplicate.ErrorCode)
}

func TestExecuteScenarioDefinitionRevisionSafeLifecycle(t *testing.T) {
	repository := newScenarioDefinitionTestRepository(t)
	catalog := debugging.NewInitialCatalog()
	definition := `{"schemaVersion":2,"id":"revision-safe","name":"Revision Safe","description":"Exercises explicit scenario operations.","baseline":{"type":"current"},"steps":[{"type":"command","commandId":"environment.wind.set","arguments":{"direction":"north"}}]}`

	validated, handled := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"validate", debugging.CommandScenarioDefinitionValidate, map[string]any{"definition": definition},
	))
	require.True(t, handled)
	require.True(t, validated.Success, validated.ErrorMessage)
	var validation scenarioDefinitionValidationResult
	require.NoError(t, json.Unmarshal(validated.Data, &validation))
	assert.True(t, validation.Valid)
	assert.Empty(t, validation.Issues)

	created, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"create", debugging.CommandScenarioDefinitionCreate, map[string]any{"source": "project", "definition": definition},
	))
	require.True(t, created.Success, created.ErrorMessage)
	var createData scenarioDefinitionMutationResultData
	require.NoError(t, json.Unmarshal(created.Data, &createData))
	assert.NotEmpty(t, createData.Definition.Revision)

	exists, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"create-again", debugging.CommandScenarioDefinitionCreate, map[string]any{"source": "project", "definition": definition},
	))
	assert.Equal(t, scenarioDefinitionExistsError, exists.ErrorCode)

	updatedDefinition := `{"schemaVersion":2,"id":"revision-safe","name":"Revision Safe Updated","description":"Exercises explicit scenario operations.","baseline":{"type":"current"},"steps":[{"type":"command","commandId":"environment.wind.set","arguments":{"direction":"south"}}]}`
	updated, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"update", debugging.CommandScenarioDefinitionUpdate, map[string]any{
			"reference": "project:revision-safe", "expectedRevision": createData.Definition.Revision, "definition": updatedDefinition,
		},
	))
	require.True(t, updated.Success, updated.ErrorMessage)
	var updateData scenarioDefinitionMutationResultData
	require.NoError(t, json.Unmarshal(updated.Data, &updateData))
	assert.NotEqual(t, createData.Definition.Revision, updateData.Definition.Revision)

	stale, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"stale-update", debugging.CommandScenarioDefinitionUpdate, map[string]any{
			"reference": "project:revision-safe", "expectedRevision": createData.Definition.Revision, "definition": updatedDefinition,
		},
	))
	assert.Equal(t, scenarioDefinitionStaleError, stale.ErrorCode)

	renamedDefinition := `{"schemaVersion":2,"id":"renamed","name":"Renamed","description":"Must not silently rename an update.","baseline":{"type":"current"},"steps":[{"type":"command","commandId":"environment.wind.set","arguments":{"direction":"south"}}]}`
	renamed, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"rename-update", debugging.CommandScenarioDefinitionUpdate, map[string]any{
			"reference": "project:revision-safe", "expectedRevision": updateData.Definition.Revision, "definition": renamedDefinition,
		},
	))
	assert.Equal(t, scenarioDefinitionInvalidError, renamed.ErrorCode)
	_, err := repository.LoadRecord("project:renamed")
	assert.ErrorIs(t, err, scenarios.ErrScenarioNotFound)

	duplicated, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"duplicate", debugging.CommandScenarioDefinitionDuplicate, map[string]any{
			"sourceReference": "project:revision-safe", "sourceRevision": updateData.Definition.Revision,
			"destinationSource": "personal", "destinationId": "revision-copy", "destinationName": "Revision Copy",
		},
	))
	require.True(t, duplicated.Success, duplicated.ErrorMessage)
	copyRecord, err := repository.LoadRecord("personal:revision-copy")
	require.NoError(t, err)
	assert.Equal(t, "Revision Copy", copyRecord.Definition.Name)
	duplicateCollision, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"duplicate-collision", debugging.CommandScenarioDefinitionDuplicate, map[string]any{
			"sourceReference": "project:revision-safe", "sourceRevision": updateData.Definition.Revision,
			"destinationSource": "personal", "destinationId": "revision-copy", "destinationName": "Other Copy",
		},
	))
	assert.Equal(t, scenarioDefinitionExistsError, duplicateCollision.ErrorCode)
	unchangedCopy, err := repository.LoadRecord("personal:revision-copy")
	require.NoError(t, err)
	assert.Equal(t, "Revision Copy", unchangedCopy.Definition.Name)

	staleDelete, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"stale-delete", debugging.CommandScenarioDefinitionDelete, map[string]any{
			"reference": "project:revision-safe", "expectedRevision": createData.Definition.Revision,
		},
	))
	assert.Equal(t, scenarioDefinitionStaleError, staleDelete.ErrorCode)
	_, err = repository.LoadRecord("project:revision-safe")
	require.NoError(t, err)

	deleted, _ := executeScenarioDefinitionDebugCommand(repository, catalog, scenarioDefinitionTestRequest(
		"delete", debugging.CommandScenarioDefinitionDelete, map[string]any{
			"reference": "project:revision-safe", "expectedRevision": updateData.Definition.Revision,
		},
	))
	require.True(t, deleted.Success, deleted.ErrorMessage)
	_, err = repository.LoadRecord("project:revision-safe")
	assert.ErrorIs(t, err, scenarios.ErrScenarioNotFound)
}

func TestExecuteScenarioDefinitionValidateReportsCatalogIssuesWithoutMutation(t *testing.T) {
	repository := newScenarioDefinitionTestRepository(t)
	definition := `{"schemaVersion":2,"id":"invalid-command","name":"Invalid Command","description":"Does not mutate the repository.","baseline":{"type":"current"},"steps":[{"type":"command","commandId":"scenario.definition.delete","arguments":{"reference":"project:any","expectedRevision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}]}`

	result, _ := executeScenarioDefinitionDebugCommand(repository, debugging.NewInitialCatalog(), scenarioDefinitionTestRequest(
		"validate", debugging.CommandScenarioDefinitionValidate, map[string]any{"definition": definition},
	))

	require.True(t, result.Success, result.ErrorMessage)
	var data scenarioDefinitionValidationResult
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.False(t, data.Valid)
	require.NotEmpty(t, data.Issues)
	assert.Equal(t, scenarios.IssueManagementCommand, data.Issues[0].Code)
	listed, err := repository.List()
	require.NoError(t, err)
	assert.Empty(t, listed)

	malformed, _ := executeScenarioDefinitionDebugCommand(repository, debugging.NewInitialCatalog(), scenarioDefinitionTestRequest(
		"validate-malformed", debugging.CommandScenarioDefinitionValidate, map[string]any{"definition": `{"schemaVersion":2}`},
	))
	require.True(t, malformed.Success, malformed.ErrorMessage)
	require.NoError(t, json.Unmarshal(malformed.Data, &data))
	assert.False(t, data.Valid)
	require.Len(t, data.Issues, 1)
	assert.Equal(t, "$", data.Issues[0].Path)
	assert.Equal(t, scenarios.IssueDefinitionInvalid, data.Issues[0].Code)
}

func TestExecuteScenarioDefinitionCommandsReturnStableRecoverableFailures(t *testing.T) {
	repository := newScenarioDefinitionTestRepository(t)
	validDefinition := `{"schemaVersion":2,"id":"valid-id","name":"Valid","description":"Valid definition.","baseline":{"type":"current"},"steps":[{"type":"note","note":"Valid."}]}`
	tests := []struct {
		name       string
		repository *scenarios.Repository
		request    debugging.Request
		code       string
	}{
		{
			name: "repository unavailable", repository: nil,
			request: debugging.Request{SessionID: "test", ID: "list", CommandID: debugging.CommandScenarioDefinitionList},
			code:    scenarioRepositoryUnavailableError,
		},
		{
			name: "invalid source", repository: repository,
			request: scenarioDefinitionTestRequest("create", debugging.CommandScenarioDefinitionCreate, map[string]any{"source": "shared", "definition": validDefinition}),
			code:    scenarioDefinitionSourceError,
		},
		{
			name: "definition is not string", repository: repository,
			request: scenarioDefinitionTestRequest("create", debugging.CommandScenarioDefinitionCreate, map[string]any{"source": "project", "definition": map[string]any{}}),
			code:    scenarioDefinitionInvalidError,
		},
		{
			name: "strict unknown field", repository: repository,
			request: scenarioDefinitionTestRequest("create", debugging.CommandScenarioDefinitionCreate, map[string]any{
				"source": "project", "definition": validDefinition[:len(validDefinition)-1] + `,"unknown":true}`,
			}),
			code: scenarioDefinitionInvalidError,
		},
		{
			name: "get missing reference", repository: repository,
			request: debugging.Request{SessionID: "test", ID: "get", CommandID: debugging.CommandScenarioDefinitionGet},
			code:    scenarioDefinitionReferenceError,
		},
		{
			name: "get malformed reference", repository: repository,
			request: scenarioDefinitionTestRequest("get", debugging.CommandScenarioDefinitionGet, map[string]any{"reference": "valid-id"}),
			code:    scenarioDefinitionReferenceError,
		},
		{
			name: "get missing definition", repository: repository,
			request: scenarioDefinitionTestRequest("get", debugging.CommandScenarioDefinitionGet, map[string]any{"reference": "personal:missing"}),
			code:    scenarioDefinitionGetError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, handled := executeScenarioDefinitionDebugCommand(test.repository, debugging.NewInitialCatalog(), test.request)
			require.True(t, handled)
			assert.False(t, result.Success)
			assert.Equal(t, test.code, result.ErrorCode)
			assert.NotEmpty(t, result.ErrorMessage)
		})
	}
}

func TestExecuteScenarioDefinitionListReturnsRecoverableRepositoryError(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	require.NoError(t, os.WriteFile(projectRoot, []byte("not a directory"), 0o600))
	repository, err := scenarios.NewRepository(projectRoot, filepath.Join(root, "personal"))
	require.NoError(t, err)

	result := executeScenarioDefinitionList(repository, debugging.Request{
		SessionID: "test", ID: "list", CommandID: debugging.CommandScenarioDefinitionList,
	})

	assert.False(t, result.Success)
	assert.Equal(t, scenarioDefinitionListError, result.ErrorCode)
}

func TestExecuteScenarioDefinitionDebugCommandDoesNotClaimUnknownCommands(t *testing.T) {
	for _, commandID := range []string{"scenario.unknown", "scenario.definition.save"} {
		result, handled := executeScenarioDefinitionDebugCommand(nil, debugging.NewInitialCatalog(), debugging.Request{CommandID: commandID})

		assert.False(t, handled, commandID)
		assert.Equal(t, debugging.Result{}, result, commandID)
	}
}

func newScenarioDefinitionTestRepository(t *testing.T) *scenarios.Repository {
	t.Helper()
	root := t.TempDir()
	repository, err := scenarios.NewRepository(filepath.Join(root, "project"), filepath.Join(root, "personal"))
	require.NoError(t, err)

	return repository
}

func scenarioDefinitionTestRequest(id, commandID string, arguments map[string]any) debugging.Request {
	encoded := make(map[string]json.RawMessage, len(arguments))
	for name, value := range arguments {
		raw, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		encoded[name] = raw
	}

	return debugging.Request{SessionID: "test", ID: id, CommandID: commandID, Arguments: encoded}
}
