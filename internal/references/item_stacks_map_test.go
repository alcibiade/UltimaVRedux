package references

import "testing"

func TestItemStacksMapSnapshotIsDetachedAndSafe(t *testing.T) {
	stacks := NewItemStacksMap()
	position := Position{X: 4, Y: 7}
	original := &ItemStack{Items: ItemStacks{
		{Item: Gold, Quantity: 10},
		{Item: Key, Quantity: 2},
	}}
	if err := stacks.Push(&position, original); err != nil {
		t.Fatalf("Push() error = %v", err)
	}

	snapshot, ok := stacks.Snapshot(&position)
	if !ok {
		t.Fatal("Snapshot() did not find existing stack")
	}
	if len(snapshot.Items) != 2 {
		t.Fatalf("Snapshot() item count = %d, want 2", len(snapshot.Items))
	}
	snapshot.Items[0].Quantity = 99
	snapshot.Items = append(snapshot.Items, ItemAndQuantity{Item: Food, Quantity: 1})

	unchanged, ok := stacks.Snapshot(&position)
	if !ok || len(unchanged.Items) != 2 || unchanged.Items[0].Quantity != 10 {
		t.Fatalf("mutating snapshot changed stored stack: %+v", unchanged.Items)
	}

	if _, ok := stacks.Snapshot(nil); ok {
		t.Fatal("Snapshot(nil) found a stack")
	}
	var nilMap *ItemStacksMap
	if _, ok := nilMap.Snapshot(&position); ok {
		t.Fatal("Snapshot() on nil map found a stack")
	}
	missing := Position{X: 8, Y: 8}
	if _, ok := stacks.Snapshot(&missing); ok {
		t.Fatal("Snapshot() found a missing stack")
	}
	stacks.itemStacks[position] = nil
	if _, ok := stacks.Snapshot(&position); ok {
		t.Fatal("Snapshot() found an invalid nil stack")
	}
}

func TestItemStacksMapPushUnderPreservesFirstCreatedPickupOrder(t *testing.T) {
	stacks := NewItemStacksMap()
	position := Position{X: 4, Y: 7}
	if err := stacks.PushUnder(&position, &ItemStack{Items: ItemStacks{{Item: Gold, Quantity: 10}}}); err != nil {
		t.Fatalf("first PushUnder() error = %v", err)
	}
	if err := stacks.PushUnder(&position, &ItemStack{Items: ItemStacks{{Item: Green, Quantity: 1}}}); err != nil {
		t.Fatalf("second PushUnder() error = %v", err)
	}

	first := stacks.Pop(&position)
	if first.Item != Gold || first.Quantity != 10 {
		t.Fatalf("first Pop() = %+v, want first-created gold", first)
	}
	second := stacks.Pop(&position)
	if second.Item != Green || second.Quantity != 1 {
		t.Fatalf("second Pop() = %+v, want later green potion", second)
	}
}
