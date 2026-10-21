package party_state

import (
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func (p *PlayerCharacter) GetNameAsString() string {
	return strings.TrimRight(string(p.Name[:]), string(rune(0)))
}

func (p *PlayerCharacter) GetKeySpriteIndex() indexes.SpriteIndex {
	if p.Class == 'A' {
		return indexes.AvatarKeyIndex
	}
	if p.Class == 'M' {
		return indexes.WizardKeyIndex
	}
	if p.Class == 'F' {
		return indexes.FighterKeyIndex
	}
	if p.Class == 'B' {
		return indexes.BardKeyIndex
	}

	return 0
}

// IncStatWithCap increments a stat by amount, capping at maxValue
// Applies the capped attribute increase used for Wanderer awards; ref:PROGRESS-0003.
// Wanderer stat boosts cap STR/DEX/INT at 30; ref:PROGRESS-0003.
func IncStatWithCap(currentValue *byte, amount byte, maxValue byte) {
	newValue := *currentValue + amount
	if newValue > maxValue {
		newValue = maxValue
	}
	*currentValue = newValue
}

func (p *PlayerCharacter) IsAlive() bool {
	return p.currentHp > 0 && !p.IsDead() && p.IsInParty()
}

// Status check helper methods
func (p *PlayerCharacter) IsDead() bool {
	return p.status == Dead
}

func (p *PlayerCharacter) IsPoisoned() bool {
	return p.status == Poisoned
}

func (p *PlayerCharacter) IsAsleep() bool {
	return p.status == Sleep
}

func (p *PlayerCharacter) IsGood() bool {
	return p.status == Good
}

// MarkAsDead sets the character status to Dead
func (p *PlayerCharacter) MarkAsDead() {
	p.status = Dead
}

// SetStatus sets the character status
func (p *PlayerCharacter) SetStatus(status CharacterStatus) {
	p.status = status
}

func (p *PlayerCharacter) IsInParty() bool {
	return p.PartyMap == PartyMapActive
}

// Getter methods for private fields
func (p *PlayerCharacter) GetCurrentHp() uint16 {
	return p.currentHp
}

func (p *PlayerCharacter) GetMaxHp() uint16 {
	return p.maxHp
}

func (p *PlayerCharacter) GetCurrentMp() byte {
	return p.currentMp
}

func (p *PlayerCharacter) GetStatus() CharacterStatus {
	return p.status
}

// HP management methods with automatic clamping
func (p *PlayerCharacter) AddHp(amount uint16) {
	if p.currentHp <= 0 {
		p.status = Good
	}
	p.currentHp += amount
	if p.currentHp > p.maxHp {
		p.currentHp = p.maxHp
	}
}

func (p *PlayerCharacter) SetHp(amount uint16) {
	if p.currentHp <= 0 {
		p.status = Good
	}
	p.currentHp = amount
	if p.currentHp > p.maxHp {
		p.currentHp = p.maxHp
	}
}

func (p *PlayerCharacter) SubtractHp(amount uint16) {
	if amount >= p.currentHp {
		p.currentHp = 0
	} else {
		p.currentHp -= amount
	}
	if p.currentHp <= 0 {
		p.status = Dead
	}
}

func (p *PlayerCharacter) SetMaxHp(amount uint16) {
	p.maxHp = amount
	// Clamp current HP if exceeds new max
	if p.currentHp > p.maxHp {
		p.currentHp = p.maxHp
	}
}

// MP management methods with Intelligence-based capping
func (p *PlayerCharacter) GetMaxMp() byte {
	return p.Intelligence // MP cap is always INT
}

// CanCastSpells reports whether the character's class can ever use magic.
func (p *PlayerCharacter) CanCastSpells() bool {
	return p.Class == Avatar || p.Class == Wizard || p.Class == Bard
}

func (p *PlayerCharacter) SetMp(amount byte) {
	maxMp := p.GetMaxMp()
	p.currentMp = amount
	if p.currentMp > maxMp {
		p.currentMp = maxMp
	}
}

func (p *PlayerCharacter) SubtractMp(amount byte) {
	if amount >= p.currentMp {
		p.currentMp = 0
	} else {
		p.currentMp -= amount
	}
}

// RestoreMpByClass restores MP based on character class
// Camping and Wanderer restoration share this cap; ref:CAMP-0005 and ref:PROGRESS-0003.
func (p *PlayerCharacter) RestoreMpByClass() {
	switch p.Class {
	case Wizard, Avatar: // 'M' and 'A' in original
		p.SetMp(p.Intelligence)
	case Bard: // 'B' in original
		p.SetMp(p.Intelligence / 2)
		// Fighter and other classes: no MP restoration
	}
}

// =============================================================================
// Equipment Management Methods
// =============================================================================

// GetEquippedItem returns the equipment in the specified slot.
// Returns references.NoEquipment if the slot is empty.
func (p *PlayerCharacter) GetEquippedItem(slot references.EquipmentSlot) references.Equipment {
	switch slot {
	case references.SlotHelm:
		return references.Equipment(p.Helmet)
	case references.SlotArmour:
		return references.Equipment(p.Armor)
	case references.SlotWeapon:
		// The persisted field name is legacy; Ready treats this byte as generic hand0.
		return references.Equipment(p.Weapon)
	case references.SlotShield:
		// The persisted field name is legacy; Ready treats this byte as generic hand1.
		return references.Equipment(p.Shield)
	case references.SlotRing:
		return references.Equipment(p.Ring)
	case references.SlotAmulet:
		return references.Equipment(p.Amulet)
	default:
		return references.NoEquipment
	}
}

// SetEquippedItem sets the equipment in the specified slot.
// Use references.NoEquipment to unequip.
func (p *PlayerCharacter) SetEquippedItem(slot references.EquipmentSlot, item references.Equipment) {
	if item < 0 || item > references.NoEquipment {
		panic("equipped item must fit the original one-byte character record")
	}
	encoded := byte(item)
	switch slot {
	case references.SlotHelm:
		p.Helmet = encoded
	case references.SlotArmour:
		p.Armor = encoded
	case references.SlotWeapon:
		p.Weapon = encoded
	case references.SlotShield:
		p.Shield = encoded
	case references.SlotRing:
		p.Ring = encoded
	case references.SlotAmulet:
		p.Amulet = encoded
	}
}

// IsSlotEmpty returns true if the specified slot has no equipment.
func (p *PlayerCharacter) IsSlotEmpty(slot references.EquipmentSlot) bool {
	return p.GetEquippedItem(slot) == references.NoEquipment
}

// GetTotalAttack returns the sum of attack values from all equipped items.
func (p *PlayerCharacter) GetTotalAttack() int {
	total := 0
	for _, slot := range []references.EquipmentSlot{
		references.SlotHelm,
		references.SlotArmour,
		references.SlotWeapon,
		references.SlotShield,
		references.SlotRing,
		references.SlotAmulet,
	} {
		item := p.GetEquippedItem(slot)
		if item != references.NoEquipment {
			total += item.GetAttackValue()
		}
	}

	return total
}

// GetTotalDefense returns the sum of defense values from all equipped items.
func (p *PlayerCharacter) GetTotalDefense() int {
	total := 0
	for _, slot := range []references.EquipmentSlot{
		references.SlotHelm,
		references.SlotArmour,
		references.SlotWeapon,
		references.SlotShield,
		references.SlotRing,
		references.SlotAmulet,
	} {
		item := p.GetEquippedItem(slot)
		if item != references.NoEquipment {
			total += item.GetDefenseValue()
		}
	}

	return total
}

// GetTotalEquippedStrRequirement returns the sum of STR requirements for all equipped items.
// This follows the cumulative equipment-strength rule (ref:EQUIP-0011), which calculates
// the cumulative "weight" of all readied items to determine if a new item can be equipped.
func (p *PlayerCharacter) GetTotalEquippedStrRequirement() int {
	sum := 0
	for _, slot := range []references.EquipmentSlot{
		references.SlotHelm,
		references.SlotArmour,
		references.SlotWeapon,
		references.SlotShield,
		references.SlotRing,
		references.SlotAmulet,
	} {
		item := p.GetEquippedItem(slot)
		if item != references.NoEquipment {
			sum += item.GetStrengthRequired()
		}
	}

	return sum
}
