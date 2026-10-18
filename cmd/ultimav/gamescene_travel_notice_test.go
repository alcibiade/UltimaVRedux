package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTravelNoticeDoesNotOwnMovementAfterTransition(t *testing.T) {
	for limit := range 4 {
		for _, travel := range []string{"enter", "exit", "waterfall"} {
			t.Run(fmt.Sprintf("%s/cap-%d", travel, limit), func(t *testing.T) {
				scene, held, clock, update := newReaderInputClockScene(t)
				state := scene.gameState
				wireFeedbackOutputForTest(scene)
				setActionPopupLimitForTest(scene, limit)
				state.SystemCallbacks.Visual.IsAnyKeyPressed = scene.IsAnyKeyPressed
				state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
				state.PartyState.ForEachCharacter(func(_ int, member *party_state.PlayerCharacter) { member.Dexterity = 99 })
				var notice string
				switch travel {
				case "enter":
					notice = state.GameReferences.LocationReferences.GetLocationReference(references.Britain).EnteringText
					state.StartEnterLocationWorkflow(state.GameReferences.LocationReferences.GetLocationReference(references.Britain))
				case "exit":
					notice = strings.TrimSpace(state.GameReferences.LocationReferences.GetLocationReference(references.Britain).GetExitingText())
					state.TransitionToLocation(references.Britain, 0, references.Position{X: 15, Y: 15}, game_state.TransitionFullLocation, false)
					state.EventQueueManager.Workflow("test_exit_notice").Priority(event_queue.PriorityNormal).
						Sequential().BlockInput().First(func() {}).ThenSequence(state.ThenExitLocationTransition()).Start()
				case "waterfall":
					notice = "F-A-L-L-S!!!"
					position := state.MapState.PlayerLocation.Position
					for step := range 4 {
						cell := references.Position{X: position.X, Y: position.Y + references.Coordinate(step)}
						tile := indexes.SpriteIndex(indexes.Grass)
						if step == 1 {
							tile = indexes.Waterfall1
						}
						state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &cell, tile)
					}
					require.True(t, state.ProcessWaterfallHazard(&position))
				}
				held[ebiten.KeyRight] = true
				for range 100 {
					*clock += 10
					update()
				}
				landing := state.MapState.PlayerLocation.Position
				delete(held, ebiten.KeyRight)
				for range 10 {
					*clock += 10
					update()
				}
				require.False(t, state.EventQueueManager.HasActiveWorkflows(), "no presentation-only dwell remains after travel")
				require.False(t, scene.inputBlocked)
				require.False(t, scene.dialogStack.HasModalDialog(), "travel cards do not own focus")
				require.Len(t, scene.transientActionFeedback, min(limit, 1), "the notice remains visible unless Off")
				assert.Equal(t, 1, strings.Count(scene.output.JournalHistory(), strings.TrimSpace(notice)))
				require.Equal(t, landing, state.MapState.PlayerLocation.Position, "the held arrow does not resume movement")
				// Isolate the next step from authored obstacles; its input route is production.
				next := landing.Right()
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &next, indexes.Grass)
				state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
				held[ebiten.KeyRight] = true
				*clock += keyPressDelay + 1
				update()
				require.Equal(t, next, state.MapState.PlayerLocation.Position)
				require.Len(t, scene.transientActionFeedback, min(limit, 1))
				assert.Equal(t, 1, strings.Count(scene.output.JournalHistory(), strings.TrimSpace(notice)))
			})
		}
	}
}
