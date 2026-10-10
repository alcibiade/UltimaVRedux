package journal

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func atlasValidNote() AtlasNoteRecord {
	return AtlasNoteRecord{ID: 1, Position: SiteLocation{Map: references.BritanniaUnderworld, X: 17, Y: 29}, Label: "Landing",
		Description: "Leave the skiff.\n\nBring food.", RecordedAt: GameTime{Year: 139, Month: 1, Day: 1, Hour: 20}}
}

func TestAtlasPersonalValidationAndDistinctAuthorities(t *testing.T) {
	note := atlasValidNote()
	state := AtlasNotesState{LastID: 1, Records: []AtlasNoteRecord{note}}
	require.NoError(t, ValidateAtlasNotes(state))
	for _, mutate := range []func(*AtlasNoteRecord){
		func(n *AtlasNoteRecord) { n.ID = 0 }, func(n *AtlasNoteRecord) { n.ID = 2 },
		func(n *AtlasNoteRecord) { n.Position.Floor = -1 }, func(n *AtlasNoteRecord) { n.Position.Map = references.Britain },
		func(n *AtlasNoteRecord) { n.Position.X = 256 }, func(n *AtlasNoteRecord) { n.RecordedAt.Year = 0 },
		func(n *AtlasNoteRecord) { n.Label = "  " }, func(n *AtlasNoteRecord) { n.Label = "A\nB" },
		func(n *AtlasNoteRecord) { n.Label = strings.Repeat("a", 33) }, func(n *AtlasNoteRecord) { n.Description = strings.Repeat("b", 501) },
		func(n *AtlasNoteRecord) { n.Description = "invisible\ttext" }, func(n *AtlasNoteRecord) { n.Label = "unsupported ☃" },
	} {
		bad := note
		mutate(&bad)
		require.ErrorIs(t, ValidateAtlasNotes(AtlasNotesState{LastID: 1, Records: []AtlasNoteRecord{bad}}), ErrInvalidAtlasPersonal)
	}
	require.NoError(t, ValidateAtlasNoteText(strings.Repeat("x", 32), strings.Repeat("\n", 500)))
	require.NoError(t, ValidateAtlasNoteText("Label", ""))
	duplicate := note
	duplicate.ID = 2
	require.Error(t, ValidateAtlasNotes(AtlasNotesState{LastID: 2, Records: []AtlasNoteRecord{note, duplicate}}))
	for _, entry := range []AtlasEntry{
		{Category: AtlasNotes, Note: note},
		{Category: AtlasVessels, Vessel: AtlasVesselPosition{ID: 3, Position: note.Position}},
	} {
		position, ok := entry.PlotPosition()
		require.True(t, ok)
		require.Equal(t, note.Position, position)
		require.False(t, entry.Survey.Measured, "authorship/parked authority does not invent a survey")
	}
	_, ok := (AtlasEntry{Category: AtlasSettlements, Note: note, Vessel: AtlasVesselPosition{ID: 3, Position: note.Position}}).PlotPosition()
	require.False(t, ok, "new authorities must not grant ordinary landmark coordinates")
}

func TestAtlasFleetValidationAcrossRepresentations(t *testing.T) {
	fleet := AtlasFleetState{LastID: 2}
	records := []AtlasVesselRecord{{ID: 1, VehicleType: references.FrigateVehicle}, {ID: 2, VehicleType: references.SkiffVehicle}}
	require.NoError(t, ValidateAtlasFleet(fleet, records))
	require.NoError(t, ValidateAtlasFleet(AtlasFleetState{}, []AtlasVesselRecord{{VehicleType: references.HorseVehicle}}))
	for _, bad := range []AtlasVesselRecord{{ID: 1, VehicleType: references.SkiffVehicle}, {ID: 3, VehicleType: references.SkiffVehicle}, {ID: 2, VehicleType: references.HorseVehicle}} {
		require.Error(t, ValidateAtlasFleet(fleet, []AtlasVesselRecord{records[0], bad}))
	}
}
