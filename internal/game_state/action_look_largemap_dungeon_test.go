package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestActionLookLargeMapNamesExteriorDungeonPortalsAndRecordsJournal(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()
	var popup string
	state.SystemCallbacks.UI.ShowActionFeedback = func(message string, alsoAddToOutput bool) {
		popup = message
		assert.False(t, alsoAddToOutput, "the transcript is added once by Look")
	}

	for _, floor := range []references.FloorNumber{
		references.FloorNumber(references.OVERWORLD),
		references.FloorNumber(references.UNDERWORLD),
	} {
		for _, candidate := range canonicalDungeonWords {
			if !DungeonEntranceExistsOnFloor(candidate.dungeon, floor) {
				continue
			}
			for _, open := range []bool{false, true} {
				name := fmt.Sprintf("%s floor %d open %t", candidate.dungeon, floor, open)
				t.Run(name, func(t *testing.T) {
					position := state.GetDungeonEntranceCoordinates()[candidate.dungeon]
					state.MapState.PlayerLocation.Floor = floor
					state.MapState.PlayerLocation.Position = position.Down()
					state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, candidate.open)
					state.GetCurrentLayeredMap().UnSetTileByLayer(map_state.MapUnitLayer, &position)
					state.SetDungeonState(candidate.dungeon, open)
					callbacks.Reset()
					popup = ""
					want := "Thou dost see the collapsed entrance to the dungeon " + candidate.dungeon.String()
					if open {
						want = "Thou dost see the entrance to the dungeon " + candidate.dungeon.String()
					}

					require.True(t, state.ActionLookLargeMap(references.Up))
					assert.Equal(t, want, popup)
					assert.Equal(t, []string{"Look-" + want}, callbacks.Messages)
					assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
					journalIndex, _ := dungeonIndex(candidate.dungeon)
					observation := state.dungeonJournal[journalIndex]
					assert.True(t, observation.Known)
					siteIndex := journal.DungeonSurfaceSite
					if floor == references.FloorNumber(references.UNDERWORLD) {
						siteIndex = journal.DungeonUnderworldSite
					}
					site := observation.Sites[siteIndex]
					assert.True(t, site.Seen)
					assert.Equal(t, int(position.X), site.Location.X)
					assert.Equal(t, int(position.Y), site.Location.Y)
					assert.Equal(t, floor, site.Location.Floor)
					assert.Equal(t, open, site.Open)
					assert.False(t, observation.WordUsed, "looking never reveals the word of power")
					entryFound := false
					for _, entry := range state.journalDungeons() {
						if entry.Name == candidate.dungeon.String() {
							entryFound = true
							assert.Equal(t, open, entry.Sites[siteIndex].Open)
							assert.Empty(t, entry.Word, "looking never reveals the word of power")
							assert.NotEmpty(t, entry.Location)
						}
					}
					assert.True(t, entryFound, "observing a portal must make its dungeon visible in the journal")
				})
			}
		}
	}
}

func TestDungeonEntranceAtNormalizesLargeMapCoordinatesAndRejectsOtherTiles(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.UNDERWORLD)).
		WithSystemCallbacks().
		Build()

	for _, candidate := range canonicalDungeonWords {
		entrance := state.GetDungeonEntranceCoordinates()[candidate.dungeon]
		for _, position := range []references.Position{
			entrance,
			{X: entrance.X + references.XLargeMapTiles, Y: entrance.Y},
			{X: entrance.X, Y: entrance.Y - references.YLargeMapTiles},
		} {
			for _, tile := range []indexes.SpriteIndex{indexes.BlockEntrance, candidate.open} {
				dungeon, found := state.dungeonEntranceAt(position, tile)
				assert.True(t, found, "%s at %+v tile %d", candidate.dungeon, position, tile)
				assert.Equal(t, candidate.dungeon, dungeon)
			}
			_, found := state.dungeonEntranceAt(position, indexes.Grass)
			assert.False(t, found, "the coordinate alone cannot identify a concealed or replaced entrance")
		}
	}
}

func TestActionLookLargeMapPreservesNonPortalDescriptions(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.OVERWORLD)).
		WithSystemCallbacks().
		Build()
	currentMap := state.GetCurrentLayeredMap()

	t.Run("open entrance with replaced terrain", func(t *testing.T) {
		position := state.GetDungeonEntranceCoordinates()[references.Deceit]
		state.MapState.PlayerLocation.Position = position.Down()
		currentMap.SetTileByLayer(map_state.MapLayer, &position, indexes.Hut)
		currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
		state.SetDungeonState(references.Deceit, true)
		callbacks.Reset()

		require.True(t, state.ActionLookLargeMap(references.Up))
		assert.Equal(t, []string{
			"Look-Thou dost see " + state.GameReferences.LookReferences.GetTileLookDescription(indexes.Hut),
		}, callbacks.Messages)
		assert.False(t, state.dungeonJournal[DeceitIndex].Known)
	})

	t.Run("different open entrance family at canonical coordinate", func(t *testing.T) {
		position := state.GetDungeonEntranceCoordinates()[references.Deceit]
		state.MapState.PlayerLocation.Position = position.Down()
		currentMap.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.CaveEntrance)
		currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
		state.SetDungeonState(references.Deceit, true)
		callbacks.Reset()

		require.True(t, state.ActionLookLargeMap(references.Up))
		assert.Equal(t, []string{
			"Look-Thou dost see " + state.GameReferences.LookReferences.GetTileLookDescription(indexes.CaveEntrance),
		}, callbacks.Messages)
		assert.False(t, state.dungeonJournal[DeceitIndex].Known)
		currentMap.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	})

	t.Run("open portal tile away from a canonical coordinate", func(t *testing.T) {
		position := references.Position{X: 10, Y: 10}
		state.MapState.PlayerLocation.Position = position.Down()
		currentMap.SetTileByLayer(map_state.MapLayer, &position, indexes.DoomEntrance)
		currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
		callbacks.Reset()

		require.True(t, state.ActionLookLargeMap(references.Up))
		assert.Equal(t, []string{
			"Look-Thou dost see " + state.GameReferences.LookReferences.GetTileLookDescription(indexes.DoomEntrance),
		}, callbacks.Messages)
		assert.False(t, state.dungeonJournal[DeceitIndex].Known)
	})

	t.Run("unrelated blocked entrance", func(t *testing.T) {
		position := references.Position{X: 10, Y: 10}
		state.MapState.PlayerLocation.Position = position.Down()
		currentMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BlockEntrance)
		currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
		callbacks.Reset()

		require.True(t, state.ActionLookLargeMap(references.Up))
		assert.Equal(t, []string{
			"Look-Thou dost see " + state.GameReferences.LookReferences.GetTileLookDescription(indexes.BlockEntrance),
		}, callbacks.Messages)
		assert.False(t, state.dungeonJournal[DeceitIndex].Known)
	})

	t.Run("Doom coordinate on Britannia", func(t *testing.T) {
		position := state.GetDungeonEntranceCoordinates()[references.Doom]
		state.MapState.PlayerLocation.Position = position.Down()
		currentMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BlockEntrance)
		currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
		callbacks.Reset()

		require.True(t, state.ActionLookLargeMap(references.Up))
		assert.Equal(t, []string{
			"Look-Thou dost see " + state.GameReferences.LookReferences.GetTileLookDescription(indexes.BlockEntrance),
		}, callbacks.Messages)
		assert.False(t, state.dungeonJournal[DoomIndex].Known)
	})

	t.Run("open Doom tile on Britannia", func(t *testing.T) {
		position := state.GetDungeonEntranceCoordinates()[references.Doom]
		state.MapState.PlayerLocation.Position = position.Down()
		currentMap.SetTileByLayer(map_state.MapLayer, &position, indexes.CaveEntrance)
		currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
		state.SetDungeonState(references.Doom, true)
		callbacks.Reset()

		require.True(t, state.ActionLookLargeMap(references.Up))
		assert.Equal(t, []string{
			"Look-Thou dost see " + state.GameReferences.LookReferences.GetTileLookDescription(indexes.CaveEntrance),
		}, callbacks.Messages)
		assert.False(t, state.dungeonJournal[DoomIndex].Known)
	})
}

func TestActionLookLargeMapPreservesMapUnitPrecedenceAtDungeonEntrance(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.OVERWORLD)).
		WithSystemCallbacks().
		Build()
	position := state.GetDungeonEntranceCoordinates()[references.Deceit]
	state.MapState.PlayerLocation.Position = position.Down()
	currentMap := state.GetCurrentLayeredMap()
	currentMap.SetTileByLayer(map_state.MapLayer, &position, indexes.DoomEntrance)
	currentMap.SetTileByLayer(map_state.MapUnitLayer, &position, indexes.HorseRight)
	for _, open := range []bool{false, true} {
		state.SetDungeonState(references.Deceit, open)
		callbacks.Reset()

		require.True(t, state.ActionLookLargeMap(references.Up))
		assert.Equal(t, []string{
			"Look-Thou dost see " + state.GameReferences.LookReferences.GetTileLookDescription(indexes.HorseRight),
		}, callbacks.Messages)
		assert.False(t, state.dungeonJournal[DeceitIndex].Known)
	}
}

func TestActionLookLargeMapWrappedPortalRecordsCanonicalPosition(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.UNDERWORLD)).
		WithSystemCallbacks().
		Build()
	position := state.GetDungeonEntranceCoordinates()[references.Deceit]
	state.MapState.PlayerLocation.Position = references.Position{X: position.X + references.XLargeMapTiles, Y: position.Y + 1}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.DoomEntrance)
	state.SetDungeonState(references.Deceit, true)
	callbacks.Reset()

	require.True(t, state.ActionLookLargeMap(references.Up))
	assert.Equal(t, []string{"Look-Thou dost see the entrance to the dungeon Deceit"}, callbacks.Messages)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	observed := state.dungeonJournal[DeceitIndex].Sites[journal.DungeonUnderworldSite]
	assert.True(t, observed.Seen)
	assert.Equal(t, int(position.X), observed.Location.X)
	assert.Equal(t, int(position.Y), observed.Location.Y)
	assert.Equal(t, references.FloorNumber(references.UNDERWORLD), observed.Location.Floor)
	assert.Equal(t, journal.SurveyedLocation{}, observed.Survey)
}

func TestActionLookDungeonSurveyUsesWrappedTargetBeforeDawn(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithFloor(0).WithRandomSeed(42).WithSystemCallbacks().Build()
	position := state.GetDungeonEntranceCoordinates()[references.Deceit]
	state.MapState.PlayerLocation.Position = references.Position{X: position.X + references.XLargeMapTiles, Y: position.Y + 1}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.DoomEntrance)
	state.SetDungeonState(references.Deceit, true)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	state.DateTime.Hour, state.DateTime.Minute = 5, 58
	require.True(t, state.ActionLookLargeMap(references.Up))
	require.False(t, state.journalDungeons()[0].Sites[0].Survey.Measured)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	callbacks.Reset()
	calls := state.GetRNG().Calls()
	require.True(t, state.ActionLookLargeMap(references.Up))
	survey := state.journalDungeons()[0].Sites[0].Survey
	require.True(t, survey.Measured)
	require.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, X: int(position.X), Y: int(position.Y)}, survey.Location)
	require.Equal(t, 5, survey.RecordedAt.Time.Hour)
	require.Equal(t, 59, survey.RecordedAt.Time.Minute)
	require.Equal(t, uint8(6), state.DateTime.Hour)
	require.Equal(t, SextantDaylight, state.CurrentSextantEligibility())
	require.Equal(t, []int{1}, callbacks.TimeAdvanced())
	require.Len(t, callbacks.Messages, 1, "survey adds no extra feedback")
	require.Equal(t, calls, state.GetRNG().Calls())
	require.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sextant))
}
