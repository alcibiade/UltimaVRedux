//nolint:mnd // Existing scroll animation color and interpolation values are domain-specific.
package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func (g *GameScene) thenUseCombatScroll(
	scroll references.Scroll,
	canCommit func() bool,
	didCommit *bool,
) event_queue.Sequence {
	//nolint:exhaustive // Field-only scrolls are rejected by the default programming-error path.
	switch scroll {
	case references.ScrollVasLor:
		return g.thenCommitCombatScroll(scroll, g.thenUseCombatLightScroll, canCommit, didCommit)
	case references.ScrollInSanct:
		return g.thenCommitCombatScroll(scroll, func() event_queue.Sequence {
			return g.thenUseCombatDurationScroll(scroll, "Protection!", func(duration int) {
				g.gameState.ActivateProtectionEffect(duration)
			})
		}, canCommit, didCommit)
	case references.ScrollInAn:
		return g.thenCommitCombatScroll(scroll, func() event_queue.Sequence {
			return g.thenUseCombatDurationScroll(scroll, "Negate magic!", func(duration int) {
				g.gameState.ActivateNegateMagicEffect(duration)
			})
		}, canCommit, didCommit)
	case references.ScrollKalXenCorp:
		return g.thenUseCombatDaemonScroll(canCommit, didCommit)
	case references.ScrollAnTym:
		return g.thenCommitCombatScroll(scroll, g.thenUseCombatTimeStopScroll, canCommit, didCommit)
	default:
		panic("combat Use workflow received a field-only scroll")
	}
}

func (g *GameScene) thenCommitCombatScroll(
	scroll references.Scroll,
	effect func() event_queue.Sequence,
	canCommit func() bool,
	didCommit *bool,
) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		committed := false
		var timing combatPresentationTiming

		return q.Then(func() {
			if !canCommit() {
				return
			}
			timing = g.captureCombatTiming()
			defer g.combatTimingScope(timing)()
			committed = g.gameState.CommitScrollUse(scroll)
			*didCommit = committed
			if !committed {
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)
			}
		}).If(func() bool { return committed }, func(effectQ *event_queue.SequentialEventBuilder) {
			effectQ.ThenSequence(func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
				defer g.combatTimingScope(timing)()

				return effect()(q)
			})
		})
	}
}

func (g *GameScene) thenUseCombatLightScroll() event_queue.Sequence {
	durationMs := g.captureCombatTiming().durationMs(lightScrollInvertDurationMs)

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			g.gameState.SystemCallbacks.Visual.SetScreenInvert(lightScrollInvertIntensity, int(durationMs))
		}).Wait(durationMs).Then(func() {
			duration := g.gameState.GameReferences.InventoryItemReferences.GetScrollEffectDuration(references.ScrollVasLor)
			g.gameState.ActivateLightEffect(duration)
		})
	}
}

func (g *GameScene) thenUseCombatDurationScroll(
	scroll references.Scroll,
	message string,
	activate func(int),
) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			duration := g.gameState.GameReferences.InventoryItemReferences.GetScrollEffectDuration(scroll)
			activate(duration)
		}).ThenSequence(g.thenShowConfirmationDialog(message))
	}
}

func (g *GameScene) thenUseCombatTimeStopScroll() event_queue.Sequence {
	durationMs := g.captureCombatTiming().durationMs(timeStopInvertDurationMs)

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			g.gameState.SystemCallbacks.Visual.SetScreenInvert(timeStopInvertIntensity, int(durationMs))
		}).Wait(durationMs).Then(func() {
			duration := g.gameState.GameReferences.InventoryItemReferences.GetScrollEffectDuration(references.ScrollAnTym)
			g.gameState.ActivateTimeStopEffect(duration)
		}).ThenSequence(g.thenShowConfirmationDialog("Negate time!"))
	}
}

// StartWindChangeScrollWorkflow initiates the Wind Change scroll workflow.
// Prompts for direction using a popup dialog, then executes the scroll effect.
// Wind scroll direction selection precedes the effect (ref:CONSUME-0003).
//
// Flow:
//  1. Display popup with "Wind change!\nDirection-" prompt
//  2. Wait for cardinal arrow/keypad input or Space (pass)
//  3. Execute wind change, consume scroll, show "Rel Hur!" and direction popup
func (g *GameScene) StartWindChangeScrollWorkflow() {
	g.startWindChangeScrollWorkflow(nil)
}

func (g *GameScene) startWindChangeScrollWorkflow(complete func()) {
	origin := g.gameState.MapState.PlayerLocation
	quantity := g.gameState.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur)
	g.gameState.EventQueueManager.Workflow("wind_change_scroll").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {

		}).ThenSequence(

		g.thenShowCastUseDirectionPrompt(
			"Wind change!\nDirection-",
			func(dir references.Direction) {
				if g.gameState.CombatScene != nil || g.gameState.MapState.PlayerLocation != origin ||
					g.gameState.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur) != quantity {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("Use context changed!", true)

					return
				}
				if complete != nil {
					g.gameState.CompleteWindChangeScrollForSurfaceUse(dir)
				} else {
					g.gameState.CompleteWindChangeScroll(dir)
				}
			},
			func() {},
		)).Finally(func() {
		g.clearCastUseDirectionPrompt()
		if complete != nil {
			complete()
		}
	}).
		Start()
}

// Light scroll visual effect constants
const (
	lightScrollInvertIntensity  = 0.8 // Screen invert intensity (0.0-1.0)
	lightScrollInvertDurationMs = 300 // Duration of invert effect in milliseconds
)

// StartLightScrollWorkflow initiates the Vas Lor (Light) scroll workflow.
// Plays magic sound, shows invert effect, then activates the light.
//
// Flow:
//  1. Play magic sound and start screen invert effect
//  2. Wait for invert effect to complete
//  3. Activate light effect (240 gameplay minutes)
func (g *GameScene) StartLightScrollWorkflow() {
	g.startLightScrollWorkflow(nil)
}

func (g *GameScene) startLightScrollWorkflow(complete func()) {
	effectApplied := false
	applyEffect := func() {
		if effectApplied {
			return
		}
		effectApplied = true
		duration := g.gameState.GameReferences.InventoryItemReferences.GetScrollEffectDuration(references.ScrollVasLor)
		g.gameState.ActivateLightEffect(duration)
	}

	g.gameState.EventQueueManager.Workflow("light_scroll").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			// Play magic sound and start invert effect
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			g.gameState.SystemCallbacks.Visual.SetScreenInvert(lightScrollInvertIntensity, lightScrollInvertDurationMs)
		}).
		Wait(lightScrollInvertDurationMs). // Wait for invert effect to complete
		Then(applyEffect).Finally(func() {
		if complete != nil {
			applyEffect()
			complete()
		}
	}).
		Start()
}

// Time Stop scroll visual effect constants
const (
	timeStopInvertIntensity    = 0.6  // Screen invert intensity (0.0-1.0)
	timeStopInvertDurationMs   = 300  // Duration of invert effect in milliseconds
	timeStopNoEffectDurationMs = 2000 // Timed popup duration for blocked locations
)

// StartTimeStopScrollWorkflow initiates the An Tym (Negate Time) scroll workflow.
// ref:CONSUME-0003: blocked in Stonegate (0x1d) and Doom (0x28).
//
// Flow:
//  1. Check location restriction (Doom/Stonegate → "No effect!" timed popup)
//  2. Play magic sound and start screen invert effect
//  3. Wait for invert effect to complete
//  4. Activate time stop effect (20 turns duration)
//  5. Show "Negate time!" persist popup (player must acknowledge)
func (g *GameScene) StartTimeStopScrollWorkflow() {
	g.startTimeStopScrollWorkflow(nil)
}

func (g *GameScene) startTimeStopScrollWorkflow(complete func()) {
	// Location validation - blocked in Doom and Stonegate
	loc := g.gameState.MapState.PlayerLocation.Location
	if loc == references.Doom || loc == references.Stonegate {
		// Timed popup (2000ms auto-dismiss) for blocked locations
		g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundSpellFailure)

		overlay := widgets.NewOverlayDialog("No effect!").
			AsOverlayFeedback().
			WithExternallyManagedDuration(timeStopNoEffectDurationMs).
			Build()
		g.gameState.EventQueueManager.Workflow("timestop_scroll").
			Priority(event_queue.PriorityNormal).
			Sequential().BlockInput().
			First(func() {
				g.installTimedOverlay(overlay)
			}).While(
			func() bool {
				return g.timedOverlayDialog == overlay &&
					g.gameState.ElapsedMs < g.timedOverlayDismissMs
			},
			func() {},
			DialogPollingRateMs,
		).Finally(func() {
			if g.timedOverlayDialog == overlay {
				g.clearTimedOverlay()
			}
			if complete != nil {
				complete()
			}
		}).Start()

		return
	}

	effectApplied := false
	applyEffect := func() {
		if effectApplied {
			return
		}
		effectApplied = true
		duration := g.gameState.GameReferences.InventoryItemReferences.GetScrollEffectDuration(references.ScrollAnTym)
		g.gameState.ActivateTimeStopEffect(duration)
	}

	// Execute time stop workflow with visual effects
	g.gameState.EventQueueManager.Workflow("timestop_scroll").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			g.gameState.SystemCallbacks.Visual.SetScreenInvert(timeStopInvertIntensity, timeStopInvertDurationMs)
		}).
		Wait(timeStopInvertDurationMs).
		Then(applyEffect).ThenSequence(

		g.thenShowConfirmationDialog("Negate time!")).Finally(func() {
		if complete != nil {
			applyEffect()
			complete()
		}
	}).
		Start()
}

// Resurrection scroll visual effect constants
const (
	resurrectionFlashIntensity     = 0.7  // Screen flash intensity (0.0-1.0)
	resurrectionFlashDurationMs    = 400  // White/gold flash duration in milliseconds
	resurrectionNoTargetDurationMs = 2000 // Timed popup for no dead targets
)

// StartResurrectionScrollWorkflow initiates the In Mani Corp (Resurrection) scroll workflow.
// Shows character selection for all party members and applies resurrection to dead targets.
//
// Flow:
//  1. Show character selection dialog
//  2. On confirmation: consume the scroll
//  3. If dead: resurrect, play magic sound, and show the white/gold flash
//  4. If living: report no effect
func (g *GameScene) StartResurrectionScrollWorkflow() {
	g.startResurrectionScrollWorkflow(nil)
}

func (g *GameScene) startResurrectionScrollWorkflow(complete func()) {
	origin := g.gameState.MapState.PlayerLocation
	quantity := g.gameState.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp)
	var characterSelectionDialog *widgets.CharacterSelectionDialog
	var selectionMade bool
	var selectedCharacterIndex int
	var result game_state.ResurrectionScrollUseResult

	g.gameState.EventQueueManager.Workflow("resurrection_scroll").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			characterSelectionDialog = widgets.NewCharacterSelectionDialog(widgets.CharacterSelectionConfig{
				Title: "In Mani Corp: Choose Target",
				Options: g.characterSelectionOptions(characterSelectionContext{
					Profile: characterSelectionResurrectionScroll,
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
		).
		Then(func() {
			g.keyboard.SetForceWaitAnyKey(KeyboardDebounceMs)
		}).
		Wait(KeyboardDebounceShortMs).If(

		func() bool { return selectedCharacterIndex >= 0 },
		func(subQ *event_queue.SequentialEventBuilder) {
			subQ.Then(func() {
				if g.gameState.CombatScene == nil && g.gameState.MapState.PlayerLocation == origin &&
					g.gameState.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp) == quantity {
					// REDUX DEVIATION: CONSUME-0001: original Use consumes before target selection.
					// Redux commits only after a final valid confirmation.
					result = g.gameState.CommitResurrectionScrollUse(selectedCharacterIndex)
				}
				if result == game_state.ResurrectionScrollUnavailable {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

					return
				}
				if complete == nil {
					g.gameState.AdvanceGameplayMinutes(1)
				}
			}).If(func() bool { return result == game_state.ResurrectionScrollNoEffect }, func(noEffectQ *event_queue.SequentialEventBuilder) {
				noEffectQ.ThenSequence(g.thenShowConfirmationDialog("Not dead!"))
			}).If(func() bool { return result == game_state.ResurrectionScrollSuccess }, func(successQ *event_queue.SequentialEventBuilder) {
				successQ.Then(func() {
					g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
					g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(1.0, 0.95, 0.7)
				}).
					Animate(resurrectionFlashDurationMs, func(progress float32) {
						var intensity float64
						if progress < 0.5 {
							intensity = float64(progress) * 2.0 * resurrectionFlashIntensity
						} else {
							intensity = float64(1.0-progress) * 2.0 * resurrectionFlashIntensity
						}
						g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)
					}).
					Then(func() {
						g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
					}).ThenSequence(

					g.thenShowConfirmationDialog("Resurrected!"))
			})
		}).Finally(

		func() {
			if characterSelectionDialog != nil {
				g.dialogStack.RemoveWidgetInstance(characterSelectionDialog)
			}
			g.keyboard.ResetReleaseDebouncing()
			if complete != nil {
				complete()
			} else {
				g.gameState.ProcessPendingProvisionHours()
			}
		}).
		Start()
}
