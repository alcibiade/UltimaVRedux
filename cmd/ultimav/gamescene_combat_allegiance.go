package main

import (
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

type combatAllegianceKey struct {
	identity game_state.CombatTurnIdentity
	id       game_state.CombatantID
}

type combatAllegianceObservation struct {
	loyalty game_state.CombatMonsterLoyalty
	seen    bool
}

type combatAllegianceState struct {
	encounter *game_state.CombatState
	actors    map[combatAllegianceKey]combatAllegianceObservation
}

func combatAllegianceText(actor game_state.CombatActorPresentation) string {
	if actor.Loyalty == game_state.CombatMonsterLoyaltyParty {
		return "Ally"
	}

	return "Enemy"
}

func combatActorContext(actor game_state.CombatActorPresentation) string {
	parts := []string{combatAllegianceText(actor)}
	if actor.Possessed {
		parts = append(parts, "Possessed")
	}
	if actor.Charmed {
		parts = append(parts, "Charmed")
	}
	if actor.Summoned {
		parts = append(parts, "Summoned")
	}

	return strings.Join(parts, " / ")
}

// Poll only disclosed, currently presented identities. Newly created/revealed
// entries establish a silent baseline, including after hidden allegiance changes.
// This feedback never owns a workflow, changes input, or advances gameplay time.
func (g *GameScene) updateCombatAllegiance() {
	state := &g.combatAllegiance
	if state.encounter != g.gameState.CombatScene {
		*state = combatAllegianceState{encounter: g.gameState.CombatScene}
	}
	if state.encounter == nil {
		return
	}
	if state.actors == nil {
		state.actors = make(map[combatAllegianceKey]combatAllegianceObservation)
	}
	for key, previous := range state.actors {
		previous.seen = false
		state.actors[key] = previous
	}
	g.gameState.ForEachDisclosedCombatActor(func(actor game_state.CombatActorPresentation) {
		if !g.combatAllegiancePresented(actor) {
			return
		}
		key := combatAllegianceKey{identity: actor.Identity, id: actor.CombatantID}
		previous, known := state.actors[key]
		if known && previous.loyalty != actor.Loyalty {
			message := actor.Name + " now fights against you."
			if actor.Loyalty == game_state.CombatMonsterLoyaltyParty {
				message = actor.Name + " now fights for your side."
			}
			g.gameState.SystemCallbacks.Message.AddRowStr(message)
		}
		previous.loyalty, previous.seen = actor.Loyalty, true
		state.actors[key] = previous
	})
	for key, previous := range state.actors {
		if !previous.seen {
			delete(state.actors, key)
		}
	}
}

func (g *GameScene) combatAllegiancePresented(actor game_state.CombatActorPresentation) bool {
	if g.combatTeleportEffect != nil && g.combatTeleportEffect.Caster == actor.Identity && g.combatTeleportEffect.Phase == combatTeleportHidden {
		return false
	}
	if _, visible := g.combatSelectionPresentation(actor.Identity, actor.Position); !visible {
		return false
	}

	return actor.Identity.IsPartyMember || g.combatSummonEffect == nil || !g.combatSummonEffect.hidesMonster(actor.Identity.EntityIndex)
}
