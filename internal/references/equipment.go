package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// =============================================================================
// EQUIPMENT SYSTEM
// =============================================================================
// This file contains all equipment-related types, enums, constants, and methods.
// The static equipment data (equipmentDefinitions map) is in equipment_data.go.
//
// Reference ratings and catalog prices are described in ref:EQUIP-0001 and ref:EQUIP-0003.
// See docs/gamefacts/EQUIPMENT_ORIGINAL_DATA.md for complete reference.
// =============================================================================

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

// Equipment represents an equipment item that can be worn or wielded.
type Equipment int

// ID returns the numeric ID of this equipment item.
func (e Equipment) ID() int {
	return int(e)
}

// Type returns the item type for equipment (always ItemTypeEquipment).
func (e Equipment) Type() ItemType {
	return ItemTypeEquipment
}

// MissileType identifies the visual projectile type for ranged weapons.
// Original attack-category encoding; ref:EQUIP-0002.
type MissileType int

// Missile types; MissileNone is -1 so that MissileArrow is 0.
const (
	MissileNone MissileType = iota - 1
	MissileArrow
	MissileCannonBall
	MissileAxe
	MissileRed
	MissileBlue
	MissileGreen
	MissileViolet
	MissileRock
)

// EquipmentSlot identifies which slot an equipment item occupies.
type EquipmentSlot int

// Equipment slots; SlotNone marks items that cannot be equipped, such as ammunition.
const (
	SlotNone   EquipmentSlot = iota // Not equippable (ammo, special)
	SlotHelm                        // Head slot
	SlotArmour                      // Body slot
	SlotWeapon                      // Legacy Weapon byte; Ready treats it as hand0 (or both hands for 2H)
	SlotShield                      // Legacy Shield byte; Ready treats it as hand1
	SlotRing                        // Ring slot
	SlotAmulet                      // Amulet/Neck slot
)

// EquipmentSlots returns the six equippable slots in display order.
func EquipmentSlots() []EquipmentSlot {
	return []EquipmentSlot{SlotHelm, SlotArmour, SlotWeapon, SlotShield, SlotRing, SlotAmulet}
}

// DisplayName labels the two persisted Weapon/Shield fields as generic hands.
func (s EquipmentSlot) DisplayName() string {
	if s == SlotWeapon {
		return "Hand 1"
	}
	if s == SlotShield {
		return "Hand 2"
	}

	return s.String()
}

// String returns the legacy name for an equipment slot.
func (s EquipmentSlot) String() string {
	switch s {
	case SlotHelm:
		return "Helm"
	case SlotArmour:
		return "Armour"
	case SlotWeapon:
		return "Weapon"
	case SlotShield:
		return "Shield"
	case SlotRing:
		return "Ring"
	case SlotAmulet:
		return "Amulet"
	case SlotNone:
	}

	return "None"
}

// Runic character indices for equipment icons; ref:EQUIP-0003.
const (
	runicIconHelm   byte = 0x01 // Helm icon in runic font
	runicIconShield byte = 0x02 // Shield icon in runic font
	runicIconArmour byte = 0x03 // Armor icon in runic font
	runicIconWeapon byte = 0x04 // Generic weapon icon (dagger) in runic font
	runicIconRing   byte = 0x18 // Ring icon in runic font
	runicIconAmulet byte = 0x19 // Amulet icon in runic font
)

// RunicIconChar returns the runic font character index for this slot type.
// These characters are small icons representing the equipment category.
func (s EquipmentSlot) RunicIconChar() byte {
	switch s {
	case SlotHelm:
		return runicIconHelm
	case SlotArmour:
		return runicIconArmour
	case SlotWeapon:
		return runicIconWeapon
	case SlotShield:
		return runicIconShield
	case SlotRing:
		return runicIconRing
	case SlotAmulet:
		return runicIconAmulet
	case SlotNone:
	}

	return 0x00
}

// EquipmentCategory identifies the broad category of equipment for grouping/filtering.
type EquipmentCategory int

// Equipment categories; CategoryNone is used by BareHands and NoEquipment.
const (
	CategoryNone   EquipmentCategory = iota // Not categorized (BareHands, NoEquipment)
	CategoryHelm                            // Head armor
	CategoryShield                          // Off-hand protection
	CategoryArmour                          // Body armor
	CategoryWeapon                          // Weapons (melee and ranged)
	CategoryAmmo                            // Ammunition (arrows, quarrels)
	CategoryRing                            // Finger rings
	CategoryAmulet                          // Neck amulets
)

// String returns the display name for an equipment category.
func (c EquipmentCategory) String() string {
	switch c {
	case CategoryHelm:
		return "Helm"
	case CategoryShield:
		return "Shield"
	case CategoryArmour:
		return "Armour"
	case CategoryWeapon:
		return "Weapon"
	case CategoryAmmo:
		return "Ammo"
	case CategoryRing:
		return "Ring"
	case CategoryAmulet:
		return "Amulet"
	case CategoryNone:
	}

	return "None"
}

// =============================================================================
// EQUIPMENT CONSTANTS
// =============================================================================

// Equipment ids. BareHands (-2) is the weapon used when nothing is readied and lies outside the 0-based inventory ids.
//
//goland:noinspection GoUnusedConst
const (
	BareHands        Equipment = -2
	LeatherHelm      Equipment = 0
	ChainCoif        Equipment = 1
	IronHelm         Equipment = 2
	SpikedHelm       Equipment = 3
	SmallShield      Equipment = 4
	LargeShield      Equipment = 5
	SpikedShield     Equipment = 6
	MagicShield      Equipment = 7
	JewelShield      Equipment = 8
	ClothArmour      Equipment = 9
	LeatherArmour    Equipment = 10
	RingMail         Equipment = 11
	ScaleMail        Equipment = 12
	ChainMail        Equipment = 13
	PlateMail        Equipment = 14
	MysticArmour     Equipment = 15
	Dagger           Equipment = 16
	Sling            Equipment = 17
	Club             Equipment = 18
	FlamingOil       Equipment = 19
	MainGauche       Equipment = 20
	Spear            Equipment = 21
	ThrowingAxe      Equipment = 22
	ShortSword       Equipment = 23
	Mace             Equipment = 24
	MorningStar      Equipment = 25
	Bow              Equipment = 26
	Arrows           Equipment = 27
	Crossbow         Equipment = 28
	Quarrels         Equipment = 29
	LongSword        Equipment = 30
	TwoHHammer       Equipment = 31
	TwoHAxe          Equipment = 32
	TwoHSword        Equipment = 33
	Halberd          Equipment = 34
	ChaosSword       Equipment = 35
	MagicBow         Equipment = 36
	SilverSword      Equipment = 37
	MagicAxe         Equipment = 38
	GlassSword       Equipment = 39
	JeweledSword     Equipment = 40
	MysticSword      Equipment = 41
	RingInvisibility Equipment = 42
	RingProtection   Equipment = 43
	RingRegeneration Equipment = 44
	AmuletOfTurning  Equipment = 45
	SpikedCollar     Equipment = 46
	Ankh             Equipment = 47
	NoEquipment      Equipment = 255 // 0xFF in decimal
)

// =============================================================================
// INTERNAL STRUCTURES
// =============================================================================

// equipmentDefinition contains all static data for an equipment item.
// Reference ratings and catalog prices are described in ref:EQUIP-0001 and ref:EQUIP-0003.
// See docs/gamefacts/EQUIPMENT_ORIGINAL_DATA.md for complete reference.
type equipmentDefinition struct {
	Name         string             // Full display name
	ShortName    string             // Abbreviated name (max ~12 chars), empty = use Name
	Category     EquipmentCategory  // Equipment category for grouping
	Slot         EquipmentSlot      // Which slot this item occupies
	TwoHanded    bool               // Requires both hands (blocks shield slot)
	Attack       int                // Attack/damage rating; ref:EQUIP-0001.
	Defense      int                // Defense/armor rating; ref:EQUIP-0001.
	StrRequired  int                // Strength required to equip; ref:EQUIP-0001.
	Range        int                // Missile attack range; ref:EQUIP-0001.
	RequiresAmmo Equipment          // Ammunition type required (NoEquipment if none)
	MissileType  MissileType        // Projectile visual category; ref:EQUIP-0002.
	BasePrice    int                // Shop catalog base price; ref:EQUIP-0003.
	IconChar     byte               // Runic equipment icon code; ref:EQUIP-0003.
	Abilities    []EquipmentAbility // Special magical abilities (empty for most items)
}

// getDefinition returns the equipment definition, falling back to a zero definition for unknown items.
// Note: RequiresAmmo defaults to NoEquipment since Equipment(0) is LeatherHelm.
func getDefinition(e Equipment) equipmentDefinition {
	if def, ok := equipmentDefinitions[e]; ok {
		// Handle zero-value for RequiresAmmo - Equipment(0) is LeatherHelm, not "no ammo"
		// Only Bow, MagicBow, and Crossbow have RequiresAmmo set in definitions
		if def.RequiresAmmo == Equipment(0) {
			def.RequiresAmmo = NoEquipment
		}

		return def
	}

	return equipmentDefinition{Name: "Unknown", Category: CategoryNone, Slot: SlotNone, RequiresAmmo: NoEquipment}
}

// =============================================================================
// EQUIPMENT ABILITIES
// =============================================================================

// EquipmentAbility represents a special magical ability that equipment can have.
// Original weapon and accessory effects are scoped in ref:EQUIP-0017, ref:EQUIP-0018, and ref:EQUIP-0019.
type EquipmentAbility int

// Equipment abilities; AbilityNone means no special ability.
const (
	AbilityNone EquipmentAbility = iota

	// AbilityInvisibility makes the wearer invisible in combat.
	// Readying an Invisibility Ring can immediately change combat appearance; ref:EQUIP-0018.
	AbilityInvisibility

	// AbilityRegeneration grants one capped HP when an inclusive 0..7 draw is 7.
	// Check cadence belongs to the calling update/rest/combat boundary; ref:EQUIP-0018.
	AbilityRegeneration

	// AbilitySpellReflect is the legacy label for the Turning Amulet's missile interference.
	// The original forces eligible magic-user missiles to miss, rather than reflecting them; ref:EQUIP-0019.
	AbilitySpellReflect

	// AbilityCharmsWielder causes the wielder to become charmed/possessed randomly.
	// Chaos Sword can charm its wielder at player-turn entry; ref:EQUIP-0017.
	AbilityCharmsWielder

	// AbilityShatters destroys the weapon after one successful hit.
	// Source: Glass Sword special behavior - attack value 99, single use
	AbilityShatters

	// AbilityNeverMisses always hits the target. The authoritative damage formula
	// reduces the Jeweled Sword's attack value of 1 to zero damage.
	AbilityNeverMisses

	// AbilityReturns causes thrown weapon to return after attack (infinite ammo).
	// Source: Magic Axe special behavior - range 15, returns to wielder
	AbilityReturns

	// AbilityMayVanish has a 1/16 chance to vanish when equipped.
	// Invisibility and Regeneration rings may vanish after Ready; ref:EQUIP-0018.
	AbilityMayVanish
)

// String returns a human-readable name for the ability.
func (a EquipmentAbility) String() string {
	switch a {
	case AbilityInvisibility:
		return "Invisibility"
	case AbilityRegeneration:
		return "Regeneration"
	case AbilitySpellReflect:
		return "Spell Reflect"
	case AbilityCharmsWielder:
		return "Charms Wielder"
	case AbilityShatters:
		return "Shatters"
	case AbilityNeverMisses:
		return "Never Misses"
	case AbilityReturns:
		return "Returns"
	case AbilityMayVanish:
		return "May Vanish"
	case AbilityNone:
	}

	return "None"
}

// =============================================================================
// EQUIPMENT METHODS
// =============================================================================

// String returns the display name for an equipment item.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) String() string {
	return getDefinition(e).Name
}

// newEquipmentItemReferences derives inventory presentation from the equipment
// catalog used by Ready, combat, and merchants.
func newEquipmentItemReferences() map[Equipment]InventoryItem {
	items := make(map[Equipment]InventoryItem, len(equipmentDefinitions)-1)
	for equipment, definition := range equipmentDefinitions {
		if equipment == NoEquipment {
			continue
		}
		items[equipment] = InventoryItem{ItemName: definition.Name, ItemSprite: equipment.inventorySprite()}
	}

	return items
}

func (e Equipment) inventorySprite() indexes.SpriteIndex {
	if e == BareHands {
		return indexes.ItemHelm // Preserve the existing inventory placeholder.
	}
	switch getDefinition(e).Category {
	case CategoryHelm:
		return indexes.ItemHelm
	case CategoryShield:
		return indexes.ItemShield
	case CategoryArmour:
		return indexes.ItemArmour
	case CategoryWeapon, CategoryAmmo:
		return indexes.ItemWeapon
	case CategoryRing:
		return indexes.ItemRing
	case CategoryAmulet:
		return indexes.ItemAnkh
	case CategoryNone:
		panic("inventory sprite requested for non-equipment")
	default:
		panic("unknown equipment category")
	}
}

// GetSlotType returns which slot this equipment uses.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) GetSlotType() EquipmentSlot {
	return getDefinition(e).Slot
}

// IsTwoHanded returns true if this weapon requires both hands.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) IsTwoHanded() bool {
	return getDefinition(e).TwoHanded
}

// IsWeapon returns true if this equipment is a weapon (goes in weapon slot).
func (e Equipment) IsWeapon() bool {
	return e.GetSlotType() == SlotWeapon
}

// IsArmour returns true if this equipment is body armor.
func (e Equipment) IsArmour() bool {
	return e.GetSlotType() == SlotArmour
}

// IsShield returns true if this equipment is a shield.
func (e Equipment) IsShield() bool {
	return e.GetSlotType() == SlotShield
}

// IsAmmo returns true if this equipment is ammunition (not equippable).
func (e Equipment) IsAmmo() bool {
	return e == Arrows || e == Quarrels
}

// GetAttackValue returns the attack/damage bonus for this equipment.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) GetAttackValue() int {
	return getDefinition(e).Attack
}

// GetDefenseValue returns the defense/armor bonus for this equipment.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) GetDefenseValue() int {
	return getDefinition(e).Defense
}

// GetStrengthRequired returns the strength needed to equip this item.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) GetStrengthRequired() int {
	return getDefinition(e).StrRequired
}

// GetAttackRange returns the maximum attack range for missile weapons.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) GetAttackRange() int {
	return getDefinition(e).Range
}

// GetMissileType returns the projectile visual type for ranged weapons.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) GetMissileType() MissileType {
	return getDefinition(e).MissileType
}

// GetRequiredAmmo returns the ammo type required for this weapon, or NoEquipment if none.
// Uses the centralized equipmentDefinitions map.
func (e Equipment) GetRequiredAmmo() Equipment {
	return getDefinition(e).RequiresAmmo
}

// IsSubjectToCombatInterference returns true if an adjacent enemy can prevent
// this weapon's ranged attack.
func (e Equipment) IsSubjectToCombatInterference() bool {
	//nolint:exhaustive // Only the five original interference-sensitive launchers return true.
	switch e {
	case Bow, Crossbow, MagicBow, FlamingOil, Sling:
		return true
	default:
		return false
	}
}

// IsRecoverableThrownWeapon returns true for thrown weapons recovered at
// hand-to-hand range.
func (e Equipment) IsRecoverableThrownWeapon() bool {
	//nolint:exhaustive // Only listed ammunition/thrown-item users return true; ref:EQUIP-0015 and ref:EQUIP-0016.
	switch e {
	case Dagger, ThrowingAxe, Spear:
		return true
	default:
		return false
	}
}

// BypassesTerrainOnSuccessfulHit returns true when a successful ranged hit is
// not blocked by intervening terrain.
func (e Equipment) BypassesTerrainOnSuccessfulHit() bool {
	return e == MorningStar || e == Halberd
}

// IsConsumedOnRangedAttack returns true when launching the weapon consumes the
// weapon itself.
func (e Equipment) IsConsumedOnRangedAttack() bool {
	return e == FlamingOil
}

// GetIconChar returns the runic font character index for this equipment.
// These are the equipment-specific Runic display codes; ref:EQUIP-0003.
func (e Equipment) GetIconChar() byte {
	return getDefinition(e).IconChar
}

// GetCategory returns the equipment category for grouping/filtering.
func (e Equipment) GetCategory() EquipmentCategory {
	return getDefinition(e).Category
}

// GetBasePrice returns the shop purchase price for this equipment.
func (e Equipment) GetBasePrice() int {
	return getDefinition(e).BasePrice
}

// GetAbilities returns all special abilities for this equipment.
// Most equipment has no abilities (returns empty slice).
func (e Equipment) GetAbilities() []EquipmentAbility {
	return getDefinition(e).Abilities
}

// IsBludgeoning returns true if this weapon uses STR instead of DEX for hit rolls.
// Bludgeoning weapons use attack category 8; ref:EQUIP-0002.
// These are: SpikedHelm, SpikedShield, Club, Mace, TwoHHammer.
func (e Equipment) IsBludgeoning() bool {
	return e == SpikedHelm || e == SpikedShield || e == Club || e == Mace || e == TwoHHammer
}

// HasAbility returns true if this equipment has the specified ability.
func (e Equipment) HasAbility(ability EquipmentAbility) bool {
	for _, a := range getDefinition(e).Abilities {
		if a == ability {
			return true
		}
	}

	return false
}
