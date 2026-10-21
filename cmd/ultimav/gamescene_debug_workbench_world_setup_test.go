package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestExecuteWorldSetupCommandsUseTypedSceneAdapter(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(0).
		WithSystemCallbacks().
		Build()
	scene := &GameScene{gameState: state}

	enter, handled := scene.executeWorldSetupDebugCommand(debugging.Request{
		SessionID: "test", ID: "enter", CommandID: debugging.CommandMovementEnterLocation,
		Arguments: map[string]json.RawMessage{"location": json.RawMessage(`"5"`)},
	})
	require.True(t, handled)
	require.True(t, enter.Success)
	assert.Equal(t, references.Minoc, state.MapState.PlayerLocation.Location)

	floor, handled := scene.executeWorldSetupDebugCommand(debugging.Request{
		SessionID: "test", ID: "floor", CommandID: debugging.CommandMovementFloorSet,
		Arguments: map[string]json.RawMessage{"floor": json.RawMessage(`1`)},
	})
	require.True(t, handled)
	require.True(t, floor.Success)
	assert.Equal(t, references.FloorNumber(1), state.MapState.PlayerLocation.Floor)

	landmark, handled := scene.executeWorldSetupDebugCommand(debugging.Request{
		SessionID: "test", ID: "landmark", CommandID: debugging.CommandMovementLandmarkTravel,
		Arguments: map[string]json.RawMessage{"landmark": json.RawMessage(`"honesty"`)},
	})
	require.True(t, handled)
	require.True(t, landmark.Success)
	assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
	assert.Equal(t, references.Position{X: 233, Y: 66}, state.MapState.PlayerLocation.Position)

	flame, handled := scene.executeWorldSetupDebugCommand(debugging.Request{
		SessionID: "test", ID: "flame", CommandID: debugging.CommandMovementLandmarkTravel,
		Arguments: map[string]json.RawMessage{"landmark": json.RawMessage(`"flame_courage"`)},
	})
	require.True(t, handled)
	require.True(t, flame.Success)
	assert.Equal(t, references.SerpentsHold, state.MapState.PlayerLocation.Location)
	assert.Equal(t, references.FloorNumber(-1), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.Position{X: 15, Y: 16}, state.MapState.PlayerLocation.Position)

	secretRoom, handled := scene.executeWorldSetupDebugCommand(debugging.Request{
		SessionID: "test", ID: "secret-room", CommandID: debugging.CommandMovementLandmarkTravel,
		Arguments: map[string]json.RawMessage{"landmark": json.RawMessage(`"lord_british_secret_room"`)},
	})
	require.True(t, handled)
	require.True(t, secretRoom.Success)
	assert.Equal(t, references.LordBritishsCastle, state.MapState.PlayerLocation.Location)
	assert.Equal(t, references.FloorNumber(2), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.Position{X: 17, Y: 12}, state.MapState.PlayerLocation.Position)
}
