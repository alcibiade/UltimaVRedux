package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type ordinaryDirectionOutcome uint8

const ordinaryDirectionHint = "Arrows; Space: Pass"

const (
	ordinaryDirectionActive ordinaryDirectionOutcome = iota
	ordinaryDirectionSelected
	ordinaryDirectionPassed
)

type ordinaryDirectionResult struct {
	Outcome   ordinaryDirectionOutcome
	Direction references.Direction
}

// ordinaryDirectionInput implements ordinary direction selection (ref:CMD-0003). Escape and
// unrecognized keys deliberately remain active rather than becoming cancellation.
func (g *GameScene) ordinaryDirectionInput() ordinaryDirectionResult {
	inputs := []struct {
		key       ebiten.Key
		direction references.Direction
	}{
		{ebiten.KeyDown, references.Down},
		{ebiten.KeyNumpad2, references.Down},
		{ebiten.KeyUp, references.Up},
		{ebiten.KeyNumpad8, references.Up},
		{ebiten.KeyRight, references.Right},
		{ebiten.KeyNumpad6, references.Right},
		{ebiten.KeyLeft, references.Left},
		{ebiten.KeyNumpad4, references.Left},
	}
	for _, input := range inputs {
		if g.keyboard.IsKeyPressedAndRegister(input.key) {
			g.appendDirectionToOutput(input.direction)

			return ordinaryDirectionResult{Outcome: ordinaryDirectionSelected, Direction: input.direction}
		}
	}
	if g.keyboard.IsKeyPressedAndRegister(ebiten.KeySpace) {
		g.appendToCurrentRowStr("Pass")

		return ordinaryDirectionResult{Outcome: ordinaryDirectionPassed}
	}

	return ordinaryDirectionResult{Outcome: ordinaryDirectionActive}
}

// appendOrdinaryDirectionHint retains the ordinary combat reader's guidance.
// Surface commands keep their concise prompt; their input rules live in Help.
// Tactical selectors and dungeon facing have other presentation owners.
func (g *GameScene) appendOrdinaryDirectionHint(mapType references.GeneralMapType) {
	if mapType != references.CombatMapType {
		return
	}
	//nolint:exhaustive // Only states handled by ordinaryDirectionInput need this hint.
	switch g.secondaryKeyState {
	case GetDirectionInput, PushDirectionInput, OpenDirectionInput, JimmyDoorDirectionInput,
		KlimbDirectionInput, LookDirectionInput, TalkDirectionInput, SearchDirectionInput,
		AttackDirectionInput, UseDirectionInput, FireDirectionInput:
		g.output.AppendCommandHintOnce(ordinaryDirectionHint)
	}
}
