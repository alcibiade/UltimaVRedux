package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestQuestItemLightsFollowPlacementAndInventory(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(192, 80).
		WithSystemCallbacks().
		Build()
	gs.MapState.PlayerLocation.Floor = -1
	shard := references.Position{X: 192, Y: 80} // Shard of Falsehood

	assert.Contains(t, gs.GetVisibleQuestItemLights(), map_state.QuestItemLightInfo{
		Position: shard, MarkerIndex: map_state.ShardMarkerIndex, LightRadius: map_state.ShardLightDistance,
	})

	gs.PartyState.Inventory.QuestItems.Set(references.ShardOfFalsehood, 1)
	for _, light := range gs.GetVisibleQuestItemLights() {
		assert.NotEqual(t, shard, light.Position, "a carried shard no longer lights its placement")
	}
}
