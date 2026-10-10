package save_snapshot

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func validatePersonalStateV24(snapshot SaveSnapshot, add func(string, string, string)) {
	for i, record := range snapshot.BasementChestHistory {
		validateKnowledgeDate(fmt.Sprintf("basementChestHistory[%d].lastOpened", i), record.LastOpened, add)
	}
	for _, flag := range snapshot.Party.DeadNPCs {
		if _, daily := basementDeathFlagIndex(flag); daily {
			add("party.deadNpcs", "inconsistent", "daily basement chest history replaces obsolete death flags")
		}
	}
	if err := journal.ValidateAtlasNotes(snapshot.AtlasNotes); err != nil {
		add("atlasNotes", "invalid", err.Error())
	}
	vehicles := make([]journal.AtlasVesselRecord, 0, len(snapshot.MapUnits)+1)
	appendVehicle := func(vehicle *VehicleSnapshot) {
		if vehicle != nil {
			vehicles = append(vehicles, journal.AtlasVesselRecord{ID: vehicle.VesselID, VehicleType: references.VehicleType(vehicle.Type)})
			if vehicle.VesselID != 0 && vehicle.PendingDelivery {
				add("atlasFleet", "inconsistent", "pending deliveries cannot be registered vessels")
			}
		}
	}
	appendVehicle(snapshot.PartyVehicle)
	for i, unit := range snapshot.MapUnits {
		if unit.Kind != MapUnitKindVehicle && unit.Vehicle != nil && unit.Vehicle.VesselID != 0 {
			add(fmt.Sprintf("mapUnits[%d].vehicle.vesselId", i), "inconsistent", "registered vessel identity requires a vehicle map unit")
		}
		appendVehicle(unit.Vehicle)
		if unit.SurfaceChestAuthored || unit.SurfaceChestTransient {
			if unit.Kind != MapUnitKindChest || unit.SurfaceChestAuthored && unit.SurfaceChestTransient {
				add(fmt.Sprintf("mapUnits[%d].surfaceChestAuthored", i), "inconsistent", "chest lifetime identities are exclusive and require a chest unit")
			}
		}
		if unit.SurfaceChestAuthored && (unit.Position.Location < 1 || unit.Position.Location > 32) {
			add(fmt.Sprintf("mapUnits[%d].surfaceChestAuthored", i), "inconsistent", "authored chests require a small-map NPC identity")
		}
		if unit.SurfaceChestTransient && (unit.Position.Location != snapshot.World.Position.Location || unit.Position.Floor != snapshot.World.Position.Floor) {
			add(fmt.Sprintf("mapUnits[%d].surfaceChestTransient", i), "inconsistent", "loose child chests belong only to the active map and floor")
		}
		if unit.SurfaceChestAuthored && unit.Position.Location == int(references.LordBritishsCastle) {
			for _, source := range references.LordBritishBasementChests() {
				if unit.NPCNum == source.NPCNumber {
					if !matchesBasementChestSource(unit) {
						add(fmt.Sprintf("mapUnits[%d].position", i), "inconsistent", "authored basement chest must retain its exact source position")
					}
				}
			}
		}
	}
	if err := journal.ValidateAtlasFleet(snapshot.AtlasFleet, vehicles); err != nil {
		add("atlasFleet", "invalid", err.Error())
	}
	if snapshot.CodexRevelationComplete {
		knowledge, recorded := false, false
		for _, entry := range snapshot.Redux.PlayerKnowledge {
			knowledge = knowledge || entry.Category == KnowledgeCategoryDungeon && entry.Key == DungeonKnowledgeDoomWordOfPower
		}
		for _, event := range snapshot.Redux.QuestJournal {
			recorded = recorded || event.Kind == QuestJournalEventKnowledgeLearned &&
				event.Subject == "dungeon.doom.word_of_power"
		}
		// The completion flag owns final acknowledgement; the existing entry
		// retains its already-validated first-discovery provenance.
		if !knowledge || !recorded {
			add("codexRevelationComplete", "inconsistent", "complete revelation requires word knowledge and its Chronicle event")
		}
	}
}
