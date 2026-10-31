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
// Alt+G gathers, Alt+C crafts and Alt+B builds or stakes a deed; see ref:CRAFT-0001.

// REDUX DEVIATION: CRAFT-0019: the original has no Craft menu. Its rows name reagents by short
// names and drop the inputs of an over-long row whose station is out of reach; see ref:CRAFT-0019.

const (
	// Recipe rows name the recipe, its inputs and what is missing ("Alchemy table: 6 Planks,
	// 1 Ingots (needs Workbench)"), so crafting menus use most of the map's width.
	craftingMenuButtonWidthPercent = 0.66
	craftingMenuBorderWidthPercent = 0.70
	// craftMenuRowMaxChars is the longest recipe row that fits the menu at 1280x720. A longer
	// row whose station is out of reach drops its inputs (ref:CRAFT-0019).
	craftMenuRowMaxChars   = 52
	deedNamePattern        = "^[ -~]{0,24}$"
	deedNamingWorkflowName = "deed_naming"
)

// craftingChordCommand maps an accepted key to its crafting command when Alt is held.
func (g *GameScene) craftingChordCommand(key ebiten.Key) (game_state.CraftingCommand, bool) {
	if !g.keyboard.IsModifierPressed(input.ModifierAlt) {
		return 0, false
	}
	//nolint:exhaustive // Only G, C and B carry crafting chords.
	switch key {
	case ebiten.KeyG:
		return game_state.CraftingGather, true
	case ebiten.KeyC:
		return game_state.CraftingCraft, true
	case ebiten.KeyB:
		return game_state.CraftingBuild, true
	default:
		return 0, false
	}
}

// handleCraftingChord is the one guard every map passes before its own command routing. Any
// G, C or B accepted with Alt held belongs to it, so a chord never becomes Get, Cast or Board.
// Like exact Ctrl+N, a chord counts only with Alt alone: with Ctrl, Shift or Meta also held, or
// with the option off, it answers "What?". Where the command does not apply it answers
// "Not here!". Refusals open nothing and spend no time (ref:CRAFT-0001).
func (g *GameScene) handleCraftingChord(key ebiten.Key) bool {
	command, isChord := g.craftingChordCommand(key)
	if !isChord {
		return false
	}
	exact := !g.keyboard.IsModifierPressed(input.ModifierCtrl) &&
		!g.keyboard.IsModifierPressed(input.ModifierShift) &&
		!g.keyboard.IsModifierPressed(input.ModifierMeta)
	messages := g.gameState.SystemCallbacks.Message
	switch {
	case !exact || !g.gameState.IsCraftingEnabled():
		messages.AddRowStr("What?")
	case !g.gameState.CraftingCommandApplies(command):
		messages.AddRowStr("Not here!")
	case command == game_state.CraftingGather:
		g.beginGather()
	case command == game_state.CraftingCraft:
		g.beginCraft()
	case g.gameState.IsOnDeedMap():
		g.beginDeedBuild()
	default:
		g.beginStakeDeed()
	}

	return true
}

// craftingCompletion finishes a menu-driven crafting command on the overworld. Cancelling and
// refusals are free; a completed command spends the ordinary surface turn, and nothing more
// (ref:CRAFT-0002).
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

// beginStakeDeed answers Alt+B on the overworld surface: on the party's own hut it points to
// Enter; on a suitable square it offers a deed for sale; elsewhere it explains the refusal.
func (g *GameScene) beginStakeDeed() {
	messages := g.gameState.SystemCallbacks.Message
	if deed, owned := g.gameState.DeedAt(g.gameState.MapState.PlayerLocation.Position); owned {
		messages.AddRowStr(deed.Name + " is yours. Enter to go in.")

		return
	}
	proposal := g.gameState.ProposeDeed()
	if proposal.Err != nil {
		messages.AddRowStr(g.gameState.DeedRejection(proposal.Err))

		return
	}
	messages.ShowCommandPrompt("Build-")
	finish := g.craftingCompletion(references.LargeMapType)
	var menu *widgets.ButtonListModal
	description := fmt.Sprintf("This %s can be yours. A deed costs %d gold; you carry %d. "+
		"Enter its hut to reach your own land.", proposal.Terrain.Name(), references.DeedGoldCost,
		g.gameState.PartyState.Inventory.Gold.Get())
	menu = g.newCraftingMenu("Build", description, func() {
		g.appendToCurrentRowStr("Cancelled")
		finish(surfaceCommandCancelled)
	})
	menu.AddButton(fmt.Sprintf("Stake a deed here (%d gold)", references.DeedGoldCost), func() {
		g.dialogStack.MustRemoveWidgetInstance(menu)
		g.appendToCurrentRowStr("Stake a deed")
		g.startDeedNaming(finish)
	})
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

// startDeedNaming asks for the deed's name and stakes it. Esc or an empty name stakes nothing.
func (g *GameScene) startDeedNaming(finish func(surfaceCommandState)) {
	state := surfaceCommandCancelled
	g.gameState.EventQueueManager.Workflow(deedNamingWorkflowName).
		Priority(event_queue.PriorityMedium).
		Sequential().BlockInput().
		First(func() {}).
		ThenSequence(g.thenShowInputBoxWithEscapeClear(
			"Name your land:",
			deedNamePattern,
			func(text string, submitted bool) { state = g.resolveDeedName(text, submitted) },
		)).
		Finally(func() { finish(state) }).
		Start()
}

// resolveDeedName stakes the deed once the name prompt closes and reports how the Build
// command ends: an Esc or an unusable name stakes nothing and is free.
func (g *GameScene) resolveDeedName(text string, submitted bool) surfaceCommandState {
	messages := g.gameState.SystemCallbacks.Message
	if !submitted {
		messages.AddRowStr("Cancelled")

		return surfaceCommandCancelled
	}
	if _, ok := game_state.NormalizeDeedName(text); !ok {
		messages.AddRowStr("A deed needs a name.")

		return surfaceCommandRejected
	}
	if _, err := g.gameState.StakeDeed(text); err != nil {
		return surfaceCommandRejected
	}

	return surfaceCommandCompleted
}

// beginGather asks for the direction of the node to gather (ref:CRAFT-0005). The deed map's
// secondary input routes the answer to GatherDeedNode; Esc cancels for free.
func (g *GameScene) beginGather() {
	g.gameState.SystemCallbacks.Message.ShowCommandPrompt("Gather-")
	g.secondaryKeyState = GatherDirectionInput
	g.keyboard.SetAllowKeyPressImmediately()
}

// craftingGatherResult completes a gather: an attempt spends the ordinary turn on top of the
// minutes GatherDeedNode already spent; a refusal spends nothing.
func (g *GameScene) craftingGatherResult(attempted bool, origin references.GeneralMapType) surfaceCommandResult {
	if attempted {
		return newSurfaceCommandResult(surfaceCommandCompleted, origin)
	}
	result := newSurfaceCommandResult(surfaceCommandRejected, origin)
	result.ConsumesNormalCycle = false

	return result
}

// beginCraft opens the recipe menu (ref:CRAFT-0007): first the categories, each saying how many
// of its recipes can be made now, then the recipes of one category. Every recipe is listed; one
// the party cannot make now says so in its row. Esc on the categories, or choosing a recipe that
// cannot be made, spends nothing; Esc on a category's recipes returns to the categories. A craft
// spends exactly the recipe's minutes, the ordinary turn included, and reopens the category's
// recipes on the same row so the party can craft again; Esc then closes the menu for free.
func (g *GameScene) beginCraft() {
	messages := g.gameState.SystemCallbacks.Message
	messages.ShowCommandPrompt("Craft-")
	g.openCraftCategoryMenu(g.craftingCompletion(g.gameState.MapState.PlayerLocation.Location.GetMapType()))
}

func (g *GameScene) openCraftCategoryMenu(finish func(surfaceCommandState)) {
	options := g.gameState.CraftRecipeOptions()
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu("Craft", g.craftMenuDescription(), func() {
		g.appendToCurrentRowStr("Cancelled")
		finish(surfaceCommandCancelled)
	})
	for _, category := range references.CraftCategories() {
		inCategory := craftCategoryOptions(options, category)
		ready := 0
		for _, option := range inCategory {
			if option.Available() {
				ready++
			}
		}
		if len(inCategory) == 0 {
			continue
		}
		label := fmt.Sprintf("%s (%d ready)", category.Name(), ready)
		if ready == 0 {
			label = category.Name() + " (none ready)"
		}
		menu.AddButton(label, func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			g.openCraftRecipeMenu(category, inCategory, finish, 0, false)
		})
	}
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

// craftCategoryOptions keeps the options of one category, in menu order.
func craftCategoryOptions(options []game_state.CraftRecipeOption, category references.CraftCategory) []game_state.CraftRecipeOption {
	inCategory := make([]game_state.CraftRecipeOption, 0, len(options))
	for _, option := range options {
		if option.Recipe.Category == category {
			inCategory = append(inCategory, option)
		}
	}

	return inCategory
}

// openCraftRecipeMenu lists a category's recipes with row selected. Each craft completes its own
// command with the ordinary turn, then reopens the menu on the same row with fresh rows while
// GameState allows it (repeat). After a craft, Esc only closes the menu: every craft has already
// spent its minutes, so closing spends nothing (ref:CRAFT-0007).
func (g *GameScene) openCraftRecipeMenu(
	category references.CraftCategory, options []game_state.CraftRecipeOption, finish func(surfaceCommandState),
	selected int, repeat bool,
) {
	var menu *widgets.ButtonListModal
	menu = g.newCraftingMenu(category.Name(), g.craftRecipeMenuDescription(options), func() {
		if !repeat {
			g.openCraftCategoryMenu(finish)
		}
	})
	if repeat {
		menu.SetEscapeLabel("Close")
	}
	for index, option := range options {
		recipe := option.Recipe
		menu.AddButton(craftMenuRowLabel(option), func() {
			g.dialogStack.MustRemoveWidgetInstance(menu)
			if repeat {
				g.gameState.SystemCallbacks.Message.ShowCommandPrompt("Craft-")
			}
			g.appendToCurrentRowStr(recipe.Name)
			if err := g.gameState.Craft(recipe.ID); err != nil {
				finish(surfaceCommandRejected)

				return
			}
			finish(surfaceCommandCompleted)
			if g.gameState.CraftMenuMayStayOpen() {
				g.openCraftRecipeMenu(category, craftCategoryOptions(g.gameState.CraftRecipeOptions(), category), finish,
					index, true)
			}
		})
	}
	menu.SetSelectedIndex(selected)
	g.dialogStack.PushModalDialog(menu)
	g.keyboard.PrepareForTransition(0)
}

// craftMenuRowLabel names a recipe and its inputs, reagents by their short names, and says in
// words why it cannot be made now. A row too long for the menu whose station is out of reach
// names only the recipe and the station; its inputs show at the station (ref:CRAFT-0019).
func craftMenuRowLabel(option game_state.CraftRecipeOption) string {
	label := fmt.Sprintf("%s: %s", option.Recipe.Name, references.DescribeCraftGoodsCompact(option.Recipe.Inputs))
	switch {
	case !option.StationReady:
		needs := " (needs " + option.Recipe.Station.Name() + ")"
		if len(label)+len(needs) > craftMenuRowMaxChars {
			return option.Recipe.Name + needs
		}

		return label + needs
	case !option.Affordable:
		return label + " (lacking)"
	default:
		return label
	}
}

// craftRecipeMenuDescription summarizes the stock a category's recipes draw on. A category with
// reagent inputs lists the carried reagents and Planks by short name and explains the short
// names (ref:CRAFT-0019).
func (g *GameScene) craftRecipeMenuDescription(options []game_state.CraftRecipeOption) string {
	usesReagents := false
	for _, option := range options {
		for _, input := range option.Recipe.Inputs {
			usesReagents = usesReagents || input.Kind == references.CraftGoodsReagent
		}
	}
	if !usesReagents {
		return g.craftMenuDescription()
	}
	inventory := &g.gameState.PartyState.Inventory
	carried := make([]references.CraftGoods, 0, len(references.ReagentCatalog())+1)
	if planks := inventory.Materials.Get(references.Planks); planks > 0 {
		carried = append(carried, references.MaterialGoods(references.Planks, planks))
	}
	legend := make([]string, 0, len(references.ReagentCatalog()))
	for _, reagent := range references.ReagentCatalog() {
		if quantity := inventory.Reagent.Get(reagent); quantity > 0 {
			carried = append(carried, references.ReagentGoods(reagent, quantity))
		}
		legend = append(legend, reagent.ShortName()+" "+reagent.Name())
	}
	stock := "no reagents"
	if len(carried) > 0 {
		stock = describeCarriedCompact(carried)
	}

	return fmt.Sprintf("Thou dost carry %s. Key: %s.", stock, strings.Join(legend, ", "))
}

// describeCarriedCompact renders carried stock with counts first, reagents by short name, such
// as "4 Planks, 12 SA, 3 Gi".
func describeCarriedCompact(goods []references.CraftGoods) string {
	parts := make([]string, 0, len(goods))
	for _, item := range goods {
		name := item.Name()
		if item.Kind == references.CraftGoodsReagent {
			name = references.Reagent(item.ID).ShortName()
		}
		parts = append(parts, fmt.Sprintf("%d %s", item.Quantity, name))
	}

	return strings.Join(parts, ", ")
}

// craftMenuDescription summarizes the stock recipes draw on.
func (g *GameScene) craftMenuDescription() string {
	inventory := &g.gameState.PartyState.Inventory
	carried := make([]references.CraftGoods, 0, references.NumberOfMaterials)
	for _, material := range references.MaterialCatalog() {
		if quantity := inventory.Materials.Get(material); quantity > 0 {
			carried = append(carried, references.MaterialGoods(material, quantity))
		}
	}
	stock := "no materials"
	if len(carried) > 0 {
		stock = references.DescribeCraftGoods(carried)
	}

	return fmt.Sprintf("Thou dost carry %s.", stock)
}

// craftingCommandHelpEntries describes the three chords and where they work now.
func (g *GameScene) craftingCommandHelpEntries() []widgets.CommandHelpEntry {
	entries := []widgets.CommandHelpEntry{
		{Key: "Alt+G", Purpose: "Gather resources", NextInput: "Optional Crafting and Deeds (Configure). On your deed map, " +
			"then a direction: trees and stumps give Wood (Axe), rocks Stone and ore veins Ore (Pick); fruit trees, " +
			"cacti and reagent patches are picked by hand. Bare hands sometimes win 1 Wood or Stone. Esc cancels for free."},
		{Key: "Alt+C", Purpose: "Craft", NextInput: "Optional Crafting and Deeds. On the surface overworld or your deed map, " +
			"on foot: choose a category, then a recipe: tools, refined materials, stations, building pieces, furniture, " +
			"food, torches and keys, mundane arms and armour (Forge, Workbench), and potions, scrolls and refined " +
			"reagents (Alchemy table; reagents show as SA, Gi, Ga, SS, BM, BP, NS, MR). Most need a station built " +
			"within one step; rows say what is lacking. Esc cancels for free. After a craft the menu stays on the " +
			"same recipe to craft again, each craft taking its own time; Esc then closes it."},
		{Key: "Alt+B", Purpose: "Build or stake a deed", NextInput: fmt.Sprintf("Optional Crafting and Deeds. "+
			"On open overworld land, on foot: buy a deed for %d gold and name it; a hut marks it, and Enter on the hut "+
			"opens your land. On your deed map: place a crafted piece, or Demolish one, then a direction. "+
			"Open on your storage chest stores or takes goods; Hole Up on your own bed rests as at an inn. "+
			"Esc cancels for free.", references.DeedGoldCost)},
	}
	commands := []game_state.CraftingCommand{game_state.CraftingGather, game_state.CraftingCraft, game_state.CraftingBuild}
	for index := range entries {
		switch {
		case !g.gameState.IsCraftingEnabled():
			entries[index].Unavailable = "Enable Crafting and Deeds in Configure."
		case !g.gameState.CraftingCommandApplies(commands[index]):
			entries[index].Unavailable = "Not available here."
		}
	}

	return entries
}
