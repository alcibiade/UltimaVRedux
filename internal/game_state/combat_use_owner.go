package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// CombatUseOwner retains the encounter and exact scheduled actor boundary that
// opened Use. An index alone can refer to a later turn or a replacement encounter.
type CombatUseOwner struct {
	combat     *CombatState
	actor      CombatTurnIdentity
	entry      *TurnQueueEntry
	provider   *CombatPCProvider
	character  *party_state.PlayerCharacter
	sequence   uint64
	origin     references.Position
	location   references.PlayerLocation
	combatMap  *references.CombatMapReference
	layeredMap *map_state.LayeredMap
}

// CaptureCombatUseOwner captures ownership without changing gameplay or forecasting.
func (g *GameState) CaptureCombatUseOwner() CombatUseOwner {
	if g.CombatScene == nil || g.CombatScene.TurnQueue == nil {
		return CombatUseOwner{}
	}
	actor := g.CombatScene.GetCurrentTurnIdentity()
	entry := g.CombatScene.TurnQueue.GetEntryByIdentity(actor)
	if !actor.IsPartyMember || entry == nil {
		return CombatUseOwner{}
	}
	provider, ok := entry.Combatant.(*CombatPCProvider)
	if !ok {
		return CombatUseOwner{}
	}

	return CombatUseOwner{
		combat:     g.CombatScene,
		actor:      actor,
		entry:      entry,
		provider:   provider,
		character:  provider.Character(),
		sequence:   g.CombatScene.nextPossessionTurn,
		origin:     g.CombatScene.GetCombatantPosition(actor),
		location:   g.MapState.PlayerLocation,
		combatMap:  g.CombatScene.CombatMapRef,
		layeredMap: g.MapState.LayeredMaps.GetLayeredMap(references.CombatMapType, 0),
	}
}

// isRetainedUseContext also retains exploration: entering combat invalidates its confirmation.
func (g *GameState) isRetainedUseContext(owner CombatUseOwner) bool {
	if owner.combat == nil {
		return g.CombatScene == nil
	}

	return g.IsCurrentCombatUseOwner(owner)
}

// IsCurrentCombatUseOwner rejects stale actor, clone, encounter and turn changes.
func (g *GameState) IsCurrentCombatUseOwner(owner CombatUseOwner) bool {
	if owner.combat == nil || owner.combat != g.CombatScene || g.CombatScene.TurnQueue == nil ||
		owner.actor != g.CombatScene.GetCurrentTurnIdentity() || owner.sequence != g.CombatScene.nextPossessionTurn ||
		g.MapState.PlayerLocation != owner.location || owner.location.Location.GetMapType() != references.CombatMapType {
		return false
	}
	entry := g.CombatScene.TurnQueue.GetEntryByIdentity(owner.actor)
	if entry == nil || entry != owner.entry || entry.Combatant != owner.provider || entry.status != turnQueueActive {
		return false
	}

	return owner.provider.Character() == owner.character &&
		g.PartyState.GetCharacter(owner.actor.EntityIndex) == owner.character &&
		g.CombatScene.isCombatPCActive(owner.actor) && !g.CombatScene.isCombatPCPossessed(owner.actor) &&
		g.CombatScene.GetCombatantPosition(owner.actor) == owner.origin &&
		g.CombatScene.CombatMapRef == owner.combatMap &&
		g.MapState.LayeredMaps.GetLayeredMap(references.CombatMapType, 0) == owner.layeredMap
}
