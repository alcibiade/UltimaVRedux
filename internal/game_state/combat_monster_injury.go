package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// CombatInjurySeverity is the original surviving-monster diagnosis (COMBAT-0008).
// It is independent of the condition update that changes fleeing and draws RNG.
type CombatInjurySeverity int

const (
	// CombatInjuryNone means no applicable positive, surviving monster damage.
	CombatInjuryNone CombatInjurySeverity = iota
	// CombatInjuryBarelyWounded includes HP at or above three truncated quarters.
	CombatInjuryBarelyWounded
	// CombatInjuryLightlyWounded includes HP at or above two truncated quarters.
	CombatInjuryLightlyWounded
	// CombatInjuryHeavilyWounded includes HP at or above one truncated quarter.
	CombatInjuryHeavilyWounded
	// CombatInjuryCritical means positive HP below one truncated quarter.
	CombatInjuryCritical
)

// ClassifyCombatMonsterInjury uses the static type baseline, not encounter HP or
// the last damage amount. Truncation precedes multiplication, including H < 4.
func ClassifyCombatMonsterInjury(hp, baselineHP int) CombatInjurySeverity {
	if hp <= 0 || baselineHP <= 0 {
		return CombatInjuryNone
	}
	quarter := baselineHP / 4
	switch {
	case hp < quarter:
		return CombatInjuryCritical
	case hp < 2*quarter:
		return CombatInjuryHeavilyWounded
	case hp < 3*quarter:
		return CombatInjuryLightlyWounded
	default:
		return CombatInjuryBarelyWounded
	}
}

// CombatMonsterInjury freezes post-hit severity and its disclosure/identity. It
// contains no live actor pointers and remains valid after healing or slot reuse.
// REDUX DEVIATION: COMBAT-0012: reporting admits actual positive surviving damage;
// original diagnosis can also describe an immortal zero-loss hit as barely wounded.
type CombatMonsterInjury struct {
	Severity    CombatInjurySeverity
	Disclosed   bool
	MonsterType references.MonsterType
	Loyalty     CombatMonsterLoyalty
	CombatantID CombatantID
}

// Call immediately after HP mutation, before callbacks, death or survivor effects.
// PC identity includes possessed PCs and clones, regardless of their allegiance.
func (g *GameState) captureCombatMonsterInjury(target CombatTurnIdentity, hpLost int) CombatMonsterInjury {
	if target.IsPartyMember || hpLost <= 0 {
		return CombatMonsterInjury{}
	}
	monster := g.CombatScene.MonsterManager.GetMonster(target.EntityIndex)
	if monster.IsDead() {
		return CombatMonsterInjury{}
	}
	view, disclosed := g.ProjectCombatActor(target, monster.CombatantID())

	return CombatMonsterInjury{
		Severity:  ClassifyCombatMonsterInjury(monster.CurrentHP(), monster.MaxHP()),
		Disclosed: disclosed, MonsterType: monster.MonsterType(), Loyalty: view.Loyalty,
		CombatantID: monster.CombatantID(),
	}
}
