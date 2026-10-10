package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
)

// pollAtlasMarkRequest reserves exact Ctrl+N through N release, including when
// unavailable. Releasing Ctrl first must never turn the chord into New Order.
func (g *GameScene) pollAtlasMarkRequest() bool {
	if !g.keyboard.IsKeyHeld(ebiten.KeyN) {
		g.atlasMarkChordHeld = false
	}
	if g.atlasMarkChordHeld {
		return false
	}
	if !g.keyboard.IsModifierComboPressed(input.ModifierCtrl, ebiten.KeyN) ||
		g.keyboard.IsModifierPressed(input.ModifierShift) ||
		g.keyboard.IsModifierPressed(input.ModifierAlt) ||
		g.keyboard.IsModifierPressed(input.ModifierMeta) {
		return false
	}
	g.atlasMarkChordHeld = true

	// A fresh reader shortcut is release-gated rather than movement-repeat
	// delayed. Explicit transition delays still reject this one attempt.
	return g.keyboard.IsTextKeyPressedAndRegister(ebiten.KeyN)
}

// The shortcut admits only surface-overworld exploration. The existing note
// workflow owns current-fix eligibility, captured position and explanatory errors.
func (g *GameScene) canOpenAtlasMark() bool {
	location := g.gameState.MapState.PlayerLocation

	return location.Location == references.BritanniaUnderworld &&
		location.Floor == 0 && g.canOpenAtlas()
}
