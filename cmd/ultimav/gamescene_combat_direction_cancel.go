package main

import "github.com/hajimehoshi/ebiten/v2"

func (g *GameScene) hasUncommittedCombatDirection() bool {
	//nolint:exhaustive // Specialized selectors own their separate commitment rules.
	switch g.secondaryKeyState {
	case KlimbDirectionInput, OpenDirectionInput, JimmyDoorDirectionInput,
		PushDirectionInput, GetDirectionInput, SearchDirectionInput:
		return true
	default:
		return false
	}
}

func (g *GameScene) cancelCombatDirectionInput() bool {
	if !g.hasUncommittedCombatDirection() || !g.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		return false
	}
	// REDUX DEVIATION: CMD-0003: original ordinary direction input excludes Escape.
	// Cancel accidental, uncommitted combat commands without spending the actor's action.
	g.appendToCurrentRowStr("Cancelled")
	g.secondaryKeyState = PrimaryInput
	g.keyboard.SetAllowKeyPressImmediately()

	return true
}
