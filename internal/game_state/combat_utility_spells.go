package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const combatInPorAttempts = 7

// CombatInPorOutcome identifies whether battlefield teleportation found a destination.
type CombatInPorOutcome int

const (
	// CombatInPorResolved means a valid random destination was found.
	CombatInPorResolved CombatInPorOutcome = iota
	// CombatInPorFailed means all seven random destination attempts were blocked.
	CombatInPorFailed
	// CombatInPorNotHere means teleportation is unavailable in this combat context.
	CombatInPorNotHere
)

// CombatInPorResult holds an authoritative random destination before presentation applies it.
type CombatInPorResult struct {
	Outcome CombatInPorOutcome
	Caster  CombatTurnIdentity
	From    references.Position
	To      references.Position
}

func isCombatAnSanctTarget(combat *CombatState, target references.Position) bool {
	tile := combat.CombatMapRef.GetTileNumber(target)
	if tile == indexes.LockedDoor || tile == indexes.LockedDoorView {
		return true
	}
	chest, found := combat.objectAtMatching(target, CombatObject.IsChest)

	return found && chest.IsLockedAndTrapped()
}

func isCombatAnExPorTarget(combat *CombatState, target references.Position) bool {
	//nolint:exhaustive // Only door tiles can be magically locked.
	switch combat.CombatMapRef.GetTileNumber(target) {
	case indexes.RegularDoor, indexes.LockedDoor, indexes.RegularDoorView, indexes.LockedDoorView:
		return true
	default:
		return false
	}
}

func isCombatInExPorTarget(combat *CombatState, target references.Position) bool {
	tile := combat.CombatMapRef.GetTileNumber(target)

	return tile == indexes.MagicLockDoor || tile == indexes.MagicLockDoorWithView
}

// CastCombatAnSanct unlocks an adjacent mundane door or disarms a trapped chest.
func (g *GameState) CastCombatAnSanct(casterIdx int, target references.Position) CombatUtilityResult {
	tile := g.CombatScene.CombatMapRef.GetTileNumber(target)
	var result CombatUtilityResult
	//nolint:exhaustive // All non-door tiles fall through to the validated chest target.
	switch tile {
	case indexes.LockedDoor:
		g.setCombatTerrain(target, indexes.RegularDoor)
		result = CombatUtilityDoorUnlocked
	case indexes.LockedDoorView:
		g.setCombatTerrain(target, indexes.RegularDoorView)
		result = CombatUtilityDoorUnlocked
	default:
		if !g.CombatScene.unlockChestAt(target) {
			panic("combat An Sanct resolved without a valid target")
		}
		result = CombatUtilityChestDisarmed
	}
	g.publishCombatUtilitySpell(g.combatPartyAttackerIdentity(casterIdx), references.AnSanct, target, target, result)

	return result
}

// CastCombatAnExPor applies an original magical lock to an adjacent door.
func (g *GameState) CastCombatAnExPor(casterIdx int, target references.Position) {
	tile := g.CombatScene.CombatMapRef.GetTileNumber(target)
	//nolint:exhaustive // The default detects a stale or invalid target selection.
	switch tile {
	case indexes.RegularDoor, indexes.LockedDoor:
		g.setCombatTerrain(target, indexes.MagicLockDoor)
	case indexes.RegularDoorView, indexes.LockedDoorView:
		g.setCombatTerrain(target, indexes.MagicLockDoorWithView)
	default:
		panic("combat An Ex Por resolved without a valid target")
	}
	g.publishCombatUtilitySpell(
		g.combatPartyAttackerIdentity(casterIdx), references.AnExPor, target, target, CombatUtilityDoorMagicLocked,
	)
}

// CastCombatInExPor removes an original magical lock from an adjacent door.
func (g *GameState) CastCombatInExPor(casterIdx int, target references.Position) {
	//nolint:exhaustive // The default detects a stale or invalid target selection.
	switch g.CombatScene.CombatMapRef.GetTileNumber(target) {
	case indexes.MagicLockDoor:
		g.setCombatTerrain(target, indexes.RegularDoor)
	case indexes.MagicLockDoorWithView:
		g.setCombatTerrain(target, indexes.RegularDoorView)
	default:
		panic("combat In Ex Por resolved without a valid target")
	}
	g.publishCombatUtilitySpell(
		g.combatPartyAttackerIdentity(casterIdx), references.InExPor, target, target, CombatUtilityMagicLockRemoved,
	)
}

// PlanCombatInPor chooses the original random combat destination without moving the caster.
func (g *GameState) PlanCombatInPor(casterIdx int) CombatInPorResult {
	caster := g.combatPartyAttackerIdentity(casterIdx)
	from := g.CombatScene.GetCombatantPosition(caster)
	if g.CombatScene.ExitContext != CombatExitSurface {
		return CombatInPorResult{Outcome: CombatInPorNotHere, Caster: caster, From: from, To: from}
	}
	for range combatInPorAttempts {
		candidate := references.Position{
			X: references.Coordinate(g.GetRNG().Intn(16)),
			Y: references.Coordinate(g.GetRNG().Intn(16)),
		}
		if candidate.X >= references.XCombatMapTiles || candidate.Y >= references.YCombatMapTiles ||
			!g.isCombatPositionValidForPC(&candidate, caster) {
			continue
		}

		return CombatInPorResult{Outcome: CombatInPorResolved, Caster: caster, From: from, To: candidate}
	}

	return CombatInPorResult{Outcome: CombatInPorFailed, Caster: caster, From: from, To: from}
}

// ApplyCombatInPor commits a previously planned destination at the presentation handoff point.
func (g *GameState) ApplyCombatInPor(result CombatInPorResult) {
	if result.Outcome != CombatInPorResolved || g.CombatScene.GetCombatantPosition(result.Caster) != result.From {
		panic("invalid or stale combat In Por result")
	}
	g.CombatScene.setCombatantPosition(result.Caster, result.To)
	g.publishCombatUtilitySpell(
		result.Caster, references.InPor, result.From, result.To, CombatUtilityTeleported,
	)
}

func (g *GameState) publishCombatUtilitySpell(
	caster CombatTurnIdentity,
	spell references.Spell,
	from, to references.Position,
	result CombatUtilityResult,
) {
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind:          CombatEventUtilitySpell,
		Attacker:      combatFeedbackIdentity(caster, g.CombatScene.GetCombatantPosition(caster)),
		From:          from,
		To:            to,
		UtilityResult: result,
		Diagnostics:   CombatDiagnostics{Spell: spell},
	})
}
