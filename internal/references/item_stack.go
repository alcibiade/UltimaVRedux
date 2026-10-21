package references

type ItemAndQuantity struct {
	Quantity uint16
	Item     Item
}

type ItemStacks []ItemAndQuantity

type ItemStack struct {
	Items ItemStacks
}

func randomIntInRange(rng RandomProvider, minimum, maximum int) int {
	if minimum > maximum {
		panic("minimum cannot be greater than maximum")
	}

	return rng.Intn(maximum-minimum+1) + minimum
}

func (i *ItemStack) HasItems() bool {
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

func (i *ItemStack) PeekTopItem() *ItemAndQuantity {
	if !i.HasItems() {
		panic("can't peek from empty stack")
	}

	return &i.Items[len(i.Items)-1]
}
