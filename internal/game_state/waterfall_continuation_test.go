package game_state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func newWaterfallContinuationFixture(t *testing.T, position references.Position) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(int(position.X), int(position.Y)).WithSystemCallbacks().Build()
	state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		character.PartyMap = party_state.PartyMapAtInn(references.Britain)
		character.SetStatus(party_state.Good)
		character.SetEquippedItem(references.SlotRing, references.NoEquipment)
	})
	avatar := state.PartyState.GetAvatar()
	avatar.PartyMap = party_state.PartyMapActive
	avatar.SetMaxHp(1000)
	avatar.SetHp(1000)
	avatar.Dexterity = 1 // Every transformed d30 fails, independent of seed.
	state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	state.DebugOptions.MonsterGen = false
	state.beggarKarmaCounter = 0
	state.outdoorMountedActorSkip = false
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 8}
	for _, floor := range []references.FloorNumber{references.MainFloor, references.Basement} {
		layered := state.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, floor)
		for y := range references.YLargeMapTiles {
			// A walled column deliberately prevents escaping sideways between falls.
			for dx := -1; dx <= 1; dx++ {
				cell := references.Position{X: position.X + references.Coordinate(dx), Y: references.Coordinate(y)}
				tile := indexes.SpriteIndex(indexes.TallMountains)
				if dx == 0 {
					tile = indexes.Grass
				}
				layered.SetTileByLayer(map_state.MapOverrideLayer, &cell, tile)
			}
		}
	}
	for _, controller := range state.LargeMapNPCAIController {
		controller.GetNpcs().RemoveAllEnemies()
	}
	state.SetRandomSeed(123)
	callbacks.Reset()

	return state, callbacks
}

func setWaterfallContinuationChain(state *GameState, floor references.FloorNumber, start references.Position, falls int) {
	layered := state.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, floor)
	for index := range falls {
		waterfall := references.Position{X: start.X, Y: start.Y + references.Coordinate(index*2+1)}
		layered.SetTileByLayer(map_state.MapOverrideLayer,
			waterfall.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles), indexes.Waterfall1)
	}
}

func advanceWaterfallUntil(t *testing.T, state *GameState, done func() bool) {
	t.Helper()
	for updates := 0; !done(); updates++ {
		require.Less(t, updates, 10000, "waterfall continuation failed to reach its boundary")
		state.EventQueueManager.Update()
		state.ElapsedMs += 100
	}
}

func waterfallMessageCount(callbacks *MockSystemCallbacks, message string) int {
	count := 0
	for _, recorded := range callbacks.Messages {
		if recorded == message {
			count++
		}
	}

	return count
}

func TestWaterfallContinuationConsumedSurfaceCycles(t *testing.T) {
	tests := []struct {
		name  string
		falls int
		start func(*GameState)
	}{
		{name: "non-waterfall Pass", falls: 0, start: func(state *GameState) { state.CompleteSurfacePassCycle(references.LargeMapType) }},
		{name: "single movement", falls: 1, start: func(state *GameState) { state.CompleteSurfaceMovementCycle(references.LargeMapType, true) }},
		{name: "two constrained falls", falls: 2, start: func(state *GameState) { state.CompleteSurfaceMovementCycle(references.LargeMapType, true) }},
		{name: "many constrained falls", falls: 6, start: func(state *GameState) { state.CompleteSurfaceCommandCycle(references.LargeMapType) }},
		{name: "Pass while already eligible", falls: 3, start: func(state *GameState) { state.CompleteSurfacePassCycle(references.LargeMapType) }},
		{name: "direct action completion", falls: 2, start: (*GameState).FinishTurn},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := references.Position{X: 50, Y: 50}
			state, callbacks := newWaterfallContinuationFixture(t, start)
			setWaterfallContinuationChain(state, references.MainFloor, start, test.falls)
			before := state.DateTime
			test.start(state)
			charged := state.DateTime
			callsBeforeFalls := state.GetRNG().Calls()
			assert.Equal(t, byte(2), charged.Minute-before.Minute)
			assert.Equal(t, before.Turn+1, charged.Turn)
			assert.Equal(t, 7, state.ActiveDurationEffect.TurnsRemaining)
			assert.Equal(t, 1, state.beggarKarmaCounter)
			assert.Equal(t, test.falls != 0, state.IsWaterfallHazardActive())
			advanceWaterfallUntil(t, state, func() bool {
				if state.IsWaterfallHazardActive() {
					assert.True(t, callbacks.InputBlocked, "input must remain blocked between descents")
				}
				return !state.IsWaterfallHazardActive()
			})
			assert.Equal(t, references.Position{X: 50, Y: references.Coordinate(50 + test.falls*2)}, state.MapState.PlayerLocation.Position)
			assert.Equal(t, uint16(1000-test.falls), state.PartyState.GetAvatar().GetCurrentHp())
			assert.Equal(t, test.falls, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
			assert.Equal(t, charged, state.DateTime, "falls do not replay the ordinary calendar cycle")
			assert.Equal(t, 7, state.ActiveDurationEffect.TurnsRemaining)
			assert.Equal(t, 1, state.beggarKarmaCounter, "only the initiating player update")
			// Preserve both the transformed-d30 draw and the existing fixed-damage
			// builder's Intn(1) request. Animation and landing reviews draw nothing.
			expected := beradrng.New(123)
			for range test.falls {
				expected.Intn(originalD30SourceValueCount)
				expected.Intn(1)
			}
			assert.Equal(t, callsBeforeFalls+expected.Calls(), state.GetRNG().Calls())
			assert.False(t, callbacks.InputBlocked)
			if test.falls != 0 {
				assert.Equal(t, 1, callbacks.BlockInputCalls)
				assert.Equal(t, 1, callbacks.UnblockInputCalls)
			}
		})
	}
}

func TestWaterfallContinuationBlockedMovementDoesNotCreateCheckpoint(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, callbacks := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 2)
	before := state.DateTime
	state.CompleteSurfaceMovementCycle(references.LargeMapType, false)
	assert.False(t, state.IsWaterfallHazardActive())
	assert.Equal(t, before, state.DateTime)
	assert.Zero(t, state.GetRNG().Calls())
	assert.Equal(t, 8, state.ActiveDurationEffect.TurnsRemaining)
	assert.Empty(t, callbacks.Messages)
	assert.False(t, callbacks.InputBlocked)
}

func TestWaterfallContinuationRepeatedReviewsAreReadOnly(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, callbacks := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 3)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	charged := state.DateTime
	advanceWaterfallUntil(t, state, func() bool {
		if !state.IsWaterfallHazardActive() {
			return true
		}
		position, hp, calls := state.MapState.PlayerLocation, state.PartyState.GetAvatar().GetCurrentHp(), state.GetRNG().Calls()
		for range 2 {
			require.True(t, state.ProcessWaterfallHazard(&start))
			state.CompleteSurfacePassCycle(references.LargeMapType)
			state.CompleteSurfaceCommandCycle(references.LargeMapType)
			state.CompleteSurfaceMovementCycle(references.LargeMapType, true)
			state.FinishTurn()
		}
		assert.Equal(t, position, state.MapState.PlayerLocation)
		assert.Equal(t, hp, state.PartyState.GetAvatar().GetCurrentHp())
		assert.Equal(t, calls, state.GetRNG().Calls())
		assert.Equal(t, charged, state.DateTime)

		return false
	})
	assert.Equal(t, uint16(997), state.PartyState.GetAvatar().GetCurrentHp())
	assert.Equal(t, 3, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Equal(t, 1, callbacks.BlockInputCalls)
	assert.Equal(t, 1, callbacks.UnblockInputCalls)
}

func TestWaterfallContinuationUsesAuthoritativePosition(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, _ := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 1)
	state.SetPlayerPosition(references.Position{X: 50, Y: 60})
	assert.False(t, state.ProcessWaterfallHazard(&start), "stale triggering coordinate cannot start a fall")
	state.SetPlayerPosition(start)
	stale := references.Position{X: 50, Y: 60}
	assert.True(t, state.ProcessWaterfallHazard(&stale), "current eligibility takes precedence")
	advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
	assert.Equal(t, references.Position{X: 50, Y: 52}, state.MapState.PlayerLocation.Position)
}

func TestWaterfallContinuationWrapAndCycleBound(t *testing.T) {
	for _, test := range []struct {
		name    string
		startY  references.Coordinate
		falls   int
		landing references.Coordinate
		cycle   bool
	}{
		{name: "wrapped trigger", startY: 255, falls: 2, landing: 3},
		{name: "wrapped second step", startY: 254, falls: 2, landing: 2},
		{name: "closed wrapped orbit", startY: 0, falls: int(references.YLargeMapTiles) / 2, landing: 0, cycle: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			start := references.Position{X: 50, Y: test.startY}
			state, callbacks := newWaterfallContinuationFixture(t, start)
			setWaterfallContinuationChain(state, references.MainFloor, start, test.falls)
			state.CompleteSurfacePassCycle(references.LargeMapType)
			advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
			assert.Equal(t, references.Position{X: 50, Y: test.landing}, state.MapState.PlayerLocation.Position)
			assert.Equal(t, uint16(1000-test.falls), state.PartyState.GetAvatar().GetCurrentHp())
			assert.Equal(t, test.falls, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
			notices := 0
			for _, message := range callbacks.Messages {
				if strings.Contains(message, "current circles back") {
					notices++
				}
			}
			if test.cycle {
				assert.Equal(t, 1, notices)
			} else {
				assert.Zero(t, notices)
			}
			assert.False(t, callbacks.InputBlocked)
			assert.Equal(t, 1, state.beggarKarmaCounter)
		})
	}
}

func TestWaterfallContinuationCanonicalTransitionRechecksDestinationPlane(t *testing.T) {
	start := references.Position{X: 54, Y: 136}
	state, callbacks := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 1)
	setWaterfallContinuationChain(state, references.Basement, references.Position{X: 54, Y: 138}, 2)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
	assert.Equal(t, references.PlayerLocation{
		Location: references.BritanniaUnderworld, Floor: references.Basement, Position: references.Position{X: 54, Y: 142},
	}, state.MapState.PlayerLocation)
	assert.Equal(t, state.GetCurrentLargeMapNPCAIController(), state.CurrentNPCAIController)
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "Falling into underworld!!"))
	assert.Equal(t, 3, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Equal(t, uint16(997), state.PartyState.GetAvatar().GetCurrentHp())
	require.NotEmpty(t, callbacks.MapFadeLevels)
	assert.Zero(t, callbacks.MapFadeLevels[len(callbacks.MapFadeLevels)-1])
	assert.Equal(t, byte(2), state.DateTime.Minute)
	assert.Equal(t, 7, state.ActiveDurationEffect.TurnsRemaining)
	assert.False(t, callbacks.InputBlocked)
}

func TestWaterfallContinuationPlaneIsPartOfVisitedKey(t *testing.T) {
	start := references.Position{X: 54, Y: 136}
	state, callbacks := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 1)
	// A complete Underworld orbit revisits the surface trigger's coordinates.
	// That first Underworld visit must still fall; only its own repeat terminates.
	setWaterfallContinuationChain(state, references.Basement, references.Position{X: 54, Y: 138}, int(references.YLargeMapTiles)/2)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
	assert.Equal(t, references.Basement, state.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.Position{X: 54, Y: 138}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, 1+int(references.YLargeMapTiles)/2, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "Falling into underworld!!"), "no repeated Underworld reload")
}

func TestWaterfallContinuationStopsSailingButRetainsOneActorTail(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, _ := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 2)
	state.isSailing = true
	// The actor is adjacent only to the FIRST landing, proving the initiating
	// actor tail is not delayed until the end or replayed on a later landing.
	addOverworldTestEnemy(t, state, references.MonsterSandtrap, references.Position{X: 51, Y: 52})
	state.CompleteSurfaceMovementCycle(references.LargeMapType, true)
	assert.False(t, state.IsSailing())
	assert.Equal(t, uint16(1000), state.PartyState.GetAvatar().GetCurrentHp())
	advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
	require.NoError(t, state.ResolvePendingOverworldContact())
	expected := beradrng.New(123)
	expected.Intn(originalD30SourceValueCount)
	expected.Intn(1)
	hit := expected.Intn(8) + 1
	expected.Intn(originalD30SourceValueCount)
	expected.Intn(1)
	assert.Equal(t, uint16(998-hit), state.PartyState.GetAvatar().GetCurrentHp())
	assert.Equal(t, expected.Calls(), state.GetRNG().Calls())
	assert.Equal(t, 1, state.beggarKarmaCounter)
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Equal(t, expected.Calls(), state.GetRNG().Calls(), "the actor continuation cannot replay")
}

func TestWaterfallContinuationPartyDefeatEndsChainOnce(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, callbacks := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 3)
	state.PartyState.GetAvatar().SetHp(1)
	deaths := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deaths++ }
	state.CompleteSurfacePassCycle(references.LargeMapType)
	advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
	assert.Equal(t, 1, deaths)
	assert.True(t, state.PartyState.IsAllPartyDead())
	assert.Equal(t, references.Position{X: 50, Y: 52}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Equal(t, 1, state.beggarKarmaCounter)
	assert.False(t, callbacks.InputBlocked)
}

func TestWaterfallContinuationShutdownAtEveryDescentStage(t *testing.T) {
	tests := []struct {
		name  string
		ready func(*GameState, *MockSystemCallbacks) bool
	}{
		{name: "queued before first step", ready: func(*GameState, *MockSystemCallbacks) bool { return true }},
		{name: "first forced step", ready: func(state *GameState, _ *MockSystemCallbacks) bool {
			return state.MapState.PlayerLocation.Position.Y == 137
		}},
		{name: "second step and glide", ready: func(state *GameState, _ *MockSystemCallbacks) bool {
			return state.MapState.PlayerLocation.Position.Y == 138
		}},
		{name: "damage flash", ready: func(_ *GameState, callbacks *MockSystemCallbacks) bool {
			return len(callbacks.ScreenFlashIntensities) > 1
		}},
		{name: "fade out", ready: func(state *GameState, callbacks *MockSystemCallbacks) bool {
			return state.MapState.PlayerLocation.Floor == references.MainFloor && len(callbacks.MapFadeLevels) > 1
		}},
		{name: "fade in on new plane", ready: func(state *GameState, _ *MockSystemCallbacks) bool {
			return state.MapState.PlayerLocation.Floor == references.Basement
		}},
		{name: "second descent", ready: func(state *GameState, _ *MockSystemCallbacks) bool {
			return state.MapState.PlayerLocation.Position.Y == 139
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := references.Position{X: 54, Y: 136}
			state, callbacks := newWaterfallContinuationFixture(t, start)
			setWaterfallContinuationChain(state, references.MainFloor, start, 1)
			setWaterfallContinuationChain(state, references.Basement, references.Position{X: 54, Y: 138}, 2)
			state.CompleteSurfacePassCycle(references.LargeMapType)
			advanceWaterfallUntil(t, state, func() bool { return test.ready(state, callbacks) })
			position, hp, calls, date := state.MapState.PlayerLocation, state.PartyState.GetAvatar().GetCurrentHp(), state.GetRNG().Calls(), state.DateTime
			state.EventQueueManager.Shutdown()
			state.EventQueueManager.Shutdown()
			for range 10 {
				state.ElapsedMs += 3000
				state.EventQueueManager.Update()
			}
			assert.Equal(t, position, state.MapState.PlayerLocation)
			assert.Equal(t, hp, state.PartyState.GetAvatar().GetCurrentHp())
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.Equal(t, date, state.DateTime)
			assert.False(t, state.IsWaterfallHazardActive())
			assert.False(t, callbacks.InputBlocked)
			assert.Equal(t, 1, callbacks.UnblockInputCalls)
			if len(callbacks.MapFadeLevels) > 0 {
				assert.Zero(t, callbacks.MapFadeLevels[len(callbacks.MapFadeLevels)-1])
			}
			if len(callbacks.ScreenFlashIntensities) > 0 {
				assert.Zero(t, callbacks.ScreenFlashIntensities[len(callbacks.ScreenFlashIntensities)-1])
			}
		})
	}
}

func TestWaterfallContinuationForeignRelocationOrTerminalOwnerStopsWork(t *testing.T) {
	for _, test := range []struct {
		name string
		stop func(*GameState)
	}{
		{name: "position replaced", stop: func(state *GameState) { state.SetPlayerPosition(references.Position{X: 50, Y: 90}) }},
		{name: "map replaced", stop: func(state *GameState) {
			state.TransitionToLocation(references.BritanniaUnderworld, references.Basement,
				references.Position{X: 50, Y: 90}, TransitionFullLocation, false)
		}},
		{name: "terminal handoff", stop: func(state *GameState) { state.finaleHandoffPending = true }},
		{name: "combat takes ownership", stop: func(state *GameState) {
			enemy := addOverworldTestEnemy(t, state, references.MonsterOrc, references.Position{X: 51, Y: 51})
			require.NoError(t, state.StartOverworldCombatEncounter(enemy))
		}},
		{name: "external defeat", stop: func(state *GameState) { state.PartyState.GetAvatar().SubtractHp(1000) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			start := references.Position{X: 50, Y: 50}
			state, callbacks := newWaterfallContinuationFixture(t, start)
			setWaterfallContinuationChain(state, references.MainFloor, start, 3)
			state.CompleteSurfacePassCycle(references.LargeMapType)
			advanceWaterfallUntil(t, state, func() bool { return state.MapState.PlayerLocation.Position.Y == 51 })
			test.stop(state)
			position, hp, calls := state.MapState.PlayerLocation, state.PartyState.GetAvatar().GetCurrentHp(), state.GetRNG().Calls()
			advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
			assert.Equal(t, position, state.MapState.PlayerLocation)
			assert.Equal(t, hp, state.PartyState.GetAvatar().GetCurrentHp())
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.False(t, callbacks.InputBlocked)
		})
	}
}

func TestWaterfallContinuationDeferredAndPendingOwnership(t *testing.T) {
	for _, test := range []struct {
		name           string
		cancelDeferred bool
	}{
		{name: "cancel callback-deferred cascade", cancelDeferred: true},
		{name: "cancel pending cascade"},
	} {
		t.Run(test.name, func(t *testing.T) {
			start := references.Position{X: 50, Y: 50}
			state, callbacks := newWaterfallContinuationFixture(t, start)
			setWaterfallContinuationChain(state, references.MainFloor, start, 2)
			state.EventQueueManager.Workflow("waterfall_test_owner").Priority(event_queue.PriorityHigh).
				Sequential().BlockInput().First(func() {
				state.CompleteSurfacePassCycle(references.LargeMapType)
				require.True(t, state.IsWaterfallHazardActive(), "callback-deferred Start reserves ownership immediately")
				charged := state.DateTime
				state.CompleteSurfacePassCycle(references.LargeMapType)
				require.True(t, state.ProcessWaterfallHazard(&start))
				assert.Equal(t, charged, state.DateTime)
				if test.cancelDeferred {
					state.EventQueueManager.Shutdown()
				}
			}).Wait(1000).Start()
			state.EventQueueManager.Update()
			assert.Equal(t, start, state.MapState.PlayerLocation.Position, "pending cascade must await its workflow lane")
			assert.Equal(t, !test.cancelDeferred, callbacks.InputBlocked)
			state.EventQueueManager.Shutdown()
			assert.False(t, state.IsWaterfallHazardActive())
			assert.False(t, callbacks.InputBlocked)
			assert.Zero(t, state.GetRNG().Calls())
			assert.Equal(t, uint16(1000), state.PartyState.GetAvatar().GetCurrentHp())
		})
	}
}

func TestWaterfallContinuationFinalizingOwnershipDoesNotRecharge(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(map[bool]string{false: "release held key", true: "shutdown while awaiting release"}[shutdown], func(t *testing.T) {
			start := references.Position{X: 50, Y: 50}
			state, callbacks := newWaterfallContinuationFixture(t, start)
			setWaterfallContinuationChain(state, references.MainFloor, start, 1)
			state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, start, references.MainFloor)
			held := true
			state.SystemCallbacks.Visual.IsAnyKeyPressed = func() bool { return held }
			state.CompleteSurfacePassCycle(references.LargeMapType)
			advanceWaterfallUntil(t, state, func() bool { return state.outdoorMountedActorSkip })
			require.True(t, state.IsWaterfallHazardActive(), "name stays reserved until the held key releases")
			require.True(t, callbacks.InputBlocked)
			date, calls := state.DateTime, state.GetRNG().Calls()
			state.CompleteSurfacePassCycle(references.LargeMapType)
			require.True(t, state.ProcessWaterfallHazard(&start))
			assert.Equal(t, date, state.DateTime)
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.True(t, state.outdoorMountedActorSkip)
			if shutdown {
				state.EventQueueManager.Shutdown()
			} else {
				held = false
				state.EventQueueManager.Update()
			}
			assert.False(t, state.IsWaterfallHazardActive())
			assert.False(t, callbacks.InputBlocked)
			assert.Equal(t, 1, callbacks.UnblockInputCalls)
			assert.Equal(t, date, state.DateTime)
			assert.Equal(t, calls, state.GetRNG().Calls())
		})
	}
}

func TestWaterfallContinuationKeepsOrdinaryPoisonAndFoodSeparate(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, _ := newWaterfallContinuationFixture(t, start)
	control, _ := newWaterfallContinuationFixture(t, start)
	for _, candidate := range []*GameState{state, control} {
		candidate.DateTime.Hour, candidate.DateTime.Minute = 5, 59
		candidate.lastProcessedHour = 5
		candidate.PartyState.GetAvatar().SetStatus(party_state.Poisoned)
	}
	setWaterfallContinuationChain(state, references.MainFloor, start, 3)
	control.CompleteSurfacePassCycle(references.LargeMapType)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	ordinaryHP := control.PartyState.GetAvatar().GetCurrentHp()
	assert.Equal(t, ordinaryHP, state.PartyState.GetAvatar().GetCurrentHp(), "player update precedes any fall")
	advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
	assert.Equal(t, ordinaryHP-3, state.PartyState.GetAvatar().GetCurrentHp(), "poison cannot repeat per descent")
	assert.Equal(t, control.PartyState.Inventory.Provisions.Food.Get(), state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, control.DateTime, state.DateTime)
	assert.Equal(t, control.ActiveDurationEffect, state.ActiveDurationEffect)
	assert.Equal(t, control.beggarKarmaCounter, state.beggarKarmaCounter)
}

func TestWaterfallContinuationPerFallOrderedRNGAndInactiveMembers(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, _ := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 4)
	for index, dexterity := range []uint8{30, 1, 15} {
		member := state.PartyState.GetCharacter(index)
		member.PartyMap = party_state.PartyMapActive
		member.SetMaxHp(1000)
		member.SetHp(1000)
		member.Dexterity = dexterity
	}
	dead := state.PartyState.GetCharacter(3)
	dead.PartyMap = party_state.PartyMapActive
	dead.SetStatus(party_state.Dead)
	deadHP := dead.GetCurrentHp()
	inn := state.PartyState.GetCharacter(4)
	innHP := inn.GetCurrentHp()
	wantedHP := []uint16{1000, 1000, 1000}
	expected := beradrng.New(123)
	for range 4 {
		var targets []int
		for index, dexterity := range []int{30, 1, 15} {
			roll := max(expected.Intn(originalD30SourceValueCount)/2, 1)
			if roll >= dexterity {
				targets = append(targets, index)
			}
		}
		for _, index := range targets {
			expected.Intn(1) // Existing fixed damage request after the full DEX sweep.
			wantedHP[index]--
		}
	}
	state.CompleteSurfacePassCycle(references.LargeMapType)
	advanceWaterfallUntil(t, state, func() bool { return !state.IsWaterfallHazardActive() })
	for index, hp := range wantedHP {
		assert.Equal(t, hp, state.PartyState.GetCharacter(index).GetCurrentHp())
	}
	assert.Equal(t, deadHP, dead.GetCurrentHp())
	assert.Equal(t, innHP, inn.GetCurrentHp())
	assert.Equal(t, expected.Calls(), state.GetRNG().Calls())
	assert.Equal(t, expected.Intn(1000), state.GetRNG().Intn(1000), "retain the next random value, not just a call count")
}

func TestWaterfallContinuationDoesNotReplaceAlreadyOwedOutdoorWork(t *testing.T) {
	start := references.Position{X: 50, Y: 50}
	state, _ := newWaterfallContinuationFixture(t, start)
	setWaterfallContinuationChain(state, references.MainFloor, start, 2)
	phase := &outdoorActorContinuation{beforeSlot: 7, world: references.OVERWORLD, ready: true}
	state.outdoorActorContinuation = phase
	state.isSailing = true
	// This caller already owns the actor phase; a waterfall may stop sailing but
	// must not clear that retained cursor or manufacture another phase.
	state.completeSurfaceCycleWithActorPhase(references.LargeMapType, DefaultLargeMapMinutesPerTurn, false)
	assert.False(t, state.IsSailing())
	assert.Same(t, phase, state.outdoorActorContinuation)
	advanceWaterfallUntil(t, state, func() bool { return !state.EventQueueManager.HasWorkflow(waterfallWorkflowName) })
	assert.Same(t, phase, state.outdoorActorContinuation)
	assert.True(t, state.IsWaterfallHazardActive(), "retain the cascade until its caller's actor cursor drains")
	assert.Equal(t, 7, phase.beforeSlot)
	assert.Equal(t, byte(2), state.DateTime.Minute)
	assert.Equal(t, 1, state.beggarKarmaCounter)
	state.EventQueueManager.Shutdown()
	assert.Nil(t, state.activeWaterfall)
}

func TestWaterfallContinuationWaitsForAlreadyOwedRoughSeasAndStoppedSailing(t *testing.T) {
	for _, test := range []struct {
		name string
		hp   uint16
	}{
		{name: "living party continues the cascade", hp: 1000},
		{name: "rough-sea defeat ends before first descent", hp: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			start := references.Position{X: 50, Y: 50}
			state, callbacks := newWaterfallContinuationFixture(t, start)
			setWaterfallContinuationChain(state, references.MainFloor, start, 2)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &start, indexes.Water1)
			state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, start, references.MainFloor)
			state.PartyState.GetAvatar().SetHp(test.hp)
			deaths := 0
			state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deaths++ }
			// Model an automatic-travel owner still draining its current callback
			// after a vehicle change. The hazard must stop it without discarding an
			// already admitted rough-sea hit or charging another ordinary cycle.
			state.isSailing = true
			state.EventQueueManager.Workflow("automatic_sailing").Priority(event_queue.PriorityNormal).
				Sequential().First(func() {}).While(state.IsSailing, func() {}, 1).Start()
			state.ProcessSkiffDeepWaterDamage(&start)
			require.True(t, state.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName))
			state.CompleteSurfaceMovementCycle(references.LargeMapType, true)
			charged := state.DateTime
			require.True(t, state.IsWaterfallHazardActive())
			assert.False(t, state.IsSailing())
			assert.Equal(t, test.hp, state.PartyState.GetAvatar().GetCurrentHp(), "admission must not replay pending damage")
			advanceWaterfallUntil(t, state, func() bool {
				if state.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName) {
					assert.Equal(t, start, state.MapState.PlayerLocation.Position, "rough seas finish before the first forced step")
					assert.Zero(t, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
				}
				return !state.EventQueueManager.HasActiveWorkflows()
			})
			assert.Equal(t, 1, waterfallMessageCount(callbacks, "Rough seas!"))
			assert.Equal(t, charged, state.DateTime)
			assert.Equal(t, byte(2), charged.Minute)
			assert.Equal(t, 1, state.beggarKarmaCounter)
			assert.Equal(t, 7, state.ActiveDurationEffect.TurnsRemaining)
			assert.False(t, callbacks.InputBlocked)
			assert.False(t, state.IsWaterfallHazardActive())
			assert.False(t, state.EventQueueManager.HasWorkflow("automatic_sailing"))
			assert.Zero(t, state.pendingSkiffDeepWaterHits)
			expected := beradrng.New(123)
			roughDamage := expected.Intn(8) + 1
			if test.hp == 1 {
				assert.Equal(t, 1, deaths)
				assert.Equal(t, start, state.MapState.PlayerLocation.Position)
				assert.Zero(t, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
				assert.True(t, state.PartyState.IsAllPartyDead())
			} else {
				for range 2 {
					expected.Intn(originalD30SourceValueCount)
					expected.Intn(1)
				}
				assert.Zero(t, deaths)
				assert.Equal(t, uint16(998-roughDamage), state.PartyState.GetAvatar().GetCurrentHp())
				assert.Equal(t, references.Position{X: 50, Y: 54}, state.MapState.PlayerLocation.Position)
				assert.Equal(t, 2, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
			}
			assert.Equal(t, expected.Calls(), state.GetRNG().Calls(), "one rough-sea hit followed by one DEX/damage sweep per fall")
		})
	}
}
