package game_state

import (
	"fmt"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) registerAtlasVessel(vehicle *map_units.VehicleDetails) {
	if vehicle.VesselID != 0 || vehicle.VehicleType != references.FrigateVehicle && vehicle.VehicleType != references.SkiffVehicle ||
		!g.PartyState.Inventory.QuestItems.HasSome(references.Sextant) {
		return
	}
	// REDUX DEVIATION: ordinary sextant readings require eligible context/hour.
	// Automatic fleet bookkeeping is the user-directed possession-only convenience.
	if g.atlasFleet.LastID >= journal.MaxAtlasIdentity {
		panic("atlas vessel identity exhausted")
	}
	g.atlasFleet.LastID++
	vehicle.VesselID = g.atlasFleet.LastID
}

func (g *GameState) appendPersonalAtlas(dossier *journal.AtlasDossier) {
	for _, note := range g.atlasNotes.Records {
		dossier.Entries = append(dossier.Entries, journal.AtlasEntry{Key: journal.AtlasNoteKey(note.ID), Name: note.Label,
			Kind: "note", Category: journal.AtlasNotes, Phase: -1, NameKnown: true, Note: note})
	}
	dossier.VesselsUnlocked = g.PartyState.Inventory.QuestItems.HasSome(references.Sextant)
	if !dossier.VesselsUnlocked {
		return
	}
	controller := g.LargeMapNPCAIController[references.OVERWORLD]
	if controller == nil || controller.GetNpcs() == nil {
		return
	}
	var entries []journal.AtlasEntry
	for _, unit := range *controller.GetNpcs() {
		boat, ok := unit.(*map_units.NPCFriendly)
		if !ok || boat.NPCReference.GetNPCType() != references.Vehicle || boat.Floor() != 0 || !boat.IsVisible() {
			continue
		}
		vehicle := boat.GetVehicleDetails()
		if vehicle.VesselID == 0 || vehicle.IsPendingDelivery() ||
			vehicle.VehicleType != references.FrigateVehicle && vehicle.VehicleType != references.SkiffVehicle ||
			vehicle.VehicleType == references.FrigateVehicle && vehicle.IsShipDestroyed() {
			continue
		}
		kind, name := "skiff", "Skiff"
		if vehicle.VehicleType == references.FrigateVehicle {
			kind, name = "frigate", "Frigate"
		}
		position := journal.SiteLocation{Map: references.BritanniaUnderworld, X: int(boat.Pos().X), Y: int(boat.Pos().Y)}
		if !journal.ValidAtlasSurfacePosition(position) {
			continue
		}
		entries = append(entries, journal.AtlasEntry{Key: fmt.Sprintf("vessel-%d", vehicle.VesselID),
			Name: fmt.Sprintf("%s %d", name, vehicle.VesselID), Kind: kind, Category: journal.AtlasVessels,
			Phase: -1, NameKnown: true, Vessel: journal.AtlasVesselPosition{ID: vehicle.VesselID, Position: position}})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Vessel.ID < entries[j].Vessel.ID })
	dossier.Entries = append(dossier.Entries, entries...)
}

// CaptureAtlasFleet returns the persistent allocation watermark, not a coordinate cache.
func (g *GameState) CaptureAtlasFleet() journal.AtlasFleetState { return g.atlasFleet }

// RestoreAtlasFleet restores only the watermark. Native validation must check
// all parked and boarded VesselIDs together for uniqueness, boat type and <= LastID.
func (g *GameState) RestoreAtlasFleet(state journal.AtlasFleetState) error {
	if state.LastID > journal.MaxAtlasIdentity {
		return fmt.Errorf("%w: fleet identity watermark", journal.ErrInvalidAtlasPersonal)
	}
	g.atlasFleet = state

	return nil
}
