package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
)

// startPotionWorkflow selects a character, consumes the potion, rolls its random
// effect and applies it. Potion effects and substitution are documented in
// ref:CONSUME-0002.
func (g *GameScene) startPotionWorkflow(commit *game_state.PotionUseCommit, complete func()) {
	g.startCharacterSelectPotionWorkflow(commit, complete)
}

func (g *GameScene) startDungeonPotionWorkflow(potion references.Potion, complete func()) {
	commit, _ := g.gameState.BeginPotionUse(potion, game_state.UsageDungeon)
	if commit == nil {
		g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)
		complete()

		return
	}
	g.startCharacterSelectPotionWorkflow(commit, complete)
}

func (g *GameScene) thenUseCombatPotion(
	potion references.Potion,
	canCommit func() bool,
	didCommit *bool,
) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		var effect game_state.PotionEffectResult
		var outcome *game_state.PotionUseOutcome
		var timing combatPresentationTiming
		committed := false
		casterIdx := g.gameState.CombatScene.GetActiveCombatantIndex()

		return q.Then(func() {
			if !canCommit() {
				return
			}
			timing = g.captureCombatTiming()
			defer g.combatTimingScope(timing)()
			commit, _ := g.gameState.BeginPotionUse(potion, game_state.UsageCombat)
			if commit == nil {
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

				return
			}
			outcome, committed = g.gameState.CommitPotionUse(commit)
			*didCommit = committed
			if !committed {
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

				return
			}
			effect = g.gameState.CommitPotionEffect(outcome, casterIdx)
			if outcome.Mislabelled {
				g.gameState.SystemCallbacks.Message.AddRowStr(
					"The potion must have been mislabelled - it has a different effect!",
				)
			}
		}).If(func() bool { return committed }, func(effectQ *event_queue.SequentialEventBuilder) {
			effectQ.ThenSequence(func(presentationQ *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
				return g.thenPresentPotionEffectWithTiming(timing, effect, casterIdx)(presentationQ)
			})
		})
	}
}

func (g *GameScene) thenPresentPotionEffect(
	effect game_state.PotionEffectResult,
	characterIndex int,
) event_queue.Sequence {
	return g.thenPresentPotionEffectWithTiming(g.capturePresentationTiming(g.gameState.IsInCombat()), effect, characterIndex)
}

func (g *GameScene) thenPresentPotionEffectWithTiming(
	timing combatPresentationTiming,
	effect game_state.PotionEffectResult,
	characterIndex int,
) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		switch effect {
		case game_state.PotionEffectNoEffect:
			return q.Then(func() {
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundSpellFailure)
			}).ThenSequence(g.thenShowPotionEffectDialog("No effect!"))
		case game_state.PotionEffectAwakened:
			return q.ThenSequence(game_state.NewPartyStateBuilder(g.gameState).
				ToCharacter(characterIndex).FlashWhite().WithSound(game_state.SoundAwaken).BuildWithPresentationTiming(timing.duration)).
				ThenSequence(g.thenShowPotionEffectDialog("Awakened!"))
		case game_state.PotionEffectHealed:
			return q.ThenSequence(game_state.NewPartyStateBuilder(g.gameState).
				ToCharacter(characterIndex).FlashInverted().WithSound(game_state.SoundHeal).BuildWithPresentationTiming(timing.duration)).
				ThenSequence(g.thenShowPotionEffectDialog("Healed!"))
		case game_state.PotionEffectCured:
			return q.ThenSequence(game_state.NewPartyStateBuilder(g.gameState).
				ToCharacter(characterIndex).FlashInverted().WithSound(game_state.SoundCure).BuildWithPresentationTiming(timing.duration)).
				ThenSequence(g.thenShowPotionEffectDialog("Poison cured!"))
		case game_state.PotionEffectPoisoned:
			return q.ThenSequence(game_state.NewPartyStateBuilder(g.gameState).
				ToCharacter(characterIndex).FlashGreen().WithSound(game_state.SoundPoison).BuildWithPresentationTiming(timing.duration)).
				ThenSequence(g.thenShowPotionEffectDialog("POISONED!"))
		case game_state.PotionEffectSlept:
			return q.ThenSequence(game_state.NewPartyStateBuilder(g.gameState).
				ToCharacter(characterIndex).FlashPurple().WithSound(game_state.SoundSleep).BuildWithPresentationTiming(timing.duration)).
				ThenSequence(g.thenShowPotionEffectDialog("Slept!"))
		case game_state.PotionEffectRat:
			return q.Then(func() {
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			}).ThenSequence(g.thenShowPotionEffectDialog("Poof!"))
		case game_state.PotionEffectInvisible:
			pulse := g.thenScreenPulseWithTiming(timing,
				combatLightBlueFlashRed,
				combatLightBlueFlashGreen,
				combatLightBlueFlashBlue,
				combatRelTymPulsePeak,
			)

			return q.Then(func() {
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			}).ThenSequence(pulse).
				Wait(timing.durationMs(combatRelTymPulseGapMs)).
				ThenSequence(pulse).
				ThenSequence(g.thenShowPotionEffectDialog("Invisible!"))
		case game_state.PotionEffectNoNotice:
			return g.thenShowPotionEffectDialog("No noticeable effect now!")(q)
		case game_state.PotionEffectXRay:
			return g.thenPresentXRayVision(false)(q)
		default:
			panic("unknown potion effect result")
		}
	}
}

func (g *GameScene) thenShowPotionEffectDialog(message string) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		dialog := widgets.NewOverlayDialog(message).
			AsConfirmation().
			WithUserDismissed(g.keyboard).
			OnDismiss(g.clearPotionEffectDialog).
			Build()

		return q.Then(func() {
			g.potionEffectDialog = dialog
		}).ThenSequence(g.showOverlayAndWait(dialog))
	}
}

func (g *GameScene) clearPotionEffectDialog() {
	if g.potionEffectDialog != nil {
		g.dialogStack.RemoveWidgetInstance(g.potionEffectDialog)
		g.potionEffectDialog = nil
	}
}

// startCharacterSelectPotionWorkflow delays commitment until target confirmation.
func (g *GameScene) startCharacterSelectPotionWorkflow(
	commit *game_state.PotionUseCommit,
	complete func(),
) {
	var characterSelectionDialog *widgets.CharacterSelectionDialog
	var outcome *game_state.PotionUseOutcome
	committed := false
	var selectionMade bool
	var selectedCharacterIndex int
	var effect game_state.PotionEffectResult

	sequence := g.gameState.EventQueueManager.Workflow("potion_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			characterSelectionDialog = widgets.NewCharacterSelectionDialog(widgets.CharacterSelectionConfig{
				Title: characterSelectionPotionName(commit.Potion()) + " Potion: Choose Target",
				Options: g.characterSelectionOptions(characterSelectionContext{
					Profile: characterSelectionPotionTarget,
					Potion:  commit.Potion(),
				}),
				OnSelect: func(characterIndex int) {
					g.dialogStack.RemoveWidgetInstance(characterSelectionDialog)
					selectedCharacterIndex = characterIndex
					selectionMade = true
				},
				OnCancel: func() {
					g.dialogStack.RemoveWidgetInstance(characterSelectionDialog)
					selectedCharacterIndex = -1
					selectionMade = true
				},
				Keyboard:    g.keyboard,
				SpriteSheet: g.spriteSheet,
				Placement:   gameScreenPercents,
			})
			g.dialogStack.PushModalDialog(characterSelectionDialog)
		}).
		While(
			func() bool { return !selectionMade },
			func() {},
			DialogPollingRateMs,
		).Then(func() {
		g.keyboard.SetForceWaitAnyKey(KeyboardDebounceMs)
	}).Wait(KeyboardDebounceShortMs).If(
		func() bool { return selectedCharacterIndex >= 0 },
		func(effectQ *event_queue.SequentialEventBuilder) {
			effectQ.Then(func() {
				if outcome == nil {
					// REDUX DEVIATION: CONSUME-0001: original Use consumes before target selection.
					// Redux waits for final target confirmation in every context.
					outcome, committed = g.gameState.CommitPotionUseForTarget(commit, selectedCharacterIndex)
					if !committed {
						g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

						return
					}
				}
				effect = g.gameState.CommitPotionEffect(outcome, selectedCharacterIndex)
				if outcome.Mislabelled {
					g.gameState.SystemCallbacks.Message.AddRowStr(
						"The potion must have been mislabelled - it has a different effect!")
				}
			}).If(func() bool { return committed }, func(presentationQ *event_queue.SequentialEventBuilder) {
				presentationQ.ThenSequence(g.thenPresentPotionEffect(effect, selectedCharacterIndex))
			})
		})

	sequence.Finally(func() {
		g.clearPotionEffectDialog()
		if characterSelectionDialog != nil {
			g.dialogStack.RemoveWidgetInstance(characterSelectionDialog)
		}
		if outcome != nil {
			g.gameState.CancelPotionEffect(outcome)
		}
		g.xRayVisionActive = false
		g.keyboard.ResetReleaseDebouncing()
		if complete != nil {
			complete()
		} else if committed {
			g.gameState.AdvanceGameplayMinutesAndProcessProvisions(1)
		}
	}).Start()
}

func (g *GameScene) thenPresentXRayVision(showMislabelMsg bool) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			if showMislabelMsg {
				g.gameState.SystemCallbacks.Message.AddRowStr(
					"The potion must have been mislabelled - it has a different effect!")
			}

			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundXRay)

			g.xRayVisionActive = true
		}).ThenSequence(

			game_state.NewPartyStateBuilder(g.gameState).
				FlashInverted().
				Build()).ThenSequence(

			g.thenShowPotionEffectDialog("You can see clearly now!")).
			WhileAwaitingInput(
				func() {

				},
				func(key *semantic_input.Key) bool {
					return key != nil
				},
				int64(DialogPollingRateMs),
			)
	}
}
