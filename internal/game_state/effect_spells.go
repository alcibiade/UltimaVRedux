package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// effect_spells.go - Reusable spell effect functions
// These can be triggered by: scrolls, cast spells, potions, or magic items.
// Each effect function handles its own feedback messages via SystemCallbacks.
// Returns true if the spell/scroll was successfully used.

// Spell effect constants from original Ultima V
const (
	// Light spell durations in gameplay minutes.
	InLorLightDuration  = 100 // In Lor (Light) spell duration
	VasLorLightDuration = 255 // Vas Lor (Great Light) prepared spell duration

	// Food creation
	CreateFoodDiceSize = 3 // In Xen Mani creates 1-3 food

	// Resurrection
	ResurrectInitialHP = 1 // HP after resurrection
)

// PartySpellEffectResult is the authoritative outcome of a party-target spell.
type PartySpellEffectResult struct {
	Applied bool
	Amount  int
}

// ActivateLightEffect provides magical illumination. Recasting refreshes
// its independent timer without replacing the active duration effect. ref:SPELL-0004
func (gs *GameState) ActivateLightEffect(minutes int) bool {
	gs.MagicLightMinutesRemaining = max(minutes, 0)
	gs.MapState.Lighting.SetMagicLight(gs.MagicLightMinutesRemaining > 0)
	gs.GetCurrentLayeredMap().InvalidateLightSourceCache()
	gs.SystemCallbacks.UI.ShowActionFeedback("Light!", true)

	return true
}

// ActivateProtectionEffect provides combat defense bonus (+3 defense).
// Called by: In Sanct scroll (100 turns), In Sanct spell (variable).
// Returns true if the effect was activated or refreshed.
func (gs *GameState) ActivateProtectionEffect(turns int) bool {
	activated := gs.setDurationEffect(DurationEffectProtection, turns)
	if activated {
		gs.SystemCallbacks.UI.ShowActionFeedback("Protection!", true)
	}

	return true
}

// ActivateNegateMagicEffect blocks monster spellcasting.
// Called by: In An scroll (20 turns), In An spell (10 turns).
// Returns true if the effect was activated or refreshed.
func (gs *GameState) ActivateNegateMagicEffect(turns int) bool {
	activated := gs.setDurationEffect(DurationEffectNegateMagic, turns)
	if activated {
		gs.SystemCallbacks.UI.ShowActionFeedback("Negate magic!", true)
	}

	return true
}

// ActivateTimeStopEffect suppresses affected actors' actions and calendar time.
// Presentation independently slows their body animations to a near standstill;
// feedback and environmental clocks continue normally. See ref:SPELL-0004.
// Called by: An Tym scroll (20 turns), An Tym spell (10 turns).
// Returns true if the effect was activated or refreshed.
// NOTE: Does NOT show message - caller (UI layer) handles popup dialog.
func (gs *GameState) ActivateTimeStopEffect(turns int) bool {
	gs.setDurationEffect(DurationEffectTimeStop, turns)

	return true
}

// ActivateQuicknessEffect doubles party speed (time passes half as fast).
// Called by: Rel Tym spell (30 turns).
// Returns true if the effect was activated or refreshed.
func (gs *GameState) ActivateQuicknessEffect(turns int) bool {
	activated := gs.setDurationEffect(DurationEffectQuickness, turns)
	if activated {
		gs.SystemCallbacks.UI.ShowActionFeedback("Quickness!", true)
	}

	return true
}

// ActivateConfusionEffect causes monsters to attack randomly.
// Called by: An Quas spell (20 turns).
// Returns true if the effect was activated or refreshed.
func (gs *GameState) ActivateConfusionEffect(turns int) bool {
	activated := gs.setDurationEffect(DurationEffectConfusion, turns)
	if activated {
		gs.SystemCallbacks.UI.ShowActionFeedback("Confusion!", true)
	}

	return true
}

// ResurrectCharacter applies the original resurrection stat recalculation.
// Called by: In Mani Corp scroll, Resurrect spell.
// Returns true if character was resurrected, false if character is not dead.
// NOTE: Does NOT show message - caller (UI layer) handles popup dialog.
func (gs *GameState) ResurrectCharacter(characterIndex int) bool {
	pc := gs.PartyState.GetCharacter(characterIndex)
	if pc == nil || !pc.IsDead() {
		return false
	}

	pc.RestoreMpByClass()
	if gs.PartyState.Karma.Value < 98 {
		pc.Exp = uint16((int(pc.Exp) * gs.PartyState.Karma.Value) / KarmaPercentageBase)
	}
	gs.recalculateLevelFromExperience(pc)
	pc.SetStatus(party_state.Good)
	pc.SetHp(ResurrectInitialHP)

	return true
}

// AwakenCharacter wakes a sleeping party member.
// Changes status from Sleep to Good.
// Called by: An Zu spell, Awaken potion.
// Returns true if character was awakened, false if character is not asleep.
func (gs *GameState) AwakenCharacter(characterIndex int) bool {
	pc := gs.PartyState.GetCharacter(characterIndex)
	if pc == nil {
		return false
	}

	// Can't awaken dead characters
	if pc.IsDead() {
		return false
	}

	// Check if actually asleep
	if !pc.IsAsleep() {
		return false
	}

	if gs.CombatScene != nil {
		for sourceSlot := range sourceCombatQueueSlots {
			entry := gs.CombatScene.TurnQueue.getActiveEntryAtSourceSlot(sourceSlot)
			if entry == nil || !entry.Combatant.IsPC() || entry.EntityIndex != characterIndex {
				continue
			}
			identity := turnQueueIdentity(entry)
			if gs.CombatScene.isCombatPCSleeping(identity) {
				gs.setCombatPCSleeping(identity, false)

				return true
			}
		}
	}

	// Wake the character
	pc.SetStatus(party_state.Good)

	return true
}

// CreateFood conjures 1-3 food and returns the amount actually added after saturation.
// Called by: In Xen Mani spell.
func (gs *GameState) CreateFood() uint16 {
	food := &gs.PartyState.Inventory.Provisions.Food
	before := food.Get()
	food.IncrementBy(uint16(gs.RollDice(CreateFoodDiceSize)))

	return food.Get() - before
}

// CastMani applies the prepared spell's original 1d30 healing mutation.
func (gs *GameState) CastMani(characterIndex int) PartySpellEffectResult {
	pc := gs.PartyState.GetCharacter(characterIndex)
	if pc == nil || pc.IsDead() {
		return PartySpellEffectResult{}
	}
	hpBefore := pc.GetCurrentHp()
	// Original: healing calls transformed d30 for Mani and yellow potions.
	// See ref:RNG-0001.
	amount := gs.RollOriginalD30()
	pc.AddHp(uint16(amount))
	changes := transientStatusChanges{}
	changes.observeHealing(hpBefore, pc)
	gs.publishTransientStatusChanges(changes)

	return PartySpellEffectResult{Applied: true, Amount: amount}
}

// CastVasMani restores a living, wounded target to maximum HP.
func (gs *GameState) CastVasMani(characterIndex int) PartySpellEffectResult {
	pc := gs.PartyState.GetCharacter(characterIndex)
	if pc == nil || pc.IsDead() ||
		(gs.GetCurrentSpellContext() == references.ContextCombat && gs.CombatScene != nil && !gs.CombatScene.Victory) {
		return PartySpellEffectResult{}
	}
	before := pc.GetCurrentHp()
	pc.SetHp(pc.GetMaxHp())
	changes := transientStatusChanges{}
	changes.observeHealing(before, pc)
	gs.publishTransientStatusChanges(changes)

	return PartySpellEffectResult{Applied: true, Amount: int(pc.GetCurrentHp() - before)}
}

// CastAnNox cures poison on a living party member.
func (gs *GameState) CastAnNox(characterIndex int) PartySpellEffectResult {
	pc := gs.PartyState.GetCharacter(characterIndex)
	if pc == nil || pc.IsDead() || !pc.IsPoisoned() {
		return PartySpellEffectResult{}
	}
	statusBefore := pc.GetStatus()
	pc.SetStatus(party_state.Good)
	changes := transientStatusChanges{}
	changes.observePoison(statusBefore, pc)
	gs.publishTransientStatusChanges(changes)

	return PartySpellEffectResult{Applied: true}
}
