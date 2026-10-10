package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	errEquipmentSlotOperationString  = errors.New("equipment slot operation must be a string")
	errEquipmentSlotChangeRequired   = errors.New("at least one equipment slot change is required")
	errEquipmentOperationUnsupported = errors.New("unsupported equipment operation")
	errEquipmentIDInvalid            = errors.New("invalid equipment ID")
	errEquipmentSourceUnsupported    = errors.New("unsupported equipment source")
)

func executePartyEquipmentLoadoutUpdate(
	request debugging.Request,
	state *game_state.GameState,
) debugging.Result {
	before := newInventoryLockerResult(state)
	if before.SnapshotFingerprint != mustDebugStringArgument(request, inventoryExpectedSnapshotArgument) {
		return debugging.FailureResult(
			request,
			"stale_equipment_state",
			"the carried equipment or a roster loadout changed after review",
		)
	}
	changes, err := equipmentSlotChangesFromRequest(request)
	if err != nil {
		return debugging.FailureResult(request, "invalid_loadout", err.Error())
	}
	targetValue := mustDebugStringArgument(request, "character")
	targets := make([]game_state.EquipmentLoadoutAlteration, 0, 1)
	if targetValue == allActivePartyValue {
		characters := state.ActivePartyCharacters()
		targets = make([]game_state.EquipmentLoadoutAlteration, 0, len(characters))
		for _, character := range characters {
			targets = append(targets, game_state.EquipmentLoadoutAlteration{
				CharacterSlot: character.Slot, ExpectedOccupant: character.Name, Changes: changes,
			})
		}
	} else {
		slot, expectedOccupant, parseErr := parseActiveCharacterToken(targetValue)
		if parseErr != nil {
			return debugging.FailureResult(request, "invalid_character", parseErr.Error())
		}
		targets = append(targets, game_state.EquipmentLoadoutAlteration{
			CharacterSlot: slot, ExpectedOccupant: expectedOccupant, Changes: changes,
		})
	}
	alteration, err := state.AlterEquipmentLoadouts(game_state.EquipmentLoadoutBatchAlteration{Targets: targets})
	if err != nil {
		return debugging.FailureResult(request, "invalid_loadout", err.Error())
	}

	names := make([]string, 0, len(alteration.Targets))
	for _, target := range alteration.Targets {
		names = append(names, target.CharacterName)
	}
	summary := fmt.Sprintf("Equipment updated for %s.", strings.Join(names, ", "))
	if targetValue == allActivePartyValue {
		summary = fmt.Sprintf("Equipment loadout applied to %d active party members.", len(alteration.Targets))
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionAlteration,
		summary,
		partyEquipmentLoadoutResult{
			Target: targetValue, TargetCount: len(alteration.Targets), CharacterNames: names,
			Inventory: newInventoryLockerResult(state),
		},
	)
	result.Changes = make([]debugging.StateChange, 0, len(alteration.Changes))
	for _, change := range alteration.Changes {
		appendSemanticChange(&result.Changes, change.Path, change.Before, change.After)
	}

	return result
}

type partyEquipmentLoadoutResult struct {
	Target         string                `json:"target"`
	TargetCount    int                   `json:"targetCount"`
	CharacterNames []string              `json:"characterNames"`
	Inventory      inventoryLockerResult `json:"inventory"`
}

func equipmentSlotChangesFromRequest(request debugging.Request) ([]game_state.EquipmentSlotChange, error) {
	changes := make([]game_state.EquipmentSlotChange, 0, len(partyEquipmentRequestSlots()))
	for _, entry := range partyEquipmentRequestSlots() {
		raw, supplied := request.Arguments[entry.id]
		if !supplied {
			continue
		}
		var operation string
		if err := json.Unmarshal(raw, &operation); err != nil {
			return nil, fmt.Errorf("%w: %s", errEquipmentSlotOperationString, entry.id)
		}
		change, err := parseEquipmentSlotOperation(entry.slot, operation)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.id, err)
		}
		changes = append(changes, change)
	}
	if len(changes) == 0 {
		return nil, errEquipmentSlotChangeRequired
	}

	return changes, nil
}

func parseEquipmentSlotOperation(
	slot references.EquipmentSlot,
	operation string,
) (game_state.EquipmentSlotChange, error) {
	if operation == "unequip" {
		return game_state.EquipmentSlotChange{Slot: slot, Equipment: references.NoEquipment, Source: ""}, nil
	}
	sourceValue, equipmentValue, found := strings.Cut(operation, ":")
	if !found {
		return game_state.EquipmentSlotChange{}, fmt.Errorf("%w: %q", errEquipmentOperationUnsupported, operation)
	}
	id, err := strconv.Atoi(equipmentValue)
	if err != nil {
		return game_state.EquipmentSlotChange{}, fmt.Errorf("%w: %q", errEquipmentIDInvalid, equipmentValue)
	}
	source := game_state.EquipmentSource(sourceValue)
	if source != game_state.EquipmentSourceInventory && source != game_state.EquipmentSourceManufacture {
		return game_state.EquipmentSlotChange{}, fmt.Errorf("%w: %q", errEquipmentSourceUnsupported, sourceValue)
	}

	return game_state.EquipmentSlotChange{Slot: slot, Equipment: references.Equipment(id), Source: source}, nil
}

type partyEquipmentRequestSlot struct {
	id   string
	slot references.EquipmentSlot
}

func partyEquipmentRequestSlots() []partyEquipmentRequestSlot {
	return []partyEquipmentRequestSlot{
		{id: "helm", slot: references.SlotHelm},
		{id: "armour", slot: references.SlotArmour},
		{id: "weapon", slot: references.SlotWeapon},
		{id: "shield", slot: references.SlotShield},
		{id: "ring", slot: references.SlotRing},
		{id: "amulet", slot: references.SlotAmulet},
	}
}
