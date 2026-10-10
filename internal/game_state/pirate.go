package game_state

import (
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Pirate spawning constants matching original Ultima V
// See ref:SHIP-0005.
const (
	// pirateMinDistanceFromPlayer is the minimum distance from player for pirate spawning
	// Original: spawn-location validation requires >6 tiles from player
	// See ref:SHIP-0005.
	pirateMinDistanceFromPlayer = 6

	// pirateSpawnAreaSize is the size of the spawn search area around the player
	// Original: world creature generation searches 32x32 area
	// See ref:SHIP-0005.
	pirateSpawnAreaSize = 32

	// pirateMaxSpawnAttempts is how many times to try finding a valid spawn location
	// Original: world creature generation tries up to 128 times
	// See ref:SHIP-0005.
	pirateMaxSpawnAttempts = 128
)

// isPiratePredicate returns a predicate function that matches pirate ship enemies
func isPiratePredicate() map_units.EnemyMatchFunc {
	return func(enemy *map_units.NPCEnemy) bool {
		return enemy.Variant() == map_units.EnemyVariantPirate
	}
}

// ForceSpawnPirateAtRandomLocation spawns a pirate ship at a random off-screen
// deep water location, bypassing probability checks. Used for debug commands.
//
// Returns true if pirate was spawned, false if no valid location found
func (g *GameState) ForceSpawnPirateAtRandomLocation() bool {
	_, spawned := g.ForceSpawnPirateAtRandomLocationWithPosition()

	return spawned
}

// ForceSpawnPirateAtRandomLocationWithPosition returns the inserted position on success.
func (g *GameState) ForceSpawnPirateAtRandomLocationWithPosition() (references.Position, bool) {
	// Only spawn on large maps
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return references.Position{X: 0, Y: 0}, false
	}

	playerPos := g.MapState.PlayerLocation.Position
	layeredMap := g.GetCurrentLayeredMap()

	// Try up to pirateMaxSpawnAttempts times to find a valid location
	for range pirateMaxSpawnAttempts {
		// Generate random position within spawn area
		randX := references.Coordinate(g.GetRNG().Intn(pirateSpawnAreaSize))
		randY := references.Coordinate(g.GetRNG().Intn(pirateSpawnAreaSize))

		// Calculate position relative to player
		candidateX := (playerPos.X - pirateSpawnAreaSize/2 + randX) & references.MapCoordinateWrapMask
		candidateY := (playerPos.Y - pirateSpawnAreaSize/2 + randY) & references.MapCoordinateWrapMask
		candidatePos := references.Position{X: candidateX, Y: candidateY}

		// Check distance from player (must be >6 tiles on each axis)
		distX, distY := playerPos.GetWrappedAxisDistances(candidatePos)
		if distX < pirateMinDistanceFromPlayer || distY < pirateMinDistanceFromPlayer {
			continue // Too close to player
		}

		// Check if tile is deep water
		tile := layeredMap.GetTopTile(&candidatePos)
		if !tile.Is(indexes.Water1) {
			continue // Not deep water
		}

		// Spawn pirate facing toward player
		if !g.spawnPirateAt(candidatePos, playerPos) {
			return references.Position{X: 0, Y: 0}, false
		}

		return candidatePos, true
	}

	// Never found a valid location after pirateMaxSpawnAttempts
	return references.Position{X: 0, Y: 0}, false
}

// spawnPirateAt creates a pirate ship enemy at the specified position
// facing toward the target position (usually the player)
func (g *GameState) spawnPirateAt(pos references.Position, targetPos references.Position) bool {
	// Determine initial direction toward player
	direction := g.calculateDirectionToward(pos, targetPos)
	spriteIndex := g.getPirateSpriteForDirection(direction)

	pirateTile := g.GameReferences.TileReferences.GetTile(spriteIndex)
	pirateEnemy := references.NewPirateShipEnemyReference(pirateTile)

	// Create NPC enemy instance
	npcEnemy := map_units.NewEnemyNPC(pirateEnemy, references.DynamicSpawnIndex)
	npcEnemy.SetPos(pos)
	npcEnemy.SetFloor(g.MapState.PlayerLocation.Floor)
	npcEnemy.SetVisible(true)

	// Add to current large map NPC controller
	return g.CurrentNPCAIController.GetNpcs().AddEnemy(npcEnemy)
}

// calculateDirectionToward returns the primary direction from source toward target
func (g *GameState) calculateDirectionToward(source, target references.Position) references.Direction {
	dx := int(target.X) - int(source.X)
	dy := int(target.Y) - int(source.Y)

	// Handle wraparound on large map
	if dx > references.MapWrapThreshold {
		dx -= references.MapCoordinateSpace
	} else if dx < -references.MapWrapThreshold {
		dx += references.MapCoordinateSpace
	}

	if dy > references.MapWrapThreshold {
		dy -= references.MapCoordinateSpace
	} else if dy < -references.MapWrapThreshold {
		dy += references.MapCoordinateSpace
	}

	// Choose primary direction based on larger axis
	if absInt(dx) > absInt(dy) {
		if dx > 0 {
			return references.Right
		}

		return references.Left
	}

	if dy > 0 {
		return references.Down
	}

	return references.Up
}

// getPirateSpriteForDirection returns the appropriate pirate ship sprite for a direction
func (g *GameState) getPirateSpriteForDirection(direction references.Direction) indexes.SpriteIndex {
	switch direction {
	case references.Up:
		return indexes.PirateShipUp
	case references.Right:
		return indexes.PirateShipRight
	case references.Down:
		return indexes.PirateShipDown
	case references.Left:
		return indexes.PirateShipLeft
	case references.NoneDirection:
		return indexes.PirateShipDown // No direction specified - default facing down
	default:
		return indexes.PirateShipDown // Unknown direction - default facing down
	}
}

// HasAnyPirate returns true if any pirate ship exists on the current map
func (g *GameState) HasAnyPirate() bool {
	return g.CurrentNPCAIController.GetNpcs().HasEnemyMatching(isPiratePredicate())
}

// Pirate attack constants matching original Ultima V
// See ref:SHIP-0005.
const (
	pirateAttackWorkflowName = "pirate_attack"

	// pirateAttackRange admits an aligned target less than four tiles away;
	// cardinal adjacency is resolved before the ranged opportunity. ref:SHIP-0005
	pirateAttackRange = 4
)

// PirateAttackOutcome identifies one source-ordered pirate actor action.
type PirateAttackOutcome uint8

// Pirate attack outcomes distinguish warning, blocked travel, and authoritative hits.
const (
	PirateAttackWarning PirateAttackOutcome = iota
	PirateAttackBlocked
	PirateAttackHit
)

// PirateAttackResult is detached authoritative state for one pirate action.
type PirateAttackResult struct {
	Outcome       PirateAttackOutcome
	Source        references.Position
	Target        references.Position
	ProjectileEnd references.Position
	Hit           SourceHitResult
}

type pirateAttackRequest struct {
	result PirateAttackResult
	// Fireballs share the FIFO projectile workflow with pirate broadsides so
	// mixed outdoor attackers cannot overwrite the single projectile channel.
	fireball bool
}

// CheckPirateAttack handles aligned pirate fire at a distance below
// four tiles. A misaligned pirate retains one warning/rotation phase.
// Already aligned pirates fire in descending object-slot order within
// the phase; cardinal contact has its separate earlier route. ref:SHIP-0005
func (g *GameState) CheckPirateAttack() bool {
	// Only check on large maps
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return false
	}

	// Get all pirate ships using generic lookup
	npcs := g.CurrentNPCAIController.GetNpcs()
	pirates := npcs.GetEnemiesMatching(isPiratePredicate())
	playerPos := g.MapState.PlayerLocation.Position
	floor := g.MapState.PlayerLocation.Floor
	slices.SortFunc(pirates, func(left, right *map_units.NPCEnemy) int {
		return right.MapUnitDetails().NPCNum - left.MapUnitDetails().NPCNum
	})

	acted := false
	for _, pirate := range pirates {
		if pirate.IsVisible() && pirate.Floor() == floor && g.checkPirateActorAttack(pirate, playerPos) {
			acted = true
		}
	}

	return acted
}

func (g *GameState) checkPirateActorAttack(pirate *map_units.NPCEnemy, target references.Position) bool {
	dx, dy := pirate.PosPtr().GetWrappedAxisDistances(target)
	if !pirateInCannonRange(dx, dy) {
		return false
	}
	result := PirateAttackResult{Outcome: PirateAttackWarning, Source: pirate.Pos(), Target: target}
	if !g.isPirateInBroadsidePosition(pirate, target) {
		// REDUX DEVIATION: pirate cannon fire rotates and fires in one call.
		// Redux preserves the resolved one-phase warning before later eligible fire.
		// See ref:SHIP-0005.
		g.rotatePirateToBroadside(pirate, target)
	} else {
		result = g.resolvePirateCannon(pirate, target)
	}
	g.pirateAttackQueue = append(g.pirateAttackQueue, pirateAttackRequest{result: result})
	g.startPirateAttackWorkflow()

	return true
}

func pirateInCannonRange(absDisX, absDisY int) bool {
	inRangeOnXAxis := absDisX == 0 && absDisY > 0 && absDisY < pirateAttackRange
	inRangeOnYAxis := absDisY == 0 && absDisX > 0 && absDisX < pirateAttackRange

	return inRangeOnXAxis || inRangeOnYAxis
}

// FirePirateCannon fires a cannon from pirate ship to player position
// See ref:SHIP-0005.
func (g *GameState) FirePirateCannon(pirate *map_units.NPCEnemy, targetPos references.Position) {
	result := g.resolvePirateCannon(pirate, targetPos)
	g.pirateAttackQueue = append(g.pirateAttackQueue, pirateAttackRequest{
		result: result,
	})
	g.startPirateAttackWorkflow()
}

func (g *GameState) resolvePirateCannon(
	pirate *map_units.NPCEnemy,
	target references.Position,
) PirateAttackResult {
	result := PirateAttackResult{
		Outcome: PirateAttackBlocked,
		Source:  pirate.Pos(),
		Target:  target,
	}
	var pathClear bool
	result.ProjectileEnd, pathClear = g.traceOutdoorMissile(result.Source, target, references.MissileCannonBall)
	if !pathClear {
		return result
	}

	result.Outcome = PirateAttackHit
	result.Hit = g.ResolveSourceHit()

	return result
}

func (g *GameState) startPirateAttackWorkflow() {
	if g.EventQueueManager.HasWorkflow(pirateAttackWorkflowName) {
		return
	}

	q := g.EventQueueManager.Workflow(pirateAttackWorkflowName).
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput()
	g.buildPirateAttackQueue(q, true)
	q.Finally(func() {
		g.pirateAttackQueue = nil
		g.SystemCallbacks.Visual.ClearProjectile()
		g.SystemCallbacks.Visual.SetScreenFlashIntensity(0.0)
		g.SystemCallbacks.Visual.SetScreenShakeOffset(0.0, 0.0)
	})
	q.Start()
}

func (g *GameState) buildPirateAttackQueue(q *event_queue.SequentialEventBuilder, first bool) {
	if len(g.pirateAttackQueue) == 0 {
		panic("pirate attack workflow started without a queued request")
	}
	request := g.pirateAttackQueue[0]
	start := func() {
		if request.result.Outcome == PirateAttackWarning {
			g.SystemCallbacks.Message.AddRowStr("* Pirate ship turns *")

			return
		}
		if !request.fireball {
			g.SystemCallbacks.Message.AddRowStr("* BOOOM! *")
		}
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundCannonFire)
	}
	if first {
		q.First(start)
	} else {
		q.Then(start)
	}
	missile := references.MissileCannonBall
	if request.fireball {
		missile = references.MissileRed
	}
	q.If(func() bool { return request.result.Outcome != PirateAttackWarning }, func(attack *event_queue.SequentialEventBuilder) {
		attack.ThenSequence(g.ProjectileAnimationSequence(
			request.result.Source,
			request.result.ProjectileEnd,
			missile,
			func() {
				if request.result.Outcome == PirateAttackHit {
					g.SystemCallbacks.Visual.PlayKapowAtPosition(request.result.Target)
				}
			},
		)).If(func() bool { return request.result.Outcome == PirateAttackHit }, func(hit *event_queue.SequentialEventBuilder) {
			hit.ThenSequence(
				NewPartyStateBuilder(g).
					ToAllParty().
					FlashRed().
					ShakeModerate().
					WithSound(SoundBroadside).
					Build()).
				Then(func() {
					g.presentSourceHitOutcome(request.result.Hit)
					if request.result.Hit.PartyWiped {
						g.SystemCallbacks.Flow.Consequences.TriggerDeathSequence()
					}
				})
		})
	}).
		Then(func() {
			g.SystemCallbacks.Visual.ClearProjectile()
			g.pirateAttackQueue = g.pirateAttackQueue[1:]
		}).
		If(func() bool { return len(g.pirateAttackQueue) > 0 }, func(next *event_queue.SequentialEventBuilder) {
			g.buildPirateAttackQueue(next, false)
		})
}

// getPirateCurrentDirection returns the pirate ship's current facing direction based on sprite
func (g *GameState) getPirateCurrentDirection(pirate *map_units.NPCEnemy) references.Direction {
	return pirate.Facing()
}

// isPirateInBroadsidePosition checks if pirate can fire at target with current facing
// Ships must fire perpendicular to their facing direction (broadsides only)
// - Ship facing N/S can only fire E/W
// - Ship facing E/W can only fire N/S
func (g *GameState) isPirateInBroadsidePosition(pirate *map_units.NPCEnemy, targetPos references.Position) bool {
	pirateDir := g.getPirateCurrentDirection(pirate)
	piratePos := pirate.Pos()

	dx := int(targetPos.X) - int(piratePos.X)
	dy := int(targetPos.Y) - int(piratePos.Y)

	// Handle wraparound on large map
	if dx > references.MapWrapThreshold {
		dx -= references.MapCoordinateSpace
	} else if dx < -references.MapWrapThreshold {
		dx += references.MapCoordinateSpace
	}

	if dy > references.MapWrapThreshold {
		dy -= references.MapCoordinateSpace
	} else if dy < -references.MapWrapThreshold {
		dy += references.MapCoordinateSpace
	}

	// Target on same X axis (above/below pirate) → firing N/S → ship must face E/W
	if dx == 0 {
		return pirateDir == references.Left || pirateDir == references.Right
	}

	// Target on same Y axis (left/right of pirate) → firing E/W → ship must face N/S
	if dy == 0 {
		return pirateDir == references.Up || pirateDir == references.Down
	}

	// Not on axis - shouldn't happen in attack check
	return false
}

// rotatePirateToBroadside rotates pirate ship to a broadside position for firing at target
// Pirate spends this turn rotating, will fire on next turn when in position
func (g *GameState) rotatePirateToBroadside(pirate *map_units.NPCEnemy, targetPos references.Position) {
	piratePos := pirate.Pos()

	dx := int(targetPos.X) - int(piratePos.X)

	// Handle wraparound
	if dx > references.MapWrapThreshold {
		dx -= references.MapCoordinateSpace
	} else if dx < -references.MapWrapThreshold {
		dx += references.MapCoordinateSpace
	}

	var broadsideDir references.Direction

	// Target on same X axis (above/below) → need to face E or W to fire N/S broadside
	if dx == 0 {
		// Choose Right or Left - could be smarter but Right is fine
		broadsideDir = references.Right
	} else {
		// Target on same Y axis (left/right) → need to face N or S to fire E/W broadside
		broadsideDir = references.Up
	}

	newSprite := g.getPirateSpriteForDirection(broadsideDir)
	newTile := g.GameReferences.TileReferences.GetTile(newSprite)
	pirate.SetFacing(broadsideDir, newTile)
}
