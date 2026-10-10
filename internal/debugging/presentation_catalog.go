package debugging

const (
	// CommandLightingOverlaySet changes the game-window lighting diagnostic overlay.
	CommandLightingOverlaySet = "overlay.lighting.set"
	// CommandSearchOverlaySet changes the game-window search-location overlay.
	CommandSearchOverlaySet = "overlay.search.set"
	// CommandMoongateOverlaySet changes the game-window moonstone overlay.
	CommandMoongateOverlaySet = "overlay.moongate.set"
	// CommandCombatPresentationSpeedSet changes persisted combat movement presentation speed.
	CommandCombatPresentationSpeedSet = "combat.presentation.speed.set"
)

func presentationCommands() []CommandDefinition {
	minimumSpeed, maximumSpeed := 25, 400
	setBoolean := func(id, name, description string, aliases []string) CommandDefinition {
		return CommandDefinition{
			ID: id, Category: "Presentation", Name: name, Description: description, Aliases: aliases,
			Parameters:     []ParameterDefinition{{ID: "enabled", Name: "Enabled", Kind: ParameterBoolean, Required: true, ControlHint: controlHintSelect, Availability: []string{contextGameplay}}},
			ExecutionModes: []ExecutionMode{ExecutionPresentation}, Destructive: false,
		}
	}

	return []CommandDefinition{
		setBoolean(CommandLightingOverlaySet, "Set Lighting Overlay", "Show or hide game-window light-distance diagnostics.", []string{"light overlay"}),
		setBoolean(CommandSearchOverlaySet, "Set Search Overlay", "Show or hide authentic search locations in the game window.", []string{"command search overlay"}),
		setBoolean(CommandMoongateOverlaySet, "Set Moongate Overlay", "Show or hide moonstone and moongate diagnostics in the game window.", []string{"moongate overlay"}),
		{
			ID: CommandCombatPresentationSpeedSet, Category: "Presentation", Name: "Set Combat Speed",
			Description: "Set and persist combat movement animation speed for subsequent movement.", Aliases: []string{"combat speed"},
			Parameters:     []ParameterDefinition{{ID: "percent", Name: "Speed Percent", Kind: ParameterInteger, Required: true, Minimum: &minimumSpeed, Maximum: &maximumSpeed, ControlHint: controlHintNumber, Availability: []string{contextGameplay}}},
			ExecutionModes: []ExecutionMode{ExecutionPresentation}, Destructive: false,
		},
	}
}
