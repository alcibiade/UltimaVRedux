package game_state

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newVehicleInspectorGameState(vehicleType references.VehicleType, hull int) *GameState {
	vehicle := map_units.NewNPCFriendlyVehiceNewRef(vehicleType, references.Position{}, 0)
	vehicle.GetVehicleDetails().SetHullHP(hull)

	return &GameState{
		PartyVehicle: *vehicle,
		MapState: map_state.MapState{
			PlayerLocation: references.PlayerLocation{
				Location: references.BritanniaUnderworld,
				Position: references.Position{X: 23, Y: 41},
				Floor:    -1,
			},
		},
	}
}

// TestCurrentPartyVehicleSnapshotReportsAuthoritativeState verifies all fields required by the vehicle inspector.
func TestCurrentPartyVehicleSnapshotReportsAuthoritativeState(t *testing.T) {
	gs := newVehicleInspectorGameState(references.FrigateVehicle, 63)
	details := gs.PartyVehicle.GetVehicleDetails()
	details.SetPartyVehicleDirection(references.Left)
	details.IsOwnedByNPC = true
	details.SetSailState(map_units.SailRaised)
	details.SetSkiffQuantity(1)
	gs.SetHMSCapeActivated(true)

	got := gs.CurrentPartyVehicleSnapshot()
	want := PartyVehicleSnapshot{
		VehicleDetailsSnapshot: map_units.VehicleDetailsSnapshot{
			VehicleType:       references.FrigateVehicle,
			CurrentDirection:  references.Left,
			PreviousDirection: references.Right,
			IsOwnedByNPC:      true,
			HullHP:            63,
			MaxHullHP:         map_units.MaxHullHP,
			SailState:         map_units.SailRaised,
			SkiffQuantity:     1,
		},
		Position:                references.Position{X: 23, Y: 41},
		Floor:                   -1,
		HMSCapeActive:           true,
		RawHullDamageAvailable:  true,
		FullHullRepairAvailable: true,
	}
	if got != want {
		t.Fatalf("CurrentPartyVehicleSnapshot() = %+v, want %+v", got, want)
	}

	gs.MapState.PlayerLocation.Position = references.Position{X: 1, Y: 2}
	if got != want {
		t.Fatalf("snapshot changed after PlayerLocation mutation: got %+v, want %+v", got, want)
	}
}

// TestApplyRawPartyVehicleHullDamageReturnsExactSnapshots documents non-sinking raw damage behavior.
func TestApplyRawPartyVehicleHullDamageReturnsExactSnapshots(t *testing.T) {
	gs := newVehicleInspectorGameState(references.FrigateVehicle, 63)
	before := gs.CurrentPartyVehicleSnapshot()

	result, err := gs.ApplyRawPartyVehicleHullDamage(17)
	if err != nil {
		t.Fatalf("ApplyRawPartyVehicleHullDamage() error = %v", err)
	}
	wantAfter := before
	wantAfter.HullHP = 46
	if result.Before != before || result.After != wantAfter {
		t.Fatalf("ApplyRawPartyVehicleHullDamage() = %+v, want before %+v after %+v", result, before, wantAfter)
	}
	if got := gs.CurrentPartyVehicleSnapshot(); got != wantAfter {
		t.Fatalf("current snapshot = %+v, want %+v", got, wantAfter)
	}
}

// TestApplyRawPartyVehicleHullDamageRejectsInvalidInputAtomically prevents raw edits from invoking sinking or healing.
func TestApplyRawPartyVehicleHullDamageRejectsInvalidInputAtomically(t *testing.T) {
	tests := []struct {
		name   string
		amount int
	}{
		{name: "negative damage", amount: -1},
		{name: "zero damage", amount: 0},
		{name: "damage to zero", amount: 20},
		{name: "damage below zero", amount: 21},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs := newVehicleInspectorGameState(references.FrigateVehicle, 20)
			before := gs.CurrentPartyVehicleSnapshot()

			result, err := gs.ApplyRawPartyVehicleHullDamage(test.amount)
			if !errors.Is(err, ErrInvalidPartyVehicleHullOperation) {
				t.Fatalf("error = %v, want ErrInvalidPartyVehicleHullOperation", err)
			}
			if result.Before != before || result.After != before {
				t.Fatalf("result changed on rejection: got %+v, want before/after %+v", result, before)
			}
			if got := gs.CurrentPartyVehicleSnapshot(); got != before {
				t.Fatalf("state mutated on rejection: got %+v, want %+v", got, before)
			}
		})
	}
}

// TestPartyVehicleHullOperationsRejectNonFrigates documents that synthetic hull fields on other vehicles are not editable.
func TestPartyVehicleHullOperationsRejectNonFrigates(t *testing.T) {
	for _, vehicleType := range []references.VehicleType{
		references.NoPartyVehicle,
		references.HorseVehicle,
		references.CarpetVehicle,
		references.SkiffVehicle,
	} {
		t.Run(fmt.Sprintf("vehicle type %d", vehicleType), func(t *testing.T) {
			gs := newVehicleInspectorGameState(vehicleType, 50)
			before := gs.CurrentPartyVehicleSnapshot()

			damageResult, damageErr := gs.ApplyRawPartyVehicleHullDamage(1)
			if !errors.Is(damageErr, ErrInvalidPartyVehicleHullOperation) ||
				damageResult.Before != before || damageResult.After != before {
				t.Fatalf("damage result/error = %+v/%v, want unchanged invalid operation", damageResult, damageErr)
			}
			repairResult, repairErr := gs.RepairPartyVehicleHullToMaximum()
			if !errors.Is(repairErr, ErrInvalidPartyVehicleHullOperation) ||
				repairResult.Before != before || repairResult.After != before {
				t.Fatalf("repair result/error = %+v/%v, want unchanged invalid operation", repairResult, repairErr)
			}
			if got := gs.CurrentPartyVehicleSnapshot(); got != before {
				t.Fatalf("state mutated: got %+v, want %+v", got, before)
			}
		})
	}
}

// TestRepairPartyVehicleHullToMaximumIsExactAndIdempotent prevents full repair from changing unrelated vehicle state.
func TestRepairPartyVehicleHullToMaximumIsExactAndIdempotent(t *testing.T) {
	gs := newVehicleInspectorGameState(references.FrigateVehicle, 37)
	before := gs.CurrentPartyVehicleSnapshot()

	result, err := gs.RepairPartyVehicleHullToMaximum()
	if err != nil {
		t.Fatalf("RepairPartyVehicleHullToMaximum() error = %v", err)
	}
	wantAfter := before
	wantAfter.HullHP = map_units.MaxHullHP
	if result.Before != before || result.After != wantAfter {
		t.Fatalf("first repair = %+v, want before %+v after %+v", result, before, wantAfter)
	}

	second, err := gs.RepairPartyVehicleHullToMaximum()
	if err != nil {
		t.Fatalf("second RepairPartyVehicleHullToMaximum() error = %v", err)
	}
	if second.Before != wantAfter || second.After != wantAfter {
		t.Fatalf("idempotent repair = %+v, want before/after %+v", second, wantAfter)
	}
}

// TestSetPartyFrigateSkiffQuantityUsesAuthenticSixBitRange prevents vehicle flag corruption.
func TestSetPartyFrigateSkiffQuantityUsesAuthenticSixBitRange(t *testing.T) {
	gs := newVehicleInspectorGameState(references.FrigateVehicle, map_units.MaxHullHP)
	for _, quantity := range []int{0, MaxFrigateSkiffs} {
		result, err := gs.SetPartyFrigateSkiffQuantity(quantity)
		if err != nil {
			t.Fatalf("SetPartyFrigateSkiffQuantity(%d) error = %v", quantity, err)
		}
		if result.After.SkiffQuantity != quantity {
			t.Fatalf("quantity after edit = %d, want %d", result.After.SkiffQuantity, quantity)
		}
	}

	before := gs.CurrentPartyVehicleSnapshot()
	for _, quantity := range []int{-1, MaxFrigateSkiffs + 1} {
		result, err := gs.SetPartyFrigateSkiffQuantity(quantity)
		if !errors.Is(err, ErrInvalidPartyVehicleSkiffOperation) {
			t.Fatalf("quantity %d error = %v, want ErrInvalidPartyVehicleSkiffOperation", quantity, err)
		}
		if result.Before != before || result.After != before || gs.CurrentPartyVehicleSnapshot() != before {
			t.Fatalf("quantity %d mutated state after rejection", quantity)
		}
	}
}

// TestSetPartyFrigateSkiffQuantityRejectsNonFrigate prevents synthetic skiff edits on other vehicles.
func TestSetPartyFrigateSkiffQuantityRejectsNonFrigate(t *testing.T) {
	gs := newVehicleInspectorGameState(references.SkiffVehicle, 0)
	before := gs.CurrentPartyVehicleSnapshot()
	result, err := gs.SetPartyFrigateSkiffQuantity(1)
	if !errors.Is(err, ErrInvalidPartyVehicleSkiffOperation) || result.Before != before || result.After != before {
		t.Fatalf("result/error = %+v/%v, want unchanged invalid operation", result, err)
	}
}
