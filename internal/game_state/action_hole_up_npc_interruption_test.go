package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestBedRestChecksFreshLiveNPCPlacementAfterCycle(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		present bool
		visible bool
		want    bool
	}{
		{name: "live NPC occupies bed", present: true, visible: true, want: true},
		{name: "absent NPC does not occupy bed", want: false},
		{name: "hidden NPC does not occupy bed", present: true, visible: false, want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()
			controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
			require.True(t, ok)
			bed := state.MapState.PlayerLocation.Position
			units := map_units.MapUnits{}
			if testCase.present {
				npcRef := references.NPCReference{Location: references.Britain}
				npcRef.SetKeyIndex(indexes.BardKeyIndex)
				npcRef.Schedule = references.CreateNPCScheduledFixedOneLocation(
					bed,
					state.MapState.PlayerLocation.Floor,
				)
				npc := map_units.NewNPCFriendly(npcRef, 1)
				npc.SetVisible(testCase.visible)
				units = append(units, npc)
			}
			controller.GetNpcs().ReplaceWith(units)
			state.DateTime.Hour, state.DateTime.Minute = 1, 0
			rest := state.NewBedRestState(1)
			state.BeginBedRest(rest)

			result := state.ProcessBedRestCycle(rest)

			assert.Equal(t, testCase.want, result.NPCAtBed)
			assert.Equal(t, byte(1), state.DateTime.Hour)
			assert.Equal(t, byte(10), state.DateTime.Minute,
				"occupancy is first checked after a complete time/player cycle")
		})
	}
}
