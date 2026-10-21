package debugging

import "github.com/bradhannah/UltimaVRedux/internal/logging"

const (
	// CommandLoggingEnabledSet changes the process-wide logging master switch.
	CommandLoggingEnabledSet = "logging.enabled.set"
	// CommandLoggingStdoutSet changes process-wide stdout logging output.
	CommandLoggingStdoutSet = "logging.stdout.set"
	// CommandLoggingLevelSet changes the process-wide minimum logging level.
	CommandLoggingLevelSet = "logging.level.set"
	// CommandLoggingCategorySet changes one process-wide logging category.
	CommandLoggingCategorySet = "logging.category.set"
	// CommandLoggingCategoriesSetAll changes every known process-wide logging category.
	CommandLoggingCategoriesSetAll = "logging.categories.set-all"
)

func loggingCommands() []CommandDefinition {
	categoryChoices := make([]Choice, 0, len(logging.GetAllCategories()))
	for _, category := range logging.GetAllCategories() {
		categoryChoices = append(categoryChoices, Choice{Value: category, Label: category, Metadata: nil})
	}
	levelChoices := []Choice{
		{Value: "DEBUG", Label: "Debug", Metadata: nil},
		{Value: "INFO", Label: "Info", Metadata: nil},
		{Value: "WARN", Label: "Warn", Metadata: nil},
		{Value: "ERROR", Label: "Error", Metadata: nil},
		{Value: "FATAL", Label: "Fatal", Metadata: nil},
	}
	booleanParameter := func(name string) []ParameterDefinition {
		return []ParameterDefinition{{ID: parameterEnabled, Name: name, Kind: ParameterBoolean, Required: true, ControlHint: controlHintSelect}}
	}

	return []CommandDefinition{
		{
			ID: CommandLoggingEnabledSet, Category: commandCategoryDiagnostics, Name: "Set Logging",
			Description: "Explicitly enable or disable process-wide categorized logging.",
			Parameters:  booleanParameter("Logging Enabled"), ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, Destructive: false,
		},
		{
			ID: CommandLoggingStdoutSet, Category: commandCategoryDiagnostics, Name: "Set Stdout Logging",
			Description: "Explicitly enable or disable accepted log output to stdout.",
			Parameters:  booleanParameter("Stdout Enabled"), ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, Destructive: false,
		},
		{
			ID: CommandLoggingLevelSet, Category: commandCategoryDiagnostics, Name: "Set Logging Level",
			Description: "Set the process-wide minimum accepted logging level.", Aliases: []string{"debug level"},
			Parameters:     []ParameterDefinition{{ID: "level", Name: "Minimum Level", Kind: ParameterEnum, Required: true, Choices: levelChoices, ControlHint: controlHintSelect}},
			ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, Destructive: false,
		},
		{
			ID: CommandLoggingCategorySet, Category: commandCategoryDiagnostics, Name: "Set Logging Category",
			Description: "Explicitly enable or disable one categorized logging channel.", Aliases: []string{"debug log"},
			Parameters: []ParameterDefinition{
				{ID: "category", Name: "Category", Kind: ParameterEnum, Required: true, Choices: categoryChoices, ControlHint: controlHintSelect},
				{ID: parameterEnabled, Name: "Enabled", Kind: ParameterBoolean, Required: true, ControlHint: controlHintSelect},
			},
			ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, Destructive: false,
		},
		{
			ID: CommandLoggingCategoriesSetAll, Category: commandCategoryDiagnostics, Name: "Set All Logging Categories",
			Description: "Explicitly enable or disable every known categorized logging channel.",
			Parameters:  booleanParameter("All Categories Enabled"), ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, Destructive: false,
		},
	}
}
