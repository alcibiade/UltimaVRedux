package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// CombatUtilitySpellCommit is the single-use authority retained from selection
// through delayed resolution and action completion. Presentation cannot retarget it.
type CombatUtilitySpellCommit struct {
	state     *GameState
	targeting *CombatAdjacentSpellTargeting
	target    references.Position
	spell     references.Spell
	tile      indexes.SpriteIndex
	terrain   indexes.SpriteIndex
	chest     CombatObject
	hasChest  bool
	started   bool
	resolved  bool
	finished  bool
}

func (g *GameState) newCombatUtilitySpellCommit(
	targeting *CombatAdjacentSpellTargeting, target references.Position,
) *CombatUtilitySpellCommit {
	commit := &CombatUtilitySpellCommit{
		state: g, targeting: targeting, target: target, spell: targeting.Request.Spell,
		tile: g.CombatScene.CombatMapRef.GetTileNumber(target), terrain: g.combatMapOnlyTerrain(target),
	}
	if commit.spell == references.AnSanct && commit.tile != indexes.LockedDoor && commit.tile != indexes.LockedDoorView {
		commit.chest, commit.hasChest = g.CombatScene.objectAtMatching(target, CombatObject.IsChest)
	}

	return commit
}

func (g *GameState) ownsCombatUtilitySpell(commit *CombatUtilitySpellCommit) bool {
	return commit != nil && commit.state == g && !commit.finished &&
		g.IsCurrentCombatUseOwner(commit.targeting.owner) &&
		g.CombatScene.AdjacentSpellTargeting == commit.targeting && commit.targeting.utility == commit
}

// StartCombatUtilitySpell admits one presentation owner for a paid cast.
func (g *GameState) StartCombatUtilitySpell(commit *CombatUtilitySpellCommit) bool {
	if !g.ownsCombatUtilitySpell(commit) || commit.started {
		return false
	}
	commit.started = true

	return true
}

// ResolveCombatUtilitySpell applies a paid cast once, only to its retained
// encounter, actor, dispatch, maps and exact target state.
func (g *GameState) ResolveCombatUtilitySpell(commit *CombatUtilitySpellCommit) bool {
	if !g.ownsCombatUtilitySpell(commit) || !commit.started || commit.resolved {
		return false
	}
	commit.resolved = true
	if g.CombatScene.CombatMapRef.GetTileNumber(commit.target) != commit.tile ||
		g.combatMapOnlyTerrain(commit.target) != commit.terrain {
		return false
	}
	if commit.hasChest {
		chest, found := g.CombatScene.objectAtMatching(commit.target, CombatObject.IsChest)
		if !found || commit.chest.LifetimeID() == 0 || chest.LifetimeID() != commit.chest.LifetimeID() || chest != commit.chest {
			return false
		}
	}
	caster := commit.targeting.owner.actor.EntityIndex
	//nolint:exhaustive // Only the door/disarm spell family creates this commit.
	switch commit.spell {
	case references.AnSanct:
		g.CastCombatAnSanct(caster, commit.target)
	case references.AnExPor:
		g.CastCombatAnExPor(caster, commit.target)
	case references.InExPor:
		g.CastCombatInExPor(caster, commit.target)
	default:
		panic("unsupported committed combat utility spell")
	}

	return true
}

// FinishCombatUtilitySpell settles a paid cast even on presentation shutdown.
// Stale owners are retired without mutating or finishing a replacement action.
func (g *GameState) FinishCombatUtilitySpell(commit *CombatUtilitySpellCommit) {
	if commit == nil || commit.state != g || commit.finished {
		return
	}
	owned := g.ownsCombatUtilitySpell(commit) && commit.started
	if owned {
		g.ResolveCombatUtilitySpell(commit)
		owned = g.ownsCombatUtilitySpell(commit)
	}
	commit.finished = true
	combat := commit.targeting.owner.combat
	if combat.AdjacentSpellTargeting == commit.targeting {
		combat.AdjacentSpellTargeting = nil
	}
	if owned {
		g.FinishCombatAction()
	}
}
