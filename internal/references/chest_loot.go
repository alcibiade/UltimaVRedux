package references

// ChestLoot is one generated loot allocation (ref:COMBAT-0010), in generation order. A nonzero
// ChestValue denotes a world chest, never a collectible inventory item.
type ChestLoot struct {
	Item       ItemAndQuantity
	ChestValue byte
}

// RollChestLoot applies the miscellaneous/arms checks and replacement
// draws. Admission happens afterward: a full object list still consumes every
// source draw, including the discarded first quantity draw for gold and chests.
func RollChestLoot(value int, rng RandomProvider) []ChestLoot {
	if rng == nil || value < 0 || value > 127 {
		panic("invalid chest loot input")
	}
	frequencies := [...]int{25, 3, 17, 17, 9, 15, 7, 7}
	maximums := [...]int{10, 90, 8, 8, 2, 2, 2, 2}
	var loot []ChestLoot
	for index := len(frequencies) - 1; index >= 0; index-- {
		if frequencies[index] > value || frequencies[index] > randomIntInRange(rng, 1, 30) {
			continue
		}
		number := randomIntInRange(rng, 1, maximums[index])
		entry := ChestLoot{}
		switch index {
		case 0:
			entry.ChestValue = byte(randomIntInRange(rng, 1, value)) //nolint:gosec // Validated seven-bit value.
		case 1:
			entry.Item = ItemAndQuantity{Item: Gold, Quantity: uint16(randomIntInRange(rng, 1, value*3))} //nolint:gosec // At most 381.
		case 2:
			entry.Item = ItemAndQuantity{Item: Potion(number - 1), Quantity: 1}
		case 3:
			entry.Item = ItemAndQuantity{Item: Scroll(number - 1), Quantity: 1}
		default:
			items := [...]Provision{Key, Gem, Torches, Food}
			entry.Item = ItemAndQuantity{Item: items[index-4], Quantity: uint16(number)} //nolint:gosec // Source quantity is 1..2.
		}
		loot = append(loot, entry)
	}
	armaments := [...]int{
		10, 10, 15, 20, 10, 15, 20, 28, 0xff,
		15, 15, 20, 20, 20, 24, 0xff,
		5, 10, 10, 10, 10, 10, 10, 10, 15, 15, 15, 10, 15, 10,
		20, 20, 20, 20, 20, 0xff, 23, 23, 23, 0xff, 0xff, 0xff,
		23, 23, 23, 23, 15, 0xff,
	}
	for range value/2 + 1 {
		index := randomIntInRange(rng, 0, len(armaments)-1)
		if armaments[index] > value || armaments[index] > randomIntInRange(rng, 1, 30) {
			continue
		}
		loot = append(loot, ChestLoot{Item: ItemAndQuantity{Item: Equipment(index), Quantity: 1}})
	}

	return loot
}
