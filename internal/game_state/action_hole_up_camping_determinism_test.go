package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type bardCampingCycleSnapshot struct {
	cycle    int
	hour     byte
	minute   byte
	rngCalls uint64
	guard    references.Position
	hp       uint16
}

func startBardCampingTest(t *testing.T, hours int, forceAmbush bool) (*GameState, *MockSystemCallbacks, *camping.CampingScene, *int) {
	t.Helper()
	state, callbacks := newCampingAmbushTestState(t)
	state.DateTime.Hour, state.DateTime.Minute = 10, 0
	state.lastProcessedHour = 10
	state.PartyState.GetCharacter(0).Class = party_state.Bard
	state.PartyState.GetCharacter(1).SetEquippedItem(references.SlotRing, references.RingRegeneration)
	state.SetRandomSeed(12345)
	installAutomaticAnyKey(state)
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(hours, 0, state.MapState.PlayerLocation, campingMap)
	cleanupCalls := 0
	queue := state.EventQueueManager.Workflow("bard_camping_test").
		Priority(event_queue.PriorityNormal).Sequential().BlockInput().First(func() {})
	state.AddCampingSequenceToBuilder(queue, scene, false, forceAmbush, func() { cleanupCalls++ })
	queue.Start()
	t.Cleanup(state.EventQueueManager.Shutdown)

	return state, callbacks, scene, &cleanupCalls
}

func TestOutdoorBardCampingLogicalCyclesIgnoreUpdateIncrement(t *testing.T) {
	for _, ambush := range []bool{false, true} {
		t.Run(fmt.Sprintf("ambush=%t", ambush), func(t *testing.T) {
			var baseline *GameState
			var baselineTrace []bardCampingCycleSnapshot
			for _, increment := range []int64{16, 200, 500, 1500} {
				t.Run(fmt.Sprintf("update=%dms", increment), func(t *testing.T) {
					hours := 1
					if ambush {
						hours = 2
					}
					state, callbacks, scene, cleanupCalls := startBardCampingTest(t, hours, ambush)
					beforeDate := state.DateTime
					var trace []bardCampingCycleSnapshot
					sawPerformance := false
					var performanceRNGCalls uint64
					for update := 0; update < 10000 && state.EventQueueManager.HasActiveWorkflows(); update++ {
						state.ElapsedMs += increment
						state.EventQueueManager.Update()
						if callbacks.PlaybackRequestCount(SoundBardLute, SoundPlaybackLoop) == 1 &&
							callbacks.StopRequestCount(SoundBardLute) == 0 {
							if !sawPerformance {
								performanceRNGCalls = state.GetRNG().Calls()
								sawPerformance = true
							}
							require.Zero(t, scene.CurrentCycle, "performance must precede every logical cycle")
							require.Equal(t, beforeDate, state.DateTime)
							require.Equal(t, performanceRNGCalls, state.GetRNG().Calls(), "performance updates must not consume gameplay RNG")
						}
						if scene.CurrentCycle > len(trace) {
							require.Equal(t, len(trace)+1, scene.CurrentCycle)
							require.Equal(t, 1, callbacks.StopRequestCount(SoundBardLute))
							trace = append(trace, bardCampingCycleSnapshot{
								cycle: scene.CurrentCycle, hour: state.DateTime.Hour, minute: state.DateTime.Minute,
								rngCalls: state.GetRNG().Calls(), guard: scene.GetPartyMemberPosition(scene.GuardIndex),
								hp: state.PartyState.GetCharacter(1).GetCurrentHp(),
							})
						}
					}
					require.False(t, state.EventQueueManager.HasActiveWorkflows())
					assert.True(t, sawPerformance)
					assert.Equal(t, 1, *cleanupCalls)
					assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
					assert.Nil(t, state.CampingScene)
					assert.Equal(t, ambush, state.HasPendingOverworldContact())
					assert.Equal(t, 1, callbacks.StopRequestCount(SoundBardLute))
					_, overridden := state.SpriteOverrides.Get("camping", "guard")
					assert.False(t, overridden)
					require.NotEmpty(t, trace)
					assert.GreaterOrEqual(t, trace[0].rngCalls, performanceRNGCalls+2, "first cycle includes regeneration and guard rolls")
					if ambush {
						assert.LessOrEqual(t, scene.CurrentCycle, DebugAmbushCycle)
					} else {
						assert.Equal(t, camping.CyclesPerHour, scene.CurrentCycle)
					}
					if baseline == nil {
						baseline, baselineTrace = state, trace
						return
					}
					assert.Equal(t, baselineTrace, trace, "every logical cycle must be frame-rate independent")
					assert.Equal(t, baseline.DateTime, state.DateTime)
					assert.Equal(t, baseline.GetRNG().Calls(), state.GetRNG().Calls())
					for index := range party_state.MaxActiveParty {
						assert.Equal(t, *baseline.PartyState.GetCharacter(index), *state.PartyState.GetCharacter(index))
					}
					if ambush {
						require.NotNil(t, baseline.pendingCampingCombat)
						require.NotNil(t, state.pendingCampingCombat)
						want, got := baseline.pendingCampingCombat.combat, state.pendingCampingCombat.combat
						assert.Equal(t, want.CampingOrigin, got.CampingOrigin)
						assert.Equal(t, want.PartyPositions, got.PartyPositions)
						assert.Equal(t, want.PartySleeping, got.PartySleeping)
						assert.Equal(t, want.MonsterManager.monsters, got.MonsterManager.monsters)
					}
				})
			}
		})
	}
}

func TestOutdoorBardCampingCancellationCleansPerformanceWithoutCycles(t *testing.T) {
	state, callbacks, scene, cleanupCalls := startBardCampingTest(t, 1, false)
	beforeDate, beforeLocation := state.DateTime, state.MapState.PlayerLocation
	for update := 0; update < 100 && callbacks.PlaybackRequestCount(SoundBardLute, SoundPlaybackLoop) == 0; update++ {
		state.ElapsedMs += 100
		state.EventQueueManager.Update()
	}
	require.Equal(t, 1, callbacks.PlaybackRequestCount(SoundBardLute, SoundPlaybackLoop))
	require.True(t, state.EventQueueManager.HasInputBlockingWorkflows())
	performanceRNGCalls := state.GetRNG().Calls()

	state.EventQueueManager.Shutdown()
	state.EventQueueManager.Shutdown()

	assert.Equal(t, 1, *cleanupCalls)
	assert.Zero(t, scene.CurrentCycle)
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
	assert.Equal(t, performanceRNGCalls, state.GetRNG().Calls())
	assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
	assert.Nil(t, state.CampingScene)
	assert.Nil(t, state.pendingCampingCombat)
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundBardLute))
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundWind))
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundFireCrackle))
	_, overridden := state.SpriteOverrides.Get("camping", "guard")
	assert.False(t, overridden)
	for index := range party_state.MaxActiveParty {
		assert.False(t, state.PartyState.GetCharacter(index).IsAsleep())
	}
}
