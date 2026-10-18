package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// InventoryItem is player-facing metadata projected from an owning item family.
// Item identity is the typed catalog key; gameplay effects remain domain actions.
type InventoryItem struct {
	ItemName                   string
	ItemDisplayName            string // Full identified name, including any family suffix.
	ItemPickupMessage          string // Quest pickup feedback.
	ItemSprite                 indexes.SpriteIndex
	ItemDescription            string
	ItemDescriptionAttribution string
	// Scroll duration: game minutes for light, turns for other duration effects.
	EffectDuration int
}

// FullDisplayName returns the catalog's complete identified inventory label.
// Family catalogs supply qualifiers and Potion/Scroll/Moonstone suffixes here;
// callers must not append them again. ItemName remains available for stable
// internal names. Authored discovery/pickup prose is a separate contract.
func (item InventoryItem) FullDisplayName() string {
	if item.ItemDisplayName != "" {
		return item.ItemDisplayName
	}

	return item.ItemName
}
