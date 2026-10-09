package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// ShowZStatsDossier opens the read-only dossier as the active modal.
func (g *GameScene) ShowZStatsDossier(dossier game_state.ZStatsDossier) {
	g.keyboard.PrepareForTransition(0)

	var dialog *widgets.ZStatsDossierDialog
	dialog = widgets.NewZStatsDossierDialog(dossier, widgets.ZStatsDossierDialogConfig{
		Keyboard:    g.keyboard,
		SpriteSheet: g.spriteSheet,
		OnDismiss: func() {
			g.keyboard.PrepareForTransition(0)
			g.dialogStack.MustRemoveWidgetInstance(dialog)
		},
	})
	g.dialogStack.PushModalDialog(dialog)
}
