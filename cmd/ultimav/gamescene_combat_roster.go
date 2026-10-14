package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/mainscreen"
)

func combatRosterText(state *game_state.GameState) [party_state.MaxCharactersInParty]mainscreen.CharacterCombatText {
	var overlay [party_state.MaxCharactersInParty]mainscreen.CharacterCombatText
	if !state.IsInCombat() {
		return overlay
	}
	for index := range overlay {
		// Known roster entries represent Slot 0 only, even while a clone owns the
		// turn. Do not use ProjectCombatActor: underground actors are concealed
		// there, but their already-known roster row still needs condition feedback.
		identity := game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: index}
		conditions := combatConditions(state.CombatScene, identity)
		overlay[index] = mainscreen.CharacterCombatText{
			InCombat: true, Possessed: conditions.Possessed, Sleeping: conditions.Sleeping,
		}
	}

	return overlay
}

func (g *GameScene) drawCharacterSummary(screen *ebiten.Image) {
	g.characterSummary.SetCombatText(combatRosterText(g.gameState))
	g.characterSummary.Draw(&g.gameState.PartyState, screen,
		g.gameState.GetActivePlayerIndex(),
		g.gameState.IsSoloMoveMode(),
		g.gameState.GetSoloMoveCharacterIndex())
}
