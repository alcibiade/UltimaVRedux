package game_state

import (
	"encoding/json"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMoonstoneJournalTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	g, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(223, 133).WithSystemCallbacks().Build()
	g.DateTime = datetime.UltimaDate{Year: 139, Month: 4, Day: 12, Hour: 19}
	g.moonstoneSystem = NewMoonstoneSystem()
	g.moonstoneJournal = [8]journal.MoonstoneObservation{}
	for phase := range 8 {
		g.PartyState.Inventory.Moonstones.Set(references.Moonstone(phase), 0)
	}
	g.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	g.SetRandomSeed(661)
	callbacks.Reset()
	return g, callbacks
}

func TestMoonstoneJournalChartIsAlwaysVisibleWithoutCurrentDestination(t *testing.T) {
	g, _ := newMoonstoneJournalTestState(t)
	clock, calls := g.DateTime, g.GetRNG().Calls()
	origins := references.ChartedMoonstoneOrigins()
	for phase, entry := range g.CurrentQuestJournalDossier().Moonstones {
		assert.Equal(t, journal.MoonstoneEntry{Phase: datetime.MoonPhase(phase), ChartedOrigin: origins[phase].Label}, entry)
	}
	// Neither physical exposure, a moved array nor acquiring an instrument is evidence.
	g.moonstoneSystem.MarkMoonstoneFoundAtPosition(0)
	g.moonstoneSystem.BuryMoonstone(1, 211, 213, 0, 0)
	assert.False(t, g.CurrentQuestJournalDossier().Moonstones[0].Known)
	assert.False(t, g.CurrentQuestJournalDossier().Moonstones[1].Known)
	assert.Equal(t, [8]journal.MoonstoneObservation{}, g.moonstoneJournal)
	assert.Equal(t, clock, g.DateTime)
	assert.Equal(t, calls, g.GetRNG().Calls())

	g.PartyState.Inventory.Moonstones.Set(references.FullMoon, 1)
	entry := g.CurrentQuestJournalDossier().Moonstones[4]
	assert.Equal(t, journal.MoonstoneEntry{Phase: datetime.MoonPhaseFull, OriginalSiteRetired: true,
		Known: true, Carried: true, FirstSource: journal.MoonstoneSourcePossession}, entry)
	assert.Equal(t, [8]journal.MoonstoneObservation{}, g.moonstoneJournal, "projection never backfills")
}

func TestMoonstoneJournalLookIsPrivateLocalAndUnambiguous(t *testing.T) {
	g, callbacks := newMoonstoneJournalTestState(t)
	g.moongateStage = MoongateStageFull
	pos := references.Position{X: 224, Y: 133}
	g.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	require.True(t, g.LookAtMoongate(pos))
	assert.Equal(t, []string{"Moongate of New Moon", "A shimmering portal!"}, callbacks.Messages)
	first := g.moonstoneJournal[0]
	assert.Equal(t, journal.MoonstoneSourceLook, first.FirstSource)
	entry := g.CurrentQuestJournalDossier().Moonstones[0]
	assert.True(t, entry.Known && entry.Placed)
	assert.Zero(t, entry.LastObservedSurvey)
	assert.Zero(t, entry.HistoricalSurvey)
	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	for _, private := range []string{"224", "133", "Observer", "LastContext", "SiteKnown", "HasSextant"} {
		assert.NotContains(t, string(encoded), private)
	}

	// Instrument acquisition and projection do not measure an earlier observation.
	g.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	assert.Zero(t, g.CurrentQuestJournalDossier().Moonstones[0].LastObservedSurvey)
	g.DateTime.Minute = 7
	clock, calls := g.DateTime, g.GetRNG().Calls()
	callbacks.Reset()
	require.True(t, g.LookAtMoongate(pos))
	assert.Contains(t, callbacks.Messages, "Moongate of New Moon")
	measured := g.moonstoneJournal[0]
	assert.Equal(t, first.DiscoveredAt, measured.DiscoveredAt)
	assert.Equal(t, first.FirstSource, measured.FirstSource)
	assert.Equal(t, 7, measured.Survey.RecordedAt.Time.Minute)
	assert.Equal(t, g.localMoonstoneSite(pos), measured.Survey.Location)
	assert.NotEqual(t, measured.Site, measured.SurveyContext.Observer, "target is east of observer")
	assert.Equal(t, journal.MoonstoneSourceLook, measured.SurveyContext.Source)
	assert.Equal(t, clock, g.DateTime)
	assert.Equal(t, calls, g.GetRNG().Calls())
	assert.Empty(t, callbacks.TimeAdvanced())

	// Relocation alone does not move remembered evidence. A fresh live Look does.
	g.moonstoneSystem.BuryMoonstone(0, 225, 133, 0, 0)
	assert.Equal(t, measured.Survey, g.CurrentQuestJournalDossier().Moonstones[0].LastObservedSurvey)
	assert.Equal(t, measured, g.moonstoneJournal[0])
	callbacks.Reset()
	require.True(t, g.LookAtMoongate(references.Position{X: 225, Y: 133}))
	assert.Equal(t, []string{"Moongate of New Moon", "A shimmering portal!"}, callbacks.Messages)
	measured = g.moonstoneJournal[0]
	assert.Equal(t, g.localMoonstoneSite(references.Position{X: 225, Y: 133}), measured.Survey.Location)
	assert.True(t, measured.OriginalSiteRetired)

	// Two physical stones at the remembered tile make Look ambiguous.
	g.moonstoneSystem.BuryMoonstone(0, 224, 133, 0, 0)
	g.moonstoneSystem.BuryMoonstone(1, 224, 133, 0, 0)
	callbacks.Reset()
	require.True(t, g.LookAtMoongate(pos))
	assert.Equal(t, []string{"A shimmering portal!"}, callbacks.Messages)
	assert.Equal(t, measured, g.moonstoneJournal[0])
	assert.Zero(t, g.moonstoneJournal[1])
}

func TestMoonstoneJournalLookSourceRequiresFullSite(t *testing.T) {
	g, _ := newMoonstoneJournalTestState(t)
	pos := references.Position{X: 14, Y: 15}
	g.MapState.PlayerLocation.Position = pos
	g.observeMoonstone(2, g.localMoonstoneSite(pos), journal.MoonstoneSourceGate)
	for _, location := range []references.PlayerLocation{
		{Location: references.BritanniaUnderworld, Floor: -1, Position: pos},
		{Location: references.Britain, Floor: 0, Position: pos},
		{Location: references.Britain, Floor: 1, Position: pos},
	} {
		g.MapState.PlayerLocation = location
		_, identified := g.moonstoneSourceAt(pos)
		assert.False(t, identified, "same XY is not the same map/floor")
		g.moonstoneSystem.BuryMoonstone(2, 14, 15, uint8(location.Location), uint8(location.Floor)) //nolint:gosec // Legacy signed floor.
		phase, identified := g.moonstoneSourceAt(pos)
		assert.True(t, identified)
		assert.Equal(t, 2, phase)
		g.observeMoonstone(phase, g.localMoonstoneSite(pos), journal.MoonstoneSourceLook)
		entry := g.CurrentQuestJournalDossier().Moonstones[2]
		assert.Zero(t, entry.LastObservedSurvey)
		assert.Zero(t, entry.HistoricalSurvey)
		assert.Zero(t, g.moonstoneJournal[2].SurveyContext)
	}
}

func TestMoonstoneJournalSearchPickupRelocationAndRemeasurement(t *testing.T) {
	g, callbacks := newMoonstoneJournalTestState(t)
	pos := references.Position{X: 224, Y: 133}
	g.DateTime.Hour = 12
	g.moongateStage = MoongateStageHidden
	callbacks.Reset()
	require.True(t, g.ActionSearchLargeMap(references.Right))
	first := g.moonstoneJournal[0]
	assert.Equal(t, journal.MoonstoneSourceSearch, first.FirstSource)
	assert.True(t, first.Placed)
	assert.Zero(t, first.Survey)
	assert.Equal(t, g.localMoonstoneSite(pos), first.Site)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced(), "only Search's ordinary minute")
	assert.Zero(t, g.GetRNG().Calls())

	g.DateTime.Hour, g.DateTime.Minute = 5, 59
	callbacks.Reset()
	require.True(t, g.ActionGetLargeMap(references.Right))
	carried := g.CurrentQuestJournalDossier().Moonstones[0]
	assert.True(t, carried.Carried)
	assert.False(t, carried.Placed)
	assert.Zero(t, carried.LastObservedSurvey)
	assert.Zero(t, carried.HistoricalSurvey)
	assert.Zero(t, g.moonstoneJournal[0].Survey)
	assert.Zero(t, g.moonstoneJournal[0].SurveyContext)
	assert.Equal(t, 59, carried.ObservedAt.Time.Minute, "sample success before Get's ordinary minute")
	assert.True(t, carried.OriginalSiteRetired)
	assert.Empty(t, carried.ChartedOrigin)
	assert.True(t, g.moonstoneSystem.Moonstones[0].IsInInventory())
	assert.False(t, g.ItemStacksMap.HasItemStackAtPosition(&pos))
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	assert.Zero(t, g.GetRNG().Calls())

	// Rebury in daylight at a new private site; no historical coordinates survive.
	g.DateTime.Hour, g.DateTime.Minute = 12, 10
	newPos := references.Position{X: 225, Y: 134}
	g.MapState.PlayerLocation.Position = newPos
	g.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapLayer, &newPos, indexes.Grass)
	clock, calls := g.DateTime, g.GetRNG().Calls()
	require.True(t, g.CommitMoonstoneBurial(references.NewMoon))
	placed := g.CurrentQuestJournalDossier().Moonstones[0]
	assert.True(t, placed.Placed)
	assert.False(t, placed.Carried)
	assert.Zero(t, placed.LastObservedSurvey)
	assert.Equal(t, carried.HistoricalSurvey, placed.HistoricalSurvey)
	assert.Equal(t, first.DiscoveredAt, placed.DiscoveredAt)
	assert.Equal(t, first.FirstSource, placed.FirstSource)
	assert.Empty(t, placed.ChartedOrigin)
	assert.True(t, placed.OriginalSiteRetired)
	assert.Equal(t, clock, g.DateTime)
	assert.Equal(t, calls, g.GetRNG().Calls())
	assert.Equal(t, g.localMoonstoneSite(newPos), g.moonstoneJournal[0].Site)
	assert.False(t, g.moonstoneSystem.Moonstones[0].IsFound)
	assert.Zero(t, g.PartyState.Inventory.Moonstones.Get(references.NewMoon))

	g.DateTime.Hour = 19
	g.moongateStage = MoongateStageFull
	require.True(t, g.LookAtMoongate(newPos))
	measured := g.CurrentQuestJournalDossier().Moonstones[0]
	assert.Zero(t, measured.HistoricalSurvey)
	assert.Equal(t, g.localMoonstoneSite(newPos), measured.LastObservedSurvey.Location)
	assert.Equal(t, first.DiscoveredAt, measured.DiscoveredAt)

	// Ineligible confirmation updates the observation only, retaining the same-site survey.
	g.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	g.DateTime.Minute = 23
	require.True(t, g.LookAtMoongate(newPos))
	confirmed := g.CurrentQuestJournalDossier().Moonstones[0]
	assert.Equal(t, measured.LastObservedSurvey, confirmed.LastObservedSurvey)
	assert.NotEqual(t, measured.ObservedAt, confirmed.ObservedAt)
	confirmed.LastObservedSurvey.Location.X = 1
	assert.Equal(t, measured.LastObservedSurvey, g.CurrentQuestJournalDossier().Moonstones[0].LastObservedSurvey, "detached projection")
}

func TestMoonstoneJournalPhysicalFailureAndLegacyExposure(t *testing.T) {
	g, _ := newMoonstoneJournalTestState(t)
	pos := references.Position{X: 224, Y: 133}
	g.moongateStage = MoongateStageFull
	require.False(t, g.ActionSearchLargeMap(references.Right), "active gate blocks exposure")
	assert.Zero(t, g.moonstoneJournal)
	require.False(t, g.CommitMoonstoneBurial(references.NewMoon), "no inventory")
	require.False(t, g.CommitMoonstoneBurial(references.Moonstone(8)))
	assert.Zero(t, g.moonstoneJournal)

	// Exposed legacy item + buried placement is valid, but the complete site must agree.
	item := references.ItemAndQuantity{Item: references.NewMoon, Quantity: 1}
	require.NoError(t, g.ItemStacksMap.PushUnder(&pos, &references.ItemStack{Items: references.ItemStacks{item}}))
	for _, test := range []struct{ mapID, level uint8 }{{uint8(references.Britain), 0}, {0, 255}} {
		g.moonstoneSystem.BuryMoonstone(0, 224, 133, test.mapID, test.level)
		require.False(t, g.ActionGetLargeMap(references.Right))
		assert.True(t, g.ItemStacksMap.HasItemStackAtPosition(&pos))
		assert.Zero(t, g.moonstoneJournal)
		assert.Zero(t, g.PartyState.Inventory.Moonstones.Get(references.NewMoon))
	}
	g.moonstoneSystem.BuryMoonstone(0, 224, 133, 0, 0)
	g.moonstoneSystem.BuryMoonstone(1, 224, 133, 0, 0)
	require.True(t, g.ActionGetLargeMap(references.Right))
	assert.True(t, g.moonstoneSystem.Moonstones[0].IsInInventory())
	assert.True(t, g.moonstoneSystem.Moonstones[1].IsBuried())
	assert.True(t, g.moonstoneJournal[0].Known)
	assert.Zero(t, g.moonstoneJournal[1], "only the handled phase is learned")
	assert.Equal(t, journal.MoonstoneSourcePickup, g.moonstoneJournal[0].FirstSource)
	assert.Zero(t, g.CurrentQuestJournalDossier().Moonstones[0].HistoricalSurvey)
	assert.True(t, g.CurrentQuestJournalDossier().Moonstones[0].OriginalSiteRetired)

	// A stale physical item cannot duplicate the carried stone or refresh knowledge.
	before := g.moonstoneJournal
	require.False(t, g.getMoonstoneFromItemStack(&pos, references.NewMoon, &item))
	assert.Equal(t, before, g.moonstoneJournal)
	assert.Equal(t, uint16(1), g.PartyState.Inventory.Moonstones.Get(references.NewMoon))

	// Invalid terrain and dungeon contexts do not consume a carried stone.
	player := g.MapState.PlayerLocation.Position
	g.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapLayer, &player, indexes.StoneBrickWall)
	require.False(t, g.CommitMoonstoneBurial(references.NewMoon))
	g.MapState.PlayerLocation.Location = references.Deceit
	require.False(t, g.CommitMoonstoneBurial(references.NewMoon))
	assert.Equal(t, before, g.moonstoneJournal)
	assert.Equal(t, uint16(1), g.PartyState.Inventory.Moonstones.Get(references.NewMoon))
}

func TestMoonstoneJournalPositionPickupUsesFullSource(t *testing.T) {
	for _, test := range []struct {
		name                 string
		location             references.Location
		floor                references.FloorNumber
		stoneMap, stoneFloor uint8
		want                 bool
	}{
		{"wrong map", references.BritanniaUnderworld, 0, uint8(references.Britain), 0, false},
		{"wrong world", references.BritanniaUnderworld, 0, 0, 255, false},
		{"wrong town floor", references.Britain, 1, uint8(references.Britain), 0, false},
		{"underworld", references.BritanniaUnderworld, -1, 0, 255, true},
		{"town floor", references.Britain, 1, uint8(references.Britain), 1, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			g, _ := newMoonstoneJournalTestState(t)
			pos := references.Position{X: 14, Y: 15}
			g.TransitionToLocation(test.location, test.floor, references.Position{X: 13, Y: 15}, TransitionFullLocation, false)
			g.moonstoneSystem.BuryMoonstone(3, 14, 15, test.stoneMap, test.stoneFloor)
			g.moonstoneSystem.MarkMoonstoneFoundAtPosition(3)
			assert.Equal(t, test.want, g.getMoonstoneFromPosition(&pos))
			if test.want {
				assert.Equal(t, g.localMoonstoneSite(pos), g.moonstoneJournal[3].Site)
				assert.Equal(t, journal.MoonstoneSourcePickup, g.moonstoneJournal[3].FirstSource)
				assert.True(t, g.CurrentQuestJournalDossier().Moonstones[3].Carried)
				assert.Zero(t, g.CurrentQuestJournalDossier().Moonstones[3].HistoricalSurvey)
			} else {
				assert.Zero(t, g.moonstoneJournal)
				assert.True(t, g.moonstoneSystem.Moonstones[3].IsFoundNotCollected())
				assert.Zero(t, g.PartyState.Inventory.Moonstones.Get(references.GibbousWaxing))
			}
		})
	}
}

func TestMoonstoneOriginRetirementSurvivesReturnAndReload(t *testing.T) {
	g, _ := newMoonstoneJournalTestState(t)
	phase := 0
	key := references.AtlasMoonstoneChartKey(phase)
	original := references.ChartedMoonstoneOrigins()[phase]
	pos := original.Location.Position
	site := moonstoneJournalSite(original.Location)
	assertOneRow := func(state *GameState, measured bool) {
		t.Helper()
		dossier := state.CurrentAtlasDossier()
		count := 0
		for _, entry := range dossier.Entries {
			if entry.Category == journal.AtlasMoonstones && entry.Phase == phase {
				count++
				require.Equal(t, key, entry.Key)
				require.False(t, entry.Charted)
				require.Empty(t, entry.OriginReference, "retired supplied origin text stays retired")
				require.Zero(t, entry.HistoricalSurvey)
				require.True(t, entry.Discovered)
				plot, plotted := entry.PlotPosition()
				require.Equal(t, measured, entry.Survey.Measured)
				require.Equal(t, measured, plotted)
				if plotted {
					require.Equal(t, state.moonstoneJournal[phase].Site, plot)
					require.Equal(t, entry.Survey.Location, plot, "only the new current survey authorizes a point")
				} else {
					require.Zero(t, plot, "unmeasured return or relocation discloses no position")
					require.Zero(t, entry.Survey)
				}
			}
		}
		require.Equal(t, 1, count)
		moon := state.journalMoonstones()[phase]
		require.True(t, moon.OriginalSiteRetired)
		require.Empty(t, moon.ChartedOrigin)
		require.Zero(t, moon.HistoricalSurvey)
	}
	g.observeMoonstone(phase, site, journal.MoonstoneSourceSearch)
	require.True(t, g.moonstoneJournal[phase].Survey.Measured)
	require.False(t, g.moonstoneJournal[phase].OriginalSiteRetired)
	g.moonstoneSystem.MarkMoonstoneFoundAtPosition(phase)
	require.True(t, g.getMoonstoneFromPosition(&pos))
	assertOneRow(g, false)
	require.Zero(t, g.moonstoneJournal[phase].SurveyContext)
	g = restoreAtlasSnapshot(t, g)
	assertOneRow(g, false)

	// Rebury at the identical original coordinates, without a measurement.
	g.MapState.PlayerLocation.Position = pos
	g.DateTime.Day++
	g.DateTime.Hour = 12
	g.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapLayer, &pos, indexes.Grass)
	require.True(t, g.CommitMoonstoneBurial(references.NewMoon))
	assertOneRow(g, false)
	g = restoreAtlasSnapshot(t, g)
	assertOneRow(g, false)

	// A new measurement is current authority, not a resurrected chart origin.
	g.DateTime.Hour = 19
	g.observeMoonstone(phase, site, journal.MoonstoneSourceSearch)
	assertOneRow(g, true)
	g = restoreAtlasSnapshot(t, g)
	assertOneRow(g, true)
	g.moonstoneSystem.MarkMoonstoneFoundAtPosition(phase)
	require.True(t, g.getMoonstoneFromPosition(&pos))
	assertOneRow(g, false)

	newPos := references.Position{X: 225, Y: 134}
	g.MapState.PlayerLocation.Position = newPos
	g.DateTime.Day++
	g.DateTime.Hour = 12
	g.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapLayer, &newPos, indexes.Grass)
	require.True(t, g.CommitMoonstoneBurial(references.NewMoon))
	assertOneRow(g, false)
	g = restoreAtlasSnapshot(t, g)
	assertOneRow(g, false)
	g.DateTime.Hour = 19
	g.observeMoonstone(phase, g.localMoonstoneSite(newPos), journal.MoonstoneSourceSearch)
	assertOneRow(g, true)
	entry := atlasEntryByKey(t, g.CurrentAtlasDossier(), key)
	require.Equal(t, g.localMoonstoneSite(newPos), entry.Survey.Location)
	g = restoreAtlasSnapshot(t, g)
	assertOneRow(g, true)
	require.Equal(t, original, references.ChartedMoonstoneOrigins()[phase], "immutable reference is untouched")
}
