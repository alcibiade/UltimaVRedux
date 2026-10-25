package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ReadyEquipmentTotals describes equipment ratings, not predicted combat damage.
type ReadyEquipmentTotals struct {
	Attack   int
	Defense  int
	Strength int
}

func newReadyEquipmentPreview(character *party_state.PlayerCharacter) ReadyEquipmentPreview {
	preview := ReadyEquipmentPreview{Strength: int(character.Strength)}
	for slot := range preview.Before {
		preview.Before[slot] = references.NoEquipment
		preview.After[slot] = references.NoEquipment
	}
	for _, slot := range references.EquipmentSlots() {
		preview.Before[slot] = character.GetEquippedItem(slot)
	}
	preview.After = preview.Before
	preview.Current = readyEquipmentTotals(preview.Before)
	preview.Resulting = preview.Current

	return preview
}

// RequiresConfirmation reports whether equipping displaces existing gear.
func (preview ReadyEquipmentPreview) RequiresConfirmation() bool {
	if preview.Selection != ReadyEquipmentEquip {
		return false
	}
	for _, slot := range references.EquipmentSlots() {
		if preview.Removes(slot) {
			return true
		}
	}

	return false
}

// Removes reports whether this preview takes an equipped item out of a slot.
func (preview ReadyEquipmentPreview) Removes(slot references.EquipmentSlot) bool {
	return slot >= references.SlotHelm && slot <= references.SlotAmulet &&
		preview.Before[slot] != references.NoEquipment && preview.Before[slot] != preview.After[slot]
}

func (preview *ReadyEquipmentPreview) selectEquipment(equipment references.Equipment, hand references.EquipmentSlot) error {
	if hand != references.SlotNone && hand != references.SlotWeapon && hand != references.SlotShield {
		return rejectReadyEquipment("Invalid equipment hand.")
	}
	slot := equipment.GetSlotType()
	if slot == references.SlotNone {
		return rejectReadyEquipment(fmt.Sprintf("%s cannot be readied.", equipment))
	}
	if slot == references.SlotWeapon || slot == references.SlotShield {
		slot = preview.selectHand(equipment, hand)
		for _, occupied := range []references.EquipmentSlot{references.SlotWeapon, references.SlotShield} {
			if equipment.IsTwoHanded() || preview.Before[occupied].IsTwoHanded() {
				preview.After[occupied] = references.NoEquipment
			}
		}
	}
	preview.Selection, preview.Slot = ReadyEquipmentEquip, slot
	preview.After[slot] = equipment
	preview.Resulting = readyEquipmentTotals(preview.After)

	return nil
}

func (preview ReadyEquipmentPreview) selectHand(equipment references.Equipment, hand references.EquipmentSlot) references.EquipmentSlot {
	if equipment.IsTwoHanded() {
		return references.SlotWeapon
	}
	if hand != references.SlotNone {
		return hand
	}
	for _, slot := range []references.EquipmentSlot{references.SlotWeapon, references.SlotShield} {
		if preview.Before[slot] == references.NoEquipment {
			return slot
		}
	}

	return equipment.GetSlotType()
}

func readyEquipmentTotals(equipment [references.SlotAmulet + 1]references.Equipment) ReadyEquipmentTotals {
	var totals ReadyEquipmentTotals
	for _, slot := range references.EquipmentSlots() {
		item := equipment[slot]
		if item == references.NoEquipment {
			continue
		}
		totals.Attack += item.GetAttackValue()
		totals.Defense += item.GetDefenseValue()
		totals.Strength += item.GetStrengthRequired()
	}

	return totals
}

func (g *GameState) validateReadyReturns(preview ReadyEquipmentPreview) error {
	returns := make(map[references.Equipment]int)
	for _, slot := range references.EquipmentSlots() {
		if preview.Removes(slot) {
			returns[preview.Before[slot]]++
		}
	}
	// Preserve a stable rejection order when more than one returned stack is full.
	for item := references.LeatherHelm; item <= references.Ankh; item++ {
		quantity := returns[item]
		if int(g.PartyState.Inventory.Equipment.Get(item))+quantity > int(party_state.ItemQuantitySmallMax) {
			return rejectReadyEquipment(fmt.Sprintf("No room to return %s to party inventory.", item))
		}
	}

	return nil
}
