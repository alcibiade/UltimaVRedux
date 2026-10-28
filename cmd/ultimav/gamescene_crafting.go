package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// REDUX DEVIATION: CRAFT-0001: the original binds no Alt chords. With Gameplay.Crafting.Enabled,
// Alt+G gathers, Alt+C crafts and Alt+B builds on the overworld surface, on foot; see ref:CRAFT-0001.

const (
	craftingMenuButtonWidthPercent = 0.40
	craftingMenuBorderWidthPercent = 0.48
	landClaimNamePattern           = "^[ -~]{0,24}$"
)

// isCraftingChord reports whether the accepted key is G, C or B with Alt held. Every such key
// belongs to the crafting chords on every map, so it never falls through to Get, Cast or Board.
func (g *GameScene) isCraftingChord(key ebiten.Key) bool {
	return g.keyboard.IsModifierPressed(input.ModifierAlt) &&
		(key == ebiten.KeyG || key == ebiten.KeyC || key == ebiten.KeyB)
}

// refuseCraftingChord answers a crafting chord that may not run here and reports whether it did.
// Like exact Ctrl+N, a chord counts only with Alt alone: Ctrl, Shift or Meta with it answers
// "What?", as does a chord while the option is off. Off the overworld surface or aboard a vehicle
// the answer is "Not here!". A refusal spends no time and opens no prompt or menu.
func (g *GameScene) refuseCraftingChord() bool {
	exact := !g.keyboard.IsModifierPressed(input.ModifierCtrl) &&
		!g.keyboard.IsModifierPressed(input.ModifierShift) &&
		!g.keyboard.IsModifierPressed(input.ModifierMeta)
	switch {
	case !exact || !g.gameState.IsCraftingEnabled():
		g.gameState.SystemCallbacks.Message.AddRowStr("What?")
	case !g.gameState.IsCraftingSite():
		g.gameState.SystemCallbacks.Message.AddRowStr("Not here!")
	default:
		return false
	}

	return true
}

// craftingChordOffLargeMap answers a crafting chord on a town, dungeon or combat map, before that
// map's own command routing can read it as Get, Cast or Board. It reports whether it did.
func (g *GameScene) craftingChordOffLargeMap(key ebiten.Key) bool {
	if !g.isCraftingChord(key) {
		return false
	}
	if !g.refuseCraftingChord() {
		panic("crafting chord admitted off the large map")
	}

	return true
}

// largeMapCraftingChord routes Alt+G, Alt+C and Alt+B on the overworld and the underworld. It
// reports false for any other key, leaving ordinary command routing untouched.
func (g *GameScene) largeMapCraftingChord(key ebiten.Key, origin references.GeneralMapType) (surfaceCommandResult, bool) {
	if !g.isCraftingChord(key) {
		return surfaceCommandResult{}, false
	}
	if g.refuseCraftingChord() {
		result := newSurfaceCommandResult(surfaceCommandRejected, origin)
		result.ConsumesNormalCycle = false

		return result, true
	}
	//nolint:exhaustive // Only the three crafting chords reach here.
	switch key {
	case ebiten.KeyG:
		g.gameState.SystemCallbacks.Message.ShowCommandPrompt("Gather-")
		g.secondaryKeyState = GatherDirectionInput
		g.keyboard.SetAllowKeyPressImmediately()

		return newSurfaceCommandResult(surfaceCommandStillActive, origin), true
	case ebiten.KeyC:
		g.gameState.SystemCallbacks.Message.ShowCommandPrompt("Craft-")
		g.openCraftMenu(g.craftingCompletion(origin))
	default:
		g.gameState.SystemCallbacks.Message.ShowCommandPrompt("Build-")
		g.openBuildMenu(g.craftingCompletion(origin))
	}

	return newSurfaceCommandResult(surfaceCommandDeferred, origin), true
}

// craftingCompletion finishes a menu-driven crafting command. Cancelling a menu and a refused
// recipe, claim or name are free (ref:CRAFT-0003, ref:CRAFT-0004). A completed command spends the
// ordinary surface turn here; the action itself paid only its labor beyond that turn.
func (g *GameScene) craftingCompletion(origin references.GeneralMapType) func(surfaceCommandState) {
	return func(state surfaceCommandState) {
		result := newSurfaceCommandResult(state, origin)
		if state == surfaceCommandCancelled || state == surfaceCommandRejected {
			result.ConsumesNormalCycle = false
		}
		g.completeSurfaceCommand(result)
	}
}

func (g *GameScene) newCraftingMenu(title, description string, onCancel func()) *widgets.ButtonListModal {
	var menu *widgets.ButtonListModal
	menu = widgets.NewButtonListModal(title, func() {
		g.dialogStack.MustRemoveWidgetInstance(menu)
		onCancel()
	}, g.keyboard, &gameScreenPercents)
	menu.SetButtonWidthPercent(craftingMenuButtonWidthPercent)
	menu.SetBorderWidthPercent(craftingMenuBorderWidthPercent)
	menu.SetDescription(description)

	return menu
}

func (g *GameScene) pushCraftingMenu(menu *widgets.ButtonListModal) {
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

func (g *GameScene) cancelCraftingMenu(finish func(surfaceCommandState)) func() {
	return func() {
		g.appendToCurrentRowStr("Cancelled")
		finish(surfaceCommandCancelled)
	}
}

func (g *GameScene) craftingStockDescription() string {
	inventory := &g.gameState.PartyState.Inventory
	parts := make([]string, 0, references.NumberOfMaterials+references.NumberOfTools)
	for _, material := range references.MaterialCatalog() {
		parts = append(parts, fmt.Sprintf("%d %s", inventory.Materials.Get(material), material.Name()))
	}
	for _, tool := range references.ToolCatalog() {
		if count := inventory.Tools.Get(tool); count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, tool.Name()))
		}
	}

	return "You carry " + strings.Join(parts, ", ") + "."
}

// craftingShortfallDescription names the rows the party cannot afford, so availability is
// explained in text rather than left for the player to work out from the costs.
func craftingShortfallDescription(unaffordable []string) string {
	if len(unaffordable) == 0 {
		return ""
	}

	return " Not enough for: " + strings.Join(unaffordable, ", ") + "."
}

func (g *GameScene) openCraftMenu(finish func(surfaceCommandState)) {
	var menu *widgets.ButtonListModal
	options := g.gameState.CraftRecipeOptions()
	unaffordable := make([]string, 0, len(options))
	for _, option := range options {
		if !option.Affordable {
			unaffordable = append(unaffordable, option.Recipe.Name)
		}
	}
	description := g.craftingStockDescription() + craftingShortfallDescription(unaffordable)
	menu = g.newCraftingMenu("Craft", description, g.cancelCraftingMenu(finish))
	for _, option := range options {
		recipe := option.Recipe
		menu.AddButton(fmt.Sprintf("%s (%s)", recipe.Name, recipe.Cost.Describe()), func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.appendToCurrentRowStr(recipe.Name)
			if err := g.gameState.ApplyCraft(recipe.ID); err != nil {
				finish(surfaceCommandRejected)

				return
			}
			finish(surfaceCommandCompleted)
		})
	}
	g.pushCraftingMenu(menu)
}

func (g *GameScene) openBuildMenu(finish func(surfaceCommandState)) {
	var menu *widgets.ButtonListModal
	description := g.craftingStockDescription()
	claim, inside := g.gameState.LandClaimAt(g.gameState.MapState.PlayerLocation.Position)
	if inside {
		description = "You stand on " + claim.Name + ". " + description
	}
	if !inside {
		// The cost lives in the description: a button row is too narrow for it.
		description += fmt.Sprintf(" A claim costs %d gold and %s.", references.LandClaimGoldCost,
			references.LandClaimStakeCost.Describe())
	}
	menu = g.newCraftingMenu("Build", description, g.cancelCraftingMenu(finish))
	if !inside {
		menu.AddButton("Stake a claim here", func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.appendToCurrentRowStr("Claim")
			g.startLandClaim(finish)
		})
	}
	for _, category := range references.BuildCategories() {
		menu.AddButton(category.Name(), func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.openBuildCategoryMenu(category, finish)
		})
	}
	menu.AddButton("Demolish", func() {
		g.dialogStack.MustRemoveWidgetInstance(menu)
		g.beginBuildDirection(buildDemolishPiece, "Demolish-", finish)
	})
	g.pushCraftingMenu(menu)
}

// buildDemolishPiece marks a pending Demolish rather than a piece placement.
const buildDemolishPiece = -1

func (g *GameScene) openBuildCategoryMenu(category references.BuildCategory, finish func(surfaceCommandState)) {
	var menu *widgets.ButtonListModal
	options := make([]game_state.BuildPieceOption, 0)
	unaffordable := make([]string, 0)
	for _, option := range g.gameState.BuildPieceOptions() {
		if option.Piece.Category != category {
			continue
		}
		options = append(options, option)
		if !option.Affordable {
			unaffordable = append(unaffordable, option.Piece.Name)
		}
	}
	description := g.craftingStockDescription() + craftingShortfallDescription(unaffordable)
	menu = g.newCraftingMenu(category.Name(), description, g.cancelCraftingMenu(finish))
	for _, option := range options {
		piece := option.Piece
		menu.AddButton(fmt.Sprintf("%s (%s)", piece.Name, piece.Cost.Describe()), func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.beginBuildDirection(piece.ID, piece.Name+"-", finish)
		})
	}
	g.pushCraftingMenu(menu)
}

// beginBuildDirection hands the menu command over to an ordinary direction prompt. The menu's
// own completion is released; the direction input completes the command like Push does.
func (g *GameScene) beginBuildDirection(pieceID int, prompt string, finish func(surfaceCommandState)) {
	g.pendingBuildPieceID = pieceID
	g.gameState.SystemCallbacks.Message.ShowCommandPrompt(prompt)
	g.secondaryKeyState = BuildDirectionInput
	g.keyboard.SetAllowKeyPressImmediately()
	finish(surfaceCommandStillActive)
}

func (g *GameScene) largeMapCraftingDirection(direction references.Direction) bool {
	if g.pendingBuildPieceID == buildDemolishPiece {
		return g.gameState.ActionDemolishLargeMap(direction)
	}

	return g.gameState.ActionBuildLargeMap(g.pendingBuildPieceID, direction)
}

func (g *GameScene) startLandClaim(finish func(surfaceCommandState)) {
	if proposal := g.gameState.ProposeLandClaim(); proposal.Err != nil {
		g.gameState.SystemCallbacks.Message.AddRowStr(game_state.LandClaimRejection(proposal.Err))
		finish(surfaceCommandRejected)

		return
	}
	state := surfaceCommandCancelled
	g.gameState.EventQueueManager.Workflow("land_claim_naming").
		Priority(event_queue.PriorityMedium).
		Sequential().BlockInput().
		First(func() {}).
		ThenSequence(g.thenShowInputBoxWithEscapeClear(
			"Name your land:",
			landClaimNamePattern,
			func(text string, submitted bool) {
				if !submitted {
					g.gameState.SystemCallbacks.Message.AddRowStr("Cancelled")

					return
				}
				if _, ok := game_state.NormalizeLandClaimName(text); !ok {
					g.gameState.SystemCallbacks.Message.AddRowStr("A claim needs a name.")
					state = surfaceCommandRejected

					return
				}
				if _, err := g.gameState.StakeLandClaim(text); err != nil {
					state = surfaceCommandRejected

					return
				}
				state = surfaceCommandCompleted
			},
		)).
		Finally(func() { finish(state) }).
		Start()
}

// largeMapHomeChestOpen intercepts Open on an own storage chest and reports whether it did.
// REDUX DEVIATION: CRAFT-0007: home chests hold the party's goods and cost no karma to open.
func (g *GameScene) largeMapHomeChestOpen(direction references.Direction, finish func(surfaceCommandState)) bool {
	position, ok := g.gameState.HomeChestInDirection(direction)
	if !ok {
		return false
	}
	g.appendToCurrentRowStr("Storage chest")
	g.openHomeChestMenu(position, false, finish)

	return true
}

func (g *GameScene) openHomeChestMenu(position references.Position, used bool, finish func(surfaceCommandState)) {
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu("Storage Chest", g.gameState.HomeChestSummary(position), func() {
		if used {
			finish(surfaceCommandCompleted)

			return
		}
		g.appendToCurrentRowStr("Cancelled")
		finish(surfaceCommandCancelled)
	})
	for _, deposit := range []bool{true, false} {
		label := "Take..."
		if deposit {
			label = "Store..."
		}
		menu.AddButton(label, func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.openHomeChestCategoryMenu(position, deposit, used, finish)
		})
	}
	g.pushCraftingMenu(menu)
}

func (g *GameScene) openHomeChestCategoryMenu(position references.Position, deposit, used bool, finish func(surfaceCommandState)) {
	verb := "Take"
	if deposit {
		verb = "Store"
	}
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu(verb, g.gameState.HomeChestSummary(position), func() {
		g.openHomeChestMenu(position, used, finish)
	})
	transfer := func(categories []game_state.HomeChestCategory, label string) {
		g.dialogStack.MustRemoveWidgetInstance(menu)
		moved := 0
		for _, category := range categories {
			moved += g.gameState.TransferHomeChest(position, category, deposit)
		}
		if moved == 0 {
			g.gameState.SystemCallbacks.Message.AddRowStr("Nothing to move.")
		} else {
			g.gameState.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("%s %s: %d.", verb, label, moved))
		}
		g.openHomeChestMenu(position, used || moved > 0, finish)
	}
	menu.AddButton("Everything", func() { transfer(game_state.HomeChestCategories(), "everything") })
	for _, category := range game_state.HomeChestCategories() {
		// Category names are lowercase for messages; rows start with a capital like "Everything".
		label := strings.ToUpper(category.Name()[:1]) + category.Name()[1:]
		menu.AddButton(label, func() {
			transfer([]game_state.HomeChestCategory{category}, category.Name())
		})
	}
	g.pushCraftingMenu(menu)
}
