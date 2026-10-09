package game_state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestLookBlueFlamesDiscoverOnceAtAllKeeps(t *testing.T) {
	for flame := references.FlameOfTruth; flame < references.FlameCount; flame++ {
		t.Run(flame.String(), func(t *testing.T) {
			location := references.FlameLocations[flame]
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(location.Location).WithFloor(references.FloorNumber(location.Floor)).
				WithPlayerAt(location.X, location.Y).WithSystemCallbacks().Build()
			position := state.MapState.PlayerLocation.Position.Up()
			currentMap := state.GetCurrentLayeredMap()
			currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
			require.Equal(t, indexes.SpriteIndex(indexes.BlueFlame), currentMap.GetTopTile(&position).Index, "Look target is north of ritual standing position")
			var changes [][2]int
			state.SystemCallbacks.UI.JournalChanged = func(tab, entry int) { changes = append(changes, [2]int{tab, entry}) }
			before := state.DateTime
			require.True(t, state.ActionLookSmallMap(references.Up))
			assert.True(t, callbacks.MessagesContain("Thou dost see the "+flame.GetPrinciple()))
			expected := before
			expected.Advance(1)
			assert.Equal(t, expected, state.DateTime)
			entry, known := state.JournalKnowledge("flame." + strings.ToLower(flame.String()))
			require.True(t, known)
			assert.Equal(t, ObservationKnowledgeSource("Look ("+questJournalKeepName(location.Location)+")"), entry.Source)
			assert.Equal(t, before.DateOnly(), *entry.DiscoveredOn)
			assert.Equal(t, [][2]int{{journal.ChronicleTab, 1}}, changes)
			require.True(t, state.ActionLookSmallMap(references.Up))
			assert.Len(t, changes, 1)
			assert.Len(t, state.QuestJournalEvents(), 1)
			expected.Advance(1)
			assert.Equal(t, expected, state.DateTime)
			row := state.CurrentQuestJournalDossier().Shadowlords[int(flame)]
			assert.Empty(t, row.Flame, "an observed flame cannot reveal the unknown row's association")
		})
	}
}

func TestLookFlameDiscoveryPreservesTopUnitAndContext(t *testing.T) {
	for _, test := range []struct {
		name     string
		location references.Location
		tile     indexes.SpriteIndex
		cover    bool
	}{
		{name: "red brazier", location: references.Lycaeum, tile: indexes.Brazier},
		{name: "blue outside keep", location: references.Britain, tile: indexes.BlueFlame},
		{name: "covered blue flame", location: references.Lycaeum, tile: indexes.BlueFlame, cover: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(test.location).
				WithFloor(0).WithPlayerAt(15, 15).WithSystemCallbacks().Build()
			position := state.MapState.PlayerLocation.Position.Up()
			currentMap := state.GetCurrentLayeredMap()
			currentMap.SetTileByLayer(map_state.MapLayer, &position, test.tile)
			currentMap.UnSetTileByLayer(map_state.MapUnitLayer, &position)
			if test.cover {
				currentMap.SetTileByLayer(map_state.MapUnitLayer, &position, indexes.HorseRight)
			}
			state.SystemCallbacks.UI.JournalChanged = func(int, int) { t.Fatal("non-discovery published a change") }
			before := state.DateTime
			require.True(t, state.ActionLookSmallMap(references.Up))
			assert.Empty(t, state.QuestJournalEvents())
			before.Advance(1)
			assert.Equal(t, before, state.DateTime)
			assert.False(t, callbacks.MessagesContain("Flame of Truth"))
			if test.cover {
				assert.True(t, callbacks.MessagesContain(state.GameReferences.LookReferences.GetTileLookDescription(indexes.HorseRight)))
			}
		})
	}
}
