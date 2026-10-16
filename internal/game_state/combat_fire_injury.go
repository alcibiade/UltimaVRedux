package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// CombatFireInjury is a detached resolved injury. Every field is scalar/value
// data: durable event tapes must never retain a live combat graph or provider.
// EntryID is captured before damage can retire or replace the target entry.
type CombatFireInjury struct {
	Target      CombatFeedbackEntity
	EncounterID CombatEncounterID
	EntryID     uint64
	CombatantID CombatantID
	Disclosed   bool
	HPLost      int
	Fatal       bool
}

// CombatFireInjuryOwner is an ephemeral presentation binding, never event data.
// The scene captures it synchronously at admission to reject a replaced combat
// object even when an imported/reconstructed encounter has the same numeric ID.
type CombatFireInjuryOwner struct {
	combat *CombatState
	queue  *CombatTurnQueue
}

// BindCombatFireInjury captures presentation ownership without changing the
// detached result. Projection validates scalar lifetime IDs on every use.
func (g *GameState) BindCombatFireInjury(injury CombatFireInjury) CombatFireInjuryOwner {
	if g.CombatScene == nil || g.CombatScene.EncounterID != injury.EncounterID {
		return CombatFireInjuryOwner{}
	}

	return CombatFireInjuryOwner{combat: g.CombatScene, queue: g.CombatScene.TurnQueue}
}

func (g *GameState) captureCombatFireInjury(identity CombatTurnIdentity) CombatFireInjury {
	view, disclosed := g.ProjectCombatActor(identity, 0)
	entry := g.CombatScene.TurnQueue.GetEntryByIdentity(identity)
	injury := CombatFireInjury{
		Target:      combatFeedbackIdentity(identity, g.CombatScene.GetCombatantPosition(identity)),
		CombatantID: view.CombatantID, Disclosed: disclosed,
		EncounterID: g.CombatScene.EncounterID,
	}
	if entry != nil {
		injury.EntryID = entry.lifetimeID
	}
	if !identity.IsPartyMember {
		injury.CombatantID = g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex).CombatantID()
	}

	return injury
}

// CombatSpellFireInjury adapts the existing typed Flame Wind result. Disclosure,
// HP loss and monster birth identity come from resolution, including fatal hits;
// the exact retained queue entry supplies PC lifetime without consulting the active PC.
func (g *GameState) CombatSpellFireInjury(result CombatAreaSpellTargetResult) CombatFireInjury {
	identity := CombatTurnIdentity{
		IsPartyMember: result.Target.IsPartyMember, EntityIndex: result.Target.Index, Slot: result.Target.Slot,
	}
	injury := CombatFireInjury{
		Target: result.Target, CombatantID: result.CombatantID, Disclosed: result.Disclosed,
		HPLost: result.HPLost, Fatal: result.Fatal,
		EncounterID: g.CombatScene.EncounterID,
	}
	if entry := g.CombatScene.TurnQueue.GetEntryByIdentity(identity); entry != nil {
		injury.EntryID = entry.lifetimeID
	}

	return injury
}

// ProjectCombatFireInjury returns the current disclosed actor position, or the
// recorded fatal square after retirement. A dead actor needs no live sprite to
// keep its label; a replaced entry, hidden actor or different encounter cannot
// inherit it. This read-only projection consumes neither RNG nor gameplay time.
func (g *GameState) ProjectCombatFireInjury(injury CombatFireInjury, owner CombatFireInjuryOwner) (references.Position, bool) {
	combat := g.CombatScene
	if combat == nil || combat != owner.combat || combat.TurnQueue != owner.queue || combat.EncounterID != injury.EncounterID ||
		injury.EntryID == 0 || !injury.Disclosed || injury.HPLost <= 0 {
		return references.Position{}, false
	}
	identity := CombatTurnIdentity{
		IsPartyMember: injury.Target.IsPartyMember, EntityIndex: injury.Target.Index, Slot: injury.Target.Slot,
	}
	entry := combat.TurnQueue.GetEntryByIdentity(identity)
	if (entry != nil && entry.lifetimeID != injury.EntryID) || (entry == nil && !injury.Fatal) {
		return references.Position{}, false
	}
	if identity.IsPartyMember && combat.IsCombatPCUnderground(identity) {
		return references.Position{}, false
	}
	if !identity.IsPartyMember {
		monster := combat.MonsterManager.GetMonster(identity.EntityIndex)
		if monster != nil && (monster.CombatantID() != injury.CombatantID || monster.IsInvisible() || monster.IsCorpserHidden()) {
			return references.Position{}, false
		}
		if injury.Fatal && monster != nil && !monster.IsDead() {
			return references.Position{}, false
		}
	} else if injury.Fatal && g.PartyState.GetCharacter(identity.EntityIndex).GetCurrentHp() != 0 {
		return references.Position{}, false
	}
	if injury.Fatal {
		return injury.Target.Position, g.CombatTerrainAllowsObjectDisplay(injury.Target.Position)
	}
	view, disclosed := g.ProjectCombatActor(identity, injury.CombatantID)

	return view.Position, disclosed
}
