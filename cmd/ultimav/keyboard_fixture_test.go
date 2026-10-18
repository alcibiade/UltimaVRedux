package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestSceneUnscriptedModifierBlocksEscapeMenuFirstNavigation(t *testing.T) {
	for _, polluted := range []bool{true, false} {
		t.Run(boolLabel(polluted), func(t *testing.T) {
			scene := newEscapeMenuTestScene()
			if polluted {
				scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool {
					return key == ebiten.KeyControlLeft
				})
			}
			scene.DoEscapeMenu(NewGame(scene))
			require.Equal(t, polluted, scene.keyboard.IsWaitingForRelease(),
				"menu opening must not capture unscripted desktop keys in synthetic fixtures")
			require.Equal(t, !polluted, scene.keyboard.TryToRegisterKeyPress(ebiten.KeyDown))
		})
	}
}

func TestSceneUnscriptedControlPreemptsSimulatedGuardAnswer(t *testing.T) {
	for _, polluted := range []bool{true, false} {
		t.Run(boolLabel(polluted), func(t *testing.T) {
			fixture := newGuardTributeFixture(t, references.Britain, 1)
			t.Cleanup(fixture.state.EventQueueManager.Shutdown)
			if polluted {
				fixture.scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool {
					return key == ebiten.KeyControlRight
				})
			}
			fixture.scene.StartTalkWorkflow(references.Right)
			dialog := waitForGuardYesNo(t, fixture)
			fixture.scene.keyboard.SimulateKeyPress(ebiten.KeyY)
			dialog.Update()
			fixture.scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
			dialog.Update()
			if polluted {
				require.Empty(t, dialog.Response(), "held Control takes TextInput's shortcut branch before scripted Y")
				require.False(t, dialog.InputComplete())
			} else {
				require.Equal(t, "Yes", dialog.Response())
				require.True(t, dialog.InputComplete())
			}
		})
	}
}

func TestSceneUnscriptedShiftPreemptsSaveOverwriteSelection(t *testing.T) {
	for _, polluted := range []bool{true, false} {
		t.Run(boolLabel(polluted), func(t *testing.T) {
			scene := newEscapeMenuTestScene()
			if polluted {
				scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool {
					return key == ebiten.KeyShiftLeft
				})
			}
			capture := &playerSaveTestScene{
				playthrough: playthroughMetadata{id: testPlaythroughID}, snapshot: validCommandTestSnapshot(),
			}
			repository := &playerSaveTestRepository{
				summaries: []player_save.SlotSummary{{Slot: player_save.Slot02, Status: player_save.SlotReady, Revision: "reviewed"}},
			}
			game := NewGame(capture)
			game.ConfigurePlayerSaveRepository(repository, nil)
			activateEscapeMenuAction(t, scene, game, 1)
			browser := requireEscapeMenuDialog[*widgets.SaveSlotBrowser](t, scene)
			pressEscapeMenuWidgetKey(scene, browser, ebiten.KeyDown)
			pressEscapeMenuWidgetKey(scene, browser, ebiten.KeyEnter)
			if polluted {
				require.Same(t, browser, scene.dialogStack.PeekTopModalDialog(),
					"unscripted Shift changes Down to unsupported PageDown, leaving no overwrite selection")
			} else {
				requireEscapeMenuDialog[*widgets.SaveDescriptionEntryModal](t, scene)
			}
			require.Zero(t, capture.captures, "selection alone must not save")
			require.Empty(t, repository.saves)
		})
	}
}
