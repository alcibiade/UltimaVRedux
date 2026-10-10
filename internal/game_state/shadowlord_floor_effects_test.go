package game_state

import (
	"fmt"
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shadowlordSourceFloor(state *GameState, floor references.FloorNumber, damage bool) []indexes.SpriteIndex {
	source := state.GetCurrentSmallLocationReference()
	terrain := make([]indexes.SpriteIndex, 0, references.XSmallMapTiles*references.YSmallMapTiles)
	rng := deterministic_random.New(uint64(state.DateTime.Day))
	for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			sprite := source.GetTileNumber(references.Position{X: x, Y: y}, floor)
			// Vegetation effects: candidate-only, Y-major, nonzero destroys.
			// See ref:SHADOW-0002.
			if damage && (sprite == indexes.Tree || sprite == indexes.WheatInField) && rng.Intn(8) != 0 {
				if sprite == indexes.Tree {
					sprite = indexes.DeadTree
				} else {
					sprite = indexes.PlowedField
				}
			}
			terrain = append(terrain, sprite)
		}
	}

	return terrain
}

func changeSmallMapTestFloor(t *testing.T, state *GameState, floor references.FloorNumber, direct bool) {
	t.Helper()
	if direct {
		if floor > state.MapState.PlayerLocation.Floor {
			require.True(t, state.GoUpOneFloor())
		} else {
			require.True(t, state.GoDownOneFloor())
		}
	} else {
		result := state.TransitionToLocation(state.MapState.PlayerLocation.Location, floor,
			state.MapState.PlayerLocation.Position, TransitionFloorOnly, false)
		assert.Empty(t, result.ShadowlordsPresent, "floor travel must not replay town-entry effects")
	}
	require.Equal(t, floor, state.MapState.PlayerLocation.Floor)
}

func TestShadowlordFloorEffectsRetainSameDayPattern(t *testing.T) {
	for _, lord := range []references.Shadowlord{references.Faulinei, references.Astaroth, references.Nosfentor} {
		for _, direct := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/direct%t", lord, direct), func(t *testing.T) {
				state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
					WithRandomSeed(12345).WithSystemCallbacks().Build()
				state.DateTime.Day = 9
				state.ShadowlordState.TownAssignments[lord] = references.Britain
				result := state.TransitionToLocation(references.Britain, 0, references.Position{X: 15, Y: 10}, TransitionFullLocation, false)
				require.Equal(t, []references.Shadowlord{lord}, result.ShadowlordsPresent)
				first := currentSmallMapTerrain(state)
				require.NotEqual(t, shadowlordSourceFloor(state, 0, false), first, "fixture must actually destroy vegetation")
				require.Equal(t, shadowlordSourceFloor(state, 0, true), first)
				controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
				require.True(t, ok)
				units, err := state.captureMapUnits()
				require.NoError(t, err)
				calls, date := state.GetRNG().Calls(), state.DateTime
				messages := slices.Clone(callbacks.Messages)
				presenceCalls := len(callbacks.ShadowlordPresenceEffectsCalled)
				// Y=4 is only an entry bypass, never a floor-reset presence check.
				state.SetPlayerPosition(references.Position{X: 15, Y: 4})
				for range 3 {
					changeSmallMapTestFloor(t, state, 1, direct)
					assert.Equal(t, shadowlordSourceFloor(state, 1, true), currentSmallMapTerrain(state))
					changeSmallMapTestFloor(t, state, 0, direct)
					assert.Equal(t, first, currentSmallMapTerrain(state), "no revival or compounded survivor destruction")
					assert.Same(t, controller, state.CurrentNPCAIController)
					assert.Equal(t, lord, state.ShadowlordState.AutomaticPresence)
					assert.Equal(t, 1, countShadowlordNPCs(controller))
				}
				after, err := state.captureMapUnits()
				require.NoError(t, err)
				require.Len(t, after, len(units))
				for i := range units {
					assert.Equal(t, units[i].OverriddenAI, after[i].OverriddenAI)
					assert.Equal(t, units[i].AIOverrideProvenance, after[i].AIOverrideProvenance)
				}
				assert.Equal(t, calls, state.GetRNG().Calls())
				assert.Equal(t, date, state.DateTime)
				assert.Equal(t, messages, callbacks.Messages)
				assert.Len(t, callbacks.ShadowlordPresenceEffectsCalled, presenceCalls)
			})
		}
	}
}

func TestShadowlordFloorEffectsDestinationSourceCandidates(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().WithRandomSeed(12345).Build()
	state.DateTime.Day = 9
	destroyedTrees, destroyedCrops := 0, 0
	// Real DAT floors, never runtime-only injected candidates that a correct
	// rebuild would discard. Exercise arrival on every floor of the eight towns.
	for location := references.Moonglow; location <= references.NewMagincia; location++ {
		state.ShadowlordState.Reset()
		state.TransitionToLocation(location, 0, references.Position{X: 15, Y: 10}, TransitionFullLocation, false)
		// Isolate floor loading from town-entry spawn occupancy; the Britain
		// matrix above verifies actual physical entry for all three identities.
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		for _, floor := range state.GetCurrentSmallLocationReference().ListOfFloors {
			state.TransitionToLocation(location, floor, references.Position{X: 15, Y: 10}, TransitionFloorOnly, false)
			actual, source := currentSmallMapTerrain(state), shadowlordSourceFloor(state, floor, false)
			assert.Equal(t, shadowlordSourceFloor(state, floor, true), actual, "%s floor %d", location, floor)
			for i := range source {
				if source[i] == indexes.Tree && actual[i] == indexes.DeadTree {
					destroyedTrees++
				}
				if source[i] == indexes.WheatInField && actual[i] == indexes.PlowedField {
					destroyedCrops++
				}
			}
		}
	}
	require.Positive(t, destroyedTrees)
	require.Positive(t, destroyedCrops)
}

func TestShadowlordFloorEffectsAfterActualMidnightAndReentry(t *testing.T) {
	for _, lord := range []references.Shadowlord{references.Faulinei, references.Astaroth, references.Nosfentor} {
		t.Run(lord.String(), func(t *testing.T) { testShadowlordFloorEffectsAfterMidnight(t, lord) })
	}
}

func testShadowlordFloorEffectsAfterMidnight(t *testing.T, lord references.Shadowlord) {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().WithRandomSeed(12345).Build()
	state.DateTime.Day = 9
	state.ShadowlordState.TownAssignments[lord] = references.Britain
	state.TransitionToLocation(references.Britain, 0, references.Position{X: 15, Y: 10}, TransitionFullLocation, false)
	before := currentSmallMapTerrain(state)
	controller := state.CurrentNPCAIController
	state.DateTime.Hour, state.DateTime.Minute = 23, 59
	state.advanceElapsedGameplayMinutes(1, false)
	require.Equal(t, byte(10), state.DateTime.Day)
	require.NotEqual(t, references.Britain, state.ShadowlordState.TownAssignments[lord])
	require.Equal(t, lord, state.ShadowlordState.AutomaticPresence)
	assert.Equal(t, before, currentSmallMapTerrain(state), "midnight itself does not reload terrain")
	calls := state.GetRNG().Calls()
	for _, direct := range []bool{true, false} {
		changeSmallMapTestFloor(t, state, 1, direct)
		changeSmallMapTestFloor(t, state, 0, direct)
		assert.Equal(t, shadowlordSourceFloor(state, 0, true), currentSmallMapTerrain(state), "new day gets exactly one fresh-source pass")
		assert.Same(t, controller, state.CurrentNPCAIController)
	}
	assert.Equal(t, calls, state.GetRNG().Calls())
	state.TransitionToLocation(references.BritanniaUnderworld, 0, references.Position{X: 100, Y: 100}, TransitionFullLocation, false)
	state.TransitionToLocation(references.Britain, 0, references.Position{X: 15, Y: 10}, TransitionFullLocation, false)
	require.False(t, state.ShadowlordState.HasAutomaticPresence())
	assert.Equal(t, shadowlordSourceFloor(state, 0, false), currentSmallMapTerrain(state))
}

func TestShadowlordFloorEffectsRequireAutomaticPresence(t *testing.T) {
	for _, scenario := range []string{"absent", "jail entry", "ritual yell"} {
		t.Run(scenario, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().WithRandomSeed(12345).Build()
			location, entry := references.Britain, references.Position{X: 15, Y: 10}
			if scenario == "jail entry" {
				state.ShadowlordState.TownAssignments[references.Faulinei] = location
				entry.Y = 4
			}
			if scenario == "ritual yell" {
				location = references.Lycaeum
			}
			state.TransitionToLocation(location, 0, entry, TransitionFullLocation, false)
			if scenario == "ritual yell" {
				require.True(t, state.TrySummonShadowlord("FAULINEI"))
			}
			require.False(t, state.ShadowlordState.HasAutomaticPresence())
			controller, calls := state.CurrentNPCAIController, state.GetRNG().Calls()
			for _, direct := range []bool{true, false} {
				changeSmallMapTestFloor(t, state, 1, direct)
				assert.Equal(t, shadowlordSourceFloor(state, 1, false), currentSmallMapTerrain(state))
				changeSmallMapTestFloor(t, state, 0, direct)
				assert.Equal(t, shadowlordSourceFloor(state, 0, false), currentSmallMapTerrain(state))
			}
			assert.Same(t, controller, state.CurrentNPCAIController)
			assert.Equal(t, calls, state.GetRNG().Calls())
		})
	}
}

type shadowlordFloorResetObserver struct {
	ai.NPCAIController
	beforeReset func(bool)
}

func (o *shadowlordFloorResetObserver) HardResetNPCs(preserve bool) {
	o.beforeReset(preserve)
	o.NPCAIController.HardResetNPCs(preserve)
}

func TestShadowlordFloorEffectsPrecedeNPCReset(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().WithRandomSeed(12345).Build()
	state.DateTime.Day = 9
	state.ShadowlordState.TownAssignments[references.Astaroth] = references.Britain
	state.TransitionToLocation(references.Britain, 0, references.Position{X: 15, Y: 10}, TransitionFullLocation, false)
	calls := 0
	state.CurrentNPCAIController = &shadowlordFloorResetObserver{
		NPCAIController: state.CurrentNPCAIController,
		beforeReset: func(preserve bool) {
			calls++
			require.True(t, preserve)
			assert.Equal(t, shadowlordSourceFloor(state, state.MapState.PlayerLocation.Floor, true), currentSmallMapTerrain(state),
				"NPC repositioning must observe the already-damaged destination floor")
		},
	}
	require.True(t, state.GoUpOneFloor())
	require.True(t, state.GoDownOneFloor())
	assert.Equal(t, 2, calls)
}
