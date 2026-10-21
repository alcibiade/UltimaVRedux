package debugging

import "encoding/json"

const (
	// CommandScenarioStatus reports context and state used by forced scenarios.
	CommandScenarioStatus = "scenario.status"
	// CommandScenarioDefinitionList lists durable scripted scenario definitions.
	CommandScenarioDefinitionList = "scenario.definition.list"
	// CommandScenarioDefinitionGet returns one complete durable scripted scenario definition.
	CommandScenarioDefinitionGet = "scenario.definition.get"
	// CommandScenarioDefinitionValidate validates a scripted scenario without persisting it.
	CommandScenarioDefinitionValidate = "scenario.definition.validate"
	// CommandScenarioDefinitionCreate creates one absent durable scripted scenario definition.
	CommandScenarioDefinitionCreate = "scenario.definition.create"
	// CommandScenarioDefinitionUpdate conditionally updates one durable scripted scenario definition.
	CommandScenarioDefinitionUpdate = "scenario.definition.update"
	// CommandScenarioDefinitionDelete conditionally deletes one durable scripted scenario definition.
	CommandScenarioDefinitionDelete = "scenario.definition.delete"
	// CommandScenarioDefinitionDuplicate copies one revision to an absent explicit destination.
	CommandScenarioDefinitionDuplicate = "scenario.definition.duplicate"
	// CommandScenarioRunStart starts one durable scripted scenario workflow.
	CommandScenarioRunStart = "scenario.run.start"
	// CommandScenarioRunStatus reports the active run and bounded durable history.
	CommandScenarioRunStatus = "scenario.run.status"
	// CommandScenarioRunCancel cancels only the remaining steps of one identified run.
	CommandScenarioRunCancel = "scenario.run.cancel"
)

func scenarioCommands() []CommandDefinition {
	referencePattern := `^(project|personal):[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`
	slugPattern := `^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`
	revisionPattern := `^[a-f0-9]{64}$`
	minimumSceneGeneration, maximumSceneGeneration := 1, 2147483647
	sourceParameter := func(id, name string) ParameterDefinition {
		return ParameterDefinition{
			ID: id, Name: name, Kind: ParameterEnum, Required: true,
			Choices:     []Choice{{Value: "project", Label: "Project"}, {Value: "personal", Label: "Personal"}},
			ControlHint: controlHintSelect,
		}
	}
	definitionParameter := func() ParameterDefinition {
		return ParameterDefinition{
			ID: "definition", Name: "JSON Definition", Description: "Complete scripted scenario definition encoded as JSON.",
			Kind: ParameterString, Required: true,
		}
	}
	referenceParameter := func(id, name string) ParameterDefinition {
		return ParameterDefinition{ID: id, Name: name, Kind: ParameterString, Required: true, Pattern: referencePattern}
	}
	revisionParameter := func(id, name string) ParameterDefinition {
		return ParameterDefinition{ID: id, Name: name, Kind: ParameterString, Required: true, Pattern: revisionPattern}
	}
	optionalRevisionParameter := func(id, name string) ParameterDefinition {
		return ParameterDefinition{ID: id, Name: name, Kind: ParameterString, Default: json.RawMessage(`""`), Pattern: `^(?:[a-f0-9]{64})?$`}
	}

	return []CommandDefinition{
		{
			ID: CommandScenarioStatus, Category: "Scenarios", Name: "Scenario Lab",
			Description: "Inspect and run exceptional deterministic gameplay scenarios.", ExecutionModes: []ExecutionMode{ExecutionQuery},
		},
		{
			ID: CommandScenarioDefinitionList, Category: "Scenarios", Name: "List Scripted Scenarios",
			Description:    "List durable scripted scenario definitions from project and personal sources.",
			ExecutionModes: []ExecutionMode{ExecutionQuery},
		},
		{
			ID: CommandScenarioDefinitionGet, Category: "Scenarios", Name: "Get Scripted Scenario",
			Description:    "Get one complete versioned scripted scenario definition by source-qualified reference.",
			Parameters:     []ParameterDefinition{referenceParameter("reference", "Source-Qualified Reference")},
			ExecutionModes: []ExecutionMode{ExecutionQuery},
		},
		{
			ID: CommandScenarioDefinitionValidate, Category: "Scenarios", Name: "Validate Scripted Scenario",
			Description:    "Validate one JSON scripted scenario definition against the current command catalog without mutation.",
			Parameters:     []ParameterDefinition{definitionParameter()},
			ExecutionModes: []ExecutionMode{ExecutionQuery},
		},
		{
			ID: CommandScenarioDefinitionCreate, Category: "Scenarios", Name: "Create Scripted Scenario",
			Description:    "Validate and create one JSON scripted scenario definition only when its source-qualified destination is absent.",
			Parameters:     []ParameterDefinition{sourceParameter("source", "Source"), definitionParameter()},
			ExecutionModes: []ExecutionMode{ExecutionAlteration}, Destructive: true,
		},
		{
			ID: CommandScenarioDefinitionUpdate, Category: "Scenarios", Name: "Update Scripted Scenario",
			Description: "Validate and update one scripted scenario only when its stored revision matches; the definition ID cannot rename or move it.",
			Parameters: []ParameterDefinition{
				referenceParameter("reference", "Source-Qualified Reference"),
				revisionParameter("expectedRevision", "Expected Revision"),
				definitionParameter(),
			},
			ExecutionModes: []ExecutionMode{ExecutionAlteration}, Destructive: true,
		},
		{
			ID: CommandScenarioDefinitionDelete, Category: "Scenarios", Name: "Delete Scripted Scenario",
			Description: "Delete one scripted scenario only when its stored revision matches.",
			Parameters: []ParameterDefinition{
				referenceParameter("reference", "Source-Qualified Reference"),
				revisionParameter("expectedRevision", "Expected Revision"),
			},
			ExecutionModes: []ExecutionMode{ExecutionAlteration}, Destructive: true,
		},
		{
			ID: CommandScenarioDefinitionDuplicate, Category: "Scenarios", Name: "Duplicate Scripted Scenario",
			Description: "Copy one reviewed scenario revision to an explicit absent destination without overwriting.",
			Parameters: []ParameterDefinition{
				referenceParameter("sourceReference", "Source Reference"),
				revisionParameter("sourceRevision", "Source Revision"),
				sourceParameter("destinationSource", "Destination Source"),
				{ID: "destinationId", Name: "Destination ID", Kind: ParameterString, Required: true, Pattern: slugPattern},
				{ID: "destinationName", Name: "Destination Name", Kind: ParameterString, Required: true},
			},
			ExecutionModes: []ExecutionMode{ExecutionAlteration}, Destructive: true,
		},
		{
			ID: CommandScenarioRunStart, Category: "Scenarios", Name: "Start Scripted Scenario",
			Description: "Start a reviewed durable scripted scenario revision against the reviewed active scene generation.",
			Parameters: []ParameterDefinition{
				{
					ID: "reference", Name: "Source-Qualified Reference", Kind: ParameterString, Required: true,
					Pattern: `^(project|personal):[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`, Availability: []string{contextGameplay},
				},
				revisionParameter("expectedRevision", "Reviewed Definition Revision"),
				optionalRevisionParameter("expectedBaselineRevision", "Reviewed Baseline Revision"),
				{
					ID: "reviewedSceneGeneration", Name: "Reviewed Scene Generation", Kind: ParameterInteger, Required: true,
					Minimum: &minimumSceneGeneration, Maximum: &maximumSceneGeneration, ControlHint: controlHintNumber,
					Availability: []string{contextGameplay},
				},
				{
					ID: "parameterValues", Name: "JSON Parameter Values", Description: "Launch parameter values encoded as a JSON object.",
					Kind: ParameterString, Default: json.RawMessage(`"{}"`), Availability: []string{contextGameplay},
				},
			},
			ExecutionModes: []ExecutionMode{ExecutionWorkflow}, Destructive: true,
		},
		{
			ID: CommandScenarioRunStatus, Category: "Scenarios", Name: "Scripted Scenario Run Status",
			Description:    "Report the active scripted scenario run and bounded private history with ordered step results.",
			ExecutionModes: []ExecutionMode{ExecutionQuery},
		},
	}
}

func scenarioRunCancelCommands() []CommandDefinition {
	return []CommandDefinition{{
		ID: CommandScenarioRunCancel, Category: "Scenarios", Name: "Cancel Remaining Steps",
		Description: "Stop remaining steps of the identified run when processed; retain completed effects, the applied baseline, and production workflows. Does not restore rollback.",
		Parameters: []ParameterDefinition{{
			ID: "runId", Name: "Run ID", Kind: ParameterString, Required: true, Pattern: `^\S+$`,
		}},
		ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, ScenarioExcluded: true,
	}}
}
