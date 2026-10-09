package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
)

func (g *GameScene) closeRunicReference() {
	if sign, ok := g.runicReference.owner.(*widgets.SignDialog); ok {
		sign.SetTranslated(false)
	}
	g.runicReference.open, g.runicReference.owner = false, nil
}

// captureRunicTranslationChord also guards workflow raw AnyKey polling, which
// may happen before scene input routing. Capturing does not toggle or register T.
func (g *GameScene) captureRunicTranslationChord() bool {
	state := &g.runicReference
	if !g.keyboard.IsKeyHeld(ebiten.KeyT) {
		state.translationHeld, state.translationHandled = false, false

		return false
	}
	if g.keyboard.IsModifierPressed(input.ModifierCtrl) {
		state.translationHeld = true
	}

	return state.translationHeld
}

// handleRunicTranslationShortcut resolves a captured chord before modal Update.
// SignDialog polls physical AnyKey rather than the key-consumption registry.
func (g *GameScene) handleRunicTranslationShortcut() {
	state := &g.runicReference
	if state.translationHandled {
		return
	}
	if !state.open || !g.runicReferenceAvailable() {
		state.translationHandled = true // Gaining eligibility while held must not toggle.

		return
	}
	sign, ok := g.dialogStack.PeekTopModalDialog().(*widgets.SignDialog)
	if !ok || state.owner != sign || sign.IsDismissed() {
		state.translationHandled = true

		return
	}
	// The physical-release latch already prevents held repeats. A freshly
	// released/repressed T must not be rejected by the ordinary 165ms repeat
	// interval. Keep transition/release barriers and retry if they block input.
	g.keyboard.SetAllowKeyPressImmediately()
	if !g.keyboard.IsKeyPressedAndRegister(ebiten.KeyT) {
		return
	}
	state.translationHandled = true
	sign.SetTranslated(!sign.IsTranslated())
}
