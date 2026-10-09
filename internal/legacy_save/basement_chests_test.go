package legacy_save

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestLegacyBasementHistoryUsesOnlyAuthoritativeActiveFloor(t *testing.T) {
	for _, context := range []string{"basement", "castle upstairs", "outside"} {
		t.Run(context, func(t *testing.T) {
			bundle := validConversionBundle(t)
			bundle.Game.Position.Location = MapID(references.LordBritishsCastle)
			bundle.Game.Position.X, bundle.Game.Position.Y, bundle.Game.Position.Floor = 15, 21, 0xff
			bundle.Game.ActiveObjects = [objectCount]ObjectRecord{}
			// The active list has one surviving trapped chest; object slot 7
			// is an allocation slot, not CASTLE.NPC authored record 24.
			bundle.Game.ActiveObjects[7] = ObjectRecord{Tile: originalChestShape, X: 17, Y: 22, Level: 0xff, Number: 0x9e}
			// Neither off-floor data nor stale OOL objects can restore chest 23.
			bundle.Game.ActiveObjects[8] = ObjectRecord{Tile: originalChestShape, X: 16, Y: 21, Level: 0, Number: 30}
			bundle.Objects.Britannia[1] = ObjectRecord{Tile: originalChestShape, X: 16, Y: 21, Level: 0, Number: 30}
			if context == "castle upstairs" {
				bundle.Game.Position.Floor = 0
			}
			if context == "outside" {
				bundle.Game.Position.Location, bundle.Game.Position.Floor = MapID(references.BritanniaUnderworld), 0
				bundle.Game.ActiveObjects = [objectCount]ObjectRecord{}
			}
			before := bundle
			converted, err := Convert(bundle)
			require.NoError(t, err)
			require.Equal(t, before, bundle, "conversion is read-only")
			snapshot := converted.Snapshot
			require.Equal(t, journal.AtlasNotesState{}, snapshot.AtlasNotes)
			require.Equal(t, journal.AtlasFleetState{}, snapshot.AtlasFleet)
			require.False(t, snapshot.CodexRevelationComplete)
			for _, unit := range snapshot.MapUnits {
				if unit.Vehicle != nil {
					require.Zero(t, unit.Vehicle.VesselID)
				}
			}
			if snapshot.PartyVehicle != nil {
				require.Zero(t, snapshot.PartyVehicle.VesselID)
			}
			if context != "basement" {
				require.Equal(t, [3]save_snapshot.BasementChestDailyRecordSnapshot{}, snapshot.BasementChestHistory)
				return
			}
			date := snapshot.World.Calendar
			want := &save_snapshot.BritannianDateSnapshot{Year: date.Year, Month: date.Month, Day: date.Day}
			require.Equal(t, want, snapshot.BasementChestHistory[0].LastOpened)
			require.Nil(t, snapshot.BasementChestHistory[1].LastOpened)
			require.Equal(t, want, snapshot.BasementChestHistory[2].LastOpened)
			require.True(t, hasConversionWarning(converted.Warnings, "gam.activeObjects", "basement_chest_saved_date"))
			found := false
			for _, unit := range snapshot.MapUnits {
				if unit.Position.Location == int(references.LordBritishsCastle) && unit.Position.Floor == -1 && unit.Position.X == 17 && unit.Position.Y == 22 {
					found = true
					require.Equal(t, 0x9e, *unit.SurfaceChestNumber)
					require.Zero(t, unit.SurfaceChestAssessment)
				}
			}
			require.True(t, found)
		})
	}
}

func TestLegacyChestFlagsCannotInventDailyHistory(t *testing.T) {
	bundle := validConversionBundle(t)
	// Original CHEST objects were not legitimate npc_dead victims. Clear just
	// these identities without treating arbitrary imported bits as opening dates.
	location := int(references.LordBritishsCastle - references.Moonglow)
	for _, npc := range []int{23, 24, 25, 31} {
		bundle.Game.DeadNPCs[location*npcFlagBytesPerLocation+npc/bitsPerByte] |= 1 << uint(npc%bitsPerByte)
	}
	converted, err := Convert(bundle)
	require.NoError(t, err)
	require.Equal(t, [3]save_snapshot.BasementChestDailyRecordSnapshot{}, converted.Snapshot.BasementChestHistory)
	for _, npc := range []int{23, 24, 25} {
		require.False(t, hasNPCFlag(converted.Snapshot.Party.DeadNPCs, location, npc))
	}
	require.True(t, hasNPCFlag(converted.Snapshot.Party.DeadNPCs, location, 31))
}

func TestLegacyBasementChildDoesNotMasqueradeAsAuthoredChest(t *testing.T) {
	bundle := validConversionBundle(t)
	bundle.Game.Position.Location = MapID(references.LordBritishsCastle)
	bundle.Game.Position.X, bundle.Game.Position.Y, bundle.Game.Position.Floor = 15, 21, 0xff
	bundle.Game.ActiveObjects = [objectCount]ObjectRecord{}
	bundle.Game.ActiveObjects[7] = ObjectRecord{Tile: originalChestShape, X: 16, Y: 21, Level: 0xff, Number: 7}
	converted, err := Convert(bundle)
	require.NoError(t, err)
	require.NotNil(t, converted.Snapshot.BasementChestHistory[0].LastOpened)
	found := false
	for _, unit := range converted.Snapshot.MapUnits {
		if unit.Position.Location != int(references.LordBritishsCastle) {
			continue
		}
		found = true
		require.True(t, unit.SurfaceChestTransient)
		require.False(t, unit.SurfaceChestAuthored)
		require.Equal(t, 7, *unit.SurfaceChestNumber)
	}
	require.True(t, found)
	require.True(t, hasConversionWarning(converted.Warnings, "gam.activeObjects[7].number", "basement_child_chest"))
}
