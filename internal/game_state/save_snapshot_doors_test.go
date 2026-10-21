package game_state

import (
	"fmt"
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDoorSnapshotHelpersContinuation(t *testing.T) {
	for _, closed := range []indexes.SpriteIndex{indexes.RegularDoor, indexes.RegularDoorView} {
		for _, automatic := range []bool{false, true} {
			for _, remaining := range []int{2, 1, 0} {
				t.Run(fmt.Sprintf("tile%d/auto%t/turns%d", closed, automatic, remaining), func(t *testing.T) {
					state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
						WithPlayerAt(10, 14).WithRandomSeed(673).WithSystemCallbacks().Build()
					door := references.Position{X: 11, Y: 14}
					state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &door, closed)
					if automatic {
						require.True(t, state.CanAutoOpenDoor(&door))
						require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Right))
					} else {
						require.True(t, state.ActionOpenSmallMap(references.Right))
					}
					for turn := 2; turn > remaining; turn-- {
						require.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
					}
					date, calls, elapsed := state.DateTime, state.GetRNG().Calls(), state.ElapsedMs
					party := save_snapshot.SaveSnapshot{}
					state.captureParty(&party)
					state.captureInventory(&party)
					raw := state.captureMapMutations()
					rawBefore := slices.Clone(raw)
					saved := state.captureOpenDoor()
					require.NotNil(t, saved)
					assert.Equal(t, remaining, saved.RemainingTurns)
					assert.Equal(t, int(closed), saved.ClosedTile)
					canonical := state.canonicalizeOpenDoorMutations(raw)
					assert.Equal(t, rawBefore, raw, "capture cannot rewrite its caller's mutation list")
					require.NoError(t, validateOpenDoorSnapshotCompatibility(saved, positionSnapshot(state.MapState.PlayerLocation), canonical, state.GameReferences))
					assert.Equal(t, date, state.DateTime)
					assert.Equal(t, calls, state.GetRNG().Calls())
					assert.Equal(t, elapsed, state.ElapsedMs)
					afterParty := save_snapshot.SaveSnapshot{}
					state.captureParty(&afterParty)
					state.captureInventory(&afterParty)
					assert.Equal(t, party, afterParty)
					assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&door))

					restored, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
						WithPlayerAt(10, 14).WithRandomSeed(673).WithSystemCallbacks().Build()
					require.NoError(t, restored.restoreMapMutations(canonical))
					require.NoError(t, restored.restoreOpenDoor(saved))
					assert.Equal(t, saved, restored.captureOpenDoor())
					assert.Equal(t, canonical, restored.canonicalizeOpenDoorMutations(restored.captureMapMutations()))
					for step := remaining; step >= 0; step-- {
						for _, current := range []*GameState{state, restored} {
							assert.Equal(t, step == 0, current.MapState.SmallMapProcessTurnDoors(current.isSmallMapDoorOccupied))
							want := indexes.SpriteIndex(indexes.BrickFloor)
							if step == 0 {
								want = closed
							}
							assert.Equal(t, want, current.GetEffectiveTile(&door))
						}
					}
					assert.Nil(t, restored.captureOpenDoor())
				})
			}
		}
	}
}

func TestOpenDoorSnapshotRestoredExpiryDefersFinalOccupant(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 14).WithSystemCallbacks().Build()
	door := references.Position{X: 11, Y: 14}
	saved := &save_snapshot.OpenDoorSnapshot{
		Position:       save_snapshot.PositionSnapshot{Location: int(references.SerpentsHold), X: 11, Y: 14, Floor: 0},
		RemainingTurns: 0, ClosedTile: int(indexes.RegularDoor),
	}
	require.NoError(t, state.restoreOpenDoor(saved))
	npc := &SimpleTestMapUnit{position: door, floor: 0, visible: true}
	state.CurrentNPCAIController = &SimpleTestController{mapUnits: map_units.MapUnits{npc}}
	state.SetPlayerPosition(door)
	assert.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
	state.SetPlayerPosition(door.Left())
	assert.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
	npc.SetFloor(1)
	assert.True(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), state.GetEffectiveTile(&door))
}

func TestOpenDoorSnapshotZeroUnoccupiedAfterNPCActorPhase(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 14).WithSystemCallbacks().Build()
	door, outside := references.Position{X: 11, Y: 14}, references.Position{X: 12, Y: 14}
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Right))
	for range 2 {
		require.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
	}
	reference := references.NPCReference{Schedule: references.CreateNPCScheduledFixedOneLocation(door, 0)}
	reference.SetKeyIndex(indexes.TownsPersonKeyIndex)
	npc := map_units.NewNPCFriendly(reference, 1)
	npc.PlaceAtScheduleSlot(0)
	npc.NPCReference.Schedule.X[1], npc.NPCReference.Schedule.Y[1] = byte(outside.X), byte(outside.Y)
	state.DateTime.Hour = 1
	npc.MapUnitDetails().SetCurrentPath([]references.Position{outside})
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(map_units.MapUnits{npc})
	state.processNonCombatWorldActorPhaseForCommand(false)
	require.Equal(t, outside, npc.Pos(), "real NPC actor phase must leave after door expiry deferral")
	require.False(t, state.isSmallMapDoorOccupied(door, 0))
	saved := state.captureOpenDoor()
	require.NotNil(t, saved)
	require.Zero(t, saved.RemainingTurns)
	mutations := state.canonicalizeOpenDoorMutations(state.captureMapMutations())
	require.NoError(t, validateOpenDoorSnapshotCompatibility(saved, positionSnapshot(state.MapState.PlayerLocation), mutations, state.GameReferences))
	state.processNonCombatWorldActorPhaseForCommand(false)
	assert.False(t, state.MapState.HasTransientDoor())
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), state.GetEffectiveTile(&door))
}

func TestOpenDoorSnapshotCompatibilityRejectsMismatchedTerrain(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 14).WithSystemCallbacks().Build()
	world := positionSnapshot(state.MapState.PlayerLocation)
	valid := save_snapshot.OpenDoorSnapshot{
		Position:       save_snapshot.PositionSnapshot{Location: int(references.SerpentsHold), X: 11, Y: 14, Floor: 0},
		RemainingTurns: 0, ClosedTile: int(indexes.RegularDoor),
	}
	require.NoError(t, validateOpenDoorSnapshotCompatibility(&valid, world, nil, state.GameReferences), "zero unoccupied countdown is a stable actor-phase boundary")
	for _, scenario := range []string{"floor", "location", "coordinate", "variant", "timer", "open floor mutation", "lock mutation"} {
		t.Run(scenario, func(t *testing.T) {
			door := valid
			var mutations []save_snapshot.MapMutationSnapshot
			switch scenario {
			case "floor":
				door.Position.Floor = 7
			case "location":
				door.Position.Location = int(references.Britain)
			case "coordinate":
				door.Position.X = 32
			case "variant":
				door.ClosedTile = int(indexes.RegularDoorView)
			case "timer":
				door.RemainingTurns = 3
			case "open floor mutation", "lock mutation":
				value := int(indexes.BrickFloor)
				if scenario == "lock mutation" {
					value = int(indexes.MagicLockDoor)
				}
				mutations = []save_snapshot.MapMutationSnapshot{{Position: door.Position, Kind: save_snapshot.MapMutationSetObject, Value: value}}
			}
			assert.Error(t, validateOpenDoorSnapshotCompatibility(&door, world, mutations, state.GameReferences))
		})
	}
	// A valid explicit view-door override on another loaded floor supersedes its
	// source wall. The timer's floor is independent of the Avatar's current floor.
	valid.Position.Floor = -1
	valid.ClosedTile = int(indexes.RegularDoorView)
	mutations := []save_snapshot.MapMutationSnapshot{{Position: valid.Position, Kind: save_snapshot.MapMutationSetObject, Value: valid.ClosedTile}}
	require.NoError(t, validateOpenDoorSnapshotCompatibility(&valid, world, mutations, state.GameReferences))
	require.NoError(t, state.restoreMapMutations(mutations))
	require.NoError(t, state.restoreOpenDoor(&valid))
	assert.False(t, state.MapState.IsPositionOpenDoor(&references.Position{X: 11, Y: 14}))
	require.True(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), state.MapState.LayeredMaps.
		GetLayeredMap(references.SmallMapType, -1).GetTileTopMapOnlyTile(&references.Position{X: 11, Y: 14}).Index)
}

func TestOccupiedDoorNativeHydrationUsesFinalActorsAndPersistentLocks(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 14).WithSystemCallbacks().Build()
	door := references.Position{X: 11, Y: 14}
	require.Equal(t, MagicLockSuccess, state.MagicLockDoor(&door))
	var occupant *map_units.NPCFriendly
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		if friendly, ok := unit.(*map_units.NPCFriendly); ok && friendly.IsVisible() && !friendly.IsEmptyMapUnit() {
			occupant = friendly
			break
		}
	}
	require.NotNil(t, occupant)
	occupant.SetPos(door)
	occupant.SetFloor(0)
	require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&door))
	require.False(t, state.MapState.HasTransientDoor())
	restored := restoreDoorVisitSnapshot(t, state)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), restored.GetEffectiveTile(&door))
	assert.True(t, restored.MapState.IsPositionOpenDoor(&door))
	for _, mutation := range restored.captureMapMutations() {
		if mutation.Position.X == int(door.X) && mutation.Position.Y == int(door.Y) && mutation.Position.Floor == 0 {
			assert.Equal(t, int(indexes.MagicLockDoor), mutation.Value, "temporary occupancy floor is never persisted")
		}
	}
	for _, unit := range *restored.CurrentNPCAIController.GetNpcs() {
		if unit.Floor() == 0 && unit.Pos() == door {
			unit.SetPos(door.Right())
		}
	}
	assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoor), restored.GetEffectiveTile(&door))
}

func TestSaveSnapshotTimedDoorNativeContinuation(t *testing.T) {
	for _, closed := range []indexes.SpriteIndex{indexes.RegularDoor, indexes.RegularDoorView} {
		for _, occupant := range []string{"none", "Avatar", "NPC"} {
			for _, remaining := range []int{2, 1, 0} {
				t.Run(fmt.Sprintf("tile%d/%s/turns%d", closed, occupant, remaining), func(t *testing.T) {
					state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
						WithPlayerAt(10, 14).WithRandomSeed(673).WithSystemCallbacks().Build()
					door := references.Position{X: 11, Y: 14}
					state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &door, closed)
					require.True(t, state.ActionOpenSmallMap(references.Right))
					for turn := 2; turn > remaining; turn-- {
						require.False(t, state.MapState.SmallMapProcessTurnDoors(state.isSmallMapDoorOccupied))
					}
					switch occupant {
					case "Avatar":
						state.SetPlayerPosition(door)
					case "NPC":
						units := state.CurrentNPCAIController.GetNpcs()
						require.NotEmpty(t, *units)
						npc := (*units)[0]
						require.False(t, npc.IsEmptyMapUnit())
						npc.SetVisible(true)
						npc.SetFloor(0)
						npc.SetPos(door)
					}
					before := state.captureOpenDoor()
					restored := restoreDoorVisitSnapshot(t, state)
					assert.Equal(t, before, state.captureOpenDoor(), "capture never force-closes the live door")
					require.Equal(t, before, restored.captureOpenDoor())
					for _, current := range []*GameState{state, restored} {
						for turn := remaining; turn > 0; turn-- {
							assert.False(t, current.MapState.SmallMapProcessTurnDoors(current.isSmallMapDoorOccupied))
						}
						assert.Equal(t, occupant == "none", current.MapState.SmallMapProcessTurnDoors(current.isSmallMapDoorOccupied))
						if occupant != "none" {
							assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), current.GetEffectiveTile(&door))
							current.SetPlayerPosition(door.Left())
							for _, unit := range *current.CurrentNPCAIController.GetNpcs() {
								if unit.Floor() == 0 && unit.Pos() == door {
									unit.SetPos(door.Right())
								}
							}
							assert.True(t, current.MapState.SmallMapProcessTurnDoors(current.isSmallMapDoorOccupied))
						}
						assert.Equal(t, closed, current.GetEffectiveTile(&door))
						assert.Nil(t, current.captureOpenDoor())
					}
					assert.Equal(t, state.RollDice(100), restored.RollDice(100))
				})
			}
		}
	}
}
