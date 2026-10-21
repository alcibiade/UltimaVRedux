package references

// ItemAndQuantity is a number of identical items.
type ItemAndQuantity struct {
	Quantity uint16
	Item     Item
}

// ItemStacks is the list of item groups lying on one tile.
type ItemStacks []ItemAndQuantity

// ItemStack is the pile of items lying on one tile; the last entry is on top.
type ItemStack struct {
	Items ItemStacks
}

func randomIntInRange(rng RandomProvider, minimum, maximum int) int {
	if minimum > maximum {
		panic("minimum cannot be greater than maximum")
	}

	return rng.Intn(maximum-minimum+1) + minimum
}

func (i *ItemStack) hasItems() bool {
	return len(i.Items) > 0
}

func (i *ItemStack) popTopItem() ItemAndQuantity {
	if len(i.Items) == 0 {
		panic("can't pop from empty stack")
	}

	item := i.Items[len(i.Items)-1]
	i.Items = i.Items[:len(i.Items)-1]

	return item
}

func (i *ItemStack) peekTopItem() *ItemAndQuantity {
	if !i.hasItems() {
		panic("can't peek from empty stack")
	}

	return &i.Items[len(i.Items)-1]
}
