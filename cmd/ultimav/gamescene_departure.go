package main

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const (
	escapeDepartureButtonWidthPercent = 0.28
	escapeDepartureBorderWidthPercent = 0.396
)

type escapeDeparture uint8

const (
	escapeDepartureMainMenu escapeDeparture = iota
	escapeDepartureApplication
)

type playerSaveWriteFlow struct {
	draft     playerSaveDescriptionDraft
	onCancel  func()
	onSaved   func(player_save.Record)
	onFailure func(string, error)
}

func (g *GameScene) openEscapeDeparturePrompt(game *Game, departure escapeDeparture, notice string) {
	var prompt *widgets.ButtonListModal
	prompt = widgets.NewButtonListModal(
		escapeDepartureTitle(departure),
		func() {
			g.dialogStack.MustRemoveWidgetInstance(prompt)
			g.DoEscapeMenu(game)
		},
		g.keyboard,
		&gameScreenPercents,
	)
	prompt.SetButtonWidthPercent(escapeDepartureButtonWidthPercent)
	prompt.SetBorderWidthPercent(escapeDepartureBorderWidthPercent)
	description := escapeDepartureDescription(departure)
	if notice != "" {
		description = notice + " " + description
	}
	prompt.SetDescription(description)
	prompt.AddButton("Quick Save", func() {
		g.dialogStack.MustRemoveWidgetInstance(prompt)
		g.quickSaveBeforeDeparture(game, departure)
	})
	prompt.AddButton("Save", func() {
		g.dialogStack.MustRemoveWidgetInstance(prompt)
		g.openPlayerSaveBrowserWithState(
			game,
			playerSaveBrowserWrite,
			"",
			"",
			g.departureSaveWriteFlow(game, departure),
		)
	})
	prompt.AddButton(escapeDepartureWithoutSaveLabel(departure), func() {
		g.dialogStack.MustRemoveWidgetInstance(prompt)
		g.completeEscapeDeparture(game, departure)
	})
	prompt.AddButton("Cancel", func() {
		g.dialogStack.MustRemoveWidgetInstance(prompt)
		g.DoEscapeMenu(game)
	})
	g.dialogStack.PushModalDialog(prompt)
	g.keyboard.SetForceWaitAnyKey(escapeMenuForceWaitTimeMs)
}

func (g *GameScene) quickSaveBeforeDeparture(game *Game, departure escapeDeparture) {
	record, err := game.quickSavePlayerSave()
	if err == nil {
		g.addRowStr("Quick saved " + playerSaveSlotName(record.Slot) + ".")
		g.completeEscapeDeparture(game, departure)

		return
	}
	if errors.Is(err, ErrNoActivePlayerSaveSlot) {
		g.openPlayerSaveBrowserWithState(
			game,
			playerSaveBrowserWrite,
			"",
			"Choose a slot before continuing.",
			g.departureSaveWriteFlow(game, departure),
		)

		return
	}
	g.openEscapeDeparturePrompt(
		game,
		departure,
		"Quick Save failed: "+playerSaveErrorDetail(err),
	)
}

func (g *GameScene) departureSaveWriteFlow(game *Game, departure escapeDeparture) *playerSaveWriteFlow {
	return &playerSaveWriteFlow{
		onCancel: func() {
			g.openEscapeDeparturePrompt(game, departure, "Save cancelled.")
		},
		onSaved: func(record player_save.Record) {
			g.addRowStr("Saved " + playerSaveSlotName(record.Slot) + ".")
			g.completeEscapeDeparture(game, departure)
		},
		onFailure: func(action string, err error) {
			g.openEscapeDeparturePrompt(
				game,
				departure,
				fmt.Sprintf("%s failed: %s", action, playerSaveErrorDetail(err)),
			)
		},
	}
}

func (g *GameScene) completeEscapeDeparture(game *Game, departure escapeDeparture) {
	switch departure {
	case escapeDepartureMainMenu:
		if err := game.returnToMainMenu(g.gameConfig); err != nil {
			g.openEscapeDeparturePrompt(game, departure, "Main menu failed: "+err.Error())
		}
	case escapeDepartureApplication:
		game.requestExit()
	default:
		panic("GameScene.completeEscapeDeparture: unsupported destination")
	}
}

func escapeDepartureTitle(departure escapeDeparture) string {
	if departure == escapeDepartureMainMenu {
		return "Return to Main Menu"
	}

	return "Exit to DOS"
}

func escapeDepartureDescription(departure escapeDeparture) string {
	if departure == escapeDepartureMainMenu {
		return "Your current game will close. Save your progress before returning to the main menu?"
	}

	return "Unsaved progress will be lost. Save before leaving Ultima V Redux?"
}

func escapeDepartureWithoutSaveLabel(departure escapeDeparture) string {
	if departure == escapeDepartureMainMenu {
		return "Discard & Main Menu"
	}

	return "Exit Without Save"
}
