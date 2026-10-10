package main

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestWorldPositionStatusReturnsEveryDestinationFloorAndDefault(t *testing.T) {
	scene := newDungeonExplorerDebugTestScene(t)

	result := scene.ExecuteDebugCommand(debugging.Request{CommandID: debugging.CommandWorldPositionStatus})

	require.True(t, result.Success, result.ErrorMessage)
	var status debugging.WorldPositionStatusData
	require.NoError(t, json.Unmarshal(result.Data, &status))
	assert.Equal(t, int(references.Britain), status.LocationID)
	assert.Equal(t, 0, status.Floor)
	assert.Equal(t, 10, status.X)
	assert.Equal(t, 11, status.Y)
	assert.Len(t, status.SourceFingerprint, sha256HexLength)
	assert.Equal(t, debugging.WorldPositionBoundsData{MinimumX: 0, MaximumX: 31, MinimumY: 0, MaximumY: 31}, status.Bounds)
	require.Len(t, status.Destinations, int(references.Doom)+1)

	for location := references.BritanniaUnderworld; location <= references.Doom; location++ {
		destination := status.Destinations[int(location)]
		assert.Equal(t, int(location), destination.LocationID)
		assert.Equal(t, debugMapTypeName(location.GetMapType()), destination.MapType)
		assert.Contains(t, destination.Floors, destination.DefaultSetup.Floor)
		assert.GreaterOrEqual(t, destination.DefaultSetup.X, destination.Bounds.MinimumX)
		assert.LessOrEqual(t, destination.DefaultSetup.X, destination.Bounds.MaximumX)
		assert.GreaterOrEqual(t, destination.DefaultSetup.Y, destination.Bounds.MinimumY)
		assert.LessOrEqual(t, destination.DefaultSetup.Y, destination.Bounds.MaximumY)
		if location == references.BritanniaUnderworld {
			assert.Nil(t, destination.WorldEntrance)
		} else if location.GetMapType() != references.DungeonMapType {
			require.NotNil(t, destination.WorldEntrance)
			worldPosition, worldFloor := scene.gameState.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(location)
			assert.Equal(t, int(worldFloor), destination.WorldEntrance.Floor)
			assert.Equal(t, int(worldPosition.X), destination.WorldEntrance.X)
			assert.Equal(t, int(worldPosition.Y), destination.WorldEntrance.Y)
		} else {
			assert.Nil(t, destination.WorldEntrance)
			entrance := scene.gameState.GetDungeonEntranceCoordinates()[location]
			underworldOutside := worldPlacementTarget(
				t,
				destination,
				debugging.WorldPlacementDungeonOutside,
				debugging.WorldRouteUnderworld,
			)
			require.True(t, underworldOutside.Available)
			require.NotNil(t, underworldOutside.Setup)
			assert.Equal(t, int(references.BritanniaUnderworld), underworldOutside.LocationID)
			assert.Equal(t, -1, underworldOutside.Setup.Floor)
			assert.Equal(t, int(entrance.X), underworldOutside.Setup.X)
			assert.Equal(t, int(entrance.Y), underworldOutside.Setup.Y)

			underworldStart := worldPlacementTarget(
				t,
				destination,
				debugging.WorldPlacementDungeonInsideStart,
				debugging.WorldRouteUnderworld,
			)
			require.True(t, underworldStart.Available)
			require.NotNil(t, underworldStart.Setup)
			if location == references.Doom {
				assert.Len(t, destination.PlacementTargets, 3)
				assert.Equal(t, debugging.WorldPositionSetupData{Floor: 0, X: 1, Y: 1}, *underworldStart.Setup)
				assert.Equal(t, "East", underworldStart.FacingLabel)
				assert.Equal(t, "Arrived from above", underworldStart.LastDirectionLabel)
				exitTarget := worldPlacementTarget(
					t,
					destination,
					debugging.WorldPlacementDungeonUnderworldExit,
					debugging.WorldRouteUnderworld,
				)
				assert.False(t, exitTarget.Available)
				assert.Nil(t, exitTarget.Setup)
				assert.Contains(t, exitTarget.UnavailableReason, "no down-ladder Underworld endpoint")
				assert.Empty(t, worldPlacementTargets(destination, debugging.WorldRouteBritannia))
			} else {
				assert.Len(t, destination.PlacementTargets, 5)
				britanniaOutside := worldPlacementTarget(
					t,
					destination,
					debugging.WorldPlacementDungeonOutside,
					debugging.WorldRouteBritannia,
				)
				require.NotNil(t, britanniaOutside.Setup)
				assert.Equal(t, debugging.WorldPositionSetupData{
					Floor: 0,
					X:     int(entrance.X),
					Y:     int(entrance.Y),
				}, *britanniaOutside.Setup)
				britanniaStart := worldPlacementTarget(
					t,
					destination,
					debugging.WorldPlacementDungeonInsideStart,
					debugging.WorldRouteBritannia,
				)
				require.NotNil(t, britanniaStart.Setup)
				assert.Equal(t, debugging.WorldPositionSetupData{Floor: 0, X: 1, Y: 1}, *britanniaStart.Setup)
				assert.Equal(t, "East", britanniaStart.FacingLabel)
				assert.Equal(t, "Arrived from above", britanniaStart.LastDirectionLabel)
				assert.Equal(t, debugging.WorldPositionSetupData{Floor: 7, X: 7, Y: 7}, *underworldStart.Setup)
				assert.Equal(t, "West", underworldStart.FacingLabel)
				assert.Equal(t, "Arrived from below", underworldStart.LastDirectionLabel)
				exitTarget := worldPlacementTarget(
					t,
					destination,
					debugging.WorldPlacementDungeonUnderworldExit,
					debugging.WorldRouteUnderworld,
				)
				require.True(t, exitTarget.Available)
				require.NotNil(t, exitTarget.Setup)
				assert.Equal(t, debugging.WorldPositionSetupData{Floor: 7, X: 7, Y: 7}, *exitTarget.Setup)
				assert.Equal(t, "West", exitTarget.FacingLabel)
				assert.Equal(t, "West", exitTarget.LastDirectionLabel)
			}
		}

		switch location.GetMapType() {
		case references.LargeMapType:
			assert.Equal(t, []int{-1, 0}, destination.Floors)
			assert.Equal(t, 255, destination.Bounds.MaximumX)
		case references.DungeonMapType:
			assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7}, destination.Floors)
			assert.Equal(t, 7, destination.Bounds.MaximumX)
		case references.SmallMapType:
			locationReference := scene.gameState.GameReferences.LocationReferences.GetLocationReference(location)
			require.NotNil(t, locationReference)
			expectedFloors := make([]int, len(locationReference.ListOfFloors))
			for index, floor := range locationReference.ListOfFloors {
				expectedFloors[index] = int(floor)
			}
			assert.Equal(t, expectedFloors, destination.Floors)
			assert.Equal(t, 31, destination.Bounds.MaximumX)
		case references.CombatMapType, references.CampingMapType:
			t.Fatalf("unexpected destination map type %d", location.GetMapType())
		}
	}
}

func TestDungeonExplorerStatusReturnsDetachedYMajorSemanticCells(t *testing.T) {
	scene := newDungeonExplorerDebugTestScene(t)
	state := scene.gameState
	player := references.Position{X: 3, Y: 4}
	_, err := state.DebugSetupDungeon(
		references.Deceit,
		2,
		player,
		references.DungeonFacingSouth,
		true,
	)
	require.NoError(t, err)
	semanticPosition := references.Position{X: 2, Y: 3}
	_, err = state.TransformDungeonCell(2, semanticPosition, func(references.DungeonCell) references.DungeonCell {
		return references.DungeonCell(0xd8)
	})
	require.NoError(t, err)
	projection := state.MapState.LayeredMaps.GetLayeredMap(references.DungeonMapType, 2)
	require.NotNil(t, projection)
	projection.SetTileByLayer(map_state.MapLayer, &semanticPosition, indexes.BrickFloor)

	result := scene.ExecuteDebugCommand(debugging.Request{CommandID: debugging.CommandDungeonExplorerStatus})

	require.True(t, result.Success, result.ErrorMessage)
	var status debugging.DungeonExplorerStatusData
	require.NoError(t, json.Unmarshal(result.Data, &status))
	assert.True(t, status.Active)
	assert.True(t, status.SetupAvailable)
	assert.Equal(t, int(references.Deceit), status.LocationID)
	assert.Equal(t, 2, status.Floor)
	assert.Equal(t, 3, status.DisplayLevel)
	assert.Equal(t, int(references.DungeonFacingSouth), status.FacingID)
	assert.Equal(t, "South", status.FacingLabel)
	assert.Equal(t, int(references.DungeonLastSouth), status.LastDirectionID)
	assert.Equal(t, "South", status.LastDirectionLabel)
	assert.True(t, status.Lit)
	assert.Zero(t, status.TorchMinutesRemaining)
	assert.Equal(t, game_state.VasLorLightDuration, status.MagicLightMinutesRemaining)
	assert.Equal(t, uint64(1), status.Revision)
	assert.Len(t, status.SourceFingerprint, sha256HexLength)
	require.Len(t, status.Cells, 64)
	playerCells := 0
	for index, cell := range status.Cells {
		assert.Equal(t, index%8, cell.X)
		assert.Equal(t, index/8, cell.Y)
		if cell.Player {
			playerCells++
		}
	}
	assert.Equal(t, 1, playerCells)
	assert.True(t, status.Cells[int(player.Y)*8+int(player.X)].Player)
	semanticCell := status.Cells[int(semanticPosition.Y)*8+int(semanticPosition.X)]
	assert.Equal(t, 0xd8, semanticCell.Raw)
	assert.Equal(t, int(references.DungeonHiddenDoor), semanticCell.KindID)
	assert.Equal(t, "hidden_door", semanticCell.KindName)
	assert.Equal(t, 8, semanticCell.Payload)
	assert.True(t, semanticCell.Bit08)
}

func TestWorldPositionSetPreservesSwitchesAndClearsDungeonAuthority(t *testing.T) {
	scene := newDungeonExplorerDebugTestScene(t)
	state := scene.gameState
	_, err := state.DebugSetupDungeon(
		references.Deceit,
		0,
		references.Position{X: 1, Y: 1},
		references.DungeonFacingWest,
		false,
	)
	require.NoError(t, err)
	_, err = state.TransformDungeonCell(5, references.Position{X: 6, Y: 7}, func(references.DungeonCell) references.DungeonCell {
		return references.DungeonCell(0xc8)
	})
	require.NoError(t, err)
	beforePreserved, active := state.DungeonStateSnapshot()
	require.True(t, active)

	preserved := scene.ExecuteDebugCommand(worldPositionSetDebugRequest(
		state,
		references.Deceit,
		5,
		references.Position{X: 6, Y: 7},
	))

	require.True(t, preserved.Success, preserved.ErrorMessage)
	afterPreserved, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, beforePreserved, afterPreserved)
	assert.Equal(t, references.PlayerLocation{
		Location: references.Deceit,
		Floor:    5,
		Position: references.Position{X: 6, Y: 7},
	}, state.MapState.PlayerLocation)
	assert.Equal(t, []string{"world.floor", "world.position.x", "world.position.y"}, stateChangePaths(preserved.Changes))
	noOpPosition := scene.ExecuteDebugCommand(worldPositionSetDebugRequest(
		state,
		references.Deceit,
		5,
		references.Position{X: 6, Y: 7},
	))
	require.True(t, noOpPosition.Success, noOpPosition.ErrorMessage)
	assert.Empty(t, noOpPosition.Changes)
	state.MapState.Lighting.RestoreTorchMinutes(17)
	state.MagicLightMinutesRemaining = 23
	state.MapState.Lighting.SetMagicLight(true)

	switched := scene.ExecuteDebugCommand(worldPositionSetDebugRequest(
		state,
		references.Doom,
		2,
		references.Position{X: 2, Y: 3},
	))

	require.True(t, switched.Success, switched.ErrorMessage)
	switchSnapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonFacingEast, switchSnapshot.Facing)
	assert.Equal(t, references.DungeonLastEast, switchSnapshot.LastDirection)
	assert.Zero(t, switchSnapshot.Revision)
	assert.True(t, state.MapState.Lighting.HasAnyLight())
	assert.Equal(t, 17, state.MapState.Lighting.TorchMinutesRemaining())
	assert.Equal(t, 23, state.MagicLightMinutesRemaining)
	assert.Contains(t, stateChangePaths(switched.Changes), "world.locationId")
	assert.Contains(t, stateChangePaths(switched.Changes), "dungeon.facing")
	assert.Contains(t, stateChangePaths(switched.Changes), "dungeon.lastDirection")
	assert.Contains(t, stateChangePaths(switched.Changes), "dungeon.revision")
	assert.NotContains(t, stateChangePaths(switched.Changes), "lighting.lit")
	assert.NotContains(t, stateChangePaths(switched.Changes), "lighting.torchMinutesRemaining")
	assert.NotContains(t, stateChangePaths(switched.Changes), "lighting.magicLightMinutesRemaining")

	left := scene.ExecuteDebugCommand(worldPositionSetDebugRequest(
		state,
		references.Britain,
		1,
		references.Position{X: 15, Y: 30},
	))

	require.True(t, left.Success, left.ErrorMessage)
	_, active = state.DungeonStateSnapshot()
	assert.False(t, active)
	assert.Equal(t, references.Britain, state.MapState.PlayerLocation.Location)
	assert.Contains(t, stateChangePaths(left.Changes), "dungeon.active")
}

func TestWorldPositionSetResolvesDungeonRouteIntentsAndRestartsAuthority(t *testing.T) {
	scene := newDungeonExplorerDebugTestScene(t)
	state := scene.gameState
	state.MapState.Lighting.RestoreTorchMinutes(17)
	state.MagicLightMinutesRemaining = 23
	state.MapState.Lighting.SetMagicLight(true)
	beforeDate := state.DateTime
	beforeRNGCalls := state.GetRNG().Calls()
	entrance := state.GetDungeonEntranceCoordinates()[references.Deceit]

	outside := scene.ExecuteDebugCommand(worldRoutePositionSetDebugRequest(
		state,
		references.Deceit,
		debugging.WorldPlacementDungeonOutside,
		debugging.WorldRouteUnderworld,
	))

	require.True(t, outside.Success, outside.ErrorMessage)
	assert.Equal(t, references.PlayerLocation{
		Location: references.BritanniaUnderworld,
		Floor:    -1,
		Position: entrance,
	}, state.MapState.PlayerLocation)
	assert.False(t, state.IsDungeonOpen(references.Deceit))
	assert.False(t, state.ActionEnterLargeMap(), "outside placement must not bypass the normal sealed-entrance gate")
	_, active := state.DungeonStateSnapshot()
	assert.False(t, active)
	assert.Contains(t, outside.Summary, "floor -1")
	assert.Contains(t, outside.Summary, "via Underworld")
	var outsideStatus debugging.WorldPositionStatusData
	require.NoError(t, json.Unmarshal(outside.Data, &outsideStatus))
	assert.Equal(t, int(references.BritanniaUnderworld), outsideStatus.LocationID)
	assert.Equal(t, -1, outsideStatus.Floor)
	assert.Equal(t, int(entrance.X), outsideStatus.X)
	assert.Equal(t, int(entrance.Y), outsideStatus.Y)
	assert.Equal(t, []string{"world.locationId", "world.floor", "world.position.x", "world.position.y"}, stateChangePaths(outside.Changes))

	inside := scene.ExecuteDebugCommand(worldRoutePositionSetDebugRequest(
		state,
		references.Deceit,
		debugging.WorldPlacementDungeonInsideStart,
		debugging.WorldRouteUnderworld,
	))

	require.True(t, inside.Success, inside.ErrorMessage)
	assert.Equal(t, references.PlayerLocation{
		Location: references.Deceit,
		Floor:    7,
		Position: references.Position{X: 7, Y: 7},
	}, state.MapState.PlayerLocation)
	insideSnapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonFacingWest, insideSnapshot.Facing)
	assert.Equal(t, references.DungeonArrivedFromBelow, insideSnapshot.LastDirection)
	assert.Equal(t, 17, state.MapState.Lighting.TorchMinutesRemaining())
	assert.Equal(t, 23, state.MagicLightMinutesRemaining)
	assert.True(t, state.MapState.Lighting.HasMagicLight())
	if state.HasPendingDungeonRoomEncounter() {
		require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
		state.EndCombat(game_state.CombatDefeat)
		beforeRNGCalls = state.GetRNG().Calls()
	}

	changedPosition := references.Position{X: 6, Y: 5}
	_, err := state.TransformDungeonCell(7, changedPosition, func(cell references.DungeonCell) references.DungeonCell {
		return cell ^ 0x01
	})
	require.NoError(t, err)
	changedSnapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	require.NotZero(t, changedSnapshot.Revision)

	exitTarget := scene.ExecuteDebugCommand(worldRoutePositionSetDebugRequest(
		state,
		references.Deceit,
		debugging.WorldPlacementDungeonUnderworldExit,
		debugging.WorldRouteUnderworld,
	))

	require.True(t, exitTarget.Success, exitTarget.ErrorMessage)
	exitSnapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Zero(t, exitSnapshot.Revision)
	assert.Equal(t, references.DungeonFacingWest, exitSnapshot.Facing)
	assert.Equal(t, references.DungeonLastWest, exitSnapshot.LastDirection)
	assert.NotEqual(t, changedSnapshot.Cells, exitSnapshot.Cells)
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeRNGCalls, state.GetRNG().Calls())
	assert.Equal(t, 17, state.MapState.Lighting.TorchMinutesRemaining())
	assert.Equal(t, 23, state.MagicLightMinutesRemaining)
	if state.HasPendingDungeonRoomEncounter() {
		require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
		state.EndCombat(game_state.CombatDefeat)
	}

	doomStart := scene.ExecuteDebugCommand(worldRoutePositionSetDebugRequest(
		state,
		references.Doom,
		debugging.WorldPlacementDungeonInsideStart,
		debugging.WorldRouteUnderworld,
	))
	require.True(t, doomStart.Success, doomStart.ErrorMessage)
	assert.Equal(t, references.PlayerLocation{
		Location: references.Doom,
		Floor:    0,
		Position: references.Position{X: 1, Y: 1},
	}, state.MapState.PlayerLocation)
	doomSnapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonFacingEast, doomSnapshot.Facing)
	assert.Equal(t, references.DungeonArrivedFromAbove, doomSnapshot.LastDirection)
	if state.HasPendingDungeonRoomEncounter() {
		require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
		state.EndCombat(game_state.CombatDefeat)
	}

	beforeLocation := state.MapState.PlayerLocation
	beforeDungeon, active := state.DungeonStateSnapshot()
	require.True(t, active)
	unavailable := scene.ExecuteDebugCommand(worldRoutePositionSetDebugRequest(
		state,
		references.Doom,
		debugging.WorldPlacementDungeonUnderworldExit,
		debugging.WorldRouteUnderworld,
	))
	assert.Equal(t, "invalid_world_setup", unavailable.ErrorCode)
	assert.Contains(t, unavailable.ErrorMessage, "Doom has no down-ladder Underworld endpoint")
	assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
	afterDungeon, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, beforeDungeon, afterDungeon)
	assert.Empty(t, unavailable.Changes)
}

func TestDungeonExplorerSetupAppliesExactArgumentsAndSemanticChanges(t *testing.T) {
	scene := newDungeonExplorerDebugTestScene(t)
	state := scene.gameState
	state.MapState.Lighting.RestoreTorchMinutes(17)

	result := scene.ExecuteDebugCommand(dungeonExplorerSetupDebugRequest(
		state,
		references.Hythloth,
		7,
		references.Position{X: 7, Y: 0},
		"west",
		false,
		0,
	))

	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, debugging.ExecutionSynchronized, result.EffectiveExecutionMode)
	var status debugging.DungeonExplorerStatusData
	require.NoError(t, json.Unmarshal(result.Data, &status))
	assert.True(t, status.Active)
	assert.Equal(t, int(references.Hythloth), status.LocationID)
	assert.Equal(t, 7, status.Floor)
	assert.Equal(t, 8, status.DisplayLevel)
	assert.Equal(t, 7, status.X)
	assert.Zero(t, status.Y)
	assert.Equal(t, int(references.DungeonFacingWest), status.FacingID)
	assert.Equal(t, "West", status.FacingLabel)
	assert.False(t, status.Lit)
	assert.Zero(t, status.TorchMinutesRemaining)
	assert.Zero(t, status.MagicLightMinutesRemaining)
	require.Len(t, status.Cells, 64)
	paths := stateChangePaths(result.Changes)
	assert.Contains(t, paths, "world.locationId")
	assert.Contains(t, paths, "world.floor")
	assert.Contains(t, paths, "world.position.x")
	assert.Contains(t, paths, "world.position.y")
	assert.Contains(t, paths, "dungeon.facing")
	assert.Contains(t, paths, "dungeon.lastDirection")
	assert.Contains(t, paths, "dungeon.cells")
	assert.Contains(t, paths, "lighting.lit")
	assert.Contains(t, paths, "lighting.torchMinutesRemaining")
	for _, change := range result.Changes {
		if change.Path != "dungeon.cells" {
			continue
		}
		var before, after []int
		require.NoError(t, json.Unmarshal(change.Before, &before))
		require.NoError(t, json.Unmarshal(change.After, &after))
		assert.Empty(t, before)
		assert.Len(t, after, references.DungeonCellCount)
	}

	noOpSetup := scene.ExecuteDebugCommand(dungeonExplorerSetupDebugRequest(
		state,
		references.Hythloth,
		7,
		references.Position{X: 7, Y: 0},
		"west",
		false,
		0,
	))
	require.True(t, noOpSetup.Success, noOpSetup.ErrorMessage)
	assert.Empty(t, noOpSetup.Changes)
}

func TestWorldDungeonMutationsRejectStaleReviewsWithoutPartialMutation(t *testing.T) {
	t.Run("world position", func(t *testing.T) {
		scene := newDungeonExplorerDebugTestScene(t)
		state := scene.gameState
		request := worldPositionSetDebugRequest(
			state,
			references.Minoc,
			0,
			references.Position{X: 15, Y: 30},
		)
		request.Arguments["expectedX"] = json.RawMessage(`9`)
		before := state.MapState.PlayerLocation

		result := scene.ExecuteDebugCommand(request)

		assert.Equal(t, "stale_world_position", result.ErrorCode)
		assert.Equal(t, "world position changed after review", result.ErrorMessage)
		assert.Equal(t, before, state.MapState.PlayerLocation)
		assert.Empty(t, result.Changes)
	})

	t.Run("world source lighting", func(t *testing.T) {
		scene := newDungeonExplorerDebugTestScene(t)
		state := scene.gameState
		request := worldPositionSetDebugRequest(
			state,
			references.Minoc,
			0,
			references.Position{X: 15, Y: 30},
		)
		before := state.MapState.PlayerLocation
		state.MapState.Lighting.RestoreTorchMinutes(9)

		result := scene.ExecuteDebugCommand(request)

		assert.Equal(t, "stale_world_position", result.ErrorCode)
		assert.Equal(t, "world source state changed after review", result.ErrorMessage)
		assert.Equal(t, before, state.MapState.PlayerLocation)
		assert.Equal(t, 9, state.MapState.Lighting.TorchMinutesRemaining())
		assert.Empty(t, result.Changes)
	})

	t.Run("dungeon world position", func(t *testing.T) {
		scene := newDungeonExplorerDebugTestScene(t)
		state := scene.gameState
		request := dungeonExplorerSetupDebugRequest(
			state,
			references.Deceit,
			0,
			references.Position{X: 4, Y: 4},
			"east",
			true,
			0,
		)
		request.Arguments["expectedFloor"] = json.RawMessage(`1`)
		before := state.MapState.PlayerLocation

		result := scene.ExecuteDebugCommand(request)

		assert.Equal(t, "stale_world_position", result.ErrorCode)
		assert.Equal(t, before, state.MapState.PlayerLocation)
		_, active := state.DungeonStateSnapshot()
		assert.False(t, active)
	})

	t.Run("dungeon revision", func(t *testing.T) {
		scene := newDungeonExplorerDebugTestScene(t)
		state := scene.gameState
		_, err := state.DebugSetupDungeon(
			references.Deceit,
			0,
			references.Position{X: 1, Y: 1},
			references.DungeonFacingNorth,
			true,
		)
		require.NoError(t, err)
		request := dungeonExplorerSetupDebugRequest(
			state,
			references.Doom,
			7,
			references.Position{X: 7, Y: 7},
			"west",
			false,
			0,
		)
		_, err = state.TransformDungeonCell(0, references.Position{}, func(cell references.DungeonCell) references.DungeonCell {
			return cell ^ 0x01
		})
		require.NoError(t, err)
		beforeLocation := state.MapState.PlayerLocation
		beforeDungeon, active := state.DungeonStateSnapshot()
		require.True(t, active)
		beforeLight := state.MapState.Lighting.HasAnyLight()

		result := scene.ExecuteDebugCommand(request)

		assert.Equal(t, "stale_dungeon_state", result.ErrorCode)
		assert.Equal(t, "dungeon state changed after review", result.ErrorMessage)
		assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
		afterDungeon, active := state.DungeonStateSnapshot()
		require.True(t, active)
		assert.Equal(t, beforeDungeon, afterDungeon)
		assert.Equal(t, beforeLight, state.MapState.Lighting.HasAnyLight())
		assert.Empty(t, result.Changes)
	})

	t.Run("dungeon source lighting", func(t *testing.T) {
		scene := newDungeonExplorerDebugTestScene(t)
		state := scene.gameState
		_, err := state.DebugSetupDungeon(
			references.Deceit,
			0,
			references.Position{X: 1, Y: 1},
			references.DungeonFacingNorth,
			true,
		)
		require.NoError(t, err)
		request := dungeonExplorerSetupDebugRequest(
			state,
			references.Doom,
			7,
			references.Position{X: 7, Y: 7},
			"west",
			false,
			0,
		)
		beforeLocation := state.MapState.PlayerLocation
		beforeDungeon, active := state.DungeonStateSnapshot()
		require.True(t, active)
		state.MagicLightMinutesRemaining--

		result := scene.ExecuteDebugCommand(request)

		assert.Equal(t, "stale_dungeon_state", result.ErrorCode)
		assert.Equal(t, "world and dungeon source state changed after review", result.ErrorMessage)
		assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
		afterDungeon, active := state.DungeonStateSnapshot()
		require.True(t, active)
		assert.Equal(t, beforeDungeon, afterDungeon)
		assert.Empty(t, result.Changes)
	})
}

func TestWorldDungeonScenarioRuntimeReviewUsesStateProducedByPreviousStep(t *testing.T) {
	scene := newDungeonExplorerDebugTestScene(t)
	catalog := debugging.NewInitialCatalog()
	worldDefinition, ok := catalog.Get(debugging.CommandWorldPositionSet)
	require.True(t, ok)
	worldRequest := worldPositionSetDebugRequest(
		scene.gameState,
		references.Minoc,
		0,
		references.Position{X: 15, Y: 30},
	)
	worldRequest.Arguments["expectedLocationId"] = json.RawMessage(`0`)
	worldRequest.Arguments["expectedFloor"] = json.RawMessage(`7`)
	worldRequest.Arguments["expectedX"] = json.RawMessage(`1`)
	worldRequest.Arguments["expectedY"] = json.RawMessage(`2`)
	worldRequest.Arguments["expectedSourceFingerprint"] = json.RawMessage(`"persisted"`)

	resolvedWorld, err := scene.ResolveScenarioRuntimeParameters(worldRequest, worldDefinition)
	require.NoError(t, err)
	assert.JSONEq(t, strconv.Itoa(int(references.Britain)), string(resolvedWorld.Arguments["expectedLocationId"]))
	assert.JSONEq(t, `0`, string(resolvedWorld.Arguments["expectedFloor"]))
	assert.JSONEq(t, `10`, string(resolvedWorld.Arguments["expectedX"]))
	assert.JSONEq(t, `11`, string(resolvedWorld.Arguments["expectedY"]))
	assert.JSONEq(t, string(mustDebugJSON(debugWorldSourceFingerprint(scene.gameState))), string(resolvedWorld.Arguments["expectedSourceFingerprint"]))
	worldResult := scene.ExecuteDebugCommand(resolvedWorld)
	require.True(t, worldResult.Success, worldResult.ErrorMessage)

	dungeonDefinition, ok := catalog.Get(debugging.CommandDungeonExplorerSetup)
	require.True(t, ok)
	dungeonRequest := dungeonExplorerSetupDebugRequest(
		scene.gameState,
		references.Deceit,
		0,
		references.Position{X: 4, Y: 4},
		"east",
		true,
		99,
	)
	dungeonRequest.Arguments["expectedLocationId"] = json.RawMessage(`0`)
	dungeonRequest.Arguments["expectedFloor"] = json.RawMessage(`7`)
	dungeonRequest.Arguments["expectedX"] = json.RawMessage(`1`)
	dungeonRequest.Arguments["expectedY"] = json.RawMessage(`2`)
	dungeonRequest.Arguments["expectedSourceFingerprint"] = json.RawMessage(`"persisted"`)

	resolvedDungeon, err := scene.ResolveScenarioRuntimeParameters(dungeonRequest, dungeonDefinition)
	require.NoError(t, err)
	assert.JSONEq(t, strconv.Itoa(int(references.Minoc)), string(resolvedDungeon.Arguments["expectedLocationId"]))
	assert.JSONEq(t, `0`, string(resolvedDungeon.Arguments["expectedFloor"]))
	assert.JSONEq(t, `15`, string(resolvedDungeon.Arguments["expectedX"]))
	assert.JSONEq(t, `30`, string(resolvedDungeon.Arguments["expectedY"]))
	assert.JSONEq(t, `0`, string(resolvedDungeon.Arguments["expectedDungeonRevision"]))
	dungeonResult := scene.ExecuteDebugCommand(resolvedDungeon)
	require.True(t, dungeonResult.Success, dungeonResult.ErrorMessage)
	assert.Equal(t, references.Deceit, scene.gameState.MapState.PlayerLocation.Location)
}

func TestDungeonExplorerSetupReportsFreshAuthorityReplacement(t *testing.T) {
	scene := newDungeonExplorerDebugTestScene(t)
	state := scene.gameState
	position := references.Position{X: 2, Y: 3}
	_, err := state.DebugSetupDungeon(
		references.Deceit,
		0,
		position,
		references.DungeonFacingNorth,
		true,
	)
	require.NoError(t, err)
	changedPosition := references.Position{X: 6, Y: 5}
	_, err = state.TransformDungeonCell(7, changedPosition, func(cell references.DungeonCell) references.DungeonCell {
		return cell ^ 0x01
	})
	require.NoError(t, err)
	state.ActiveDurationEffect = game_state.DurationEffect{
		EffectType:     game_state.DurationEffectQuickness,
		TurnsRemaining: 10,
	}
	token, _ := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, game_state.DungeonCommandOnly)
	before, active := state.DungeonStateSnapshot()
	require.True(t, active)
	require.True(t, before.QuicknessSkipNext)

	result := scene.ExecuteDebugCommand(dungeonExplorerSetupDebugRequest(
		state,
		references.Deceit,
		0,
		position,
		"north",
		true,
		int(before.Revision),
	))

	require.True(t, result.Success, result.ErrorMessage)
	paths := stateChangePaths(result.Changes)
	assert.Contains(t, paths, "dungeon.cells")
	assert.Contains(t, paths, "dungeon.quicknessSkipNext")
	assert.Contains(t, paths, "dungeon.revision")
	for _, change := range result.Changes {
		if change.Path != "dungeon.cells" {
			continue
		}
		var beforeCells, afterCells []int
		require.NoError(t, json.Unmarshal(change.Before, &beforeCells))
		require.NoError(t, json.Unmarshal(change.After, &afterCells))
		require.Len(t, beforeCells, references.DungeonCellCount)
		require.Len(t, afterCells, references.DungeonCellCount)
		index := 7*64 + int(changedPosition.Y)*8 + int(changedPosition.X)
		assert.NotEqual(t, beforeCells[index], afterCells[index])
	}
}

func TestDungeonExplorerChangesReportWandererSemantically(t *testing.T) {
	before := dungeonAuthoritySnapshot{
		Active: true,
		Snapshot: game_state.DungeonStateSnapshot{
			Location: references.Deceit,
			Wanderer: &game_state.DungeonWanderer{
				Monster:          references.MonsterSlime,
				Floor:            4,
				Position:         references.Position{X: 3, Y: 4},
				PreviousPosition: references.Position{X: 2, Y: 4},
				OnCeiling:        true,
			},
		},
	}
	after := before
	after.Snapshot.Wanderer = nil
	status := debugging.DungeonExplorerStatusData{}
	changes := make([]debugging.StateChange, 0)

	appendDungeonExplorerChanges(&changes, before, after, status, status)

	require.Len(t, changes, 1)
	assert.Equal(t, "dungeon.wanderer", changes[0].Path)
	assert.JSONEq(t, `{"monster":24,"floor":4,"x":3,"y":4,"previousX":2,"previousY":4,"onCeiling":true}`, string(changes[0].Before))
	assert.JSONEq(t, `null`, string(changes[0].After))
}

func TestWorldDungeonMutationsUseWorldSetupSceneBlockers(t *testing.T) {
	blockers := []struct {
		name      string
		errorCode string
		apply     func(*GameScene)
	}{
		{
			name: "dialog", errorCode: "dialog_active",
			apply: func(scene *GameScene) {
				scene.dialogStack.PushModalDialog(&blockingDebugWorkflowDialog{})
			},
		},
		{
			name: "workflow", errorCode: "gameplay_workflow_active",
			apply: func(scene *GameScene) {
				scene.gameState.EventQueueManager.Workflow("blocking").
					Priority(event_queue.PriorityNormal).
					Sequential().
					First(func() {}).
					Finally(func() {}).
					Start()
			},
		},
		{
			name: "scripted scene", errorCode: "scripted_scene_active",
			apply: func(scene *GameScene) {
				scene.miniMapScene = minimap.NewMiniMapScene()
			},
		},
	}
	for _, blocker := range blockers {
		for _, commandID := range []string{
			debugging.CommandWorldPositionSet,
			debugging.CommandDungeonExplorerSetup,
		} {
			t.Run(blocker.name+"/"+commandID, func(t *testing.T) {
				scene := newDungeonExplorerDebugTestScene(t)
				state := scene.gameState
				var request debugging.Request
				if commandID == debugging.CommandWorldPositionSet {
					request = worldPositionSetDebugRequest(
						state,
						references.Minoc,
						0,
						references.Position{X: 15, Y: 30},
					)
				} else {
					request = dungeonExplorerSetupDebugRequest(
						state,
						references.Deceit,
						0,
						references.Position{X: 4, Y: 4},
						"east",
						true,
						0,
					)
				}
				before := state.MapState.PlayerLocation
				blocker.apply(scene)

				statusResult := scene.ExecuteDebugCommand(debugging.Request{
					CommandID: debugging.CommandDungeonExplorerStatus,
				})
				require.True(t, statusResult.Success)
				var status debugging.DungeonExplorerStatusData
				require.NoError(t, json.Unmarshal(statusResult.Data, &status))
				assert.False(t, status.SetupAvailable)
				assert.NotEmpty(t, status.SetupUnavailableReason)

				result := scene.ExecuteDebugCommand(request)

				assert.Equal(t, blocker.errorCode, result.ErrorCode)
				assert.Equal(t, before, state.MapState.PlayerLocation)
				_, active := state.DungeonStateSnapshot()
				assert.False(t, active)
			})
		}
	}
}

func newDungeonExplorerDebugTestScene(t *testing.T) *GameScene {
	t.Helper()
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(0).
		WithPlayerAt(10, 11).
		WithSystemCallbacks().
		Build()

	return &GameScene{gameState: state} //nolint:exhaustruct // Focused Workbench adapter fixture.
}

func worldPositionSetDebugRequest(
	state *game_state.GameState,
	location references.Location,
	floor references.FloorNumber,
	position references.Position,
) debugging.Request {
	arguments := reviewedWorldPositionDebugArguments(state)
	arguments["location"] = mustDebugJSON(strconv.Itoa(int(location)))
	arguments["floor"] = mustDebugJSON(strconv.Itoa(int(floor)))
	arguments["x"] = mustDebugJSON(int(position.X))
	arguments["y"] = mustDebugJSON(int(position.Y))

	return debugging.Request{CommandID: debugging.CommandWorldPositionSet, Arguments: arguments}
}

func worldRoutePositionSetDebugRequest(
	state *game_state.GameState,
	location references.Location,
	placement string,
	route string,
) debugging.Request {
	arguments := reviewedWorldPositionDebugArguments(state)
	arguments["placement"] = mustDebugJSON(placement)
	arguments["route"] = mustDebugJSON(route)
	arguments["location"] = mustDebugJSON(strconv.Itoa(int(location)))

	return debugging.Request{CommandID: debugging.CommandWorldPositionSet, Arguments: arguments}
}

func dungeonExplorerSetupDebugRequest(
	state *game_state.GameState,
	location references.Location,
	floor references.FloorNumber,
	position references.Position,
	facing string,
	lit bool,
	expectedRevision int,
) debugging.Request {
	arguments := reviewedWorldPositionDebugArguments(state)
	arguments["dungeon"] = mustDebugJSON(strconv.Itoa(int(location)))
	arguments["floor"] = mustDebugJSON(int(floor))
	arguments["x"] = mustDebugJSON(int(position.X))
	arguments["y"] = mustDebugJSON(int(position.Y))
	arguments["facing"] = mustDebugJSON(facing)
	arguments["lit"] = mustDebugJSON(lit)
	arguments["expectedDungeonRevision"] = mustDebugJSON(expectedRevision)

	return debugging.Request{CommandID: debugging.CommandDungeonExplorerSetup, Arguments: arguments}
}

func reviewedWorldPositionDebugArguments(state *game_state.GameState) map[string]json.RawMessage {
	current := state.MapState.PlayerLocation

	return map[string]json.RawMessage{
		"expectedLocationId":        mustDebugJSON(int(current.Location)),
		"expectedFloor":             mustDebugJSON(int(current.Floor)),
		"expectedX":                 mustDebugJSON(int(current.Position.X)),
		"expectedY":                 mustDebugJSON(int(current.Position.Y)),
		"expectedSourceFingerprint": mustDebugJSON(debugWorldSourceFingerprint(state)),
	}
}

func worldPlacementTarget(
	t *testing.T,
	destination debugging.WorldPositionDestinationData,
	placement string,
	route string,
) debugging.WorldPositionPlacementTargetData {
	t.Helper()
	for _, target := range destination.PlacementTargets {
		if target.Placement == placement && target.Route == route {
			return target
		}
	}
	require.FailNow(t, "world placement target not found", "%s via %s", placement, route)

	return debugging.WorldPositionPlacementTargetData{}
}

func worldPlacementTargets(
	destination debugging.WorldPositionDestinationData,
	route string,
) []debugging.WorldPositionPlacementTargetData {
	targets := make([]debugging.WorldPositionPlacementTargetData, 0)
	for _, target := range destination.PlacementTargets {
		if target.Route == route {
			targets = append(targets, target)
		}
	}

	return targets
}

const sha256HexLength = 64
