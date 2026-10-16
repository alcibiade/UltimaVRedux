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

func TestOutdoorWindowPushQuadrantRetentionAndReentry(t *testing.T) {
	for _, test := range []struct {
		direction  references.Direction
		stepFrom   references.Position
		returnFrom references.Position
		wantOrigin references.Position
		retained   [4]bool
	}{
		{references.Right, references.Position{X: 122, Y: 120}, references.Position{X: 117, Y: 120}, references.Position{X: 112, Y: 112}, [4]bool{false, true, false, true}},
		{references.Left, references.Position{X: 101, Y: 120}, references.Position{X: 106, Y: 120}, references.Position{X: 80, Y: 112}, [4]bool{true, false, true, false}},
		{references.Down, references.Position{X: 120, Y: 138}, references.Position{X: 120, Y: 133}, references.Position{X: 96, Y: 128}, [4]bool{false, false, true, true}},
		{references.Up, references.Position{X: 120, Y: 117}, references.Position{X: 120, Y: 122}, references.Position{X: 96, Y: 96}, [4]bool{true, true, false, false}},
	} {
		t.Run(test.direction.GetDirectionCompassName(), func(t *testing.T) {
			for _, floor := range []references.FloorNumber{0, -1} {
				state := newBlinkHistoryFixture(t, floor)
				original := state.outdoorWindow
				positions := [4]references.Position{{X: 100, Y: 116}, {X: 116, Y: 116}, {X: 100, Y: 132}, {X: 116, Y: 132}}
				for _, target := range positions {
					actor, forward := target.Left(), target.Right()
					state.SetPlayerPosition(actor)
					setLargeMapPushTile(state, actor, indexes.BrickFloor)
					setLargeMapPushTile(state, target, indexes.Barrel)
					setLargeMapPushTile(state, forward, indexes.BrickFloor)
					require.True(t, state.ActionPushLargeMap(references.Right))
					assert.Equal(t, original, state.outdoorWindow, "Push displacement never scrolls")
				}
				state.SetPlayerPosition(test.stepFrom)
				state.MovePlayerOneStep(test.direction)
				assert.Equal(t, test.wantOrigin, state.outdoorWindow.origin)
				for quadrant, target := range positions {
					if test.retained[quadrant] {
						assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), largeMapPushTile(state, target))
						assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(state, target.Right()))
					} else {
						assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(state, target))
						assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), largeMapPushTile(state, target.Right()))
					}
				}
				// Shift back. The discarded half reloads its original arrangement;
				// returning cannot resurrect its earlier Push writes.
				state.SetPlayerPosition(test.returnFrom)
				state.MovePlayerOneStep(test.direction.GetOppositeDirection())
				assert.Equal(t, original.origin, state.outdoorWindow.origin)
				for quadrant, target := range positions {
					if !test.retained[quadrant] {
						assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(state, target))
					}
				}
				assert.Len(t, state.transientLargeMapPushTiles, 4, "two copied quadrants keep both writes")
			}
		})
	}
}

func TestOutdoorWindowSameOriginInitializationExpiresOnlyOwnedPushWrites(t *testing.T) {
	state, _, positions := newLargeMapPushFixture(t, indexes.Barrel, references.Right)
	original := state.outdoorWindow
	require.True(t, state.ActionPushLargeMap(references.Right))
	require.Len(t, state.transientLargeMapPushTiles, 2)
	// A later durable mutation takes ownership of one previously pushed square.
	setLargeMapPushTile(state, positions.forward, indexes.RegularDoor)
	unrelated := references.Position{X: 52, Y: 52}
	setLargeMapPushTile(state, unrelated, indexes.MagicLockDoor)
	otherPlane := state.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, -1)
	otherPlane.SetTileByLayer(map_state.MapLayer, &positions.target, indexes.Plant)
	vehicle := map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, unrelated, 0)
	vehicle.SetPos(unrelated)
	units := state.CurrentNPCAIController.GetNpcs()
	units.ReplaceWith([]map_units.MapUnit{vehicle})
	require.True(t, state.ObjectPresentAt(&unrelated), "the fixture must contain the live vehicle before reinitialization")
	vehicleBefore := *vehicle.MapUnitDetails()
	state.DungeonStates[0] = true
	state.InitializeOutdoorWindow()
	assert.Equal(t, original, state.outdoorWindow, "unchanged origins still perform a fresh load")
	assert.Empty(t, state.transientLargeMapPushTiles)
	assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(state, positions.target))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), largeMapPushTile(state, positions.forward))
	assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoor), largeMapPushTile(state, unrelated))
	assert.Equal(t, indexes.SpriteIndex(indexes.Plant), otherPlane.SpriteIndexByLayer(map_state.MapLayer, positions.target))
	require.Len(t, *units, 1)
	assert.Same(t, vehicle, (*units)[0])
	assert.Equal(t, vehicleBefore, *vehicle.MapUnitDetails())
	assert.True(t, state.ObjectPresentAt(&unrelated))
	assert.True(t, state.DungeonStates[0])
}

func TestOutdoorWindowPushAcrossWorldSeamRetainsWrappedQuadrant(t *testing.T) {
	state, _, positions := newLargeMapPushFixtureAt(t, indexes.Barrel, references.Right, references.Position{X: 255, Y: 120})
	require.True(t, state.ActionPushLargeMap(references.Right))
	assert.Equal(t, references.Position{X: 240, Y: 112}, state.outdoorWindow.origin)
	for range 11 {
		state.MovePlayerOneStep(references.Right)
	}
	assert.Equal(t, references.Position{X: 0, Y: 112}, state.outdoorWindow.origin)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), largeMapPushTile(state, positions.target))
	assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(state, positions.forward))
	state.InitializeOutdoorWindow()
	assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(state, positions.target))
}
