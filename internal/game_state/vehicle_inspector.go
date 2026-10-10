package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ErrInvalidPartyVehicleHullOperation indicates that a raw hull edit would violate vehicle invariants.
var ErrInvalidPartyVehicleHullOperation = errors.New("invalid party vehicle hull operation")

// ErrInvalidPartyVehicleSkiffOperation indicates an invalid attached-skiff edit.
var ErrInvalidPartyVehicleSkiffOperation = errors.New("invalid party vehicle skiff operation")

// MaxFrigateSkiffs aliases the shared deliberate Redux frigate capacity.
const MaxFrigateSkiffs = references.MaxFrigateSkiffs

// PartyVehicleSnapshot is a detached view of the currently boarded vehicle.
type PartyVehicleSnapshot struct {
	map_units.VehicleDetailsSnapshot
	Position                references.Position
	Floor                   references.FloorNumber
	HMSCapeActive           bool
	RawHullDamageAvailable  bool
	FullHullRepairAvailable bool
}

// PartyVehicleHullOperationResult reports exact vehicle state surrounding a hull edit.
type PartyVehicleHullOperationResult struct {
	Before PartyVehicleSnapshot
	After  PartyVehicleSnapshot
}

// PartyVehicleSkiffOperationResult reports exact vehicle state surrounding a skiff edit.
type PartyVehicleSkiffOperationResult struct {
	Before PartyVehicleSnapshot
	After  PartyVehicleSnapshot
}

// CurrentPartyVehicleSnapshot returns the current boarded vehicle at the player's authoritative location.
func (g *GameState) CurrentPartyVehicleSnapshot() PartyVehicleSnapshot {
	details := g.PartyVehicle.GetVehicleDetails().Snapshot()
	isFrigate := details.VehicleType == references.FrigateVehicle

	return PartyVehicleSnapshot{
		VehicleDetailsSnapshot:  details,
		Position:                g.MapState.PlayerLocation.Position,
		Floor:                   g.MapState.PlayerLocation.Floor,
		HMSCapeActive:           isFrigate && g.IsHMSCapeActivated(),
		RawHullDamageAvailable:  isFrigate && details.HullHP > 1,
		FullHullRepairAvailable: isFrigate,
	}
}

// ApplyRawPartyVehicleHullDamage applies positive non-sinking damage to the current frigate.
func (g *GameState) ApplyRawPartyVehicleHullDamage(amount int) (PartyVehicleHullOperationResult, error) {
	before := g.CurrentPartyVehicleSnapshot()
	result := PartyVehicleHullOperationResult{Before: before, After: before}
	if before.VehicleType != references.FrigateVehicle {
		return result, fmt.Errorf(
			"%w: current vehicle type %d is not a frigate",
			ErrInvalidPartyVehicleHullOperation,
			before.VehicleType,
		)
	}
	if amount <= 0 {
		return result, fmt.Errorf("%w: damage must be positive", ErrInvalidPartyVehicleHullOperation)
	}
	if amount >= before.HullHP {
		return result, fmt.Errorf(
			"%w: damage %d would reduce hull %d to zero or below",
			ErrInvalidPartyVehicleHullOperation,
			amount,
			before.HullHP,
		)
	}

	g.PartyVehicle.GetVehicleDetails().TakeHullDamage(amount)
	result.After = g.CurrentPartyVehicleSnapshot()

	return result, nil
}

// RepairPartyVehicleHullToMaximum restores the current frigate to authentic maximum hull strength.
func (g *GameState) RepairPartyVehicleHullToMaximum() (PartyVehicleHullOperationResult, error) {
	before := g.CurrentPartyVehicleSnapshot()
	result := PartyVehicleHullOperationResult{Before: before, After: before}
	if before.VehicleType != references.FrigateVehicle {
		return result, fmt.Errorf(
			"%w: current vehicle type %d is not a frigate",
			ErrInvalidPartyVehicleHullOperation,
			before.VehicleType,
		)
	}

	g.PartyVehicle.GetVehicleDetails().SetHullHP(map_units.MaxHullHP)
	result.After = g.CurrentPartyVehicleSnapshot()

	return result, nil
}

// SetPartyFrigateSkiffQuantity sets the exact attached-skiff quantity.
func (g *GameState) SetPartyFrigateSkiffQuantity(quantity int) (PartyVehicleSkiffOperationResult, error) {
	before := g.CurrentPartyVehicleSnapshot()
	result := PartyVehicleSkiffOperationResult{Before: before, After: before}
	if before.VehicleType != references.FrigateVehicle {
		return result, fmt.Errorf("%w: current vehicle is not a frigate", ErrInvalidPartyVehicleSkiffOperation)
	}
	if quantity < 0 || quantity > MaxFrigateSkiffs {
		return result, fmt.Errorf(
			"%w: quantity %d outside 0..%d",
			ErrInvalidPartyVehicleSkiffOperation,
			quantity,
			MaxFrigateSkiffs,
		)
	}

	g.PartyVehicle.GetVehicleDetails().SetSkiffQuantity(quantity)
	result.After = g.CurrentPartyVehicleSnapshot()

	return result, nil
}
