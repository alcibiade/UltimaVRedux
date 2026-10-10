package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestDungeonCampingAudioFollowsPresentedFireAndCleansUp(t *testing.T) {
	for _, terrain := range []struct {
		name string
		cell references.DungeonCell
		fire bool
	}{
		{name: "corridor campfire", cell: 0x00, fire: true},
		{name: "ladder replaces fire", cell: 0x10},
		{name: "chest replaces fire", cell: 0x40},
	} {
		for _, cancel := range []bool{false, true} {
			name := terrain.name + "/complete"
			if cancel {
				name = terrain.name + "/cancel"
			}
			t.Run(name, func(t *testing.T) {
				state, callbacks := newDungeonCampingTestStateWithCallbacks(t)
				startDungeonCampingAudioTest(t, state, terrain.cell)
				center := state.CampingScene.CampingMap.CampfirePosition
				tile := state.GetLayeredMapByCurrentLocation().GetTopTile(&center)
				require.NotNil(t, tile)
				require.Equal(t, terrain.fire, tile.Index == indexes.CampFire)
				starts := 0
				if terrain.fire {
					starts = 1
				}
				require.Equal(t, starts, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop))
				require.Zero(t, callbacks.StopRequestCount(SoundFireCrackle))
				require.Equal(t, 1, callbacks.PlaybackRequestCount(SoundWind, SoundPlaybackLoop))
				if cancel {
					state.EventQueueManager.Shutdown()
					state.EventQueueManager.Shutdown()
				} else {
					runCampingWorkflowToCompletion(t, state)
				}
				require.Equal(t, starts, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop))
				require.Equal(t, 1, callbacks.StopRequestCount(SoundFireCrackle))
				require.Equal(t, 1, callbacks.StopRequestCount(SoundWind))
				require.Nil(t, state.CampingScene)
				require.False(t, callbacks.InputBlocked)
				require.Equal(t, references.Deceit, state.MapState.PlayerLocation.Location)
			})
		}
	}
}

func TestOutdoorCampingFireDoesNotLeakIntoFirelessDungeonRest(t *testing.T) {
	state, callbacks := newCampingAmbushTestState(t)
	installAutomaticAnyKey(state)
	require.True(t, state.ActionHoleUpLargeMapWithDebug(1, NoGuardAssigned, false, false))
	state.EventQueueManager.Update()
	require.NotNil(t, state.CampingScene)
	require.Equal(t, 1, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop))
	require.Zero(t, callbacks.StopRequestCount(SoundFireCrackle))
	runCampingWorkflowToCompletion(t, state)
	require.Equal(t, 1, callbacks.StopRequestCount(SoundFireCrackle))

	startDungeonCampingAudioTest(t, state, 0x40)
	require.Equal(t, 1, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop), "dungeon must not restart the stopped outdoor loop")
	require.Equal(t, 1, callbacks.StopRequestCount(SoundFireCrackle))
	runCampingWorkflowToCompletion(t, state)
	require.Equal(t, 1, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop))
	require.Equal(t, 2, callbacks.StopRequestCount(SoundFireCrackle))
}

func startDungeonCampingAudioTest(t *testing.T, state *GameState, cell references.DungeonCell) {
	t.Helper()
	var cells references.DungeonCells
	cells.SetCell(0, references.Position{X: 1, Y: 1}, cell)
	attachDungeonStateForTest(t, state, cells)
	token, _ := state.BeginDungeonCommandCycle()
	state.ActionHoleUpDungeonMap(token)
	scene, plan, err := state.PrepareDungeonCampingPlan(1, NoGuardAssigned, token)
	require.NoError(t, err)
	state.SetRandomSeed(dungeonCampingNoAmbushSeed(t, 1))
	installAutomaticAnyKey(state)
	workflow := state.EventQueueManager.Workflow("dungeon_camping_audio_test").
		Priority(event_queue.PriorityNormal).Sequential().BlockInput().First(func() {})
	state.AddDungeonCampingSequenceToBuilder(workflow, scene, plan, func() {
		state.CompleteDungeonCampingCommand(token)
	})
	workflow.Start()
	state.EventQueueManager.Update()
	require.NotNil(t, state.CampingScene)
}
