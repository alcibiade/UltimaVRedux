package map_units

import "github.com/bradhannah/UltimaVRedux/internal/references"

// VehicleDetailsSnapshot is a detached view of a vehicle's inspectable state.
type VehicleDetailsSnapshot struct {
	VesselID          uint64
	VehicleType       references.VehicleType
	CurrentDirection  references.Direction
	PreviousDirection references.Direction
	IsOwnedByNPC      bool
	HullHP            int
	MaxHullHP         int
	SailState         SailState
	SkiffQuantity     int
	PendingDelivery   bool
}

// Snapshot returns a detached view of the vehicle details.
func (v *VehicleDetails) Snapshot() VehicleDetailsSnapshot {
	return VehicleDetailsSnapshot{
		VesselID:          v.VesselID,
		VehicleType:       v.VehicleType,
		CurrentDirection:  v.currentDirection,
		PreviousDirection: v.previousDirection,
		IsOwnedByNPC:      v.IsOwnedByNPC,
		HullHP:            v.HullHP,
		MaxHullHP:         MaxHullHP,
		SailState:         v.SailState,
		SkiffQuantity:     v.skiffQuantity,
		PendingDelivery:   v.pendingDelivery,
	}
}

// Restore replaces all persistent vehicle details from a detached snapshot.
func (v *VehicleDetails) Restore(snapshot VehicleDetailsSnapshot) {
	v.VesselID = snapshot.VesselID
	v.VehicleType = snapshot.VehicleType
	v.currentDirection = snapshot.CurrentDirection
	v.previousDirection = snapshot.PreviousDirection
	v.IsOwnedByNPC = snapshot.IsOwnedByNPC
	v.HullHP = clampHullHP(snapshot.HullHP)
	v.SailState = snapshot.SailState
	v.skiffQuantity = snapshot.SkiffQuantity
	v.pendingDelivery = snapshot.PendingDelivery
}
