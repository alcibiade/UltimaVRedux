package main

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestTelescopeDaytimeUsesSourceBoundaries(t *testing.T) {
	for hour := range uint8(24) {
		assert.Equal(t, hour >= 6 && hour <= 17, isTelescopeDaytime(hour), "hour %d", hour)
	}
}

func TestTownTelescopeLookCompletesCommandExactlyOnce(t *testing.T) {
	state, callbacks := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithRandomSeed(443).
		WithSystemCallbacks().
		Build()
	state.DateTime.Hour = 18
	scene := newSurfaceCompletionTestScene(state)
	state.SystemCallbacks.Flow.Observation.ShowTelescopeInteraction = scene.ShowTelescopeInteraction
	target := references.Right.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, target, indexes.Telescope)
	callbacks.Reset()
	scene.secondaryKeyState = LookDirectionInput
	scene.keyboard.SimulateKeyPress(ebiten.KeyRight)

	scene.smallMapHandleSecondaryInput()
	waitForAstronomyDialog(t, scene)

	assert.Empty(t, callbacks.TimeAdvanced())
	require.NotPanics(t, state.EventQueueManager.Shutdown)
	assert.Equal(t, []int{game_state.DefaultSmallMapMinutesPerTurn}, callbacks.TimeAdvanced())
	state.EventQueueManager.Shutdown()
	assert.Len(t, callbacks.TimeAdvanced(), 1)
}

func TestTelescopeSunDamageTargetsActivePlayerOrLivingFallback(t *testing.T) {
	tests := []struct {
		name         string
		activePlayer int
		killAvatar   bool
		wantTarget   int
	}{
		{
			name: "active player", activePlayer: 1, wantTarget: 1,
		},
		{
			name: "first living fallback", activePlayer: -1, killAvatar: true, wantTarget: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := game_state.NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithRandomSeed(443).
				WithSystemCallbacks().
				Build()
			state.DateTime.Hour = 6
			if test.killAvatar {
				state.PartyState.GetCharacter(0).SetStatus(party_state.Dead)
			}
			if test.activePlayer >= 0 {
				require.True(t, state.SetActivePlayer(test.activePlayer))
			}
			hpBefore := make([]uint16, 2)
			for index := range hpBefore {
				hpBefore[index] = state.PartyState.GetCharacter(index).GetCurrentHp()
			}
			scene := newSurfaceCompletionTestScene(state)

			scene.StartTelescopeWorkflow()
			drainNamedSurfaceUseWorkflow(t, scene, "telescope_workflow")

			for index := range hpBefore {
				want := hpBefore[index]
				if index == test.wantTarget {
					want--
				}
				assert.Equal(t, want, state.PartyState.GetCharacter(index).GetCurrentHp(), "party slot %d", index)
			}
		})
	}
}

func TestSurfaceKeyURoutesSpyglassThroughCallbackAndCompletesOnce(t *testing.T) {
	scene, callbacks := newSurfaceUseOwnershipSceneAt(
		t,
		references.BritanniaUnderworld,
		references.FloorNumber(references.OVERWORLD),
		references.Position{X: 0x4e, Y: 0xab},
	)
	state := scene.gameState
	state.PartyState.Inventory.QuestItems.Set(references.Spyglass, 1)
	state.DateTime.Hour = 19
	expectedTime := state.DateTime
	expectedTime.Advance(game_state.DefaultLargeMapMinutesPerTurn)
	callbackRequests := 0
	state.SystemCallbacks.Flow.Observation.ShowSpyglassInteraction = func(complete func()) {
		callbackRequests++
		scene.ShowSpyglassInteraction(complete)
	}
	var feedback []string
	state.SystemCallbacks.UI.ShowActionFeedback = func(message string, _ bool) {
		feedback = append(feedback, message)
	}
	callbacks.Reset()

	scene.largeMapInputHandler(ebiten.KeyU)
	itemDialog := waitForCombatItemDialog(t, scene)
	itemDialog.HandleEnter()
	drainUseSelectionWorkflow(t, scene)
	astronomyDialog := waitForAstronomyDialog(t, scene)

	assert.Equal(t, 1, callbackRequests)
	assert.Contains(t, feedback, "Thou dost see the night sky!")
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Spyglass))

	require.NotNil(t, astronomyDialog)
	require.NotPanics(t, state.EventQueueManager.Shutdown)

	assert.Equal(t, []int{game_state.DefaultLargeMapMinutesPerTurn}, callbacks.TimeAdvanced())
	assert.Equal(t, expectedTime, state.DateTime)
	assert.False(t, callbacks.InputBlocked)
	assert.Equal(t, callbacks.BlockInputCalls, callbacks.UnblockInputCalls)
	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Zero(t, scene.dialogStack.Count())

	require.NotPanics(t, state.EventQueueManager.Shutdown)
	assert.Len(t, callbacks.TimeAdvanced(), 1)
}

func TestSurfaceKeyURoutesSextantOutputAndCompletesOnce(t *testing.T) {
	scene, callbacks := newSurfaceUseOwnershipSceneAt(
		t,
		references.BritanniaUnderworld,
		references.FloorNumber(references.OVERWORLD),
		references.Position{X: 0x4e, Y: 0xab},
	)
	state := scene.gameState
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 19
	expectedTime := state.DateTime
	expectedTime.Advance(game_state.DefaultLargeMapMinutesPerTurn)
	callbacks.Reset()

	scene.largeMapInputHandler(ebiten.KeyU)
	itemDialog := waitForCombatItemDialog(t, scene)
	itemDialog.HandleEnter()
	drainUseSelectionWorkflow(t, scene)

	assert.Contains(t, callbacks.Messages, "Position:")
	assert.Contains(t, callbacks.Messages, "K'L\", E'O\"")
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sextant))
	assert.Equal(t, []int{game_state.DefaultLargeMapMinutesPerTurn}, callbacks.TimeAdvanced())
	assert.Equal(t, expectedTime, state.DateTime)
	assert.False(t, callbacks.InputBlocked)
	assert.Equal(t, callbacks.BlockInputCalls, callbacks.UnblockInputCalls)
	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Zero(t, scene.dialogStack.Count())

	for range 3 {
		state.EventQueueManager.Update()
	}
	assert.Len(t, callbacks.TimeAdvanced(), 1)
}

func TestSpyglassAcceptsBritanniaAtSourceNightBoundaries(t *testing.T) {
	for _, hour := range []uint8{0, 5, 19, 23} {
		t.Run(fmt.Sprintf("hour %d", hour), func(t *testing.T) {
			state, _ := game_state.NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithFloor(references.FloorNumber(references.OVERWORLD)).
				WithSystemCallbacks().
				Build()
			state.PartyState.Inventory.QuestItems.Set(references.Spyglass, 1)
			state.DateTime.Hour = hour
			var feedback []string
			state.SystemCallbacks.UI.ShowActionFeedback = func(message string, _ bool) {
				feedback = append(feedback, message)
			}
			scene := newSurfaceCompletionTestScene(state)
			completed := 0

			scene.startSpyglassWorkflow(func() { completed++ })
			waitForAstronomyDialog(t, scene)
			state.EventQueueManager.Shutdown()

			assert.Equal(t, 1, completed)
			assert.Contains(t, feedback, "Thou dost see the night sky!")
			assert.Zero(t, scene.dialogStack.Count())
		})
	}
}

func TestSpyglassUsesSourceDayAndLocationFeedback(t *testing.T) {
	t.Run("18 has no stars", func(t *testing.T) {
		state, _ := game_state.NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithFloor(references.FloorNumber(references.OVERWORLD)).
			WithSystemCallbacks().
			Build()
		state.PartyState.Inventory.QuestItems.Set(references.Spyglass, 1)
		state.DateTime.Hour = 18
		var feedback []string
		state.SystemCallbacks.UI.ShowActionFeedback = func(message string, _ bool) {
			feedback = append(feedback, message)
		}
		scene := newSurfaceCompletionTestScene(state)
		completed := 0

		scene.startSpyglassWorkflow(func() { completed++ })
		drainNamedSurfaceUseWorkflow(t, scene, "spyglass_workflow")

		assert.Equal(t, 1, completed)
		assert.Contains(t, feedback, "No stars!")
	})

	t.Run("basement rejected", func(t *testing.T) {
		state, _ := game_state.NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithFloor(-1).
			WithSystemCallbacks().
			Build()
		state.PartyState.Inventory.QuestItems.Set(references.Spyglass, 1)
		var feedback []string
		state.SystemCallbacks.UI.ShowActionFeedback = func(message string, _ bool) {
			feedback = append(feedback, message)
		}
		scene := newSurfaceCompletionTestScene(state)
		completed := 0

		scene.startSpyglassWorkflow(func() { completed++ })

		assert.Equal(t, 1, completed)
		assert.Contains(t, feedback, "Not here!")
		assert.False(t, state.EventQueueManager.HasWorkflow("spyglass_workflow"))
	})
}

func waitForAstronomyDialog(t *testing.T, scene *GameScene) *widgets.AstronomyDialog {
	t.Helper()
	for range 30 {
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
		if dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.AstronomyDialog); ok {
			return dialog
		}
	}
	t.Fatal("spyglass workflow did not show astronomy")

	return nil
}
