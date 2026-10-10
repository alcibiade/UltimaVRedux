package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestAutomaticDoorCloseWaitsForFinalSurfaceOccupant(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	door := references.Position{X: 11, Y: 10}
	layer := state.GetCurrentLayeredMap()
	layer.SetTileByLayer(map_state.MapLayer, &door, indexes.RegularDoorView)
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Right))
	require.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
	require.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))

	npc := &SimpleTestMapUnit{position: door, floor: state.MapState.PlayerLocation.Floor, visible: true}
	controller := &SimpleTestController{mapUnits: map_units.MapUnits{npc}}
	state.CurrentNPCAIController = controller
	state.SetPlayerPosition(door)

	state.processNonCombatWorldActorPhaseForCommand(false)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetTileTopMapOnlyTile(&door).Index)
	assert.True(t, state.MapState.HasTransientDoor())
	assert.Empty(t, callbacks.SoundEffectsPlayed)

	state.SetPlayerPosition(references.Position{X: 10, Y: 10})
	state.processNonCombatWorldActorPhaseForCommand(false)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetTileTopMapOnlyTile(&door).Index)
	assert.True(t, state.MapState.HasTransientDoor(), "NPC still occupies the doorway")
	assert.Empty(t, callbacks.SoundEffectsPlayed)

	npc.SetPos(references.Position{X: 12, Y: 10})
	state.processNonCombatWorldActorPhaseForCommand(false)
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), layer.GetTileTopMapOnlyTile(&door).Index)
	assert.False(t, state.MapState.HasTransientDoor())
	assert.Equal(t, []SoundEffect{SoundCloseDoor}, callbacks.SoundEffectsPlayed)
}

func TestAutomaticDoorCloseIgnoresActorsOnAnotherFloor(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	position := references.Position{X: 11, Y: 10}
	state.CurrentNPCAIController = &SimpleTestController{mapUnits: map_units.MapUnits{
		&SimpleTestMapUnit{position: position, floor: 1, visible: true},
	}}

	assert.False(t, state.isSmallMapDoorOccupied(position, 0))
	state.MapState.PlayerLocation.Position = position
	assert.True(t, state.isSmallMapDoorOccupied(position, 0))
}
