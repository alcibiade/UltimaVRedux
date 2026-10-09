package legacy_save

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	originalBasementChestValue = 30
	originalChestValueMask     = 0x7f
)

// Original town initialization constructs only the active level's objects (ref:CMD-0009),
// with authored chest value 30. SAVED.GAM's active list is resume authority;
// neither stale OOL halves nor an off-floor missing object proves consumption.
func convertBasementChestHistory(
	game SavedGame,
	snapshot *save_snapshot.SaveSnapshot,
	warnings *[]ConversionWarning,
) {
	flags := snapshot.Party.DeadNPCs[:0]
	for _, flag := range snapshot.Party.DeadNPCs {
		if !isDailyBasementDeadFlag(flag) {
			flags = append(flags, flag)
		} else {
			addConversionWarning(warnings, "gam.deadNpcs", "basement_chest_death_flag_ignored",
				fmt.Sprintf("ordinary basement chest %d cannot be permanently killed; "+
					"its legacy flag supplies no opening date", flag.NPC))
		}
	}
	snapshot.Party.DeadNPCs = flags
	if references.Location(game.Position.Location) != references.LordBritishsCastle ||
		signedFloor(game.Position.Floor) != int(references.Basement) {
		return
	}
	markKnownBasementChildren(snapshot, warnings)
	for index, source := range references.LordBritishBasementChests() {
		if activeBasementChestPresent(game, source.Position) {
			continue
		}
		date := snapshot.World.Calendar
		snapshot.BasementChestHistory[index].LastOpened = &save_snapshot.BritannianDateSnapshot{
			Year: date.Year, Month: date.Month, Day: date.Day,
		}
		addConversionWarning(warnings, "gam.activeObjects", "basement_chest_saved_date",
			fmt.Sprintf("absent authored basement chest %d defaults its last opening to the imported date", source.NPCNumber))
	}
}

func isDailyBasementDeadFlag(flag save_snapshot.NPCFlagSnapshot) bool {
	for _, source := range references.LordBritishBasementChests() {
		if flag.Location == lordBritishNPCFlagLocation && flag.NPC == source.NPCNumber {
			return true
		}
	}

	return false
}

func activeBasementChestPresent(game SavedGame, position references.Position) bool {
	for slot, record := range game.ActiveObjects {
		if slot == 0 {
			continue
		}
		if record.Tile == originalChestShape && signedFloor(record.Level) == int(references.Basement) &&
			int(record.X) == int(position.X) && int(record.Y) == int(position.Y) &&
			record.Number&originalChestValueMask == originalBasementChestValue {
			// A value-30 same-square chest is ambiguous (additem can generate
			// one too). Preserve it rather than inventing a past opening.
			return true
		}
	}

	return false
}

func markKnownBasementChildren(snapshot *save_snapshot.SaveSnapshot, warnings *[]ConversionWarning) {
	for index := range snapshot.MapUnits {
		unit := &snapshot.MapUnits[index]
		if unit.Kind != save_snapshot.MapUnitKindChest || unit.SurfaceChestNumber == nil ||
			unit.Position.Location != int(references.LordBritishsCastle) ||
			unit.Position.Floor != int(references.Basement) ||
			*unit.SurfaceChestNumber&originalChestValueMask == originalBasementChestValue {
			continue
		}
		// Authored chests have value 30; ref:CMD-0009.
		// Generated chest values use independent rolls; ref:COMBAT-0010.
		unit.SurfaceChestTransient = true
		addConversionWarning(warnings, fmt.Sprintf("gam.activeObjects[%d].number", unit.NPCNum),
			"basement_child_chest", "non-authored basement chest value preserves a transient child lifetime")
	}
}
