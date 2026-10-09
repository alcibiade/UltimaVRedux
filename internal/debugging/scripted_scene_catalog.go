package debugging

import "encoding/json"

const (
	// CommandScenarioBlackthornInterrogation starts the production Blackthorn interrogation from deterministic state.
	CommandScenarioBlackthornInterrogation = "scenario.blackthorn-interrogation.start"
	// CommandScenarioLordBritishFinale starts the production terminal finale with exact Wooden Box ownership.
	CommandScenarioLordBritishFinale = "scenario.lord-british-finale.start"
)

func scriptedSceneScenarioCommands() []CommandDefinition {
	reviewedParty := ParameterDefinition{
		ID: "expectedPartyFingerprint", Name: "Reviewed Party", Kind: ParameterString, Required: true,
		Availability: []string{contextGameplay}, ScenarioRuntime: true,
	}

	return []CommandDefinition{
		{
			ID: CommandScenarioBlackthornInterrogation, Category: categoryThirdBatchScenarios, Name: "Test Blackthorn Interrogation",
			Description:    "Open the first shrine, move to Blackthorn's Palace, and start the production interrogation workflow.",
			Aliases:        []string{"blackthorn interrogation", "blackthorn prison"},
			Parameters:     []ParameterDefinition{reviewedParty},
			ExecutionModes: []ExecutionMode{ExecutionWorkflow}, Destructive: true,
		},
		{
			ID: CommandScenarioLordBritishFinale, Category: categoryThirdBatchScenarios, Name: "Test Lord British Finale",
			Description: "Choose Wooden Box ownership and start the production terminal Lord British rescue and ending workflow.",
			Aliases:     []string{"lord british finale", "endgame", "ending"},
			Parameters: []ParameterDefinition{
				reviewedParty,
				{
					ID: "woodenBox", Name: "Wooden Box Ownership", Kind: ParameterBoolean, Required: true,
					Default: json.RawMessage(`true`), ControlHint: controlHintSelect, Availability: []string{contextGameplay},
				},
			},
			ExecutionModes: []ExecutionMode{ExecutionWorkflow}, Destructive: true, ScenarioTerminal: true,
		},
	}
}
