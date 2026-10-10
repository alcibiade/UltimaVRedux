package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/astar"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const combatAutoPathMinimumPositions = 2

// CombatAutoActionKind identifies the single action selected by combat assistance.
type CombatAutoActionKind int

const (
	// CombatAutoActionUnavailable means no visible enemy has a reachable melee square.
	CombatAutoActionUnavailable CombatAutoActionKind = iota
	// CombatAutoActionMove advances one cardinal tile along a complete route.
	CombatAutoActionMove
	// CombatAutoActionAttack attacks a visible enemy with an eligible equipped source.
	CombatAutoActionAttack
)

// CombatAutoAction describes one non-mutating auto-advance decision.
type CombatAutoAction struct {
	Kind      CombatAutoActionKind
	Direction references.Direction
	Target    references.Position
}

type combatAutoPath struct {
	positions    []references.Position
	target       references.Position
	distance     int
	currentHP    int
	monsterIndex int
}

// PlanCombatAutoAction selects one attack or movement action without mutating
// combat state. Routes use the same terrain and occupancy rules as PC movement.
func (g *GameState) PlanCombatAutoAction() CombatAutoAction {
	if g.CombatScene == nil || g.CombatScene.AttackTargeting != nil ||
		g.CombatScene.SpellTargeting != nil || g.CombatScene.ConeTargeting != nil ||
		g.CombatScene.AdjacentSpellTargeting != nil || g.CombatScene.pendingSpell != nil ||
		g.CombatScene.pendingAttack != nil {
		return unavailableCombatAutoAction()
	}

	activeIdentity, active := g.activeCombatPCIdentity()
	if !active || g.CombatScene.HasScheduledCombatStep() ||
		g.CombatScene.isCombatPCSleeping(activeIdentity) || g.CombatScene.isCombatPCPossessed(activeIdentity) ||
		g.CombatScene.IsPartyTurnAutoPass(activeIdentity.EntityIndex) {
		return unavailableCombatAutoAction()
	}

	start := g.CombatScene.GetCombatantPosition(activeIdentity)
	if preview := g.PreviewCombatAttack(); len(preview.Targets) > 0 {
		target := preview.Targets[0]
		targetHP := g.CombatScene.MonsterManager.GetMonster(target.MonsterIndex).CurrentHP()
		for _, candidate := range preview.Targets[1:] {
			candidateHP := g.CombatScene.MonsterManager.GetMonster(candidate.MonsterIndex).CurrentHP()
			if candidate.Distance > target.Distance ||
				(candidate.Distance == target.Distance && candidateHP > targetHP) ||
				(candidate.Distance == target.Distance && candidateHP == targetHP &&
					candidate.MonsterIndex > target.MonsterIndex) {
				continue
			}
			target = candidate
			targetHP = candidateHP
		}

		return CombatAutoAction{
			Kind:      CombatAutoActionAttack,
			Direction: references.NoneDirection,
			Target:    target.Position,
		}
	}

	path := g.findShortestCombatAutoPath(activeIdentity, start)
	if len(path.positions) < combatAutoPathMinimumPositions {
		return unavailableCombatAutoAction()
	}

	direction := directionBetweenCombatPositions(path.positions[0], path.positions[1])
	if direction == references.NoneDirection {
		return unavailableCombatAutoAction()
	}

	return CombatAutoAction{
		Kind:      CombatAutoActionMove,
		Direction: direction,
		Target:    path.target,
	}
}

func unavailableCombatAutoAction() CombatAutoAction {
	return CombatAutoAction{
		Kind:      CombatAutoActionUnavailable,
		Direction: references.NoneDirection,
		Target:    references.Position{X: 0, Y: 0},
	}
}

// BeginCombatAutoAttack retains the chosen monster, not merely its square or a
// reusable queue slot. REDUX DEVIATION: Tab assistance automates the original
// manual per-item aiming sequence, without changing physical attack rules.
func (g *GameState) BeginCombatAutoAttack(position references.Position) bool {
	if g.CombatScene == nil || g.CombatScene.hasPlayerActionOwner() {
		return false
	}
	for _, target := range g.PreviewCombatAttack().Targets {
		if target.Position != position {
			continue
		}
		if !g.BeginCombatAttack() {
			return false
		}
		g.CombatScene.AttackTargeting.autoTarget = g.CombatScene.MonsterManager.GetMonster(target.MonsterIndex)

		return true
	}

	return false
}

// ContinueCombatAutoAttack revalidates each source after the preceding impact.
// It never retargets, and leaves committed ranged attacks to the shared impact owner.
func (g *GameState) ContinueCombatAutoAttack() CombatAttackTargetResult {
	if g.CombatScene == nil || g.CombatScene.AttackTargeting == nil || g.CombatScene.pendingAttack != nil {
		return combatAttackResult(CombatAttackTargetIgnored)
	}
	targeting := g.CombatScene.AttackTargeting
	target := targeting.autoTarget
	if target == nil {
		return combatAttackResult(CombatAttackTargetIgnored)
	}
	actor, active := g.activeCombatPCIdentity()
	occupant, _ := g.CombatScene.MonsterManager.GetMonsterIndexAtPosition(target.Position())
	if !active || actor != targeting.Attacker || occupant != target || !g.isVisibleCombatAutoTarget(target) ||
		g.CombatScene.isCombatPCSleeping(actor) || g.CombatScene.isCombatPCPossessed(actor) {
		g.CombatScene.AttackTargeting = nil

		return combatAttackResult(CombatAttackSequenceComplete)
	}
	targeting.Origin = g.CombatScene.GetCombatantPosition(actor)
	for {
		review := g.reviewCombatAttackPosition(actor, targeting.Origin, targeting.CurrentSource(), target.Position())
		if review.Status.Attackable() {
			targeting.Cursor = target.Position()

			return g.ConfirmCombatAttackTarget()
		}
		if review.Status == CombatShotInterference {
			engager, _ := g.validRememberedMeleeEngager(actor)
			g.publishCombatInterference(actor, engager, CombatInterruptedAttack, targeting.CurrentSource().Equipment)
		}
		if g.advanceCombatAttackSource() {
			return combatAttackResult(CombatAttackSequenceComplete)
		}
	}
}

func (g *GameState) findShortestCombatAutoPath(
	activeIdentity CombatTurnIdentity,
	start references.Position,
) combatAutoPath {
	pathfinder := astar.NewAStarMap()
	pathfinder.InitializeByMovementCost(
		start,
		references.XCombatMapTiles,
		references.YCombatMapTiles,
		func(position references.Position) int {
			if g.isCombatPositionValidForPC(&position, activeIdentity) {
				return 1
			}

			return -1
		},
	)

	best := combatAutoPath{
		positions:    nil,
		target:       references.Position{X: 0, Y: 0},
		distance:     0,
		currentHP:    0,
		monsterIndex: -1,
	}
	g.CombatScene.MonsterManager.ForEachActive(func(monsterIndex int, monster *CombatMonster) {
		if !g.isVisibleCombatAutoTarget(monster) {
			return
		}

		distance := integerDistance(start, monster.Position())
		currentHP := monster.CurrentHP()
		for _, offset := range combatMeleeOffsets() {
			candidate := references.Position{
				X: monster.Position().X + offset.X,
				Y: monster.Position().Y + offset.Y,
			}
			if !g.isCombatPositionValidForPC(&candidate, activeIdentity) {
				continue
			}

			path := pathfinder.AStar(candidate)
			if len(path) < 2 || !isBetterCombatAutoPath(path, distance, currentHP, monsterIndex, best) {
				continue
			}

			best = combatAutoPath{
				positions:    path,
				target:       monster.Position(),
				distance:     distance,
				currentHP:    currentHP,
				monsterIndex: monsterIndex,
			}
		}
	})

	return best
}

func combatMeleeOffsets() [8]references.Position {
	return [...]references.Position{
		{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
		{X: -1, Y: 0}, {X: 1, Y: 0},
		{X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
	}
}

func isBetterCombatAutoPath(path []references.Position, distance, currentHP, monsterIndex int, best combatAutoPath) bool {
	if len(best.positions) == 0 || distance != best.distance {
		return len(best.positions) == 0 || distance < best.distance
	}
	if currentHP != best.currentHP {
		return currentHP < best.currentHP
	}
	if len(path) != len(best.positions) {
		return len(path) < len(best.positions)
	}

	return monsterIndex < best.monsterIndex
}

func directionBetweenCombatPositions(from, destination references.Position) references.Direction {
	switch {
	case destination.X == from.X && destination.Y == from.Y-1:
		return references.Up
	case destination.X == from.X && destination.Y == from.Y+1:
		return references.Down
	case destination.X == from.X-1 && destination.Y == from.Y:
		return references.Left
	case destination.X == from.X+1 && destination.Y == from.Y:
		return references.Right
	default:
		return references.NoneDirection
	}
}
