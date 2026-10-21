package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// SearchResult represents the outcome of a search operation
type SearchResult int

// SearchResult outcomes of one search attempt.
const (
	SearchResultNotFound SearchResult = iota
	SearchResultFoundItem
	SearchResultFoundSecretDoor
	SearchResultFoundMoonstone
	SearchResultFoundReagent
	SearchResultNoAccess
)

// Special item indices from original Search/Get/Open/Jimmy rules algorithm
// These correspond to specific items in the search location data
// See ref:INV-0008.
const (
	SpecialItemCastleKeys = 13 // Keys in Blackthorn's Castle (requires no keys in inventory)
	SpecialItemSkullKeys  = 14 // Skull keys (daily regeneration)
	SpecialItemGlassSword = 15 // Glass swords (requires no glass swords in inventory)
)

// SearchLocationStatus represents the status of a location for overlay rendering
type SearchLocationStatus int

// SearchLocationStatus values, one per overlay colour.
const (
	SearchLocationNotSearchable SearchLocationStatus = iota // No overlay
	SearchLocationUnsearched                                // Green overlay
	SearchLocationAlreadyFound                              // Red overlay
)

// SearchResponse represents the complete result of a search operation
type SearchResponse struct {
	Result    SearchResult
	Items     []references.ItemAndQuantity // Items found during search
	Message   string                       // Message to display to user
	TimeSpent int                          // Time consumed by search action
}

// Common search messages
const (
	MessageCantSearchThere   = "Can't search there!"
	MessageNothingThere      = "Nothing there!"
	MessageMapNotAvailable   = "Map not available!"
	MessageAlreadyFoundItems = "Already found items there! Use Get to retrieve them."
	MessageNothingOfNote     = "Thou dost find nothing of note."
	MessageFoundPrefix       = "Thou dost find\n"
	MessageCantPlaceItems    = "Can't place items here! "
)
