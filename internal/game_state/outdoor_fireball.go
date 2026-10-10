package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	outdoorFireballRange = 3
	outdoorFireballOdds  = 8
)

// checkOutdoorActorAttack is outdoor contact and ranged attack: contact takes precedence over
// any firing roll. The controller invokes this even after an earlier attack.
// See ref:ENEMY-0006.
func (g *GameState) checkOutdoorActorAttack(enemy *map_units.NPCEnemy) ai.OutdoorActorAction {
	target := g.MapState.PlayerLocation.Position
	if positionsAreOrthogonallyAdjacentWrapped(target, enemy.Pos()) {
		contact := ClassifyOverworldEnemyContact(enemy)
		if enemy.Variant() == map_units.EnemyVariantWhirlpool {
			return g.resolveOutdoorWhirlpoolContact(enemy)
		}
		if isResolvableOverworldContact(contact) {
			if g.usesDirectDamageForOverworldContact(contact) {
				// A movement collision can already have marked this same contact.
				// Resolve it here, once, before the next actor's RNG opportunity.
				g.pendingDirectMonsterContact = contact
				g.processPendingDirectMonsterDamage()
			} else {
				g.pendingOverworldEncounter = enemy
				g.suspendOutdoorActorPhase(enemy)

				return ai.OutdoorActorDeferred
			}

			return ai.OutdoorActorAttacked
		}
	}
	if enemy.Variant() == map_units.EnemyVariantPirate {
		if g.checkPirateActorAttack(enemy, target) {
			return ai.OutdoorActorAttacked
		}
	} else if g.checkOutdoorFireball(enemy, target) {
		return ai.OutdoorActorAttacked
	}

	return ai.OutdoorActorNoAttack
}

func outdoorWhirlpoolWorkflowName(slot int) string {
	return fmt.Sprintf("outdoor_whirlpool_%d", slot)
}

// Outdoor contact resolution completes outdoor collision damage/outdoor initialization before the next outdoor contact and ranged attack. Foot hits
// can commit immediately; transport retains the source cursor in its owning
// workflow so the cinematic can still change maps at full fade-out.
// See ref:ENEMY-0006.
func (g *GameState) resolveOutdoorWhirlpoolContact(enemy *map_units.NPCEnemy) ai.OutdoorActorAction {
	slot := enemy.MapUnitDetails().NPCNum
	name := outdoorWhirlpoolWorkflowName(slot)
	if g.EventQueueManager.HasWorkflow(name) {
		// The same actor cannot acquire a second live workflow from a reentrant
		// phase request while input is owned by its first contact.
		return ai.OutdoorActorDeferred
	}
	q := g.EventQueueManager.Workflow(name).Priority(event_queue.PriorityHigh).Sequential()
	if g.PartyVehicle.GetVehicleDetails().VehicleType == references.NoPartyVehicle {
		result := g.ResolveWhirlpoolHit()
		g.buildWhirlpoolTransportationQueue(q, func() WhirlpoolHitResult { return result })
		q.Start()

		return ai.OutdoorActorAttacked
	}
	g.RemoveWhirlpoolAt(enemy.Pos())
	phase := g.suspendOutdoorActorPhase(enemy)
	g.buildOutdoorWhirlpoolTransportationQueue(q, phase)
	q.Then(func() {
		// Outdoor initialization reloads terrain, not the shared DOS object array. Redux has
		// per-world controllers: continue the cursor on the current one, never
		// on a captured old-map actor slice (original outdoor rules, original outdoor effect rules).
		// See ref:ENEMY-0006.
		phase.world = references.World(g.MapState.PlayerLocation.Floor)
		phase.ready = true
		g.continueOutdoorActorPhase(phase)
	})
	q.Start()

	return ai.OutdoorActorDeferred
}

// The outdoor owner installs one composed finalizer. Keeping transport steps
// separate from their default cleanup prevents a runtime ThenSequence builder
// from replacing the continuation's cancellation cleanup.
func (g *GameState) buildOutdoorWhirlpoolTransportationQueue(
	queue *event_queue.SequentialEventBuilder,
	phase *outdoorActorContinuation,
) {
	g.buildWhirlpoolTransportationSteps(queue, g.resolveWhirlpoolTransportation)
	queue.Finally(func() {
		g.cleanupWhirlpoolTransportation()
		if g.outdoorActorContinuation == phase && !phase.ready {
			g.cancelOutdoorActorPhase()
		}
	})
}

func (g *GameState) checkOutdoorFireball(enemy *map_units.NPCEnemy, target references.Position) bool {
	monster := enemy.EnemyReference.MonsterType
	if monster != references.MonsterSeaSerpent && monster != references.MonsterDragon {
		return false
	}
	dx, dy := enemy.PosPtr().GetWrappedAxisDistances(target)
	if dx > outdoorFireballRange || dy > outdoorFireballRange || !g.OneInXOdds(outdoorFireballOdds) {
		return false
	}
	result := PirateAttackResult{Outcome: PirateAttackBlocked, Source: enemy.Pos(), Target: target}
	var reached bool
	result.ProjectileEnd, reached = g.traceOutdoorMissile(result.Source, target, references.MissileRed)
	if reached {
		result.Outcome = PirateAttackHit
		result.Hit = g.ResolveSourceHit()
	}
	g.pirateAttackQueue = append(g.pirateAttackQueue, pirateAttackRequest{result: result, fireball: true})
	g.startPirateAttackWorkflow()

	// A launched shot consumes the opportunity even when blocked.
	// See ref:ENEMY-0006.
	return true
}
