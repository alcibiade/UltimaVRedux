package debugging

const (
	commandCategoryDiagnostics = "Diagnostics"
	commandCategoryDisplay     = "Display"
	debugDisplayModeMaximized  = "maximized"
	debugDisplayModeFullscreen = "fullscreen"

	// CommandGuardsStatus reports current town-guard availability and hostility.
	CommandGuardsStatus = "guards.status"
	// CommandShadowlordStatus reports current Shadowlord life, presence, and assignment state.
	CommandShadowlordStatus = "shadowlord.status"
	// CommandDeathStatus reports current party death state and resurrection consequences.
	CommandDeathStatus = "death.status"
	// CommandDebugOptionsStatus reports frequently used gameplay and renderer diagnostics.
	CommandDebugOptionsStatus = "debug.options.status"
	// CommandMovementFreeMoveSet explicitly enables or disables collision-free movement.
	CommandMovementFreeMoveSet = "movement.free-move.set"
	// CommandTextPlaceholdersSet explicitly enables or disables empty-line placeholders.
	CommandTextPlaceholdersSet = "overlay.text-placeholders.set"
	// CommandDialogBoundsSet explicitly enables or disables dialog text-area bounds.
	CommandDialogBoundsSet = "overlay.dialog-bounds.set"
	// CommandUnlimitedCastingResourcesSet explicitly enables or disables casting resource enforcement.
	CommandUnlimitedCastingResourcesSet = "magic.unlimited-casting-resources.set"
	// CommandSpellLevelRestrictionsBypassSet explicitly enables or disables spell-circle level enforcement.
	CommandSpellLevelRestrictionsBypassSet = "magic.spell-level-restrictions-bypass.set"
	// CommandDisplayStatus reports requested and observed runtime display state.
	CommandDisplayStatus = "display.status"
	// CommandDisplayModeSet requests one runtime game-window mode.
	CommandDisplayModeSet = "display.mode.set"
	// CommandDisplayResolutionSet requests one runtime game-window preset.
	CommandDisplayResolutionSet = "display.resolution.set"
	// CommandDisplayFullscreenSet is the compatibility runtime fullscreen toggle.
	CommandDisplayFullscreenSet = "display.fullscreen.set"
)

func operationalCommands() []CommandDefinition {
	minimumResolution, maximumResolution := 0, 4
	setBoolean := func(id, category, name, description string, aliases []string, mode ExecutionMode) CommandDefinition {
		return CommandDefinition{
			ID: id, Category: category, Name: name, Description: description, Aliases: aliases,
			Parameters: []ParameterDefinition{{
				ID: "enabled", Name: "Enabled", Kind: ParameterBoolean, Required: true,
				ControlHint: controlHintSelect, Availability: []string{contextGameplay},
			}},
			ExecutionModes: []ExecutionMode{mode}, Destructive: false,
		}
	}

	return []CommandDefinition{
		{ID: CommandGuardsStatus, Category: "Guards", Name: "Guard Status", Description: "Inspect criminal state, potential guards, and current hostility.", Aliases: []string{"guards status"}, ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandShadowlordStatus, Category: "Shadowlords", Name: "Shadowlord Status", Description: "Inspect life, summoned presence, and deterministic town assignments.", Aliases: []string{"shadowlord status"}, ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandDeathStatus, Category: "Death", Name: "Death Status", Description: "Inspect party mortality and authoritative resurrection consequences.", Aliases: []string{"death status"}, ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{ID: CommandDebugOptionsStatus, Category: commandCategoryDiagnostics, Name: "Debug Options", Description: "Inspect gameplay overrides and game-window diagnostic rendering options.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		setBoolean(CommandMovementFreeMoveSet, "Movement", "Set Free Move", "Explicitly enable or disable collision-free Avatar movement.", []string{"mv freemove"}, ExecutionLiveSafe),
		setBoolean(CommandTextPlaceholdersSet, commandCategoryDiagnostics, "Set Text Placeholders", "Show or hide empty-line placeholders in game-window text output.", []string{"debug conversation textplaceholders"}, ExecutionPresentation),
		setBoolean(CommandDialogBoundsSet, commandCategoryDiagnostics, "Set Dialog Bounds", "Show or hide dialog text-area bounds in the game window.", []string{"debug dialog boxes"}, ExecutionPresentation),
		setBoolean(CommandUnlimitedCastingResourcesSet, "Magic", "Set Unlimited Casting Resources", "Cast without prepared spells or current MP consumption while preserving magical-ability requirements.", []string{"unlimited magic", "infinite spells", "infinite mp"}, ExecutionLiveSafe),
		setBoolean(CommandSpellLevelRestrictionsBypassSet, "Magic", "Set Spell Level Bypass", "Allow magical characters to cast spells above their current level.", []string{"all spell circles", "ignore spell level"}, ExecutionLiveSafe),
		{ID: CommandDisplayStatus, Category: commandCategoryDisplay, Name: "Game Window Display", Description: "Inspect requested and observed runtime game-window modes and dimensions.", ExecutionModes: []ExecutionMode{ExecutionQuery}},
		{
			ID: CommandDisplayResolutionSet, Category: commandCategoryDisplay, Name: "Set Game Window Resolution",
			Description: "Request one supported runtime window size without changing player display settings.", Aliases: []string{"screen ru", "screen rd"},
			Parameters: []ParameterDefinition{{
				ID: "index", Name: "Resolution", Kind: ParameterInteger, Required: true,
				Minimum: &minimumResolution, Maximum: &maximumResolution, ControlHint: controlHintSelect,
				Availability: []string{contextGameplay},
			}},
			ExecutionModes: []ExecutionMode{ExecutionPresentation},
		},
		setBoolean(CommandDisplayFullscreenSet, commandCategoryDisplay, "Set Game Window Fullscreen", "Compatibility control that requests runtime fullscreen or windowed mode without changing player display settings.", []string{"screen fs"}, ExecutionPresentation),
	}
}

// displayModeCommands is appended separately to preserve the stable catalog
// ordering of every command that predates display.mode.set.
func displayModeCommands() []CommandDefinition {
	return []CommandDefinition{{
		ID: CommandDisplayModeSet, Category: commandCategoryDisplay, Name: "Set Game Window Mode",
		Description: "Request a runtime game-window mode without changing player display settings.",
		Aliases:     []string{"screen mode"},
		Parameters: []ParameterDefinition{{
			ID: "mode", Name: "Display Mode", Kind: ParameterEnum, Required: true,
			Choices: []Choice{
				{Value: "windowed", Label: "Windowed"},
				{Value: debugDisplayModeMaximized, Label: "Maximized Window"},
				{Value: debugDisplayModeFullscreen, Label: "Borderless Fullscreen"},
			},
			ControlHint: controlHintSelect, Availability: []string{contextGameplay},
		}},
		ExecutionModes: []ExecutionMode{ExecutionPresentation},
	}}
}
