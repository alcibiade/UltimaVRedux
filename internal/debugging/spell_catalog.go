package debugging

import (
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	spellDebugCategory = "Spells"

	// CommandInventorySpellStatus reports prepared spells and the active duration effect.
	CommandInventorySpellStatus = "inventory.spell.status"
	// CommandInventorySpellSet assigns one prepared spell quantity.
	CommandInventorySpellSet = "inventory.spell.set"
	// CommandInventorySpellPresetMax sets every castable spell quantity to 99.
	CommandInventorySpellPresetMax = "inventory.spell.preset-max"
	// CommandSpellEffectSetRemaining sets exact remaining turns with stale-state preconditions.
	CommandSpellEffectSetRemaining = "spell.effect.set-remaining"
	// CommandSpellEffectActivate explicitly activates Light or Time Stop at its authentic duration.
	CommandSpellEffectActivate = "spell.effect.activate"
)

func spellDebugCommands() []CommandDefinition {
	minimumQuantity := 0
	maximumQuantity := int(party_state.ItemQuantitySmallMax)
	minimumReviewedEffectTurns, maximumEffectTurns := 0, 9999

	return []CommandDefinition{
		{
			ID: CommandInventorySpellStatus, Category: spellDebugCategory, Name: "Prepared Spell Inventory",
			Description: "Inspect all 48 prepared spell quantities, independent magical light, and the active duration spell.",
			Aliases:     nil, Parameters: nil, ExecutionModes: []ExecutionMode{ExecutionQuery}, Destructive: false,
		},
		{
			ID: CommandInventorySpellSet, Category: spellDebugCategory, Name: "Set Prepared Spell",
			Description: "Set one exact prepared spell quantity after verifying its reviewed value.",
			Aliases:     nil,
			Parameters: []ParameterDefinition{
				inventoryEnumParameter("spell", "Spell", spellChoices()),
				inventoryIntegerParameter("targetQuantity", "Target Quantity", &minimumQuantity, &maximumQuantity),
				inventoryIntegerParameter("expectedQuantity", "Expected Current Quantity", &minimumQuantity, &maximumQuantity),
			},
			ExecutionModes: []ExecutionMode{ExecutionSynchronized}, Destructive: true,
		},
		{
			ID: CommandInventorySpellPresetMax, Category: spellDebugCategory, Name: "Maximum Prepared Spells",
			Description: "Set all 48 castable prepared spell quantities to 99 after reviewing the spell snapshot.",
			Aliases:     []string{"spells giveall"}, Parameters: []ParameterDefinition{spellSnapshotParameter()},
			ExecutionModes: []ExecutionMode{ExecutionAlteration}, Destructive: true,
		},
		{
			ID: CommandSpellEffectSetRemaining, Category: spellDebugCategory, Name: "Set Effect Duration",
			Description: "Set exact active-spell turns or magical-light minutes after checking both reviewed effects.", Aliases: []string{"spell reduceto1turn"},
			Parameters: []ParameterDefinition{
				{ID: "effectKind", Name: "Effect Kind", Kind: ParameterEnum, Required: true, ControlHint: controlHintSelect, Choices: []Choice{{Value: "active_spell", Label: "Active Spell"}, {Value: "magic_light", Label: "Magical Light"}}},
				{ID: "expectedEffect", Name: "Expected Effect", Kind: ParameterString, Required: true, ControlHint: activeCombatTextControlHint},
				{ID: "expectedTurns", Name: "Expected Spell Turns", Kind: ParameterInteger, Required: true, Minimum: &minimumReviewedEffectTurns, Maximum: &maximumEffectTurns, ControlHint: controlHintNumber},
				{ID: "expectedLightMinutes", Name: "Expected Light Minutes", Kind: ParameterInteger, Required: true, Minimum: &minimumReviewedEffectTurns, Maximum: &maximumEffectTurns, ControlHint: controlHintNumber},
				{ID: "remaining", Name: "Remaining Turns Or Minutes", Kind: ParameterInteger, Required: true, Minimum: &minimumReviewedEffectTurns, Maximum: &maximumEffectTurns, ControlHint: controlHintNumber},
			},
			ExecutionModes: []ExecutionMode{ExecutionSynchronized}, Destructive: false,
		},
		{
			ID: CommandSpellEffectActivate, Category: spellDebugCategory, Name: "Activate Duration Effect",
			Description: "Activate independent Light or the Time Stop spell at authentic duration after checking reviewed effect state.", Aliases: []string{"spell light", "spell timestop"},
			Parameters: []ParameterDefinition{
				{ID: "effect", Name: "Effect", Kind: ParameterEnum, Required: true, ControlHint: controlHintSelect, Choices: []Choice{{Value: "light", Label: "Light"}, {Value: "time_stop", Label: "Time Stop"}}},
				{ID: "expectedEffect", Name: "Expected Active Spell", Kind: ParameterString, Required: true, ControlHint: activeCombatTextControlHint},
				{ID: "expectedTurns", Name: "Expected Spell Turns", Kind: ParameterInteger, Required: true, Minimum: &minimumReviewedEffectTurns, Maximum: &maximumEffectTurns, ControlHint: controlHintNumber},
				{ID: "expectedLightMinutes", Name: "Expected Light Minutes", Kind: ParameterInteger, Required: true, Minimum: &minimumReviewedEffectTurns, Maximum: &maximumEffectTurns, ControlHint: controlHintNumber},
			},
			ExecutionModes: []ExecutionMode{ExecutionSynchronized}, Destructive: false,
		},
	}
}

func spellSnapshotParameter() ParameterDefinition {
	return ParameterDefinition{
		ID: inventoryExpectedSnapshotID, Name: "Expected Snapshot",
		Description: "Exact prepared spell snapshot fingerprint returned by Prepared Spell Inventory.",
		Kind:        ParameterString, Required: true, Default: nil, Minimum: nil, Maximum: nil,
		Pattern: "", Choices: nil, OptionsProvider: "", ControlHint: activeCombatTextControlHint, Availability: nil,
	}
}

func spellChoices() []Choice {
	choices := make([]Choice, 0, references.TotalSpellCount)
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		choices = append(choices, choice(strconv.Itoa(spell.ID()), references.GetSpellName(spell)))
	}

	return choices
}
