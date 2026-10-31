package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// REDUX DEVIATION: CRAFT-0001: the original binds no Alt chords. With Gameplay.Crafting.Enabled,
// Alt+G gathers, Alt+C crafts and Alt+B builds or stakes a deed; see ref:CRAFT-0001.

const (
	craftingMenuButtonWidthPercent = 0.40
	craftingMenuBorderWidthPercent = 0.48
	deedNamePattern                = "^[ -~]{0,24}$"
	deedNamingWorkflowName         = "deed_naming"
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
		// Gathering arrives with resource nodes in a later step (ADR-014 step 2).
		messages.AddRowStr("Nothing to gather yet!")
	case command == game_state.CraftingCraft:
		// Recipes arrive with stations in a later step (ADR-014 step 3).
		messages.AddRowStr("Nothing to craft yet!")
	case g.gameState.IsOnDeedMap():
		// Building pieces arrive in a later step (ADR-014 step 3).
		messages.AddRowStr("Nothing to build yet!")
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
		messages.AddRowStr(game_state.DeedRejection(proposal.Err))

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

// craftingCommandHelpEntries describes the three chords and where they work now.
func (g *GameScene) craftingCommandHelpEntries() []widgets.CommandHelpEntry {
	entries := []widgets.CommandHelpEntry{
		{Key: "Alt+G", Purpose: "Gather resources", NextInput: "Optional Crafting and Deeds (Configure). On your deed map. " +
			"Gathering arrives in a later version."},
		{Key: "Alt+C", Purpose: "Craft", NextInput: "Optional Crafting and Deeds. On the surface overworld or your deed map. " +
			"Recipes arrive in a later version."},
		{Key: "Alt+B", Purpose: "Build or stake a deed", NextInput: fmt.Sprintf("Optional Crafting and Deeds. "+
			"On open overworld land, on foot: buy a deed for %d gold and name it; a hut marks it, and Enter on the hut "+
			"opens your land. Esc cancels for free.", references.DeedGoldCost)},
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
