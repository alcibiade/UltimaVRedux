package debugging

const (
	expectedFingerprintID = "expectedFingerprint"
	// CommandCodexQuestStateSetAll atomically normalizes every reviewed shrine quest.
	CommandCodexQuestStateSetAll = "codex.quest-state.set-all"
)

//nolint:exhaustruct // Catalog entries intentionally omit zero-value optional wire fields.
func seventhBatchCommands() []CommandDefinition {
	return []CommandDefinition{{
		ID:          CommandCodexQuestStateSetAll,
		Category:    categoryShrines,
		Name:        "Set All Shrine Quest States",
		Description: "Atomically clear or complete every shrine quest after reviewing the complete shrine quest state.",
		Aliases:     []string{"codex clear", "codex endgame"},
		Parameters: []ParameterDefinition{
			{
				ID: "targetState", Name: "Target State", Kind: ParameterEnum, Required: true,
				Choices: []Choice{
					{Value: "not_assigned", Label: "Clear All", Metadata: nil},
					{Value: "completed", Label: "Prepare Endgame", Metadata: nil},
				},
				ControlHint: controlHintSelect, Availability: []string{contextGameplay},
			},
			{
				ID: expectedFingerprintID, Name: "Reviewed Shrine Fingerprint", Kind: ParameterString, Required: true,
				Availability: []string{contextGameplay},
			},
		},
		ExecutionModes: []ExecutionMode{ExecutionAlteration},
		Destructive:    true,
	}}
}
