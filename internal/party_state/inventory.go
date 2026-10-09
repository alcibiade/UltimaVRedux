package party_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	lockerProvisionCount  = 6
	lockerConsumableCount = 8
	lockerEquipmentCount  = 48
	lockerSpellCount      = 48
)

type ProvisionsQuantity struct {
	Food      ItemQuantityLarge
	Gems      ItemQuantitySmall
	Torches   ItemQuantitySmall
	Keys      ItemQuantitySmall
	SkullKeys ItemQuantitySmall
}

type Inventory struct {
	Provisions ProvisionsQuantity
	Gold       ItemQuantityLarge

	Equipment  InventoryQuantities[references.Equipment, *ItemQuantitySmall]
	Spells     InventoryQuantities[references.Spell, *ItemQuantitySmall]
	Scrolls    InventoryQuantities[references.Scroll, *ItemQuantityLarge]
	Potions    InventoryQuantities[references.Potion, *ItemQuantitySmall]
	QuestItems InventoryQuantities[references.QuestItem, *ItemQuantitySmall]
	Reagent    InventoryQuantities[references.Reagent, *ItemQuantitySmall]
	Moonstones InventoryQuantities[references.Moonstone, *ItemQuantitySmall]
}

// InventoryLockerSnapshot is a complete, detached view of locker-managed inventory.
// Every supported identity is present, including identities whose quantity is zero.
type InventoryLockerSnapshot struct {
	Provisions map[references.Provision]uint16
	Reagents   map[references.Reagent]uint16
	Potions    map[references.Potion]uint16
	Scrolls    map[references.Scroll]uint16
	Equipment  map[references.Equipment]uint16
}

// SpellInventorySnapshot is a complete detached view of all 48 castable spell quantities.
type SpellInventorySnapshot map[references.Spell]uint16

func NewInventory() *Inventory {
	inv := &Inventory{}
	inv.Equipment = NewInventoryQuantities[references.Equipment, *ItemQuantitySmall]()
	inv.Spells = NewInventoryQuantities[references.Spell, *ItemQuantitySmall]()
	inv.Scrolls = NewInventoryQuantities[references.Scroll, *ItemQuantityLarge]()
	inv.Potions = NewInventoryQuantities[references.Potion, *ItemQuantitySmall]()
	inv.QuestItems = NewInventoryQuantities[references.QuestItem, *ItemQuantitySmall]()
	inv.Reagent = NewInventoryQuantities[references.Reagent, *ItemQuantitySmall]()
	inv.Moonstones = NewInventoryQuantities[references.Moonstone, *ItemQuantitySmall]()

	return inv
}

// LockerSnapshot returns a complete value copy without allocating missing inventory entries.
func (i *Inventory) LockerSnapshot() InventoryLockerSnapshot {
	snapshot := InventoryLockerSnapshot{
		Provisions: make(map[references.Provision]uint16, lockerProvisionCount),
		Reagents:   make(map[references.Reagent]uint16, lockerConsumableCount),
		Potions:    make(map[references.Potion]uint16, lockerConsumableCount),
		Scrolls:    make(map[references.Scroll]uint16, lockerConsumableCount),
		Equipment:  make(map[references.Equipment]uint16, lockerEquipmentCount),
	}

	snapshot.Provisions[references.Gold] = i.Gold.Get()
	snapshot.Provisions[references.Food] = i.Provisions.Food.Get()
	snapshot.Provisions[references.Gem] = i.Provisions.Gems.Get()
	snapshot.Provisions[references.Torches] = i.Provisions.Torches.Get()
	snapshot.Provisions[references.Key] = i.Provisions.Keys.Get()
	snapshot.Provisions[references.SkullKeys] = i.Provisions.SkullKeys.Get()
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		snapshot.Reagents[reagent] = i.Reagent.Get(reagent)
	}
	for potion := references.Blue; potion <= references.White; potion++ {
		snapshot.Potions[potion] = i.Potions.Get(potion)
	}
	for scroll := references.ScrollVasLor; scroll <= references.ScrollAnTym; scroll++ {
		snapshot.Scrolls[scroll] = i.Scrolls.Get(scroll)
	}
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		snapshot.Equipment[equipment] = i.Equipment.Get(equipment)
	}

	return snapshot
}

// SpellSnapshot returns all castable spell quantities without allocating missing inventory entries.
func (i *Inventory) SpellSnapshot() SpellInventorySnapshot {
	snapshot := make(SpellInventorySnapshot, lockerSpellCount)
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		snapshot[spell] = i.Spells.Get(spell)
	}

	return snapshot
}

func (i *Inventory) PutItemInInventory(item *references.ItemAndQuantity) error {
	if item.Item.Type() == references.ItemTypeProvision {
		switch references.Provision(item.Item.ID()) {
		case references.Food:
			i.Provisions.Food.IncrementBy(item.Quantity)
		case references.Key:
			i.Provisions.Keys.IncrementBy(item.Quantity)
		case references.Gem:
			i.Provisions.Gems.IncrementBy(item.Quantity)
		case references.Torches:
			i.Provisions.Torches.IncrementBy(item.Quantity)
		case references.SkullKeys:
			i.Provisions.SkullKeys.IncrementBy(item.Quantity)
		case references.Gold:
			i.Gold.IncrementBy(item.Quantity)
		case references.NoProvision:
			return fmt.Errorf("invalid provision type: NoProvision should not be added to inventory")
		default:
			return fmt.Errorf("unsupported provision type: %v", references.Provision(item.Item.ID()))
		}
	} else if item.Item.Type() == references.ItemTypeQuestItem {
		questItem := references.QuestItem(item.Item.ID())
		switch questItem {
		case references.Crown:
			i.QuestItems.IncrementBy(references.Crown, item.Quantity)
		case references.Sceptre:
			i.QuestItems.IncrementBy(references.Sceptre, item.Quantity)
		case references.Amulet:
			i.QuestItems.IncrementBy(references.Amulet, item.Quantity)
		case references.ShardOfFalsehood:
			i.QuestItems.IncrementBy(references.ShardOfFalsehood, item.Quantity)
		case references.ShardOfHatred:
			i.QuestItems.IncrementBy(references.ShardOfHatred, item.Quantity)
		case references.ShardOfCowardice:
			i.QuestItems.IncrementBy(references.ShardOfCowardice, item.Quantity)
		case references.Carpet:
			i.QuestItems.IncrementBy(references.Carpet, item.Quantity)
		case references.Grapple:
			i.QuestItems.IncrementBy(references.Grapple, item.Quantity)
		case references.Spyglass:
			i.QuestItems.IncrementBy(references.Spyglass, item.Quantity)
		case references.HMSCape:
			i.QuestItems.IncrementBy(references.HMSCape, item.Quantity)
		case references.PocketWatch:
			i.QuestItems.IncrementBy(references.PocketWatch, item.Quantity)
		case references.BlackBadge:
			i.QuestItems.IncrementBy(references.BlackBadge, item.Quantity)
		case references.WoodenBox:
			i.QuestItems.IncrementBy(references.WoodenBox, item.Quantity)
		case references.Sextant:
			i.QuestItems.IncrementBy(references.Sextant, item.Quantity)
		default:
			return fmt.Errorf("unsupported quest item: %v", questItem)
		}
	} else if item.Item.Type() == references.ItemTypeMoonstone {
		moonstone := references.Moonstone(item.Item.ID())
		i.Moonstones.IncrementBy(moonstone, item.Quantity)
	} else if item.Item.Type() == references.ItemTypeEquipment {
		equipment := references.Equipment(item.Item.ID())
		i.Equipment.IncrementBy(equipment, item.Quantity)
	} else if item.Item.Type() == references.ItemTypePotion {
		i.Potions.IncrementBy(references.Potion(item.Item.ID()), item.Quantity)
	} else if item.Item.Type() == references.ItemTypeScroll {
		i.Scrolls.IncrementBy(references.Scroll(item.Item.ID()), item.Quantity)
	} else if item.Item.Type() == references.ItemTypeReagent {
		i.Reagent.IncrementBy(references.Reagent(item.Item.ID()), item.Quantity)
	} else {
		return fmt.Errorf("unsupported item type: %v", item.Item.Type())
	}

	return nil
}

// ConsumeMealTimeFood handles food consumption at meal times (6, 12, 18 hours)
// Returns (isStarving bool, message string) indicating if party is starving
func (i *Inventory) ConsumeMealTimeFood(hour int, activeMembers int) (bool, string) {
	if activeMembers < 0 || activeMembers > MaxActiveParty {
		panic("meal diner count exceeds the active party capacity")
	}
	// Only consume food at meal times
	if hour != 6 && hour != 12 && hour != 18 {
		return false, ""
	}

	if activeMembers == 0 {
		return false, ""
	}

	if i.Provisions.Food.Get() > 0 {
		// A meal has at most six diners. Decrement one unit per diner until
		// exhausted, without narrowing an untrusted count to a uint16.
		for range activeMembers {
			if !i.Provisions.Food.DecrementByOne() {
				break
			}
		}

		return false, ""
	} else {
		// Party is starving - UI feedback handled by caller
		return true, ""
	}
}
