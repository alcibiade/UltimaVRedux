package party_state

import (
	"fmt"
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

func (p *PlayerCharacter) IsFullyHealed() bool {
	return p.currentHp >= p.maxHp
}

// MP management methods with Intelligence-based capping
func (p *PlayerCharacter) GetMaxMp() byte {
	return p.Intelligence // MP cap is always INT
}

// CanCastSpells reports whether the character's class can ever use magic.
func (p *PlayerCharacter) CanCastSpells() bool {
	return p.Class == Avatar || p.Class == Wizard || p.Class == Bard
}

func (p *PlayerCharacter) AddMp(amount byte) {
	maxMp := p.GetMaxMp()
	p.currentMp += amount
	if p.currentMp > maxMp {
		p.currentMp = maxMp
	}
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

// UnequipSlot removes equipment from the specified slot.
// Returns the previously equipped item (or NoEquipment if slot was empty).
func (p *PlayerCharacter) UnequipSlot(slot references.EquipmentSlot) references.Equipment {
	prev := p.GetEquippedItem(slot)
	p.SetEquippedItem(slot, references.NoEquipment)

	return prev
}

// IsSlotEmpty returns true if the specified slot has no equipment.
func (p *PlayerCharacter) IsSlotEmpty(slot references.EquipmentSlot) bool {
	return p.GetEquippedItem(slot) == references.NoEquipment
}

// HasTwoHandedWeaponEquipped returns true if the character has a two-handed weapon equipped.
func (p *PlayerCharacter) HasTwoHandedWeaponEquipped() bool {
	weapon := p.GetEquippedItem(references.SlotWeapon)
	if weapon == references.NoEquipment {
		return false
	}

	return weapon.IsTwoHanded()
}

// HasShieldEquipped returns true if the character has a shield equipped.
func (p *PlayerCharacter) HasShieldEquipped() bool {
	return !p.IsSlotEmpty(references.SlotShield)
}

// CanEquipItem checks if a character can equip an item.
// Returns (canEquip bool, reason string).
// Validates:
//   - Strength requirement met
//   - Two-handed weapon conflicts (shield must be free)
//   - Shield conflicts with two-handed weapons
//
// Note: This method allows REPLACING items in the same slot.
// The caller is responsible for unequipping the old item before equipping the new one.
// The only conflicts checked are cross-slot conflicts (2H weapon vs shield).
func (p *PlayerCharacter) CanEquipItem(item references.Equipment) (bool, string) {
	if item == references.NoEquipment {
		return false, "Nothing to equip"
	}

	slot := item.GetSlotType()
	strRequired := item.GetStrengthRequired()
	twoHanded := item.IsTwoHanded()

	// Check cumulative equipment strength requirement; ref:EQUIP-0011.
	// The original game sums ALL equipped items' STR requirements, then checks if adding
	// the new item would exceed the character's STR.
	currentStrLoad := p.GetTotalEquippedStrRequirement()

	// If replacing an item in the same slot, subtract the old item's STR (not double-counted)
	existingItem := p.GetEquippedItem(slot)
	if existingItem != references.NoEquipment {
		currentStrLoad -= existingItem.GetStrengthRequired()
	}

	totalRequired := currentStrLoad + strRequired
	if totalRequired > int(p.Strength) {
		return false, fmt.Sprintf("Thou art not strong enough! (Need %d, have %d STR)", totalRequired, p.Strength)
	}

	// Check two-handed weapon conflicts
	if twoHanded {
		// Two-handed weapon: shield slot must be free (weapon slot will be replaced)
		// Original Ultima V message: "Both hands must be free before thou canst wield that!"
		if !p.IsSlotEmpty(references.SlotShield) {
			return false, "Both hands must be free!"
		}
	} else if slot == references.SlotShield {
		// Shield: cannot equip if two-handed weapon is equipped
		if p.HasTwoHandedWeaponEquipped() {
			return false, "Cannot use shield with two-handed weapon!"
		}
	}

	// Other slot conflicts (same slot replacement) are handled by caller
	// which unequips the old item before calling SetEquippedItem

	return true, ""
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

// GetWeaponRange returns the attack range of the equipped weapon.
// Returns 1 for melee (range 0 or no weapon).
func (p *PlayerCharacter) GetWeaponRange() int {
	weapon := p.GetEquippedItem(references.SlotWeapon)
	if weapon == references.NoEquipment {
		return 1 // Unarmed melee
	}
	r := weapon.GetAttackRange()
	if r == 0 {
		return 1 // Melee weapon
	}

	return r
}
