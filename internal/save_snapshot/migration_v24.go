package save_snapshot

import (
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// promotePersonalStateV24 is the single explicit V12-V23 promotion boundary.
// Old checkpoints never prove personal notes, post-sextant boat use, or final
// Codex acknowledgement. Only the three obsolete authored death flags prove a
// consumed daily chest; off-floor absence is not evidence of an opening.
func (snapshot *SaveSnapshot) promotePersonalStateV24() {
	snapshot.AtlasNotes = journal.AtlasNotesState{}
	snapshot.AtlasFleet = journal.AtlasFleetState{}
	snapshot.CodexRevelationComplete = false
	kept := make([]NPCFlagSnapshot, 0, len(snapshot.Party.DeadNPCs))
	for _, flag := range snapshot.Party.DeadNPCs {
		index, daily := basementDeathFlagIndex(flag)
		if !daily {
			kept = append(kept, flag)

			continue
		}
		date := snapshot.World.Calendar
		snapshot.BasementChestHistory[index].LastOpened = &BritannianDateSnapshot{Year: date.Year, Month: date.Month, Day: date.Day}
	}
	if snapshot.Party.DeadNPCs != nil {
		snapshot.Party.DeadNPCs = kept
	}
	for i := range snapshot.MapUnits {
		unit := &snapshot.MapUnits[i]
		if matchesBasementChestSource(*unit) {
			unit.SurfaceChestAuthored = true
			unit.Kind, unit.ReferenceID = MapUnitKindChest, 0
		}
	}
}

func basementDeathFlagIndex(flag NPCFlagSnapshot) (int, bool) {
	if flag.Location != int(references.LordBritishsCastle-references.Moonglow) {
		return 0, false
	}
	for i, source := range references.LordBritishBasementChests() {
		if flag.NPC == source.NPCNumber {
			return i, true
		}
	}

	return 0, false
}

func matchesBasementChestSource(unit MapUnitSnapshot) bool {
	if (unit.Kind != MapUnitKindChest && unit.Kind != MapUnitKindNPC) || unit.SurfaceChestNumber == nil ||
		unit.Position.Location != int(references.LordBritishsCastle) || unit.Position.Floor != int(references.Basement) {
		return false
	}
	for _, source := range references.LordBritishBasementChests() {
		if unit.NPCNum == source.NPCNumber && unit.Position.X == int(source.Position.X) && unit.Position.Y == int(source.Position.Y) {
			return true
		}
	}

	return false
}
