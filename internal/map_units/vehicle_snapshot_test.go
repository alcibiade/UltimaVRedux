package map_units

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAtlasVesselIdentitySurvivesVehicleSnapshot(t *testing.T) {
	vehicle := NewVehicleDetails(references.SkiffVehicle)
	vehicle.VesselID = 73
	snapshot := vehicle.Snapshot()
	var restored VehicleDetails
	restored.Restore(snapshot)
	if restored.VesselID != 73 {
		t.Fatalf("restored vessel identity = %d", restored.VesselID)
	}
	if NewVehicleDetails(references.SkiffVehicle).VesselID != 0 {
		t.Fatal("new vessel inherited fleet identity")
	}
}

// TestVehicleDetailsSnapshotIsExactAndDetached prevents inspectors from retaining mutable vehicle state.
func TestVehicleDetailsSnapshotIsExactAndDetached(t *testing.T) {
	details := NewVehicleDetails(references.FrigateVehicle)
	details.SetPartyVehicleDirection(references.Up)
	details.IsOwnedByNPC = true
	details.SetHullHP(47)
	details.SetSailState(SailRaised)
	details.SetSkiffQuantity(1)

	snapshot := details.Snapshot()
	want := VehicleDetailsSnapshot{
		VehicleType:       references.FrigateVehicle,
		CurrentDirection:  references.Up,
		PreviousDirection: references.Right,
		IsOwnedByNPC:      true,
		HullHP:            47,
		MaxHullHP:         MaxHullHP,
		SailState:         SailRaised,
		SkiffQuantity:     1,
	}
	if snapshot != want {
		t.Fatalf("Snapshot() = %+v, want %+v", snapshot, want)
	}

	details.SetHullHP(12)
	details.SetSkiffQuantity(0)
	if snapshot != want {
		t.Fatalf("snapshot changed after source mutation: got %+v, want %+v", snapshot, want)
	}
}
