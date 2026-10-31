package main

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const deedBedRestWorkflowName = "deed_bed_rest"

// beginDeedBuild answers Alt+B on the party's own deed map (ref:CRAFT-0010): a menu of the piece
// categories the party carries, then the pieces, then a direction; or Demolish and a direction.
// Esc on any menu or prompt is free.
func (g *GameScene) beginDeedBuild() {
	messages := g.gameState.SystemCallbacks.Message
	messages.ShowCommandPrompt("Build-")
	finish := g.craftingCompletion(references.SmallMapType)
	stocks := g.gameState.BuildPieceStocks()
	description := "Thou carriest no pieces to build. Craft them at a workbench with Alt+C. " +
		"Demolish takes back a piece thou didst build."
	if len(stocks) > 0 {
		description = "Choose what to build, then a direction. Demolish takes back a piece thou didst build."
	}
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu("Build", description, func() {
		g.appendToCurrentRowStr("Cancelled")
		finish(surfaceCommandCancelled)
	})
	for _, category := range references.CraftCategories() {
		inCategory := buildStocksInCategory(stocks, category)
		if len(inCategory) == 0 {
			continue
		}
		kinds := "kinds"
		if len(inCategory) == 1 {
			kinds = "kind"
		}
		menu.AddButton(fmt.Sprintf("%s (%d %s)", category.Name(), len(inCategory), kinds), func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.openBuildCategoryMenu(category, inCategory, finish)
		})
	}
	menu.AddButton("Demolish", func() {
		g.dialogStack.MustRemoveWidgetInstance(menu)
		g.appendToCurrentRowStr("Demolish")
		g.gameState.SystemCallbacks.Message.ShowCommandPrompt("Demolish-")
		g.secondaryKeyState = DemolishDirectionInput
		g.keyboard.SetAllowKeyPressImmediately()
	})
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

func buildStocksInCategory(stocks []game_state.BuildPieceStock, category references.CraftCategory) []game_state.BuildPieceStock {
	inCategory := make([]game_state.BuildPieceStock, 0, len(stocks))
	for _, stock := range stocks {
		if stock.Piece.Category == category {
			inCategory = append(inCategory, stock)
		}
	}

	return inCategory
}

// openBuildCategoryMenu lists the carried pieces of one category with their counts. Choosing one
// asks for the direction to place it; Esc is free.
func (g *GameScene) openBuildCategoryMenu(
	category references.CraftCategory, stocks []game_state.BuildPieceStock, finish func(surfaceCommandState),
) {
	description := fmt.Sprintf("Choose a piece, then the direction to place it. Placing takes %d minutes.",
		references.BuildPlaceMinutes)
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu(category.Name(), description, func() {
		g.appendToCurrentRowStr("Cancelled")
		finish(surfaceCommandCancelled)
	})
	for _, stock := range stocks {
		piece := stock.Piece
		menu.AddButton(fmt.Sprintf("%s (%d carried)", piece.Name, stock.Quantity), func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.appendToCurrentRowStr(piece.Name)
			g.pendingBuildPiece = piece.ID
			g.gameState.SystemCallbacks.Message.ShowCommandPrompt("Place-")
			g.secondaryKeyState = BuildDirectionInput
			g.keyboard.SetAllowKeyPressImmediately()
		})
	}
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

// openStorageChest shows the party's storage chest (ref:CRAFT-0011): its contents, Store and
// Take. Moving goods costs no karma. The Open command completes once the chest is closed: with
// the ordinary turn after any move, for free when nothing moved.
func (g *GameScene) openStorageChest(position references.Position, origin references.GeneralMapType) {
	g.appendToCurrentRowStr("Storage chest")
	g.openStorageChestMenu(position, false, g.craftingCompletion(origin))
}

func (g *GameScene) openStorageChestMenu(position references.Position, used bool, finish func(surfaceCommandState)) {
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu("Storage Chest", g.gameState.StorageChestSummary(position), func() {
		if used {
			finish(surfaceCommandCompleted)

			return
		}
		g.appendToCurrentRowStr("Closed")
		finish(surfaceCommandCancelled)
	})
	for _, store := range []bool{true, false} {
		label := "Take..."
		if store {
			label = "Store..."
		}
		menu.AddButton(label, func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.openStorageChestCategoryMenu(position, store, used, finish)
		})
	}
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

func (g *GameScene) openStorageChestCategoryMenu(position references.Position, store, used bool, finish func(surfaceCommandState)) {
	verb, description := "Take", "Take goods from the chest, up to what thou canst carry."
	if store {
		verb, description = "Store", "Store goods in the chest, up to what each stack can hold."
	}
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu(verb, description+" "+g.gameState.StorageChestSummary(position), func() {
		g.openStorageChestMenu(position, used, finish)
	})
	transfer := func(categories []game_state.StorageChestCategory, label string) {
		g.dialogStack.MustRemoveWidgetInstance(menu)
		moved := 0
		for _, category := range categories {
			moved += g.gameState.TransferStorageChest(position, category, store)
		}
		messages := g.gameState.SystemCallbacks.Message
		if moved == 0 {
			messages.AddRowStr("Nothing to move.")
		} else {
			messages.AddRowStr(fmt.Sprintf("%s %s: %d.", verb, strings.ToLower(label), moved))
		}
		g.openStorageChestMenu(position, used || moved > 0, finish)
	}
	menu.AddButton("Everything", func() { transfer(game_state.StorageChestCategories(), "everything") })
	for _, category := range game_state.StorageChestCategories() {
		menu.AddButton(category.Name(), func() {
			transfer([]game_state.StorageChestCategory{category}, category.Name())
		})
	}
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

// beginDeedBedRest answers Hole Up on the head of a bed the party built on its own deed
// (ref:CRAFT-0012): the party sleeps until 06:00 like a paid inn stay, for free. Members still
// poisoned at dawn die, as at an inn, so the party is warned first and Esc there is free.
func (g *GameScene) beginDeedBedRest(origin references.GeneralMapType) {
	poisoned := g.poisonedActiveMemberNames()
	if len(poisoned) == 0 {
		g.startDeedBedRest(origin)

		return
	}
	finish := g.craftingCompletion(origin)
	verb := "are"
	if len(poisoned) == 1 {
		verb = "is"
	}
	description := fmt.Sprintf("%s %s poisoned. As at an inn, poison not cured by dawn is fatal.",
		strings.Join(poisoned, ", "), verb)
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu("Sleep", description, func() {
		g.appendToCurrentRowStr("Cancelled")
		finish(surfaceCommandCancelled)
	})
	menu.AddButton("Sleep until morning anyway", func() {
		g.dialogStack.MustRemoveWidgetInstance(menu)
		g.startDeedBedRest(origin)
	})
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

func (g *GameScene) poisonedActiveMemberNames() []string {
	names := make([]string, 0)
	for index := range g.gameState.PartyState.CountActiveMembers() {
		character := g.gameState.PartyState.GetCharacter(index)
		if character.IsPoisoned() {
			names = append(names, character.GetNameAsString())
		}
	}

	return names
}

// startDeedBedRest fades out, sleeps the night through GameState, fades back and moves the party
// out of bed as after any bed rest. The ordinary command cycle completes after the workflow.
func (g *GameScene) startDeedBedRest(origin references.GeneralMapType) {
	var result game_state.DeedBedRestResult
	g.gameState.EventQueueManager.Workflow(deedBedRestWorkflowName).
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundGoodNight)
			g.gameState.SystemCallbacks.Message.AddRowStr("Zzzzzzz...")
		}).
		Wait(game_state.BedRestInitialDelayMs).
		Animate(game_state.BedRestFadeToBlackMs, func(progress float32) {
			g.gameState.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
		}).
		Then(func() {
			result = g.gameState.RestInOwnDeedBed()
			for _, name := range result.PoisonDeaths {
				g.gameState.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("Alas! %s has died!", name))
			}
		}).
		Wait(game_state.BedRestPreWakeDelayMs).
		Animate(game_state.BedRestFadeFromBlackMs, func(progress float32) {
			g.gameState.SystemCallbacks.Visual.SetMapFadeLevel(float64(1 - progress))
		}).
		Then(func() {
			if g.gameState.PartyState.IsAllPartyDead() {
				return
			}
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundRoosterCrow)
			g.gameState.SystemCallbacks.Message.AddRowStr("Rested!")
			g.gameState.MoveAvatarOutOfBed()
		}).
		Finally(func() {
			g.gameState.SystemCallbacks.Visual.SetMapFadeLevel(0)
			g.keyboard.ResetReleaseDebouncing()
		}).
		Start()
	g.deferSurfaceCommandCompletion(newSurfaceCommandResult(surfaceCommandDeferred, origin))
}
