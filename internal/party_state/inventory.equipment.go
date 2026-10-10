package party_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type InventoryItemType interface {
	references.Equipment | references.Spell | references.Scroll |
		references.QuestItem | references.Potion | references.Reagent |
		references.Moonstone
}

type InventoryQuantities[TK InventoryItemType, TV ItemQuantity] struct {
	quantities map[TK]*TV
}

func NewInventoryQuantities[TK InventoryItemType, TV ItemQuantity]() InventoryQuantities[TK, TV] {
	ei := InventoryQuantities[TK, TV]{}
	ei.quantities = make(map[TK]*TV)

	return ei
}

func (iq *InventoryQuantities[TK, TV]) ensureQuantity(itemType TK) *TV {
	if iq.quantities == nil {
		iq.quantities = make(map[TK]*TV)
	}

	itemQuantity, ok := iq.quantities[itemType]
	if !ok {
		newItemQuantity := new(TV)
		// Try ItemQuantitySmall first (most common), then ItemQuantityLarge
		if converted, ok := any(&ItemQuantitySmall{}).(TV); ok {
			*newItemQuantity = converted
		} else if converted, ok := any(&ItemQuantityLarge{}).(TV); ok {
			*newItemQuantity = converted
		} else {
			panic(fmt.Sprintf("failed to convert to type %T - programming error in inventory system", newItemQuantity))
		}
		iq.quantities[itemType] = newItemQuantity

		return newItemQuantity
	}

	return itemQuantity
}

// GetQuantity returns the mutable quantity storage for legacy mutation call sites.
// New read-only code must use Get or HasSome, neither of which inserts map entries.
func (iq *InventoryQuantities[TK, TV]) GetQuantity(itemType TK) *TV {
	return iq.ensureQuantity(itemType)
}

func (iq *InventoryQuantities[TK, TV]) IncrementByOne(itemType TK) {
	(*iq.ensureQuantity(itemType)).IncrementByOne()
}

func (iq *InventoryQuantities[TK, TV]) DecrementByOne(itemType TK) bool {
	return (*iq.ensureQuantity(itemType)).DecrementByOne()
}

func (iq *InventoryQuantities[TK, TV]) IncrementBy(itemType TK, incBy uint16) {
	(*iq.ensureQuantity(itemType)).IncrementBy(incBy)
}

func (iq *InventoryQuantities[TK, TV]) DecrementBy(itemType TK, decBy uint16) bool {
	return (*iq.ensureQuantity(itemType)).DecrementBy(decBy)
}

func (iq *InventoryQuantities[TK, TV]) Set(itemType TK, quantity uint16) {
	(*iq.ensureQuantity(itemType)).Set(quantity)
}

func (iq *InventoryQuantities[TK, TV]) Get(itemType TK) uint16 {
	if iq.quantities == nil {
		return 0
	}
	itemQuantity, ok := iq.quantities[itemType]
	if !ok || itemQuantity == nil {
		return 0
	}

	return (*itemQuantity).Get()
}

func (iq *InventoryQuantities[TK, TV]) HasSome(itemType TK) bool {
	return iq.Get(itemType) > 0
}

func (iq *InventoryQuantities[TK, TV]) SetHasOne(itemType TK) {
	(*iq.ensureQuantity(itemType)).SetHasOne()
}
