package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func newAtlasMoongateLookState(t *testing.T) (*GameState, *MockSystemCallbacks, references.Position) {
	t.Helper()
	state, callbacks := newMoonstoneJournalTestState(t)
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = false
	state.DateTime.Hour = 21
	state.moongateStage = MoongateStageFull
	position := references.Position{X: 224, Y: 133}
	state.GetCurrentLayeredMap().UnSetTileByLayer(map_state.MapUnitLayer, &position)
	return state, callbacks, position
}

func TestActionLookLargeMapMoongateSurveyEligibility(t *testing.T) {
	for _, test := range []struct {
		name         string
		hour, minute uint8
		stage        MoongateStage
		sextant      uint16
		flexible     bool
		underworld   bool
		measured     bool
	}{
		{name: "full active", hour: 21, stage: MoongateStageFull, sextant: 1, measured: true},
		{name: "partly visible", hour: 21, stage: 1, sextant: 1, measured: true},
		{name: "fully closed", hour: 21, stage: MoongateStageHidden, sextant: 1},
		{name: "no sextant", hour: 21, stage: MoongateStageFull},
		{name: "strict dusk", hour: 19, stage: 1, sextant: 1, measured: true},
		{name: "strict dawn boundary", hour: 5, minute: 59, stage: 1, sextant: 1, measured: true},
		{name: "strict day boundary", hour: 6, stage: 1, sextant: 1},
		{name: "strict before dusk", hour: 18, minute: 59, stage: 1, sextant: 1},
		{name: "strict daytime", hour: 12, stage: 1, sextant: 1},
		{name: "flexible daytime", hour: 12, stage: 1, sextant: 1, flexible: true, measured: true},
		{name: "flexible closed", hour: 12, stage: MoongateStageHidden, sextant: 1, flexible: true},
		{name: "flexible no sextant", hour: 12, stage: 1, flexible: true},
		{name: "underworld", hour: 21, stage: MoongateStageFull, sextant: 1, flexible: true, underworld: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks, target := newAtlasMoongateLookState(t)
			state.DateTime.Hour, state.DateTime.Minute = test.hour, test.minute
			state.moongateStage = test.stage
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, test.sextant)
			state.GameConfig.GameplayConfiguration.FlexibleSextantUse = test.flexible
			if test.underworld {
				state.MapState.PlayerLocation.Floor = -1
				state.moonstoneSystem.Moonstones[0].Level = 255
			}
			before := state.CurrentAtlasDossier()
			require.Len(t, before.Symbols, 27, "supplied artwork is not a personal survey")
			for _, entry := range before.Entries {
				require.False(t, entry.Survey.Measured)
			}
			stones := append([]MoonstoneLocation(nil), state.moonstoneSystem.Moonstones...)
			observer, at, calls := state.MapState.PlayerLocation, state.sampleDungeonObservation().at, state.GetRNG().Calls()
			var changes []journal.AtlasChange
			state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) {
				require.True(t, state.moonstoneJournal[0].SiteKnown, "notice follows the committed observation")
				changes = append(changes, change)
			}
			callbacks.Reset() // Calendar setup is not part of the Look command's time cost.

			require.True(t, state.ActionLookLargeMap(references.Right))
			entry := atlasEntryByKey(t, state.CurrentAtlasDossier(), references.AtlasMoonstoneChartKey(0))
			active := test.stage != MoongateStageHidden
			require.Equal(t, active, entry.Discovered)
			require.Equal(t, test.measured, entry.Survey.Measured)
			point, plotted := entry.PlotPosition()
			require.Equal(t, test.measured, plotted)
			if test.measured {
				require.Equal(t, state.localMoonstoneSite(target), point)
				require.NotEqual(t, moonstoneJournalSite(observer), point)
				require.Equal(t, at, entry.Survey.RecordedAt, "sample before Look advances its minute")
				require.Equal(t, test.flexible, entry.Survey.Flexible)
				require.Equal(t, journal.MoonstoneSourceLook.String(), entry.SurveySource)
			} else {
				require.Zero(t, point)
				require.Zero(t, entry.Survey)
			}
			if active {
				require.Equal(t, journal.MoonstoneSourceLook, state.moonstoneJournal[0].FirstSource)
				require.Equal(t, at, state.moonstoneJournal[0].ObservedAt)
				require.Contains(t, callbacks.Messages, "Moongate of New Moon")
				if test.underworld {
					require.Empty(t, changes, "no Britannia map notice for an underground gate")
				} else {
					require.Equal(t, []journal.AtlasChange{{EntryKey: references.AtlasMoonstoneChartKey(0),
						Name: "New Moon", Kind: "moonstone", Discovered: true, Surveyed: test.measured, Measured: test.measured}}, changes)
				}
			} else {
				require.Zero(t, state.moonstoneJournal)
				require.Empty(t, changes)
			}
			require.Equal(t, []int{1}, callbacks.TimeAdvanced())
			require.Equal(t, calls, state.GetRNG().Calls())
			require.Equal(t, observer, state.MapState.PlayerLocation)
			require.Equal(t, stones, state.moonstoneSystem.Moonstones, "Look neither exposes nor moves a stone")
			for phase := range 8 {
				require.Zero(t, state.PartyState.Inventory.Moonstones.Get(references.Moonstone(phase)))
				if phase != 0 {
					require.Zero(t, state.moonstoneJournal[phase])
				}
			}
		})
	}
}

func TestActionLookLargeMapMoongateRequiresUnobscuredPhysicalSource(t *testing.T) {
	for _, scenario := range []string{"map unit layer", "live unit before draw", "item stack", "quest overlay", "closed sprite", "sprite without source", "other floor", "other map", "carried", "ambiguous source", "world seam"} {
		t.Run(scenario, func(t *testing.T) {
			state, callbacks, target := newAtlasMoongateLookState(t)
			layer := state.GetCurrentLayeredMap()
			switch scenario {
			case "map unit layer":
				layer.SetTileByLayer(map_state.MapUnitLayer, &target, indexes.HorseRight)
			case "live unit before draw":
				vehicle := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, target, 0)
				require.NotNil(t, state.CurrentNPCAIController.GetNpcs().InsertVehicle(*vehicle))
			case "item stack":
				require.NoError(t, state.ItemStacksMap.PushUnder(&target, &references.ItemStack{
					Items: references.ItemStacks{{Item: references.NewMoon, Quantity: 1}},
				}))
			case "quest overlay":
				target = references.Position{X: 130, Y: 65}
				state.MapState.PlayerLocation.Floor = -1
				state.MapState.PlayerLocation.Position = target.Left()
				state.moonstoneSystem.BuryMoonstone(0, 130, 65, 0, 255)
				require.True(t, state.IsQuestItemVisible(target, references.ShardOfHatred))
			case "closed sprite":
				state.moongateStage = MoongateStageHidden
				layer.SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.Moongate)
			case "sprite without source":
				state.moonstoneSystem.BuryMoonstone(0, 225, 133, 0, 0)
				layer.SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.Moongate)
			case "other floor":
				state.moonstoneSystem.Moonstones[0].Level = 255
			case "other map":
				state.moonstoneSystem.Moonstones[0].Map = uint8(references.Britain)
			case "carried":
				state.moonstoneSystem.MarkMoonstoneFoundAtPosition(0)
				state.moonstoneSystem.TakeMoonstoneToInventory(0)
				state.PartyState.Inventory.Moonstones.Set(references.NewMoon, 1)
			case "ambiguous source":
				state.moonstoneSystem.BuryMoonstone(1, 224, 133, 0, 0)
			case "world seam":
				state.MapState.PlayerLocation.Position = references.Position{X: 255, Y: 133}
				state.moonstoneSystem.BuryMoonstone(0, 0, 133, 0, 0)
			}
			state.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("unobserved gate notice") }
			require.True(t, state.ActionLookLargeMap(references.Right))
			require.Zero(t, state.moonstoneJournal)
			if scenario != "ambiguous source" {
				require.NotContains(t, callbacks.Messages, "A shimmering portal!")
			}
		})
	}
}

func TestActionLookLargeMapMoongateUsesMovedSourcePhaseAndRetiresOrigin(t *testing.T) {
	for phase := range 8 {
		t.Run(fmt.Sprint(phase), func(t *testing.T) {
			state, _, _ := newAtlasMoongateLookState(t)
			// Select a source different from the travel destination, regardless of
			// which moon supplies that destination on this date.
			for int(state.DateTime.GetMoongateMoonPhase()) == phase {
				state.DateTime.Day++
			}
			target := references.Position{X: 80, Y: 90}
			state.MapState.PlayerLocation.Position = target.Left()
			state.moonstoneSystem.BuryMoonstone(phase, 80, 90, 0, 0)
			before := state.CurrentAtlasDossier()
			require.Zero(t, state.moonstoneJournal, "live physical placement does not backfill readers")
			require.True(t, state.ActionLookLargeMap(references.Right))
			key := references.AtlasMoonstoneChartKey(phase)
			entry := atlasEntryByKey(t, state.CurrentAtlasDossier(), key)
			require.Equal(t, journal.AtlasMoonstones, entry.Category)
			require.Equal(t, phase, entry.Phase)
			require.Equal(t, datetime.MoonPhase(phase).String()+" (last seen)", entry.Name)
			require.True(t, entry.Discovered)
			require.False(t, entry.Charted)
			require.Empty(t, entry.OriginReference)
			require.Equal(t, state.localMoonstoneSite(target), entry.Survey.Location)
			require.Zero(t, entry.HistoricalSurvey)
			require.Len(t, state.CurrentAtlasDossier().Entries, len(before.Entries), "one existing phase row, no gate category")
			for _, other := range before.Entries {
				if other.Key != key {
					require.Equal(t, other, atlasEntryByKey(t, state.CurrentAtlasDossier(), other.Key))
				}
			}

			// A remote return cannot update last seen or restore the origin label.
			origin := references.ChartedMoonstoneOrigins()[phase].Location.Position
			state.moonstoneSystem.BuryMoonstone(phase, uint8(origin.X), uint8(origin.Y), 0, 0)
			require.Equal(t, entry, atlasEntryByKey(t, state.CurrentAtlasDossier(), key))
			state.ProcessWindThroughElapsed(state.ElapsedMs)
			state = restoreAtlasSnapshot(t, state)
			require.Equal(t, entry, atlasEntryByKey(t, state.CurrentAtlasDossier(), key), "load preserves last seen, not remote physical placement")
			state.MapState.PlayerLocation.Position = origin.Left()
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			require.True(t, state.ActionLookLargeMap(references.Right))
			require.Zero(t, atlasEntryByKey(t, state.CurrentAtlasDossier(), key).Survey, "an unmeasured new site retires the previous survey")
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			require.Zero(t, atlasEntryByKey(t, state.CurrentAtlasDossier(), key).Survey, "getting the sextant cannot remeasure that sighting")
			require.True(t, state.ActionLookLargeMap(references.Right))
			returned := atlasEntryByKey(t, state.CurrentAtlasDossier(), key)
			require.Equal(t, state.localMoonstoneSite(origin), returned.Survey.Location)
			require.False(t, returned.Charted)
			require.Empty(t, returned.OriginReference)
			require.True(t, state.moonstoneJournal[phase].OriginalSiteRetired)
			state = restoreAtlasSnapshot(t, state)
			require.Equal(t, returned, atlasEntryByKey(t, state.CurrentAtlasDossier(), key), "return and reload do not resurrect the origin label")
		})
	}
}

func TestActionLookLargeMapMoongateSaveReadersAndRepeatNotice(t *testing.T) {
	state, _, target := newAtlasMoongateLookState(t)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	var changes []journal.AtlasChange
	state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) { changes = append(changes, change) }
	require.True(t, state.ActionLookLargeMap(references.Right))
	require.Len(t, changes, 1)
	first := state.moonstoneJournal[0]
	require.Zero(t, first.Survey)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.ProcessWindThroughElapsed(state.ElapsedMs)
	state = restoreAtlasSnapshot(t, state)
	require.Equal(t, first, state.moonstoneJournal[0], "tool acquisition and hydration cannot backfill a survey")
	state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) { changes = append(changes, change) }

	// Cached visibility is presentation-only; headless adjacent Look must not
	// depend on whether Draw has populated or occluded the target tile.
	state.GetCurrentLayeredMap().SetVisible(false, &target)
	require.True(t, state.ActionLookLargeMap(references.Right))
	require.Len(t, changes, 2)
	require.Equal(t, journal.AtlasChange{EntryKey: references.AtlasMoonstoneChartKey(0), Name: "New Moon",
		Kind: "moonstone", Surveyed: true, Measured: true}, changes[1])
	require.Equal(t, first.DiscoveredAt, state.moonstoneJournal[0].DiscoveredAt)
	state.GetCurrentLayeredMap().SetVisible(true, &target)
	require.True(t, state.ActionLookLargeMap(references.Right))
	require.Len(t, changes, 2, "repeat Look changes only the observation timestamp")
	state.ProcessWindThroughElapsed(state.ElapsedMs)
	state = restoreAtlasSnapshot(t, state)
	state.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("reader emitted a map notice") }
	evidence, clock, calls := state.moonstoneJournal, state.DateTime, state.GetRNG().Calls()
	for range 3 {
		dossier := state.CurrentAtlasDossier()
		entry := atlasEntryByKey(t, dossier, references.AtlasMoonstoneChartKey(0))
		point, plotted := entry.PlotPosition()
		require.True(t, plotted)
		require.Equal(t, state.localMoonstoneSite(target), point)
		state.CurrentQuestJournalDossier()
		state.GetActiveMoongatePositions()
		state.GetMoongateVisibilityLevel()
		state.IsMoongateActiveAt(target)
	}
	require.Equal(t, evidence, state.moonstoneJournal)
	require.Equal(t, clock, state.DateTime)
	require.Equal(t, calls, state.GetRNG().Calls())
}
