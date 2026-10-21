package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestActionFeedbackAllCapsShareAdmissionAndRetainOutput(t *testing.T) {
	for limit := range 4 {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			scene, state := newTransientFeedbackTestScene(t, references.Britain)
			setActionPopupLimitForTest(scene, limit)
			_, scene.output = newMainOutput(1280)
			modal := &countingDialog{}
			scene.dialogStack.PushModalDialog(modal)
			var admitted []*transientActionFeedback
			for index := range 6 {
				if index%2 == 0 {
					scene.ShowActionFeedback("same result", true)
				} else {
					// Search's caller records its result; the adapter owns only a card.
					scene.output.AddText("search result")
					scene.ShowSearchFeedback("search result")
				}
				require.Len(t, scene.transientActionFeedback, min(limit, index+1))
				require.Equal(t, min(limit, index+1)+1, scene.dialogStack.Count())
				if limit > 0 {
					admitted = append(admitted, scene.transientActionFeedback[len(scene.transientActionFeedback)-1])
				}
			}
			for index, card := range admitted {
				retained := index >= len(admitted)-limit
				assert.Equal(t, retained, scene.dialogStack.ContainsWidgetInstance(card.dialog))
				assert.Equal(t, retained, card.dialog.IsVisible())
			}
			history := scene.output.JournalHistory()
			assert.Equal(t, 3, strings.Count(history, "same result"), "identical events are not deduplicated")
			assert.Equal(t, 3, strings.Count(history, "search result"))
			scene.ShowActionFeedback("presentation only", false)
			assert.Equal(t, history, scene.output.JournalHistory())
			assert.Same(t, modal, scene.dialogStack.PeekTopModalDialog())

			state.ElapsedMs += 10_000
			scene.updateTransientActionFeedback()
			assert.Empty(t, scene.transientActionFeedback)
			assert.Equal(t, 1, scene.dialogStack.Count(), "only the unrelated modal remains")
			assert.False(t, scene.transientActionFeedbackLayoutSet)
			assert.Equal(t, history, scene.output.JournalHistory())
		})
	}
}

func TestActionFeedbackLiveCapReductionAcrossLifecycleAndModalCoverage(t *testing.T) {
	phases := []struct {
		age   int64
		phase transientFeedbackPhase
	}{
		{0, transientFeedbackFadingIn},
		{transientFeedbackMoveDurationMs + widgets.ActionFeedbackFadeInMs + 1, transientFeedbackHolding},
		{transientFeedbackMoveDurationMs + widgets.ActionFeedbackDisplayMs + 1, transientFeedbackFadingOut},
	}
	for _, phase := range phases {
		for _, covered := range []bool{false, true} {
			for limit := range 4 {
				name := fmt.Sprintf("phase-%d/modal-%t/cap-%d", phase.phase, covered, limit)
				t.Run(name, func(t *testing.T) {
					scene, state := newChestFeedbackScopeTestScene(t)
					for index := range 3 {
						captureChestFeedbackForTest(scene, references.Position{X: references.Coordinate(5 + index), Y: 5}, "same words")
					}
					cards := append([]*transientActionFeedback(nil), scene.transientActionFeedback...)
					backing := scene.transientActionFeedback
					state.ElapsedMs = phase.age
					scene.updateTransientActionFeedback()
					scene.updateNonBlockingDialogs()
					for _, card := range cards {
						require.Equal(t, phase.phase, card.phase)
					}
					modalCount := 0
					if covered {
						scene.dialogStack.PushModalDialog(&countingDialog{})
						modalCount = 1
					}
					setActionPopupLimitForTest(scene, limit)
					// Same timestamp, no new message and no resize: only the applied cap changes.
					scene.updateTransientActionFeedback()
					require.Len(t, scene.transientActionFeedback, limit)
					require.Len(t, scene.combatChestFeedback, limit)
					assert.Equal(t, limit+modalCount, scene.dialogStack.Count())
					for index, card := range cards {
						retained := index >= len(cards)-limit
						assert.Equal(t, retained, scene.dialogStack.ContainsWidgetInstance(card.dialog))
						assert.Equal(t, retained, card.dialog.IsVisible())
						if retained {
							assert.Same(t, card, scene.transientActionFeedback[index-(len(cards)-limit)])
						} else {
							assert.NotContains(t, scene.combatChestFeedback, card)
						}
					}
					for _, released := range backing[limit:] {
						assert.Nil(t, released, "compacted storage releases retired references")
					}
					if limit == 0 {
						assert.Nil(t, scene.transientActionFeedback)
						assert.False(t, scene.transientActionFeedbackLayoutSet)
						assert.False(t, scene.transientActionFeedbackClockSet)
					}
					setActionPopupLimitForTest(scene, 3)
					scene.updateTransientActionFeedback()
					assert.Len(t, scene.transientActionFeedback, limit, "raising cannot resurrect retired cards")
					assert.Len(t, scene.combatChestFeedback, limit)
				})
			}
		}
	}
}

func TestActionFeedbackLayoutReconcilesAppliedCap(t *testing.T) {
	for limit := range 4 {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			scene, state := newTransientFeedbackTestScene(t, references.Britain)
			for range 3 {
				scene.ShowSearchFeedback("same words")
			}
			cards := append([]*transientActionFeedback(nil), scene.transientActionFeedback...)
			setActionPopupLimitForTest(scene, limit)
			scene.layoutTransientActionFeedback(state.ElapsedMs, scene.currentTransientFeedbackLayout())
			require.Len(t, scene.transientActionFeedback, limit)
			for index, card := range cards {
				assert.Equal(t, index >= 3-limit, scene.dialogStack.ContainsWidgetInstance(card.dialog))
			}
			state.ElapsedMs += transientFeedbackMoveDurationMs
			scene.updateTransientActionFeedback()
			for index := 1; index < limit; index++ {
				older := scene.transientActionFeedback[index-1].dialog.PresentationBounds()
				newer := scene.transientActionFeedback[index].dialog.PresentationBounds()
				assert.LessOrEqual(t, newer.Max.Y+scene.transientActionFeedbackLayout.gap, older.Min.Y)
			}
		})
	}
}

func TestActionFeedbackLiveCapReconcilesThroughSceneUpdateUnderModal(t *testing.T) {
	for limit := range 4 {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			scene, _, clock, update := newReaderInputClockScene(t)
			wireFeedbackOutputForTest(scene)
			for range 3 {
				scene.ShowActionFeedback("recorded", true)
			}
			cards := append([]*transientActionFeedback(nil), scene.transientActionFeedback...)
			modal := &countingDialog{}
			scene.dialogStack.PushModalDialog(modal)
			setActionPopupLimitForTest(scene, limit)
			*clock += 1
			update()
			require.Len(t, scene.transientActionFeedback, limit)
			assert.Same(t, modal, scene.dialogStack.PeekTopModalDialog())
			assert.Equal(t, limit+1, scene.dialogStack.Count())
			for index, card := range cards {
				assert.Equal(t, index >= 3-limit, scene.dialogStack.ContainsWidgetInstance(card.dialog))
			}
			assert.Equal(t, 3, strings.Count(scene.output.JournalHistory(), "recorded"))
		})
	}
}

func TestActionFeedbackAllCapsExternalRemovalAndReplacementReleaseOwnership(t *testing.T) {
	for limit := range 4 {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			scene, _ := newChestFeedbackScopeTestScene(t)
			setActionPopupLimitForTest(scene, limit)
			for index := range 3 {
				identity := admitChestFeedbackIdentityForTest(scene, references.Position{X: references.Coordinate(index + 5), Y: 5})
				scene.captureCombatChestJimmyFeedback(func() (bool, *game_state.CombatChestIdentity) {
					scene.ShowActionFeedback("disarmed", false)

					return true, identity
				})
			}
			cards := append([]*transientActionFeedback(nil), scene.transientActionFeedback...)
			if limit > 0 {
				external := cards[len(cards)-1]
				require.True(t, scene.dialogStack.RemoveWidgetInstance(external.dialog))
				scene.updateTransientActionFeedback()
				assert.False(t, external.dialog.IsVisible())
				assert.NotContains(t, scene.combatChestFeedback, external)
			}
			modal := &countingDialog{}
			scene.dialogStack.PushModalDialog(modal)
			scene.PrepareForSceneReplacement()
			scene.PrepareForSceneReplacement()
			assert.Nil(t, scene.transientActionFeedback)
			assert.Empty(t, scene.combatChestFeedback)
			assert.False(t, scene.transientActionFeedbackLayoutSet)
			assert.Equal(t, 1, scene.dialogStack.Count())
			for _, card := range cards {
				assert.False(t, card.dialog.IsVisible())
			}
		})
	}
}

func TestActionFeedbackOffRejectsPrebuiltCardsWithoutHiddenOwnership(t *testing.T) {
	scene, state := newTransientFeedbackTestScene(t, references.Britain)
	scene.ShowActionFeedback("old", false)
	old := scene.transientActionFeedback[0].dialog
	setActionPopupLimitForTest(scene, 0)
	dialog := widgets.NewActionFeedbackDialog("receipt", widgets.PositionMapCenterLower)
	scene.admitTransientActionFeedback(dialog, state.ElapsedMs)
	assert.False(t, dialog.IsVisible())
	assert.False(t, old.IsVisible())
	assert.Empty(t, scene.transientActionFeedback)
	assert.Zero(t, scene.dialogStack.Count())
	assert.False(t, scene.transientActionFeedbackLayoutSet)
}

func TestActionFeedbackConfigurationCancelAndFailedApplyPreserveLiveCards(t *testing.T) {
	for _, cancel := range []bool{true, false} {
		t.Run(fmt.Sprintf("cancel-%t", cancel), func(t *testing.T) {
			configuration, path := initializeDialogConfigFile(t, "popup_capacity")
			scene, _ := newTransientFeedbackTestScene(t, references.Britain)
			scene.gameConfig = configuration
			for range 3 {
				scene.ShowActionFeedback("existing", false)
			}
			cards := append([]*transientActionFeedback(nil), scene.transientActionFeedback...)
			var dialog *ConfigurationDialog
			dialog = newConfigurationDialogForTab(&gameScreenPercents, newPresentationTestKeyboard(),
				func() { scene.dialogStack.RemoveWidgetInstance(dialog) }, configuration, nil, audioSettingsGameplayTab)
			scene.dialogStack.PushModalDialog(dialog)
			dialog.stagedGameplay.MaxVisibleActionPopups = 0
			scene.updateTransientActionFeedback()
			assert.Equal(t, cards, scene.transientActionFeedback, "staged Off is not live")
			if cancel {
				pressAudioSettingsKey(dialog, ebiten.KeyEscape)
				scene.updateTransientActionFeedback()
				assert.Equal(t, 3, configuration.GetMaxVisibleActionPopups())
				assert.Equal(t, cards, scene.transientActionFeedback)

				return
			}
			require.NoError(t, os.Remove(path))
			require.NoError(t, os.Mkdir(path, 0o700))
			pressAudioSettingsKey(dialog, ebiten.KeyEnter)
			require.True(t, dialog.IsVisible())
			require.NotEmpty(t, dialog.saveError)
			scene.updateTransientActionFeedback()
			assert.Equal(t, 3, configuration.GetMaxVisibleActionPopups())
			assert.Equal(t, cards, scene.transientActionFeedback, "failed persistence cannot evict")
			require.NoError(t, os.Remove(path))
			pressAudioSettingsKey(dialog, ebiten.KeyEnter)
			require.False(t, dialog.IsVisible())
			require.Zero(t, configuration.GetMaxVisibleActionPopups())
			scene.updateTransientActionFeedback()
			assert.Empty(t, scene.transientActionFeedback)
			assert.Zero(t, scene.dialogStack.Count())
			for _, card := range cards {
				assert.False(t, card.dialog.IsVisible())
			}
		})
	}
}

func setActionPopupLimitForTest(scene *GameScene, limit int) {
	if scene.gameConfig == nil {
		scene.gameConfig = &config.UltimaVConfiguration{} //nolint:exhaustruct // Only the applied gameplay preference is needed.
	}
	if scene.gameConfig.GameplayConfiguration == nil {
		scene.gameConfig.GameplayConfiguration = config.DefaultGameplayConfiguration()
	}
	scene.gameConfig.GameplayConfiguration.MaxVisibleActionPopups = limit
}

// Use the production Output and callback adapters, not callback capture as a
// substitute for history. Other groups retain the integration fixture's wiring.
func wireFeedbackOutputForTest(scene *GameScene) {
	_, scene.output = newMainOutput(1280)
	scene.gameState.SystemCallbacks.Message = game_state.NewMessageCallbacks(game_state.MessageCallbacks{
		AddRowStr: scene.addRowStr, AppendToCurrentRowStr: scene.appendToCurrentRowStr,
		ShowCommandPrompt: scene.showCommandPrompt, AppendStyledMessage: scene.appendStyledMessage,
	})
	scene.gameState.SystemCallbacks.UI.ShowActionFeedback = scene.ShowActionFeedback
	scene.gameState.SystemCallbacks.UI.ShowSearchFeedback = scene.ShowSearchFeedback
}
