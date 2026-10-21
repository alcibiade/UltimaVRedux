package game_state

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestWordOfPowerSurveyRequiresKnownIdentityAndSuccessfulTarget(t *testing.T) {
	for _, dungeon := range []references.Location{references.Deceit, references.Shame, references.Doom} {
		t.Run(dungeon.String(), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithFloor(dungeonEntranceFloor(dungeon)).WithRandomSeed(42).WithSystemCallbacks().Build()
			entrance := state.GetDungeonEntranceCoordinates()[dungeon]
			state.MapState.PlayerLocation.Position = entrance.Right()
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &entrance, dungeonOpenEntranceSprite(dungeon))
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			state.DateTime.Hour = 19
			word := wordForDungeon(dungeon)
			require.True(t, state.ProcessWordOfPower(word), "blind valid Yell still changes the seal")
			require.Empty(t, state.journalDungeons(), "matching an internal identity does not disclose it")
			if dungeon == references.Shame || dungeon == references.Doom {
				require.True(t, state.DiscoverWordOfPowerKnowledge(strings.ToLower(dungeon.String()), DialogueKnowledgeSource("Speaker")))
				require.True(t, state.ProcessWordOfPower(word))
				entries := state.journalDungeons()
				if dungeon == references.Doom {
					require.Len(t, entries, 1, "the Codex word is only a gate clue")
					require.True(t, entries[0].Gate)
				} else {
					require.Empty(t, entries, "unassociated word still cannot identify Shame")
				}
			}
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			require.True(t, state.ActionLookLargeMap(references.Left))
			siteIndex := journal.DungeonSurfaceSite
			if dungeon == references.Doom {
				siteIndex = journal.DungeonUnderworldSite
			}
			require.False(t, state.journalDungeons()[0].Sites[siteIndex].Survey.Measured)
			require.Empty(t, state.journalDungeons()[0].Word, "observation never attaches a hidden word")
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			state.MapState.PlayerLocation.Position = entrance.Right().Right()
			require.False(t, state.ProcessWordOfPower(word))
			require.False(t, state.journalDungeons()[0].Sites[siteIndex].Survey.Measured, "rejected Yell does not measure")
			state.MapState.PlayerLocation.Position = entrance.Right()
			clock, calls := state.DateTime, state.GetRNG().Calls()
			callbacks.Reset()
			require.True(t, state.ProcessWordOfPower(word))
			entry := state.journalDungeons()[0]
			require.Equal(t, word, entry.Word)
			require.Equal(t, state.IsDungeonOpen(dungeon), entry.Sites[siteIndex].Open)
			require.Equal(t, dungeon != references.Doom, entry.Sites[siteIndex].Survey.Measured)
			if dungeon != references.Doom {
				require.Equal(t, int(entrance.X), entry.Sites[siteIndex].Survey.Location.X)
				require.Equal(t, int(entrance.Y), entry.Sites[siteIndex].Survey.Location.Y)
			}
			require.Equal(t, clock, state.DateTime)
			require.Equal(t, calls, state.GetRNG().Calls())
			require.Empty(t, callbacks.Messages)
		})
	}
}
