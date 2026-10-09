package references

// =============================================================================
// EQUIPMENT DATA
// =============================================================================
// This file contains the static equipment definitions map.
// All types, enums, constants, and methods are in equipment.go.
//
// Original equipment reference fields:
//   - Attack, defense, range, and category: ref:EQUIP-0001 and ref:EQUIP-0002
//   - Strength costs: ref:EQUIP-0001
//   - Slot and hand usage: ref:EQUIP-0002
//   - Runic display codes: ref:EQUIP-0003
//   - Catalog base prices: ref:EQUIP-0003
//
// See docs/gamefacts/EQUIPMENT_ORIGINAL_DATA.md for the semantic reference catalog.
// =============================================================================

// equipmentDefinitions contains all static equipment data in a single source of truth.
var equipmentDefinitions = map[Equipment]equipmentDefinition{

	// =========================================================================
	// SPECIAL
	// =========================================================================
	BareHands: {
		Name: "Bare Hands", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 3, MissileType: MissileNone, IconChar: 0x04,
	},

	// =========================================================================
	// HELMS (0-3)
	// =========================================================================
	// Icon: 1,1,1,1 | Attack: 0,0,0,4 | Defense: 1,2,3,3 | Strength: 0,1,2,3; ref:EQUIP-0001.
	LeatherHelm: {
		Name: "Leather Helm", ShortName: "Leath Helm", Category: CategoryHelm, Slot: SlotHelm,
		Defense: 1, StrRequired: 0, BasePrice: 15, IconChar: 0x01,
	},
	ChainCoif: {
		Name: "Chain Coif", Category: CategoryHelm, Slot: SlotHelm,
		Defense: 2, StrRequired: 1, BasePrice: 50, IconChar: 0x01,
	},
	IronHelm: {
		Name: "Iron Helm", Category: CategoryHelm, Slot: SlotHelm,
		Defense: 3, StrRequired: 2, BasePrice: 120, IconChar: 0x01,
	},
	SpikedHelm: {
		Name: "Spiked Helm", ShortName: "Spkd. Helm", Category: CategoryHelm, Slot: SlotHelm,
		Attack: 4, Defense: 3, StrRequired: 3, BasePrice: 150, IconChar: 0x01,
	},

	// =========================================================================
	// SHIELDS (4-8)
	// =========================================================================
	// Icon: 2,2,2,2,2 | Attack: 0,0,6,0,0 | Defense: 2,3,3,5,0 | Strength: 2,3,4,0,0; ref:EQUIP-0001.
	SmallShield: {
		Name: "Small Shield", ShortName: "Sm. Shield", Category: CategoryShield, Slot: SlotShield,
		Defense: 2, StrRequired: 2, BasePrice: 40, IconChar: 0x02,
	},
	LargeShield: {
		Name: "Large Shield", ShortName: "Lg. Shield", Category: CategoryShield, Slot: SlotShield,
		Defense: 3, StrRequired: 3, BasePrice: 70, IconChar: 0x02,
	},
	SpikedShield: {
		Name: "Spiked Shield", ShortName: "Spkd. Shld", Category: CategoryShield, Slot: SlotShield,
		Attack: 6, Defense: 3, StrRequired: 4, BasePrice: 120, IconChar: 0x02,
	},
	MagicShield: {
		Name: "Magic Shield", ShortName: "Mag. Shld", Category: CategoryShield, Slot: SlotShield,
		Defense: 5, StrRequired: 0, BasePrice: 2000, IconChar: 0x02,
	},
	JewelShield: {
		Name: "Jewel Shield", ShortName: "Jewel Shld", Category: CategoryShield, Slot: SlotShield,
		Defense: 0, StrRequired: 0, BasePrice: 0, IconChar: 0x02, // Quest item, no stats
	},

	// =========================================================================
	// ARMOUR (9-15)
	// =========================================================================
	// Icon: 3,3,3,3,3,3,3 | Defense: 1,2,3,4,5,7,10 | Strength: 0,2,4,6,10,12,0; ref:EQUIP-0001.
	ClothArmour: {
		Name: "Cloth Armour", ShortName: "Cloth", Category: CategoryArmour, Slot: SlotArmour,
		Defense: 1, StrRequired: 0, BasePrice: 20, IconChar: 0x03,
	},
	LeatherArmour: {
		Name: "Leather Armour", ShortName: "Leather", Category: CategoryArmour, Slot: SlotArmour,
		Defense: 2, StrRequired: 2, BasePrice: 50, IconChar: 0x03,
	},
	RingMail: {
		Name: "Ring Mail", Category: CategoryArmour, Slot: SlotArmour,
		Defense: 3, StrRequired: 4, BasePrice: 100, IconChar: 0x03,
	},
	ScaleMail: {
		Name: "Scale Mail", ShortName: "Scale", Category: CategoryArmour, Slot: SlotArmour,
		Defense: 4, StrRequired: 6, BasePrice: 150, IconChar: 0x03,
	},
	ChainMail: {
		Name: "Chain Mail", ShortName: "Chain", Category: CategoryArmour, Slot: SlotArmour,
		Defense: 5, StrRequired: 10, BasePrice: 300, IconChar: 0x03,
	},
	PlateMail: {
		Name: "Plate Mail", ShortName: "Plate", Category: CategoryArmour, Slot: SlotArmour,
		Defense: 7, StrRequired: 12, BasePrice: 700, IconChar: 0x03,
	},
	MysticArmour: {
		Name: "Mystic Armour", ShortName: "Myst. Armr", Category: CategoryArmour, Slot: SlotArmour,
		Defense: 10, StrRequired: 0, BasePrice: 0, IconChar: 0x03, // Quest item
	},

	// =========================================================================
	// WEAPONS (16-34)
	// =========================================================================
	// Each weapon has unique icon, atk, range, missile type
	Dagger: {
		Name: "Dagger", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 6, StrRequired: 1, Range: 3, MissileType: MissileArrow,
		BasePrice: 1, IconChar: 0x04,
	},
	Sling: {
		Name: "Sling", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 6, StrRequired: 2, Range: 4, MissileType: MissileRock,
		BasePrice: 10, IconChar: 0x05,
	},
	Club: {
		Name: "Club", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 8, StrRequired: 3, Range: 0, MissileType: MissileNone,
		BasePrice: 5, IconChar: 0x06,
	},
	FlamingOil: {
		Name: "Flaming Oil", ShortName: "Flame Oil", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 8, StrRequired: 2, Range: 4, MissileType: MissileRed,
		BasePrice: 5, IconChar: 0x07,
	},
	MainGauche: {
		Name: "Main Gauche", ShortName: "Main Gauch", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 8, Defense: 1, StrRequired: 3, Range: 0, MissileType: MissileNone,
		BasePrice: 15, IconChar: 0x04,
	},
	Spear: {
		Name: "Spear", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 10, StrRequired: 4, Range: 5, MissileType: MissileArrow,
		BasePrice: 7, IconChar: 0x08,
	},
	ThrowingAxe: {
		Name: "Throwing Axe", ShortName: "Thrwng Axe", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 10, StrRequired: 6, Range: 4, MissileType: MissileAxe,
		BasePrice: 3, IconChar: 0x09,
	},
	ShortSword: {
		Name: "Short Sword", ShortName: "Sht. Sword", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 12, StrRequired: 5, Range: 0, MissileType: MissileNone,
		BasePrice: 40, IconChar: 0x0B,
	},
	Mace: {
		Name: "Mace", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 15, StrRequired: 7, Range: 0, MissileType: MissileNone,
		BasePrice: 50, IconChar: 0x0C,
	},
	MorningStar: {
		Name: "Morning Star", ShortName: "Morn. Star", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 15, StrRequired: 8, Range: 2, MissileType: MissileArrow,
		BasePrice: 60, IconChar: 0x1E,
	},
	Bow: {
		Name: "Bow", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 10, StrRequired: 8, Range: 7, RequiresAmmo: Arrows, MissileType: MissileArrow,
		BasePrice: 75, IconChar: 0x0F,
	},
	Arrows: {
		Name: "Arrows", Category: CategoryAmmo, Slot: SlotNone,
		Attack: 1, BasePrice: 10, IconChar: 0x17,
	},
	Crossbow: {
		Name: "Crossbow", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 12, StrRequired: 6, Range: 8, RequiresAmmo: Quarrels, MissileType: MissileArrow,
		BasePrice: 150, IconChar: 0x10,
	},
	Quarrels: {
		Name: "Quarrels", Category: CategoryAmmo, Slot: SlotNone,
		Attack: 1, BasePrice: 15, IconChar: 0x17,
	},
	LongSword: {
		Name: "Long Sword", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 15, StrRequired: 9, Range: 0, MissileType: MissileNone,
		BasePrice: 70, IconChar: 0x0B,
	},
	TwoHHammer: {
		Name: "2H Hammer", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 20, StrRequired: 16, Range: 0, MissileType: MissileNone,
		BasePrice: 85, IconChar: 0x11,
	},
	TwoHAxe: {
		Name: "2H Axe", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 20, StrRequired: 15, Range: 0, MissileType: MissileNone,
		BasePrice: 150, IconChar: 0x12,
	},
	TwoHSword: {
		Name: "2H Sword", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 20, StrRequired: 13, Range: 0, MissileType: MissileNone,
		BasePrice: 200, IconChar: 0x13,
	},
	Halberd: {
		Name: "Halberd", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 30, StrRequired: 18, Range: 2, MissileType: MissileArrow,
		BasePrice: 250, IconChar: 0x14,
	},

	// =========================================================================
	// MAGIC WEAPONS (35-41)
	// =========================================================================
	ChaosSword: {
		Name: "Chaos Sword", ShortName: "Chaos Swrd", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 99, StrRequired: 0, Range: 0, MissileType: MissileNone,
		BasePrice: 0, IconChar: 0x15,
		Abilities: []EquipmentAbility{AbilityCharmsWielder},
	},
	MagicBow: {
		Name: "Magic Bow", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 15, StrRequired: 0, Range: 15, RequiresAmmo: Arrows, MissileType: MissileArrow,
		BasePrice: 800, IconChar: 0x16,
	},
	SilverSword: {
		Name: "Silver Sword", ShortName: "Silver Swd", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 12, StrRequired: 8, Range: 0, MissileType: MissileNone,
		BasePrice: 250, IconChar: 0x15,
	},
	MagicAxe: {
		Name: "Magic Axe", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 20, StrRequired: 0, Range: 15, MissileType: MissileAxe,
		BasePrice: 1000, IconChar: 0x17,
		Abilities: []EquipmentAbility{AbilityReturns},
	},
	GlassSword: {
		Name: "Glass Sword", ShortName: "Glass Swrd", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 99, StrRequired: 5, Range: 0, MissileType: MissileNone,
		BasePrice: 0, IconChar: 0x15,
		Abilities: []EquipmentAbility{AbilityShatters},
	},
	JeweledSword: {
		Name: "Jeweled Sword", ShortName: "Jewel Swrd", Category: CategoryWeapon, Slot: SlotWeapon,
		Attack: 1, StrRequired: 0, Range: 0, MissileType: MissileNone,
		BasePrice: 0, IconChar: 0x15,
		Abilities: []EquipmentAbility{AbilityNeverMisses},
	},
	MysticSword: {
		Name: "Mystic Sword", ShortName: "Myst. Swrd", Category: CategoryWeapon, Slot: SlotWeapon, TwoHanded: true,
		Attack: 30, Defense: 1, StrRequired: 0, Range: 0, MissileType: MissileNone,
		BasePrice: 0, IconChar: 0x15, // Quest item
	},

	// =========================================================================
	// RINGS (42-44)
	// =========================================================================
	// Icon: 0x18,0x18,0x18 | Defense: 0,2,0; ref:EQUIP-0001 and ref:EQUIP-0003.
	RingInvisibility: {
		Name: "Ring of Invisibility", ShortName: "Inv. Ring", Category: CategoryRing, Slot: SlotRing,
		BasePrice: 450, IconChar: 0x18,
		Abilities: []EquipmentAbility{AbilityInvisibility, AbilityMayVanish},
	},
	RingProtection: {
		Name: "Ring of Protection", ShortName: "Prot. Ring", Category: CategoryRing, Slot: SlotRing,
		Defense: 2, BasePrice: 500, IconChar: 0x18,
	},
	RingRegeneration: {
		Name: "Ring of Regeneration", ShortName: "Regen Ring", Category: CategoryRing, Slot: SlotRing,
		BasePrice: 200, IconChar: 0x18,
		Abilities: []EquipmentAbility{AbilityRegeneration, AbilityMayVanish},
	},

	// =========================================================================
	// AMULETS (45-47)
	// =========================================================================
	// Icon: 0x19,0x1a,0x1b | Defense: 0,2,0; ref:EQUIP-0001 and ref:EQUIP-0003.
	AmuletOfTurning: {
		Name: "Amulet of Turning", ShortName: "Am/Turning", Category: CategoryAmulet, Slot: SlotAmulet,
		BasePrice: 900, IconChar: 0x19,
		Abilities: []EquipmentAbility{AbilitySpellReflect},
	},
	SpikedCollar: {
		Name: "Spiked Collar", ShortName: "Sp. Collar", Category: CategoryAmulet, Slot: SlotAmulet,
		Defense: 2, BasePrice: 240, IconChar: 0x1A,
	},
	Ankh: {
		Name: "Ankh", Category: CategoryAmulet, Slot: SlotAmulet,
		BasePrice: 0, IconChar: 0x1B, // Quest item
	},

	// =========================================================================
	// SPECIAL
	// =========================================================================
	NoEquipment: {
		Name: "(empty)", Category: CategoryNone, Slot: SlotNone,
	},
}
