package debugging

import "encoding/json"

const (
	// CommandSaveSnapshotList lists detached repository metadata.
	CommandSaveSnapshotList = "save.snapshot.list"
	// CommandSaveSnapshotGet returns detached detail for one repository record.
	CommandSaveSnapshotGet = "save.snapshot.get"
	// CommandSaveSnapshotCapture creates an immutable named capture.
	CommandSaveSnapshotCapture = "save.snapshot.capture"
	// CommandSaveSnapshotLoad atomically replaces gameplay after persisting rollback state.
	CommandSaveSnapshotLoad = "save.snapshot.load"
	// CommandSaveSnapshotDelete revision-safely removes a named capture.
	CommandSaveSnapshotDelete = "save.snapshot.delete"
)

func saveSnapshotCommands() []CommandDefinition {
	referencePattern := `^(project|personal|rollback):[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`
	namedReferencePattern := `^(project|personal):[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`
	revisionPattern := `^[0-9a-f]{64}$`
	idPattern := `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`
	reference := func(pattern string) ParameterDefinition {
		return ParameterDefinition{ID: "reference", Name: "Snapshot", Kind: ParameterString, Required: true, Pattern: pattern}
	}
	revision := ParameterDefinition{ID: "expectedRevision", Name: "Expected Revision", Kind: ParameterString, Required: true, Pattern: revisionPattern}

	return []CommandDefinition{
		{ID: CommandSaveSnapshotList, Category: "Save Snapshots", Name: "List Save Snapshots", Description: "List detached metadata and current eligibility for named and rollback snapshots.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandSaveSnapshotGet, Category: "Save Snapshots", Name: "Inspect Save Snapshot", Description: "Inspect detached metadata and a compact gameplay summary without returning the full snapshot.", Parameters: []ParameterDefinition{reference(referencePattern)}, ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{
			ID: CommandSaveSnapshotCapture, Category: "Save Snapshots", Name: "Capture Save Snapshot", Description: "Capture the stable active gameplay scene as a new immutable project or personal snapshot.",
			Parameters: []ParameterDefinition{
				{ID: "source", Name: "Source", Kind: ParameterEnum, Required: true, Default: json.RawMessage(`"personal"`), Choices: []Choice{{Value: "project", Label: "Project"}, {Value: "personal", Label: "Personal"}}, ControlHint: controlHintSelect, Availability: []string{contextGameplay}},
				{ID: "id", Name: "ID", Kind: ParameterString, Required: true, Pattern: idPattern},
				{ID: "name", Name: "Name", Kind: ParameterString, Required: true, Pattern: `.*\S.*`},
				{ID: "description", Name: "Description", Kind: ParameterString, Default: json.RawMessage(`""`)},
			},
			ExecutionModes: []ExecutionMode{ExecutionAlteration},
		},
		{ID: CommandSaveSnapshotLoad, Category: "Save Snapshots", Name: "Load Save Snapshot", Description: "Persist an automatic rollback, build a dormant candidate, and atomically replace the active scene.", Parameters: []ParameterDefinition{reference(referencePattern), revision}, ExecutionModes: []ExecutionMode{ExecutionSceneRestart}, Destructive: true},
		{ID: CommandSaveSnapshotDelete, Category: "Save Snapshots", Name: "Delete Save Snapshot", Description: "Delete a project or personal snapshot only at the reviewed revision.", Parameters: []ParameterDefinition{reference(namedReferencePattern), revision}, ExecutionModes: []ExecutionMode{ExecutionAlteration}, Destructive: true},
	}
}
