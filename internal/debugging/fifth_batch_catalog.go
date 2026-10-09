package debugging

const (
	categoryCommandWorkflows = "Command Workflows"
	// CommandTestTelescope queues the production telescope workflow from compatible live state.
	CommandTestTelescope = "command-test.telescope"
	// CommandTestSpyglass queues the production spyglass workflow from compatible live state.
	CommandTestSpyglass = "command-test.spyglass"
	// CommandTestCrystalBall queues the production crystal-ball workflow from compatible live state.
	CommandTestCrystalBall = "command-test.crystal-ball"
)

func fifthBatchCommands() []CommandDefinition {
	return []CommandDefinition{
		{
			ID: CommandTestTelescope, Category: categoryCommandWorkflows, Name: "Test Telescope",
			Description:    "Queue the production telescope workflow using the current Britannian time and small-map context.",
			Aliases:        []string{"use telescope"},
			Parameters:     nil,
			ExecutionModes: []ExecutionMode{ExecutionWorkflow}, Destructive: false,
		},
		{
			ID: CommandTestSpyglass, Category: categoryCommandWorkflows, Name: "Test Spyglass",
			Description: "Queue the production spyglass workflow after validating ownership and its " +
				"above-ground small-map context.",
			Aliases:        []string{"use spyglass"},
			Parameters:     nil,
			ExecutionModes: []ExecutionMode{ExecutionWorkflow}, Destructive: false,
		},
		{
			ID: CommandTestCrystalBall, Category: categoryCommandWorkflows, Name: "Test Crystal Ball",
			Description: "Queue the production crystal-ball selection and deterministic intelligence-check " +
				"workflow on a small map.",
			Aliases:        []string{"use crystalball"},
			Parameters:     nil,
			ExecutionModes: []ExecutionMode{ExecutionWorkflow}, Destructive: false,
		},
	}
}
