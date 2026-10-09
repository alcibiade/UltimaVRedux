package debugging

import (
	"encoding/json"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	// CommandInventoryLockerStatus reports the complete locker-managed inventory.
	CommandInventoryLockerStatus = "inventory.locker.status"
	// CommandInventoryProvisionSet assigns one provision quantity.
	CommandInventoryProvisionSet = "inventory.provision.set"
	// CommandInventoryProvisionPreset applies a provision locker preset.
	CommandInventoryProvisionPreset = "inventory.provision.preset"
	// CommandInventoryReagentPreset applies a reagent locker preset.
	CommandInventoryReagentPreset = "inventory.reagent.preset"
	// CommandInventoryReagentSet assigns one reagent quantity.
	CommandInventoryReagentSet = "inventory.reagent.set"
	// CommandInventoryPotionAdd adds to one potion stack.
	CommandInventoryPotionAdd = "inventory.potion.add"
	// CommandInventoryPotionPreset applies a potion locker preset.
	CommandInventoryPotionPreset = "inventory.potion.preset"
	// CommandInventoryScrollAdd adds to one scroll stack.
	CommandInventoryScrollAdd = "inventory.scroll.add"
	// CommandInventoryScrollPreset applies a scroll locker preset.
	CommandInventoryScrollPreset = "inventory.scroll.preset"
	// CommandInventoryEquipmentPreset applies an equipment locker preset.
	CommandInventoryEquipmentPreset = "inventory.equipment.preset"
	// CommandInventoryEquipmentSet assigns one carried equipment quantity.
	CommandInventoryEquipmentSet = "inventory.equipment.set"
	// CommandInventoryEquipmentGrantPreset applies an additive equipment locker preset.
	CommandInventoryEquipmentGrantPreset = "inventory.equipment.grant-preset"
	// CommandInventoryEquipmentClear destroys all carried equipment and every roster loadout.
	CommandInventoryEquipmentClear = "inventory.equipment.clear"

	inventoryLockerCategory      = "Inventory Locker"
	inventoryAmountParameterID   = "amount"
	inventoryExpectedSnapshotID  = "expectedSnapshot"
	inventoryPresetParameterID   = "preset"
	inventorySnapshotDescription = "Exact locker snapshot fingerprint returned by Locker Status."
)

func inventoryLockerCommands() []CommandDefinition {
	commands := inventoryProvisionCommands()
	commands = append(commands, inventoryReagentCommands()...)
	commands = append(commands, inventoryPotionCommands()...)
	commands = append(commands, inventoryScrollCommands()...)
	commands = append(commands, inventoryEquipmentCommands()...)

	return commands
}

func inventoryProvisionCommands() []CommandDefinition {
	minimumQuantity := 0
	maximumLargeQuantity := int(party_state.ItemQuantityLargeMax)

	return []CommandDefinition{
		inventoryCommand(
			CommandInventoryLockerStatus,
			"Locker Status",
			"Show a detached, complete inventory locker snapshot.",
			nil,
			[]ExecutionMode{ExecutionQuery},
			false,
		),
		inventoryCommand(
			CommandInventoryProvisionSet,
			"Set Provision",
			"Synchronously set one provision quantity after verifying its reviewed value.",
			[]ParameterDefinition{
				inventoryEnumParameter("provision", "Provision", inventoryProvisionChoices()),
				inventoryIntegerParameter("targetQuantity", "Target Quantity", &minimumQuantity, &maximumLargeQuantity),
				inventoryIntegerParameter("expectedQuantity", "Expected Current Quantity", &minimumQuantity, &maximumLargeQuantity),
			},
			[]ExecutionMode{ExecutionSynchronized},
			true,
		),
		inventoryPresetCommand(
			CommandInventoryProvisionPreset,
			"Provision Preset",
			"provision",
			inventoryChoices(choice("provisions_max", "Set All Provisions to Maximum")),
		),
	}
}

func inventoryReagentCommands() []CommandDefinition {
	return []CommandDefinition{
		inventoryExactQuantityCommand(
			CommandInventoryReagentSet, "Set Reagent", "reagent", "Reagent", inventoryReagentChoices(),
		),
		inventoryPresetCommand(CommandInventoryReagentPreset, "Reagent Preset", "reagent", inventoryChoices(
			choice("reagents_all99", "Set Every Reagent to 99"),
			choice("reagents_all20_no_garlic", "Set Non-Garlic Reagents to 20 and Clear Garlic"),
		)),
	}
}

func inventoryPotionCommands() []CommandDefinition {
	minimumAmount := 1
	maximumLargeQuantity := int(party_state.ItemQuantityLargeMax)

	return []CommandDefinition{
		inventoryCommand(
			CommandInventoryPotionAdd,
			"Add Potion",
			"Add a positive amount to one potion stack.",
			[]ParameterDefinition{
				inventoryEnumParameter("potion", "Potion", inventoryPotionChoices()),
				inventoryIntegerParameter(inventoryAmountParameterID, "Amount", &minimumAmount, &maximumLargeQuantity),
			},
			[]ExecutionMode{ExecutionLiveSafe},
			false,
		),
		inventoryPresetCommand(CommandInventoryPotionPreset, "Potion Preset", "potion", inventoryChoices(
			choice("potions_all1", "Set Every Potion to 1"), choice("potions_all99", "Set Every Potion to 99"),
		)),
	}
}

func inventoryScrollCommands() []CommandDefinition {
	minimumAmount := 1
	maximumLargeQuantity := int(party_state.ItemQuantityLargeMax)

	return []CommandDefinition{
		inventoryCommand(
			CommandInventoryScrollAdd,
			"Add Scroll",
			"Add a positive amount to one scroll stack.",
			[]ParameterDefinition{
				inventoryEnumParameter("scroll", "Scroll", inventoryScrollChoices()),
				inventoryIntegerParameter(inventoryAmountParameterID, "Amount", &minimumAmount, &maximumLargeQuantity),
			},
			[]ExecutionMode{ExecutionLiveSafe},
			false,
		),
		inventoryPresetCommand(CommandInventoryScrollPreset, "Scroll Preset", "scroll", inventoryChoices(
			choice("scrolls_all1", "Set Every Scroll to 1"), choice("scrolls_allmax", "Set Every Scroll to Maximum"),
		)),
	}
}

func inventoryEquipmentCommands() []CommandDefinition {
	return []CommandDefinition{
		inventoryExactQuantityCommand(
			CommandInventoryEquipmentSet, "Set Equipment", "equipment", "Equipment", inventoryEquipmentChoices(),
		),
		inventoryPresetCommand(
			CommandInventoryEquipmentPreset,
			"Equipment Preset",
			"equipment",
			inventoryEquipmentPresetChoices(),
		),
		inventoryEquipmentGrantPresetCommand(),
		inventoryCommand(
			CommandInventoryEquipmentClear,
			"Clear All Equipment",
			"Destroy every carried equipment item and clear all 16 roster loadouts "+
				"after reviewing the complete equipment state.",
			[]ParameterDefinition{inventorySnapshotParameter()},
			[]ExecutionMode{ExecutionAlteration},
			true,
		),
	}
}

func inventoryExactQuantityCommand(id, name, itemID, itemName string, choices []Choice) CommandDefinition {
	minimumQuantity := 0
	maximumQuantity := int(party_state.ItemQuantitySmallMax)

	return inventoryCommand(
		id,
		name,
		"Synchronously set one "+itemName+" quantity after verifying its reviewed value.",
		[]ParameterDefinition{
			inventoryEnumParameter(itemID, itemName, choices),
			inventoryIntegerParameter("targetQuantity", "Target Quantity", &minimumQuantity, &maximumQuantity),
			inventoryIntegerParameter("expectedQuantity", "Expected Current Quantity", &minimumQuantity, &maximumQuantity),
		},
		[]ExecutionMode{ExecutionSynchronized},
		true,
	)
}

func inventoryEquipmentGrantPresetCommand() CommandDefinition {
	return inventoryCommand(
		CommandInventoryEquipmentGrantPreset,
		"Grant Equipment Preset",
		"Add five matching equipment items after reviewing the locker snapshot.",
		[]ParameterDefinition{
			inventoryEnumParameter(inventoryPresetParameterID, "Preset", inventoryGrantPresetChoices()),
			inventorySnapshotParameter(),
		},
		[]ExecutionMode{ExecutionAlteration},
		true,
	)
}

func inventoryPresetCommand(id, name, kind string, choices []Choice) CommandDefinition {
	return inventoryCommand(
		id,
		name,
		"Apply an absolute reviewed "+kind+" locker preset.",
		[]ParameterDefinition{
			inventoryEnumParameter(inventoryPresetParameterID, "Preset", choices),
			inventorySnapshotParameter(),
		},
		[]ExecutionMode{ExecutionAlteration},
		true,
	)
}

func inventoryCommand(
	commandID, name, description string,
	parameters []ParameterDefinition,
	modes []ExecutionMode,
	destructive bool,
) CommandDefinition {
	return CommandDefinition{
		ID:             commandID,
		Category:       inventoryLockerCategory,
		Name:           name,
		Description:    description,
		Aliases:        nil,
		Parameters:     parameters,
		ExecutionModes: modes,
		Destructive:    destructive,
	}
}

func inventoryEnumParameter(id, name string, choices []Choice) ParameterDefinition {
	return ParameterDefinition{
		ID: id, Name: name, Description: "", Kind: ParameterEnum, Required: true,
		Default: nil, Minimum: nil, Maximum: nil, Pattern: "", Choices: choices,
		OptionsProvider: "", ControlHint: controlHintSelect, Availability: []string{contextGameplay},
	}
}

func inventoryIntegerParameter(id, name string, minimum, maximum *int) ParameterDefinition {
	return ParameterDefinition{
		ID: id, Name: name, Description: "", Kind: ParameterInteger, Required: true,
		Default: nil, Minimum: minimum, Maximum: maximum, Pattern: "", Choices: nil,
		OptionsProvider: "", ControlHint: controlHintNumber, Availability: nil,
	}
}

func inventorySnapshotParameter() ParameterDefinition {
	return ParameterDefinition{
		ID: inventoryExpectedSnapshotID, Name: "Expected Snapshot", Description: inventorySnapshotDescription,
		Kind: ParameterString, Required: true, Default: nil, Minimum: nil, Maximum: nil,
		Pattern: "", Choices: nil, OptionsProvider: "", ControlHint: "text", Availability: nil,
	}
}

func inventoryChoices(choices ...Choice) []Choice {
	return choices
}

func choice(value, label string) Choice {
	return Choice{Value: value, Label: label, Metadata: json.RawMessage(nil)}
}

func inventoryProvisionChoices() []Choice {
	return inventoryChoices(
		choice("gold", "Gold"), choice("food", "Food"), choice("gems", "Gems"),
		choice("torches", "Torches"), choice("keys", "Keys"), choice("skull_keys", "Skull Keys"),
	)
}

func inventoryPotionChoices() []Choice {
	return inventoryChoices(
		choice("0", "Blue"), choice("1", "Yellow"), choice("2", "Red"), choice("3", "Green"),
		choice("4", "Orange"), choice("5", "Purple"), choice("6", "Black"), choice("7", "White"),
	)
}

func inventoryScrollChoices() []Choice {
	return inventoryChoices(
		choice("0", "Vas Lor"), choice("1", "Rel Hur"), choice("2", "In Sanct"), choice("3", "In An"),
		choice("4", "In Quas Wis"), choice("5", "Kal Xen Corp"), choice("6", "In Mani Corp"), choice("7", "An Tym"),
	)
}

func inventoryReagentChoices() []Choice {
	return inventoryChoices(
		choice("0", "Sulfurous Ash"), choice("1", "Ginseng"), choice("2", "Garlic"), choice("3", "Spider Silk"),
		choice("4", "Blood Moss"), choice("5", "Black Pearl"), choice("6", "Nightshade"), choice("7", "Mandrake Root"),
	)
}

func inventoryEquipmentChoices() []Choice {
	choices := make([]Choice, 0, int(references.Ankh-references.LeatherHelm)+1)
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		choices = append(choices, choice(strconv.Itoa(equipment.ID()), equipment.String()))
	}

	return choices
}

func inventoryEquipmentPresetChoices() []Choice {
	return inventoryChoices(
		choice("equipment_all_equippable5", "Set Every Equippable Item to 5"),
		choice("equipment_weapons5", "Set Every Weapon to 5"),
		choice("equipment_armour5", "Set Every Armour Item to 5"),
		choice("equipment_bows_no_ammo5", "Set Every Bow to 5 and Clear Ammo"),
	)
}

func inventoryGrantPresetChoices() []Choice {
	return inventoryChoices(
		choice("all_equippable5", "Add 5 of Every Equippable Item"), choice("weapons5", "Add 5 of Every Weapon"),
		choice("armour5", "Add 5 of Every Armour Item"), choice("bows_no_ammo5", "Add 5 of Every Bow and Clear Ammo"),
	)
}
