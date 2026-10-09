package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	combatRingDisappearanceRollMaximum = 15
	combatRingDisappearanceRollResult  = 11
)

func (g *GameState) initializeCombatPartyRings(combat *CombatState) {
	if combat == nil || combat.specialRingsInitialized {
		panic("combat special rings initialized more than once")
	}
	g.initializePartySpecialRings(func(index int) {
		combat.setCombatPCInvisible(CombatTurnIdentity{IsPartyMember: true, EntityIndex: index}, true)
	})
	combat.specialRingsInitialized = true
}

func (g *GameState) initializeCampingPartyRings(camp *CampingState) {
	if camp == nil || camp.specialRingsInitialized {
		panic("camping special rings initialized more than once")
	}
	g.initializePartySpecialRings(func(index int) {
		camp.partyInvisible[index] = true
	})
	camp.specialRingsInitialized = true
}

func inheritCampingPartyRings(combat *CombatState, camp *CampingState) {
	if combat == nil || camp == nil || !camp.specialRingsInitialized || combat.specialRingsInitialized {
		panic("invalid camping special ring handoff")
	}
	combat.PartyInvisible = camp.partyInvisible
	combat.specialRingsInitialized = true
}

func (g *GameState) initializePartySpecialRings(setInvisible func(int)) {
	for index := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(index)
		if !character.IsInParty() || character.IsDead() {
			continue
		}

		ring := character.GetEquippedItem(references.SlotRing)
		if isCombatSpecialRing(ring) &&
			g.RandomIntInRange(0, combatRingDisappearanceRollMaximum) == combatRingDisappearanceRollResult {
			character.SetEquippedItem(references.SlotRing, references.NoEquipment)
			g.SystemCallbacks.Message.AddRowStr("A ring has vanished!")
			//nolint:godox // TODO(audio): Recreate original combat initialization's pitch-sweep sound disappearance cue.
			// See ref:EQUIP-0018.
		}
		g.applyPartySpecialRing(index, character.IsAsleep(), setInvisible)
	}
}

func (g *GameState) processCombatPCSpecialRing(identity CombatTurnIdentity) {
	if g.CombatScene == nil || !identity.IsPartyMember ||
		!g.CombatScene.isCombatPCActive(identity) || g.CombatScene.isCombatPCSleeping(identity) {
		return
	}
	g.applyPartySpecialRing(identity.EntityIndex, false, func(_ int) {
		g.CombatScene.setCombatPCInvisible(identity, true)
	})
}

func (g *GameState) applyPartySpecialRing(index int, sleeping bool, setInvisible func(int)) {
	if sleeping {
		return
	}
	ring := g.PartyState.GetCharacter(index).GetEquippedItem(references.SlotRing)
	switch ring { //nolint:exhaustive // Only combat special rings have effects.
	case references.RingInvisibility:
		setInvisible(index)
	case references.RingRegeneration:
		g.ProcessRegenerationRingSweep()
	}
}

func isCombatSpecialRing(equipment references.Equipment) bool {
	return equipment == references.RingInvisibility || equipment == references.RingRegeneration
}
