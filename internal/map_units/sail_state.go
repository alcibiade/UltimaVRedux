package map_units

// SailState represents the current sail configuration of a ship
// Matches original Ultima V 2-state toggle: Furled (anchored) ↔ Raised (sailing)
type SailState int

const (
	// SailFurled - Sails down, ship anchored (default state, matches original Ultima V)
	SailFurled SailState = iota
	// SailRaised - Sails up, catching wind for movement (matches original Ultima V sailing state)
	SailRaised
)

// String returns the human-readable name of the sail state
func (s SailState) String() string {
	switch s {
	case SailRaised:
		return "Sailing"
	case SailFurled:
		return "Anchored"
	default:
		return "Unknown"
	}
}

// GetMovementPrefix returns the movement prefix displayed to the player
// "Sail " for sailing, "" for anchored (can't move)
func (s SailState) GetMovementPrefix() string {
	switch s {
	case SailRaised:
		return "Sail "
	case SailFurled:
		return "" // Can't move when anchored
	default:
		return ""
	}
}
