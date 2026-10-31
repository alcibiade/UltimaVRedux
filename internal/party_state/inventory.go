package party_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// PutItemInInventory rejection reasons.
var (
	ErrNoProvisionItem      = errors.New("invalid provision type: NoProvision should not be added to inventory")
	ErrUnsupportedProvision = errors.New("unsupported provision type")
	ErrUnsupportedQuestItem = errors.New("unsupported quest item")
	ErrUnsupportedItemType  = errors.New("unsupported item type")
)

const (
	lockerProvisionCount  = 6
	lockerConsumableCount = 8
	lockerEquipmentCount  = 48
	lockerSpellCount      = 48
)

// provisionsQuantity groups the consumable provisions; gold is kept separately on Inventory.
type provisionsQuantity struct {
	Food      ItemQuantityLarge
	Gems      ItemQuantitySmall
	Torches   ItemQuantitySmall
	Keys      ItemQuantitySmall
	SkullKeys ItemQuantitySmall
}

// Inventory is the party-wide stock of provisions, gold, equipment, spells, scrolls,
// potions, quest items, reagents and moonstones, capped as in the original save.
type Inventory struct {
	Provisions provisionsQuantity
	Gold       ItemQuantityLarge

	Equipment  inventoryQuantities[references.Equipment, *ItemQuantitySmall]
	Spells     inventoryQuantities[references.Spell, *ItemQuantitySmall]
	Scrolls    inventoryQuantities[references.Scroll, *ItemQuantityLarge]
	Potions    inventoryQuantities[references.Potion, *ItemQuantitySmall]
	QuestItems inventoryQuantities[references.QuestItem, *ItemQuantitySmall]
	Reagent    inventoryQuantities[references.Reagent, *ItemQuantitySmall]
	Moonstones inventoryQuantities[references.Moonstone, *ItemQuantitySmall]

	// Materials and Tools are Redux crafting stock (ref:CRAFT-0001); they have no SAVED.GAM slot.
	Materials inventoryQuantities[references.Material, *ItemQuantityLarge]
	Tools     inventoryQuantities[references.Tool, *ItemQuantitySmall]
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

// NewInventory returns an empty inventory with every quantity table allocated.
func NewInventory() *Inventory {
	inv := new(Inventory)
	inv.Equipment = newInventoryQuantities[references.Equipment, *ItemQuantitySmall]()
	inv.Spells = newInventoryQuantities[references.Spell, *ItemQuantitySmall]()
	inv.Scrolls = newInventoryQuantities[references.Scroll, *ItemQuantityLarge]()
	inv.Potions = newInventoryQuantities[references.Potion, *ItemQuantitySmall]()
	inv.QuestItems = newInventoryQuantities[references.QuestItem, *ItemQuantitySmall]()
	inv.Reagent = newInventoryQuantities[references.Reagent, *ItemQuantitySmall]()
	inv.Moonstones = newInventoryQuantities[references.Moonstone, *ItemQuantitySmall]()
	inv.Materials = newInventoryQuantities[references.Material, *ItemQuantityLarge]()
	inv.Tools = newInventoryQuantities[references.Tool, *ItemQuantitySmall]()

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

// PutItemInInventory adds the stack to the matching inventory table, or reports why
// the item identity cannot be stored.
func (i *Inventory) PutItemInInventory(item *references.ItemAndQuantity) error {
	switch item.Item.Type() {
	case references.ItemTypeProvision:
		return i.putProvision(item)
	case references.ItemTypeQuestItem:
		questItem := references.QuestItem(item.Item.ID())
		switch questItem {
		case references.Crown, references.Sceptre, references.Amulet, references.ShardOfFalsehood,
			references.ShardOfHatred, references.ShardOfCowardice, references.Carpet, references.Grapple,
			references.Spyglass, references.HMSCape, references.PocketWatch, references.BlackBadge,
			references.WoodenBox, references.Sextant:
			i.QuestItems.IncrementBy(questItem, item.Quantity)
		default:
			return fmt.Errorf("%w: %v", ErrUnsupportedQuestItem, questItem)
		}
	case references.ItemTypeMoonstone:
		i.Moonstones.IncrementBy(references.Moonstone(item.Item.ID()), item.Quantity)
	case references.ItemTypeEquipment:
		i.Equipment.IncrementBy(references.Equipment(item.Item.ID()), item.Quantity)
	case references.ItemTypePotion:
		i.Potions.IncrementBy(references.Potion(item.Item.ID()), item.Quantity)
	case references.ItemTypeScroll:
		i.Scrolls.IncrementBy(references.Scroll(item.Item.ID()), item.Quantity)
	case references.ItemTypeReagent:
		i.Reagent.IncrementBy(references.Reagent(item.Item.ID()), item.Quantity)
	case references.ItemTypeSpell:
		// Spell stacks live in Inventory.Spells and are not accepted here.
		fallthrough
	default:
		return fmt.Errorf("%w: %v", ErrUnsupportedItemType, item.Item.Type())
	}

	return nil
}

func (i *Inventory) putProvision(item *references.ItemAndQuantity) error {
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
		return ErrNoProvisionItem
	default:
		return fmt.Errorf("%w: %v", ErrUnsupportedProvision, references.Provision(item.Item.ID()))
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
	}

	// Party is starving - UI feedback handled by caller
	return true, ""
}

// CanAfford reports whether the party holds every material in the cost.
func (i *Inventory) CanAfford(cost references.MaterialCost) bool {
	for _, material := range references.MaterialCatalog() {
		if i.Materials.Get(material) < cost[material] {
			return false
		}
	}

	return true
}

// SpendMaterials removes the cost when the party can afford all of it, and reports whether it did.
func (i *Inventory) SpendMaterials(cost references.MaterialCost) bool {
	if !i.CanAfford(cost) {
		return false
	}
	for _, material := range references.MaterialCatalog() {
		if cost[material] > 0 {
			i.Materials.DecrementBy(material, cost[material])
		}
	}

	return true
}

// AddMaterials adds every material in the amount, saturating at the stack cap.
func (i *Inventory) AddMaterials(amount references.MaterialCost) {
	for _, material := range references.MaterialCatalog() {
		if amount[material] > 0 {
			i.Materials.IncrementBy(material, amount[material])
		}
	}
}
