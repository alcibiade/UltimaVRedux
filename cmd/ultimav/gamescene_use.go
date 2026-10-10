package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// StartUseWorkflow initiates the Use command workflow, showing the item selection dialog.
// Shows "No usable items!" if the inventory is empty.
//
// Flow:
//  1. Query usable items from GameState
//  2. Show message if no items available
//  3. Display ItemSelectionDialog for selection
//  4. Execute item use on selection
//  5. Close dialog on cancel (ESC)
func (g *GameScene) StartUseWorkflow() {
	g.startUseWorkflow(game_state.UsageField, false)
}

func (g *GameScene) startSurfaceUseWorkflow() {
	g.startUseWorkflow(game_state.UsageField, true)
}

// StartCombatUseWorkflow owns selection, effect presentation, and action completion.
func (g *GameScene) StartCombatUseWorkflow() {
	g.startUseWorkflow(game_state.UsageCombat, false)
}

func (g *GameScene) startUseWorkflow(context game_state.UsageContext, completeSurfaceCycle bool) {
	commitSequence := g.gameState.CastUseCommitSequence()
	origin := g.gameState.MapState.PlayerLocation
	var combatActor game_state.CombatTurnIdentity
	if context == game_state.UsageCombat {
		combatActor = g.gameState.CombatScene.GetCurrentTurnIdentity()
	}
	originLocation := g.gameState.MapState.PlayerLocation.Location
	originMapType := g.gameState.MapState.PlayerLocation.Location.GetMapType()
	// Get usable items from game state
	gsItems := g.gameState.GetUsableItemsForContext(context)

	// Check if there are any usable items
	if len(gsItems) == 0 {
		g.addRowStr("No usable items!")

		return
	}
	// Create and show the item selection dialog
	var dialog *widgets.ItemSelectionDialog
	var selectedItem *game_state.UseableItemInfo
	var executableItem *game_state.UseableItemInfo
	var hmsCapeConfirmation *hmsCapeConfirmationState
	fieldItemAccepted := false
	parentFinalized := false
	effectFinalized := false
	completionDelivered := false
	completeOwnedSurfaceCycle := func() {
		if completionDelivered || !parentFinalized || (fieldItemAccepted && !effectFinalized) {
			return
		}
		completionDelivered = true
		state := surfaceCommandCancelled
		if fieldItemAccepted {
			state = surfaceCommandCompleted
		}
		result := newSurfaceCommandResult(state, originMapType)
		result.ConsumesNormalCycle = g.gameState.CastUseCommitSequence() != commitSequence
		result.MapTransitioned = g.gameState.MapState.PlayerLocation.Location != originLocation
		g.completeSurfaceCommand(result)
	}
	effectComplete := func() {
		if effectFinalized {
			return
		}
		effectFinalized = true
		completeOwnedSurfaceCycle()
	}

	g.gameState.EventQueueManager.Workflow("use_item_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			dialog = widgets.NewItemSelectionDialog(widgets.ItemSelectionDialogConfig{
				Items:       gsItems,
				Keyboard:    g.keyboard,
				SpriteSheet: g.spriteSheet,
				OnSelect: func(item game_state.UseableItemInfo) {
					if g.gameState.MapState.PlayerLocation != origin {
						g.gameState.SystemCallbacks.UI.ShowActionFeedback("Use context changed!", true)

						return
					}
					if context == game_state.UsageCombat {
						selectedItem = &item
					} else {
						current, ok := g.gameState.RevalidateUseItemSelection(item, context)
						if !ok {
							g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

							return
						}
						if completeSurfaceCycle {
							fieldItemAccepted = g.gameState.UseItemForSurface(current, effectComplete)
						} else {
							fieldItemAccepted = g.gameState.UseItem(current)
						}
					}
				},
				OnCancel: func() {

				},
			})
			g.dialogStack.PushModalDialog(dialog)
		}).
		While(
			func() bool { return !dialog.IsDismissed() },
			func() {},
			16,
		).If(
		func() bool { return selectedItem != nil },
		func(q *event_queue.SequentialEventBuilder) {
			q.Then(func() {
				if g.gameState.CombatScene == nil || g.gameState.CombatScene.GetCurrentTurnIdentity() != combatActor {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("Not thy turn!", true)

					return
				}
				current, ok := g.gameState.RevalidateUseItemSelection(*selectedItem, context)
				if !ok {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

					return
				}
				executableItem = &current
			}).If(
				func() bool { return executableItem != nil },
				func(effectQ *event_queue.SequentialEventBuilder) {
					effectQ.ThenSequence(func(useQ *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
						return g.thenUseCombatItem(*executableItem, &hmsCapeConfirmation)(useQ)
					})
				},
			)
		}).Finally(

		func() {
			g.dialogStack.RemoveWidgetInstance(dialog)
			g.cleanupHMSCapeConfirmation(hmsCapeConfirmation)
			if context == game_state.UsageCombat {
				g.clearCastUseDirectionPrompt()
				g.clearPotionEffectDialog()
				g.clearCombatInvocationEffects()
			}
			g.keyboard.ResetReleaseDebouncing()
			if context == game_state.UsageCombat && g.gameState.CastUseCommitSequence() != commitSequence {
				g.gameState.FinishCombatAction()
			}
			if context == game_state.UsageField && completeSurfaceCycle {
				parentFinalized = true
				completeOwnedSurfaceCycle()
			}
		}).
		Start()
}

// StartDungeonUseWorkflow retains the accepted command while selecting a supported dungeon item.
func (g *GameScene) StartDungeonUseWorkflow(token game_state.DungeonCommandCycle) {
	origin := g.gameState.MapState.PlayerLocation
	gsItems := g.gameState.GetUsableItemsForContext(game_state.UsageDungeon)
	if len(gsItems) == 0 {
		g.gameState.SystemCallbacks.Message.AddRowStr("No usable items!")
		g.gameState.CompleteDungeonCommandCycle(token, game_state.DungeonFullEndPlayer)

		return
	}

	var dialog *widgets.ItemSelectionDialog
	var dungeonViewDialog *widgets.DungeonViewDialog
	var selectedItem *game_state.UseableItemInfo
	var executableItem *game_state.UseableItemInfo
	var model game_state.DungeonViewModel
	viewReady := false
	effectOwned := false
	effectFinalized := false
	parentFinalized := false
	completionDelivered := false
	completeCommand := func() {
		if completionDelivered || !parentFinalized || (effectOwned && !effectFinalized) {
			return
		}
		completionDelivered = true
		g.gameState.CompleteDungeonCommandCycle(token, game_state.DungeonFullEndPlayer)
	}
	effectComplete := func() {
		if effectFinalized {
			return
		}
		effectFinalized = true
		completeCommand()
	}

	g.gameState.EventQueueManager.Workflow("dungeon_use_item_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			dialog = widgets.NewItemSelectionDialog(widgets.ItemSelectionDialogConfig{
				Items:       gsItems,
				Keyboard:    g.keyboard,
				SpriteSheet: g.spriteSheet,
				OnSelect: func(item game_state.UseableItemInfo) {
					selectedItem = &item
				},
				OnCancel: func() {},
			})
			g.dialogStack.PushModalDialog(dialog)
		}).While(
		func() bool { return !dialog.IsDismissed() },
		func() {},
		DialogPollingRateMs,
	).If(
		func() bool { return selectedItem != nil },
		func(q *event_queue.SequentialEventBuilder) {
			q.Then(func() {
				if g.gameState.MapState.PlayerLocation != origin {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("Use context changed!", true)

					return
				}
				current, ok := g.gameState.RevalidateUseItemSelection(*selectedItem, game_state.UsageDungeon)
				if !ok {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

					return
				}
				executableItem = &current
			}).If(
				func() bool { return executableItem != nil },
				func(useQ *event_queue.SequentialEventBuilder) {
					useQ.Then(func() {
						effectOwned = true
						g.dispatchDungeonUseItem(*executableItem, &model, &viewReady, effectComplete)
					})
				},
			).If(
				func() bool { return viewReady },
				func(viewQ *event_queue.SequentialEventBuilder) {
					viewQ.ThenSequence(g.thenShowDungeonViewDialog(&model, &dungeonViewDialog)).
						Then(effectComplete)
				},
			)
		},
	).Finally(func() {
		if dialog != nil {
			g.dialogStack.RemoveWidgetInstance(dialog)
		}
		if dungeonViewDialog != nil {
			dungeonViewDialog.Dismiss()
			g.dialogStack.RemoveWidgetInstance(dungeonViewDialog)
		}
		if viewReady {
			effectComplete()
		}
		parentFinalized = true
		completeCommand()
	}).
		Start()
}

func (g *GameScene) dispatchDungeonUseItem(
	item game_state.UseableItemInfo,
	viewModel *game_state.DungeonViewModel,
	viewReady *bool,
	complete func(),
) {
	switch itemType := item.ItemType.(type) {
	case references.Potion:
		g.startDungeonPotionWorkflow(itemType, complete)
	case references.Scroll:
		g.dispatchDungeonScroll(itemType, viewModel, viewReady, complete)
	case references.QuestItem:
		g.dispatchDungeonQuestItem(item, itemType, complete)
	case references.Moonstone:
		g.gameState.SystemCallbacks.Message.AddRowStr("Moonstone cannot be buried here!")
		complete()
	case references.Provision:
		if itemType != references.SkullKeys {
			panic("dungeon Use received unsupported provision")
		}
		g.gameState.SystemCallbacks.Message.AddRowStr("Not here!")
		complete()
	default:
		panic("dungeon Use received unsupported item type")
	}
}

func (g *GameScene) dispatchDungeonScroll(
	scroll references.Scroll,
	viewModel *game_state.DungeonViewModel,
	viewReady *bool,
	complete func(),
) {
	if scroll == references.ScrollInManiCorp {
		g.startResurrectionScrollWorkflow(complete)

		return
	}
	if !g.gameState.CommitScrollUse(scroll) {
		g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)
		complete()

		return
	}

	switch scroll {
	case references.ScrollVasLor:
		g.gameState.SystemCallbacks.Message.AddRowStr("Light!")
		g.startLightScrollWorkflow(complete)
	case references.ScrollInSanct:
		g.gameState.SystemCallbacks.Message.AddRowStr("Protection!")
		g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		duration := g.gameState.GameReferences.InventoryItemReferences.GetScrollEffectDuration(scroll)
		g.gameState.ActivateProtectionEffect(duration)
		complete()
	case references.ScrollInAn:
		g.gameState.SystemCallbacks.Message.AddRowStr("Negate magic!")
		g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		duration := g.gameState.GameReferences.InventoryItemReferences.GetScrollEffectDuration(scroll)
		g.gameState.ActivateNegateMagicEffect(duration)
		complete()
	case references.ScrollInQuasWis:
		var err error
		*viewModel, err = g.gameState.CurrentDungeonViewModel()
		if err != nil {
			panic(err)
		}
		*viewReady = true
		g.gameState.SystemCallbacks.Message.AddRowStr("View!")
		g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
	case references.ScrollInManiCorp:
		panic("resurrection scroll must select its target first")
	case references.ScrollAnTym:
		g.startTimeStopScrollWorkflow(complete)
	case references.ScrollRelHur, references.ScrollKalXenCorp:
		panic("dungeon Use received unsupported scroll")
	}
}

func (g *GameScene) dispatchDungeonQuestItem(
	item game_state.UseableItemInfo,
	questItem references.QuestItem,
	complete func(),
) {
	switch questItem {
	case references.Amulet, references.Crown, references.BlackBadge, references.PocketWatch, references.HMSCape:
		if !g.gameState.UseItemWithCompletion(item, complete) {
			complete()
		}
	case references.Sceptre:
		g.gameState.SystemCallbacks.Message.AddRowStr("Sceptre")
		g.gameState.SystemCallbacks.Message.AddRowStr("Wielding the Sceptre of Lord British...")
		g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		g.gameState.UseDungeonSceptre()
		complete()
	case references.Carpet, references.Spyglass:
		g.gameState.SystemCallbacks.Message.AddRowStr("Not here!")
		complete()
	case references.Sextant:
		g.gameState.SystemCallbacks.Message.AddRowStr("Only outdoors!")
		complete()
	case references.WoodenBox:
		g.gameState.SystemCallbacks.Message.AddRowStr("How?")
		complete()
	case references.ShardOfFalsehood, references.ShardOfHatred, references.ShardOfCowardice:
		g.gameState.SystemCallbacks.Message.AddRowStr("No effect!")
		complete()
	case references.Grapple:
		panic("dungeon Use received unsupported quest item")
	}
}
func (g *GameScene) thenUseCombatItem(
	item game_state.UseableItemInfo,
	hmsCapeConfirmation **hmsCapeConfirmationState,
) event_queue.Sequence {
	switch itemType := item.ItemType.(type) {
	case references.Potion:
		return g.thenUseCombatPotion(itemType)
	case references.Scroll:
		return g.thenUseCombatScroll(itemType)
	case references.QuestItem:
		if itemType == references.Sceptre {
			return g.thenUseCombatSceptre()
		}
		if itemType == references.HMSCape {
			commit, review := g.gameState.BeginHMSCapeUse()
			if commit == nil || review.Result != game_state.HMSCapeUseReady {
				return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
					return q.Then(func() { g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true) })
				}
			}
			confirmation := &hmsCapeConfirmationState{}
			*hmsCapeConfirmation = confirmation

			return g.thenConfirmHMSCape(commit, confirmation)
		}
	}

	panic("combat Use workflow received an unsupported item")
}
