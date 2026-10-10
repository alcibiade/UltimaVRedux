package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// surfaceLootCollectibleQuantity preserves capped pickup success while retaining
// the uncollected remainder on the ground. Inventory increments clamp silently,
// so determine the actual transfer before changing either authority. Unsupported
// categories retain their existing PutItemInInventory validation.
func (g *GameState) surfaceLootCollectibleQuantity(item references.ItemAndQuantity) uint16 {
	var carried uint16
	maximum := party_state.ItemQuantitySmallMax
	inventory := &g.PartyState.Inventory
	switch value := item.Item.(type) {
	case references.Equipment:
		carried = inventory.Equipment.Get(value)
	case references.Potion:
		carried = inventory.Potions.Get(value)
	case references.Scroll:
		carried = inventory.Scrolls.Get(value)
	case references.Provision:
		switch value {
		case references.Gold:
			carried = inventory.Gold.Get()
			maximum = party_state.ItemQuantityLargeMax
		case references.Food:
			carried = inventory.Provisions.Food.Get()
			maximum = party_state.ItemQuantityLargeMax
		case references.Key:
			carried = inventory.Provisions.Keys.Get()
		case references.Gem:
			carried = inventory.Provisions.Gems.Get()
		case references.Torches:
			carried = inventory.Provisions.Torches.Get()
		case references.SkullKeys:
			carried = inventory.Provisions.SkullKeys.Get()
		case references.NoProvision:
			return item.Quantity
		default:
			return item.Quantity
		}
	default:
		return item.Quantity
	}

	return min(item.Quantity, maximum-carried)
}
