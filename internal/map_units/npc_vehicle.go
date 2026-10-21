package map_units

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// MaxHullHP is the original maximum hull strength for a player frigate.
const MaxHullHP = 99

type VehicleDetails struct {
	// VesselID is assigned only by committed post-sextant party use. Zero is unregistered.
	VesselID        uint64
	VehicleType     references.VehicleType
	skiffQuantity   int
	pendingDelivery bool

	currentDirection  references.Direction
	previousDirection references.Direction

	// IsOwnedByNPC indicates whether this vehicle (especially horses) is owned by an NPC.
	// In original Ultima V, owned horses refuse mounting with "Nay!" message.
	// false = unowned (wishing well horses, shop-bought horses)
	// true = owned by an NPC (NPC's personal horse)
	IsOwnedByNPC bool

	// Ship state
	// HullHP represents the ship's hull health points (0-99, default 99)
	// Only meaningful for frigates. 0 = ship destroyed
	HullHP int

	// SailState represents the current sail configuration
	// SailFurled (anchored, default) ↔ SailRaised (sailing) - matches original Ultima V 2-state toggle
	// Only meaningful for frigates (skiffs have no sails)
	SailState SailState

	// HasHMSCape is retained only for native snapshot compatibility. It is not
	// gameplay authority; GameState owns the global HMS Cape preference.
	HasHMSCape bool
}

// MarkPendingDelivery keeps a purchased dock vehicle inactive until Britannia resumes.
func (v *VehicleDetails) MarkPendingDelivery() {
	v.pendingDelivery = true
}

// ClearPendingDelivery activates a purchased dock vehicle in place.
func (v *VehicleDetails) ClearPendingDelivery() {
	v.pendingDelivery = false
}

// IsPendingDelivery reports whether the dock vehicle is awaiting activation.
func (v *VehicleDetails) IsPendingDelivery() bool {
	return v.pendingDelivery
}

func NewVehicleDetails(vehicleType references.VehicleType) VehicleDetails {
	// New frigates come with 2 skiffs; ref:MERCH-0008.
	initialSkiffs := 0
	if vehicleType == references.FrigateVehicle {
		initialSkiffs = 2
	}

	return VehicleDetails{
		VesselID:          0,
		VehicleType:       vehicleType,
		skiffQuantity:     initialSkiffs,
		currentDirection:  references.Right,
		previousDirection: references.Right,
		IsOwnedByNPC:      false, // Default to unowned (wishing well/shop horses)
		// Ship state
		HullHP:     clampHullHP(MaxHullHP), // Default hull health (original Ultima V max)
		SailState:  SailFurled,             // Default to anchored (sails furled, matches original Ultima V)
		HasHMSCape: false,                  // No HMS Cape by default
	}
}

func (v *VehicleDetails) SetPartyVehicleDirection(direction references.Direction) {
	switch v.VehicleType { //nolint:exhaustive
	case references.HorseVehicle, references.CarpetVehicle:
		if direction == references.Up || direction == references.Down {
			return
		}
	case references.FrigateVehicle, references.SkiffVehicle, references.NoPartyVehicle:
	default:
		// Unsupported vehicle type for direction setting - log and continue
		logging.Warn(logging.CategoryGameState, "SetPartyVehicleDirection called with unsupported vehicle type: %v", v.VehicleType)
	}

	v.previousDirection = v.currentDirection
	v.currentDirection = direction
}

func (v *VehicleDetails) GetBoardedSpriteIndex() indexes.SpriteIndex {
	// For frigates, sprite depends on sail state
	if v.VehicleType == references.FrigateVehicle {
		// SailRaised uses sailing sprites (with sails visible)
		if v.SailState == SailRaised {
			switch v.currentDirection {
			case references.Up:
				return indexes.ShipSailsUp
			case references.Down:
				return indexes.ShipSailsDown
			case references.Left:
				return indexes.ShipSailsLeft
			case references.Right:
				return indexes.ShipSailsRight
			case references.NoneDirection:
				return indexes.ShipSailsRight // Default to facing right
			default:
				return indexes.ShipSailsRight
			}
		}
		// Rowing or SailFurled use no-sails sprites
		// (Furled uses same sprites as rowing for now - TODO: find dedicated furled sprites)
		switch v.currentDirection {
		case references.Up:
			return indexes.ShipNoSailsUp
		case references.Down:
			return indexes.ShipNoSailsDown
		case references.Left:
			return indexes.ShipNoSailsLeft
		case references.Right:
			return indexes.ShipNoSailsRight
		case references.NoneDirection:
			return indexes.ShipNoSailsRight // Default to facing right
		default:
			return indexes.ShipNoSailsRight
		}
	}

	// For non-frigates, use existing vehicle type logic
	return v.VehicleType.GetBoardedSpriteByDirection(v.previousDirection, v.currentDirection)
}

func (v *VehicleDetails) GetUnBoardedSpriteIndex() indexes.SpriteIndex {
	return v.VehicleType.GetUnBoardedSpriteByDirection(v.currentDirection)
}

// CurrentDirection returns the vehicle's current facing direction
func (v *VehicleDetails) CurrentDirection() references.Direction {
	return v.currentDirection
}

func (v *VehicleDetails) DoesMoveResultInMovement(newDirection references.Direction) bool {
	if v.VehicleType != references.FrigateVehicle {
		return true
	}

	if v.currentDirection == newDirection {
		return true
	}

	return false
}

// SetSkiffQuantity sets the skiff quantity to the given quantity.
func (v *VehicleDetails) SetSkiffQuantity(quantity int) {
	v.skiffQuantity = quantity
}

// DecrementSkiffQuantity decrements the skiff quantity by one.
func (v *VehicleDetails) DecrementSkiffQuantity() {
	if v.skiffQuantity == 0 {
		panic("skiff quantity is 0, cannot decrement it")
	}

	v.skiffQuantity--
}

// HasAtLeastOneSkiff returns true if the vehicle has at least one skiff, false otherwise.
func (v *VehicleDetails) HasAtLeastOneSkiff() bool {
	return v.skiffQuantity > 0
}

func (v *VehicleDetails) IncrementSkiffQuantity() {
	v.skiffQuantity++
}

func (v *VehicleDetails) GetSkiffQuantity() int {
	return v.skiffQuantity
}

// clampHullHP ensures hull HP stays within valid range (0-99)
// Original Ultima V uses 99 as maximum hull health
func clampHullHP(hp int) int {
	return int(math.Max(0, math.Min(MaxHullHP, float64(hp))))
}

// GetHullHP returns the current hull hit points
// Only meaningful for frigates (skiffs don't use hull HP)
func (v *VehicleDetails) GetHullHP() int {
	return v.HullHP
}

// SetHullHP sets the hull HP to a specific value (primarily for testing)
// Automatically clamps to valid range (0-99)
func (v *VehicleDetails) SetHullHP(hp int) {
	v.HullHP = clampHullHP(hp)
}

// GetSailState returns the current sail state
// Only meaningful for frigates (skiffs have no sails)
func (v *VehicleDetails) GetSailState() SailState {
	return v.SailState
}

// IsSailFurled returns true if sails are furled (anchored state)
// Ship repair requires furled sails; ref:SHIP-0004.
// Only meaningful for frigates (skiffs have no sails)
func (v *VehicleDetails) IsSailFurled() bool {
	return v.SailState == SailFurled
}

// SetSailState sets the sail state with logging for debugging
// Only meaningful for frigates (skiffs have no sails)
// Logs state changes to help track unexpected sail state resets
func (v *VehicleDetails) SetSailState(state SailState) {
	if v.SailState != state {
		logging.Debug(logging.CategoryGameState, "SAIL STATE CHANGE: %v -> %v", v.SailState, state)
	}
	v.SailState = state
}

// TakeHullDamage reduces hull HP by the given amount
// Returns true if ship is destroyed (HullHP <= 0), false otherwise
// HP is clamped to 0 minimum (no negative values)
// Pirate cannon hits apply hull damage; ref:SHIP-0003.
func (v *VehicleDetails) TakeHullDamage(amount int) bool {
	v.HullHP = clampHullHP(v.HullHP - amount)

	return v.HullHP <= 0
}

// IsShipDestroyed returns true if the ship is destroyed (HullHP <= 0)
// Only relevant for frigates in gameplay
func (v *VehicleDetails) IsShipDestroyed() bool {
	return v.HullHP <= 0
}
