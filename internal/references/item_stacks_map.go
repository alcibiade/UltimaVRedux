package references

import (
	"errors"
	"fmt"
)

var (
	errCannotPushItems = errors.New("cannot push items")
	errCannotAddItem   = errors.New("cannot add item")
)

// ItemStacksMap holds the item piles of one map by position.
type ItemStacksMap struct {
	itemStacks map[Position]*ItemStack
}

// ItemStackEntry is one detached position/stack pair.
type ItemStackEntry struct {
	Position Position
	Stack    ItemStack
}

// Entries returns a complete detached enumeration. Callers requiring canonical
// order must sort by position because the runtime map is intentionally unordered.
func (i *ItemStacksMap) Entries() []ItemStackEntry {
	if i == nil {
		return nil
	}
	entries := make([]ItemStackEntry, 0, len(i.itemStacks))
	for position, stack := range i.itemStacks {
		if stack == nil {
			continue
		}
		items := append(ItemStacks(nil), stack.Items...)
		entries = append(entries, ItemStackEntry{Position: position, Stack: ItemStack{Items: items}})
	}

	return entries
}

// NewItemStacksMap returns an empty map of item piles.
func NewItemStacksMap() *ItemStacksMap {
	ism := &ItemStacksMap{}
	ism.itemStacks = make(map[Position]*ItemStack)

	return ism
}

// Push puts a pile at the position and fails when the position already has items or an argument is nil.
func (i *ItemStacksMap) Push(pos *Position, itemStack *ItemStack) error {
	if pos == nil {
		return fmt.Errorf("%w: position is nil", errCannotPushItems)
	}
	if itemStack == nil {
		return fmt.Errorf("%w: item stack is nil", errCannotPushItems)
	}
	if i.itemStacks == nil {
		i.itemStacks = make(map[Position]*ItemStack)
	}
	if i.HasItemStackAtPosition(pos) {
		return fmt.Errorf("%w: position (%d,%d) already contains items", errCannotPushItems, pos.X, pos.Y)
	}
	i.itemStacks[*pos] = itemStack

	return nil
}

// PushUnder appends a discovery beneath an existing stack so the first-created
// object remains the first object returned by Pop.
func (i *ItemStacksMap) PushUnder(pos *Position, itemStack *ItemStack) error {
	if pos == nil {
		return fmt.Errorf("%w: position is nil", errCannotPushItems)
	}
	if itemStack == nil || len(itemStack.Items) == 0 {
		return fmt.Errorf("%w: item stack is empty", errCannotPushItems)
	}
	if !i.HasItemStackAtPosition(pos) {
		return i.Push(pos, itemStack)
	}
	existing := i.itemStacks[*pos]
	if existing == nil {
		return fmt.Errorf("%w: position (%d,%d) contains an invalid nil stack", errCannotPushItems, pos.X, pos.Y)
	}

	items := make(ItemStacks, 0, len(itemStack.Items)+len(existing.Items))
	items = append(items, itemStack.Items...)
	items = append(items, existing.Items...)
	existing.Items = items

	return nil
}

// HasItemStackAtPosition reports whether any items lie at the position.
func (i *ItemStacksMap) HasItemStackAtPosition(pos *Position) bool {
	if i == nil || pos == nil {
		return false
	}
	_, exists := i.itemStacks[*pos]

	return exists
}

// Snapshot returns a detached copy of the complete item stack at pos.
func (i *ItemStacksMap) Snapshot(pos *Position) (ItemStack, bool) {
	if i == nil || pos == nil {
		return ItemStack{Items: nil}, false
	}

	stack, exists := i.itemStacks[*pos]
	if !exists || stack == nil {
		return ItemStack{Items: nil}, false
	}

	items := make(ItemStacks, len(stack.Items))
	copy(items, stack.Items)

	return ItemStack{Items: items}, true
}

// AddOrMerge appends an item to an existing stack, merging an identical item entry when present.
func (i *ItemStacksMap) AddOrMerge(pos *Position, item ItemAndQuantity) error {
	if i == nil {
		return fmt.Errorf("%w: item stacks map is nil", errCannotAddItem)
	}
	if pos == nil {
		return fmt.Errorf("%w: position is nil", errCannotAddItem)
	}
	if item.Item == nil {
		return fmt.Errorf("%w: item is nil", errCannotAddItem)
	}
	if item.Quantity == 0 {
		return fmt.Errorf("%w: quantity must be greater than zero", errCannotAddItem)
	}

	stack, exists := i.itemStacks[*pos]
	if !exists {
		return i.Push(pos, &ItemStack{Items: ItemStacks{item}})
	}
	if stack == nil {
		return fmt.Errorf("%w: position (%d,%d) contains an invalid nil stack", errCannotAddItem, pos.X, pos.Y)
	}

	merged, err := mergeItemIntoStack(stack, item, pos)
	if err != nil || merged {
		return err
	}

	stack.Items = append(stack.Items, item)

	return nil
}

func mergeItemIntoStack(stack *ItemStack, item ItemAndQuantity, pos *Position) (bool, error) {
	for index := range stack.Items {
		existing := &stack.Items[index]
		if existing.Item == nil || existing.Item.Type() != item.Item.Type() || existing.Item.ID() != item.Item.ID() {
			continue
		}
		if uint32(existing.Quantity)+uint32(item.Quantity) > uint32(^uint16(0)) {
			return false, fmt.Errorf("%w: quantity overflow at position (%d,%d)", errCannotAddItem, pos.X, pos.Y)
		}
		existing.Quantity += item.Quantity

		return true, nil
	}

	return false, nil
}

// Pop removes and returns the top item at the position, deleting the pile when it empties; the position must hold items.
func (i *ItemStacksMap) Pop(pos *Position) *ItemAndQuantity {
	item := i.itemStacks[*pos].popTopItem()

	if !i.itemStacks[*pos].hasItems() {
		delete(i.itemStacks, *pos)
	}

	return &item
}

// Peek returns the top item at the position without removing it; the position must hold items.
func (i *ItemStacksMap) Peek(pos *Position) *ItemAndQuantity {
	oof := i.itemStacks[*pos]

	return oof.peekTopItem()
}
