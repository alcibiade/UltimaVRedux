package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// InventoryItem is player-facing metadata projected from an owning item family.
// Item identity is the typed catalog key; gameplay effects remain domain actions.
type InventoryItem struct {
	ItemName                   string
	ItemDisplayName            string // Full quest name, e.g. "Amulet of Lord British".
	ItemPickupMessage          string // Quest pickup feedback.
	ItemSprite                 indexes.SpriteIndex
	ItemDescription            string
	ItemDescriptionAttribution string
	// Scroll duration: game minutes for light, turns for other duration effects.
	EffectDuration int
}
