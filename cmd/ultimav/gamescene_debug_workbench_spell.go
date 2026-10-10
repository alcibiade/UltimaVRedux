package main

import (
	"fmt"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func executeSpellDebugCommand(request debugging.Request, state *game_state.GameState) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandInventorySpellStatus:
		return debugSuccessResult(request, "Prepared spell inventory loaded.", newSpellInventoryStatusResult(state)), true
	case debugging.CommandInventorySpellSet:
		return executeSpellInventorySet(request, state), true
	case debugging.CommandInventorySpellPresetMax:
		return executeSpellInventoryPresetMax(request, state), true
	case debugging.CommandSpellEffectSetRemaining:
		return executeSpellEffectSetRemaining(request, state), true
	case debugging.CommandSpellEffectActivate:
		return executeSpellEffectActivate(request, state), true
	default:
		return debugging.Result{
			SessionID: "", RequestID: "", CommandID: "", SceneGeneration: 0,
			Invocation:             nil,
			EffectiveExecutionMode: "", Success: false, Summary: "", ErrorCode: "", ErrorMessage: "",
			Changes: nil, Data: nil,
		}, false
	}
}

func executeSpellEffectSetRemaining(request debugging.Request, state *game_state.GameState) debugging.Result {
	var operation game_state.DurationEffectOperationResult
	var err error
	switch mustDebugStringArgument(request, "effectKind") {
	case "active_spell":
		operation, err = state.SetDebugDurationEffectTurns(
			mustDebugStringArgument(request, "expectedEffect"),
			mustDebugIntegerArgument(request, "expectedTurns"),
			mustDebugIntegerArgument(request, "expectedLightMinutes"),
			mustDebugIntegerArgument(request, "remaining"),
		)
	case "magic_light":
		operation, err = state.SetDebugMagicLightMinutes(
			mustDebugStringArgument(request, "expectedEffect"),
			mustDebugIntegerArgument(request, "expectedTurns"),
			mustDebugIntegerArgument(request, "expectedLightMinutes"),
			mustDebugIntegerArgument(request, "remaining"),
		)
	default:
		return debugging.FailureResult(request, "invalid_duration_effect", "unsupported effect kind")
	}
	if err != nil {
		return debugging.FailureResult(request, "stale_spell_effects", err.Error())
	}

	return durationEffectMutationResult(request, "Duration effect updated.", operation)
}

func executeSpellEffectActivate(request debugging.Request, state *game_state.GameState) debugging.Result {
	var effectType game_state.DurationEffectType
	switch mustDebugStringArgument(request, "effect") {
	case "light":
		effectType = game_state.DurationEffectLight
	case "time_stop":
		effectType = game_state.DurationEffectTimeStop
	default:
		return debugging.FailureResult(request, "invalid_duration_effect", "unsupported duration effect")
	}
	operation, err := state.ActivateDebugDurationEffect(
		effectType,
		mustDebugStringArgument(request, "expectedEffect"),
		mustDebugIntegerArgument(request, "expectedTurns"),
		mustDebugIntegerArgument(request, "expectedLightMinutes"),
	)
	if err != nil {
		return debugging.FailureResult(request, "stale_spell_effects", err.Error())
	}

	return durationEffectMutationResult(request, "Duration effect activated.", operation)
}

func durationEffectMutationResult(
	request debugging.Request,
	summary string,
	operation game_state.DurationEffectOperationResult,
) debugging.Result {
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionSynchronized,
		summary,
		newSpellEffectsData(operation.After),
	)
	appendSemanticChange(&result.Changes, "effects.activeSpell.effectType", operation.Before.ActiveSpell.EffectCode, operation.After.ActiveSpell.EffectCode)
	appendSemanticChange(&result.Changes, "effects.activeSpell.turnsRemaining", operation.Before.ActiveSpell.TurnsRemaining, operation.After.ActiveSpell.TurnsRemaining)
	appendSemanticChange(&result.Changes, "effects.magicLight.minutesRemaining", operation.Before.MagicLight.MinutesRemaining, operation.After.MagicLight.MinutesRemaining)

	return result
}

func executeSpellInventorySet(request debugging.Request, state *game_state.GameState) debugging.Result {
	spell, ok := debugSpell(mustDebugStringArgument(request, "spell"))
	if !ok {
		return debugging.FailureResult(request, "invalid_spell", "unsupported castable spell")
	}
	status := newSpellInventoryStatusResult(state)
	current := status.Spells[spell.ID()].Quantity
	expected := inventoryLockerQuantity(request, "expectedQuantity")
	if current != expected {
		return debugging.FailureResult(request, "stale_spell_inventory", "the prepared spell quantity changed after review")
	}
	changes, err := state.SetDebugSpellQuantity(spell, inventoryLockerQuantity(request, "targetQuantity"))
	if err != nil {
		return debugging.FailureResult(request, "invalid_spell", err.Error())
	}

	return spellInventoryMutationResult(
		request, debugging.ExecutionSynchronized, "Prepared spell quantity updated.", state, changes,
	)
}

func executeSpellInventoryPresetMax(request debugging.Request, state *game_state.GameState) debugging.Result {
	status := newSpellInventoryStatusResult(state)
	if mustDebugStringArgument(request, inventoryExpectedSnapshotArgument) != status.SnapshotFingerprint {
		return debugging.FailureResult(request, "stale_spell_inventory", "the prepared spell inventory changed after review")
	}

	return spellInventoryMutationResult(
		request,
		debugging.ExecutionAlteration,
		"All 48 castable prepared spells set to 99.",
		state,
		state.SetAllDebugSpellQuantitiesToMaximum(),
	)
}

func spellInventoryMutationResult(
	request debugging.Request,
	mode debugging.ExecutionMode,
	summary string,
	state *game_state.GameState,
	changes []game_state.DebugInventoryChange,
) debugging.Result {
	result := debugSuccessResultWithMode(request, mode, summary, newSpellInventoryStatusResult(state))
	result.Changes = make([]debugging.StateChange, 0, len(changes))
	for _, change := range changes {
		appendSemanticChange(&result.Changes, "inventory."+change.Path+".quantity", change.Before, change.After)
	}

	return result
}

func newSpellInventoryStatusResult(state *game_state.GameState) debugging.SpellInventoryStatusData {
	snapshot := state.GetDebugSpellInventorySnapshot()
	spells := make([]debugging.SpellInventoryEntryData, 0, references.TotalSpellCount)
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		metadata := references.GetSpellInfo(spell)
		if metadata == nil {
			panic(fmt.Sprintf("castable spell %d has no metadata", spell))
		}
		spells = append(spells, debugging.SpellInventoryEntryData{
			ID: spell.ID(), Name: metadata.Name, Description: metadata.Description, Circle: metadata.Circle,
			Targeting: metadata.Targeting.String(), Context: metadata.Context.String(), Quantity: snapshot[spell],
			Maximum: party_state.ItemQuantitySmallMax,
		})
	}
	status := state.GetDurationEffectsStatus()
	result := debugging.SpellInventoryStatusData{
		SnapshotFingerprint: "",
		Spells:              spells,
		MagicLight:          newMagicLightEffectData(status.MagicLight),
		ActiveSpell:         newActiveSpellEffectData(status.ActiveSpell),
	}
	result.SnapshotFingerprint = spellInventoryFingerprint(result.Spells)

	return result
}

func spellInventoryFingerprint(spells []debugging.SpellInventoryEntryData) string {
	return string(mustDebugJSON(spells))
}

func newSpellEffectsData(status game_state.DurationEffectsStatus) debugging.SpellEffectsData {
	return debugging.SpellEffectsData{
		MagicLight:  newMagicLightEffectData(status.MagicLight),
		ActiveSpell: newActiveSpellEffectData(status.ActiveSpell),
	}
}

func newMagicLightEffectData(status game_state.MagicLightStatus) debugging.MagicLightEffectData {
	return debugging.MagicLightEffectData{Active: status.Active, MinutesRemaining: status.MinutesRemaining}
}

func newActiveSpellEffectData(status game_state.DurationEffectStatus) debugging.ActiveSpellEffectData {
	incantations, ambiguous := durationEffectIncantations(status.EffectCode)

	return debugging.ActiveSpellEffectData{
		Active: status.Active, EffectType: status.EffectCode, FriendlyName: status.Name,
		Incantations: incantations, SourceAmbiguous: ambiguous, Description: debugging.DurationEffectDescription(status.EffectCode),
		TurnsRemaining: status.TurnsRemaining,
	}
}

func durationEffectIncantations(effectCode string) ([]string, bool) {
	switch game_state.DurationEffectType(firstByte(effectCode)) {
	case game_state.DurationEffectLight:
		return []string{"In Lor", "Vas Lor"}, true
	case game_state.DurationEffectProtection:
		return []string{"In Sanct"}, false
	case game_state.DurationEffectNegateMagic:
		return []string{"In An"}, false
	case game_state.DurationEffectTimeStop:
		return []string{"An Tym"}, false
	case game_state.DurationEffectQuickness:
		return []string{"Rel Tym"}, false
	case game_state.DurationEffectConfusion:
		return []string{"Quas An Wis"}, false
	case game_state.DurationEffectNone:
		return []string{}, false
	default:
		return []string{}, false
	}
}

func firstByte(value string) byte {
	if value == "" {
		return 0
	}

	return value[0]
}

func debugSpell(value string) (references.Spell, bool) {
	id, err := strconv.Atoi(value)
	if err != nil || id < references.InLor.ID() || id > references.AnTym.ID() {
		return references.InLor, false
	}

	return references.Spell(id), true
}
