package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// CombatSpellOutcome separates an unsuccessful save/damage result from a
// successful status effect. Unspecified is reserved for other event producers.
type CombatSpellOutcome int

const (
	// CombatSpellUnspecified is used by combat feedback outside these area spells.
	CombatSpellUnspecified CombatSpellOutcome = iota
	// CombatSpellAvoided means the target passed the dexterity gate.
	CombatSpellAvoided
	// CombatSpellResisted means the target passed the intelligence save.
	CombatSpellResisted
	// CombatSpellImmune means resolution reached a spell-specific immunity gate.
	CombatSpellImmune
	// CombatSpellZeroDamage means a resolved damage branch removed no HP.
	CombatSpellZeroDamage
	// CombatSpellDamaged means actual HP was removed.
	CombatSpellDamaged
	// CombatSpellSlept means the sleep effect was successfully applied.
	CombatSpellSlept
	// CombatSpellPoisoned means a Good PC was successfully poisoned.
	CombatSpellPoisoned
)

// CombatAreaSpellTargetResult retains the exact actor at resolution, rather than
// asking presentation to infer victims or outcomes from affected positions.
type CombatAreaSpellTargetResult struct {
	Target        CombatFeedbackEntity
	CombatantID   CombatantID
	Disclosed     bool
	Outcome       CombatSpellOutcome
	HPLost        int
	Fatal         bool
	MonsterInjury CombatMonsterInjury
}

// CombatAreaSpellResult contains one entry per reached, eligible target, in
// resolution order. An empty Targets slice means no targets were reached, not
// that targets resisted. This result never changes paid-cast acceptance.
type CombatAreaSpellResult struct {
	Spell   references.Spell
	Targets []CombatAreaSpellTargetResult
}

// CombatAreaSpellOutcome describes the completed cast, independently of costs.
type CombatAreaSpellOutcome int

const (
	// CombatAreaSpellNoTargets means no eligible entries were reached.
	CombatAreaSpellNoTargets CombatAreaSpellOutcome = iota
	// CombatAreaSpellUnaffected means entries were reached but no effect succeeded.
	CombatAreaSpellUnaffected
	// CombatAreaSpellAffected includes both HP and status effects.
	CombatAreaSpellAffected
)

// Outcome distinguishes an empty footprint from encountered but unaffected
// targets. Status-only success is an effect even without HP loss.
func (result CombatAreaSpellResult) Outcome() CombatAreaSpellOutcome {
	if len(result.Targets) == 0 {
		return CombatAreaSpellNoTargets
	}
	for _, target := range result.Targets {
		if target.Outcome == CombatSpellDamaged || target.Outcome == CombatSpellSlept || target.Outcome == CombatSpellPoisoned {
			return CombatAreaSpellAffected
		}
	}

	return CombatAreaSpellUnaffected
}

// applied preserves the legacy processed-position contract: a zero damage roll
// was processed, but is not a successful HP/status effect.
func (result CombatAreaSpellTargetResult) applied() bool {
	return result.Outcome == CombatSpellDamaged || result.Outcome == CombatSpellZeroDamage ||
		result.Outcome == CombatSpellSlept || result.Outcome == CombatSpellPoisoned
}

func (result CombatAreaSpellResult) processedPositions() []references.Position {
	positions := make([]references.Position, 0, len(result.Targets))
	for _, target := range result.Targets {
		if target.applied() {
			positions = append(positions, target.Target.Position)
		}
	}

	return positions
}

func (g *GameState) combatSpellTargetResult(identity CombatTurnIdentity) CombatAreaSpellTargetResult {
	_, disclosed := g.ProjectCombatActor(identity, 0)
	result := CombatAreaSpellTargetResult{
		Target:    combatFeedbackIdentity(identity, g.CombatScene.GetCombatantPosition(identity)),
		Disclosed: disclosed,
	}
	if !identity.IsPartyMember {
		result.CombatantID = g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex).CombatantID()
	}

	return result
}

func combatSpellFeedbackResult(outcome CombatSpellOutcome) CombatFeedbackResult {
	switch outcome {
	case CombatSpellAvoided, CombatSpellResisted, CombatSpellImmune:
		return CombatFeedbackMiss
	case CombatSpellZeroDamage:
		return CombatFeedbackGraze
	case CombatSpellDamaged:
		return CombatFeedbackDamage
	case CombatSpellSlept:
		return CombatFeedbackSleep
	case CombatSpellPoisoned:
		return CombatFeedbackPoison
	case CombatSpellUnspecified:
		panic("area spell target requires an outcome")
	default:
		panic("unknown area spell outcome")
	}
}

// One event owns both the durable report and target-local presentation. No
// second impact callback or affected-position explosion duplicates that result.
func (g *GameState) publishCombatAreaSpellTarget(
	casterIdx int, spell references.Spell, result CombatAreaSpellTargetResult, diagnostics CombatDiagnostics,
) CombatAreaSpellTargetResult {
	caster := g.combatPartyAttackerIdentity(casterIdx)
	attacker := combatFeedbackIdentity(caster, g.CombatScene.GetCombatantPosition(caster))
	diagnostics.Spell = spell
	diagnostics.MagicDamage = true
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind: CombatEventSpell, Attacker: attacker, Target: result.Target,
		Result: combatSpellFeedbackResult(result.Outcome), HPLost: result.HPLost, Fatal: result.Fatal,
		From: attacker.Position, To: result.Target.Position, Diagnostics: diagnostics, SpellTarget: &result,
		MonsterInjury: result.MonsterInjury,
	})

	return result
}

func combatSpellDamageOutcome(hpLost int) CombatSpellOutcome {
	if hpLost == 0 {
		return CombatSpellZeroDamage
	}

	return CombatSpellDamaged
}

func (g *GameState) resolveCombatConeResults(
	spell references.Spell, casterIdx int, candidates []references.Position,
) CombatAreaSpellResult {
	result := CombatAreaSpellResult{Spell: spell}
	processed := make(map[CombatTurnIdentity]struct{})
	poisonOnset := false
	for _, position := range candidates {
		target, found := g.CombatScene.combatConeVictimAt(position, processed)
		if !found {
			continue
		}
		outcome := g.resolveCombatConeTarget(spell, casterIdx, target)
		poisonOnset = poisonOnset || outcome.Outcome == CombatSpellPoisoned
		result.Targets = append(result.Targets, outcome)
	}
	if poisonOnset {
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconPoisonOnset)
	}

	return result
}

func (g *GameState) resolveCombatConeTarget(
	spell references.Spell, casterIdx int, target CombatTurnIdentity,
) CombatAreaSpellTargetResult {
	//nolint:exhaustive // Only the four configured combat cones reach resolution.
	switch spell {
	case references.InZu:
		return g.applyCombatInZuToTarget(casterIdx, target, g.PartyState.GetCharacter(casterIdx).Intelligence)
	case references.InNoxHur:
		return g.applyCombatPoisonWindToxify(casterIdx, target, g.rollCombatDexterityThreshold(g.effectiveCombatDexterity(target)))
	case references.InFlamHur:
		return g.applyCombatInFlamHurToTarget(casterIdx, target)
	case references.InVasGravCorp:
		return g.applyCombatInVasGravCorpToTarget(casterIdx, target)
	default:
		panic("unsupported combat cone spell")
	}
}
