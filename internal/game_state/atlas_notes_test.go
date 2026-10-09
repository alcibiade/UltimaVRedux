package game_state

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestAtlasNoteCapturedFixCRUDAndCheckpointAuthority(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithPlayerAt(17, 29).WithSystemCallbacks().WithRandomSeed(42).Build()
	state.DateTime.Hour = 19
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	_, err := state.BeginAtlasNoteDraft(0)
	require.Error(t, err)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	draft, err := state.BeginAtlasNoteDraft(0)
	require.NoError(t, err)
	beforeDate, beforeCalls := state.DateTime, state.GetRNG().Calls()
	require.Empty(t, state.CaptureAtlasNotes().Records, "opening a draft is not a commitment")
	_, err = state.CommitAtlasNote(draft, "\nBad label", "")
	require.Error(t, err)
	_, err = state.CommitAtlasNote(draft, "   ", "preserved")
	require.Error(t, err)
	note, err := state.CommitAtlasNote(draft, "  Landing  ", "Walk north.\n\nBring  food.")
	require.NoError(t, err)
	require.Equal(t, "Landing", note.Label)
	require.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, X: 17, Y: 29}, note.Position)
	require.Equal(t, uint64(1), note.ID)
	_, err = state.CommitAtlasNote(draft, "twice", "")
	require.Error(t, err)
	duplicate, err := state.BeginAtlasNoteDraft(0)
	require.NoError(t, err)
	require.False(t, duplicate.Creating())
	require.Equal(t, note, duplicate.Record())
	checkpoint := state.CaptureAtlasNotes()
	entry := atlasEntryByKey(t, state.CurrentAtlasDossier(), journal.AtlasNoteKey(note.ID))
	position, ok := entry.PlotPosition()
	require.True(t, ok)
	require.Equal(t, note.Position, position)
	require.False(t, entry.Survey.Measured)
	state.DateTime.Hour = 12
	state.MapState.PlayerLocation.Location = references.Deceit
	edit, err := state.BeginAtlasNoteDraft(note.ID)
	require.NoError(t, err)
	edited, err := state.CommitAtlasNote(edit, "Renamed", "")
	require.NoError(t, err)
	require.Equal(t, note.RecordedAt, edited.RecordedAt)
	require.Equal(t, note.Position, edited.Position)
	require.Equal(t, note.ID, edited.ID)
	stale, err := state.BeginAtlasNoteDraft(note.ID)
	require.NoError(t, err)
	require.NoError(t, state.RestoreAtlasNotes(checkpoint))
	_, err = state.CommitAtlasNote(stale, "stale", "")
	require.Error(t, err)
	require.Equal(t, note, state.CaptureAtlasNotes().Records[0])
	checkpoint.Records[0].Label = "outside mutation"
	require.Equal(t, "Landing", state.CaptureAtlasNotes().Records[0].Label)
	deletion, err := state.BeginAtlasNoteDraft(note.ID)
	require.NoError(t, err)
	require.NoError(t, state.DeleteAtlasNote(deletion))
	require.Error(t, state.DeleteAtlasNote(deletion))
	require.Empty(t, state.CaptureAtlasNotes().Records)
	require.Equal(t, uint64(1), state.CaptureAtlasNotes().LastID)
	state.DateTime = beforeDate
	state.MapState.PlayerLocation.Location = references.BritanniaUnderworld
	next, err := state.BeginAtlasNoteDraft(0)
	require.NoError(t, err)
	nextNote, err := state.CommitAtlasNote(next, "Next", "")
	require.NoError(t, err)
	require.Equal(t, uint64(2), nextNote.ID)
	require.Equal(t, beforeDate, state.DateTime)
	require.Equal(t, beforeCalls, state.GetRNG().Calls())
	require.Empty(t, state.atlasLandmarks)
}

func TestAtlasNoteStaleContextAndCapacity(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 19
	for _, change := range []func(){func() { state.MapState.PlayerLocation.Position.X++ }, func() { state.DateTime.Minute++ }, func() { state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0) }} {
		state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
		draft, err := state.BeginAtlasNoteDraft(0)
		require.NoError(t, err)
		change()
		_, err = state.CommitAtlasNote(draft, "Rejected", "")
		require.Error(t, err)
		require.Empty(t, state.CaptureAtlasNotes().Records)
	}
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	for x := range journal.MaxAtlasNotes {
		state.MapState.PlayerLocation.Position = references.Position{X: references.Coordinate(x), Y: 0}
		draft, err := state.BeginAtlasNoteDraft(0)
		require.NoError(t, err)
		_, err = state.CommitAtlasNote(draft, "Same label", strings.Repeat("x", 500))
		require.NoError(t, err)
	}
	state.MapState.PlayerLocation.Position.Y = 1
	_, err := state.BeginAtlasNoteDraft(0)
	require.ErrorContains(t, err, "full")
	edit, err := state.BeginAtlasNoteDraft(1)
	require.NoError(t, err)
	_, err = state.CommitAtlasNote(edit, strings.Repeat("x", 32), "")
	require.NoError(t, err)
}

func TestAtlasNoteStrictFlexibleAndSurfaceExteriors(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithPlayerAt(5, 7).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 12
	_, err := state.BeginAtlasNoteDraft(0)
	require.Error(t, err)
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
	draft, err := state.BeginAtlasNoteDraft(0)
	require.NoError(t, err)
	require.Equal(t, "Settlement entrance on Britannia", draft.ContextLabel())
	exterior, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Britain)
	require.EqualValues(t, exterior.X, draft.Record().Position.X)
	require.EqualValues(t, exterior.Y, draft.Record().Position.Y)
	note, err := state.CommitAtlasNote(draft, "Entrance", "Return here.")
	require.NoError(t, err)
	duplicate, err := state.BeginAtlasNoteDraft(0)
	require.NoError(t, err)
	require.False(t, duplicate.Creating())
	require.Equal(t, note, duplicate.Record())
	require.Equal(t, "Settlement entrance on Britannia", duplicate.ContextLabel(), "N must explain an existing exterior note too")
	require.Len(t, state.CaptureAtlasNotes().Records, 1)
	for _, floor := range []references.FloorNumber{-2, 7} {
		state.MapState.PlayerLocation.Floor = floor
		_, err = state.BeginAtlasNoteDraft(0)
		require.Error(t, err)
	}
	state.MapState.PlayerLocation.Location = references.BritanniaUnderworld
	state.MapState.PlayerLocation.Floor = -1
	_, err = state.BeginAtlasNoteDraft(0)
	require.Error(t, err)
}

func TestAtlasNoteNightBoundaryAndForeignSession(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = false
	for _, hour := range []uint8{0, 5, 6, 18, 19, 23} {
		state.DateTime.Hour = hour
		draft, err := state.BeginAtlasNoteDraft(0)
		if hour >= 6 && hour < 19 {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		foreign := *state
		_, err = foreign.CommitAtlasNote(draft, "Foreign", "")
		require.Error(t, err)
		// An in-place checkpoint replacement must not revive an old draft even
		// when position, clock and notes are byte-for-byte identical.
		require.NoError(t, state.RestoreAtlasNotes(state.CaptureAtlasNotes()))
		_, err = state.CommitAtlasNote(draft, "Stale", "")
		require.Error(t, err)
	}
	require.Empty(t, state.CaptureAtlasNotes().Records)
}

func TestAtlasStaticSymbolsRemainDetachedThroughMoonstoneRetirement(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	baseline := state.CurrentAtlasDossier()
	require.Len(t, baseline.Symbols, 27)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
	state.DateTime.Hour = 23
	state.moonstoneJournal[1] = journal.MoonstoneObservation{Known: true, SiteKnown: true, OriginalSiteRetired: true, Site: journal.SiteLocation{X: 12, Y: 34}, Placed: true}
	current := state.CurrentAtlasDossier()
	require.Equal(t, baseline.Symbols, current.Symbols)
	_, ok := atlasEntryByKey(t, current, references.AtlasMoonstoneChartKey(1)).PlotPosition()
	require.False(t, ok)
	require.Empty(t, state.atlasLandmarks)
	require.Empty(t, state.CaptureAtlasNotes().Records)
}
