package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestBeginSmallMapKlimbRoutesWithoutProbeSideEffects(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.NorthBritanny).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	position := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
	callbacks.Reset()

	assert.Equal(t, SmallMapKlimbNeedsDirection, state.BeginSmallMapKlimb())
	assert.Empty(t, callbacks.Messages)

	horse := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, state.MapState.PlayerLocation.Floor)
	state.PartyVehicle = *horse
	assert.Equal(t, SmallMapKlimbRejected, state.BeginSmallMapKlimb())
	assert.Equal(t, []string{"On foot!"}, callbacks.Messages)

	carpet := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, position, state.MapState.PlayerLocation.Floor)
	state.PartyVehicle = *carpet
	callbacks.Reset()
	assert.Equal(t, SmallMapKlimbNeedsDirection, state.BeginSmallMapKlimb())
	assert.Empty(t, callbacks.Messages)
}

func TestActionKlimbSmallMapAcceptsSourceDirectionalTargets(t *testing.T) {
	for _, tile := range []indexes.SpriteIndex{indexes.SmallRockWall, indexes.FenceHoriz, indexes.FenceVert} {
		t.Run(fmt.Sprint(tile), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.NorthBritanny).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()
			target := references.Up.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, target, tile)
			callbacks.Reset()

			require.True(t, state.ActionKlimbSmallMap(references.Up))

			assert.Equal(t, *target, state.MapState.PlayerLocation.Position)
			assert.Equal(t, []string{"Klimb!"}, callbacks.Messages)
			assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
		})
	}
}

func TestBeginSmallMapKlimbCompletesUnderfootTransition(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithFloor(0).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	position := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.LadderUp)

	assert.Equal(t, SmallMapKlimbCompleted, state.BeginSmallMapKlimb())
	assert.True(t, state.EventQueueManager.HasWorkflow("klimb_floor"))
}
