package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Ship cannon constants matching original Ultima V (original command rules ship cannon fire)
// See ref:KARMA-0004.
const (
	smallCannonWorkflowName = "small_cannon_fire"
	shipCannonWorkflowName  = "ship_cannon_fire"

	// shipCannonRange is the maximum firing range in tiles
	// Original: for k=0; k<3; k++
	shipCannonRange = 3

	// shipCannonDamageMax is the maximum damage per hit
	// Original: random(1,20)
	shipCannonDamageMax = 20

	// destroyedMessageDurationMs is how long the "Destroyed!" overlay message displays
	// Shown concurrently with visual effects (flash, shake, kapow) during cannon hits
	destroyedMessageDurationMs = 3000
)

// sourceMissilePassage is missile terrain validation's complete missile passage table table. A set
// bit means the underlying terrain permits missile travel.
// See ref:KARMA-0004.
var sourceMissilePassage = [32]byte{ //nolint:gochecknoglobals // Immutable source data table.
	0xff, 0xf3, 0xc3, 0x8f, 0xff, 0xff, 0xff, 0xc0,
	0xdd, 0xf8, 0x03, 0xdf, 0xff, 0xff, 0x00, 0x00,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f,
	0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xff, 0xff,
}

// Small map cannon constants matching original Ultima V
// See ref:KARMA-0004.
const (
	// smallMapCannonRange is the maximum firing range for town cannons
	// Four positions beyond the muzzle are searched. ref:KARMA-0004.
	smallMapCannonRange = 4

	// cannonNPCKarmaPenalty is karma lost when killing NPC with cannon
	// Original: karma -= 5
	cannonNPCKarmaPenalty = 5
)

// ActionFireSmallMap handles the Fire command on small maps (towns).
// Original Ultima V: original command rules town cannon fire
// - Auto-detects adjacent cannon tile (no direction input needed)
// - Cannon direction determined by sprite variant
// - Range: 4 tiles beyond the cannon
// - Door hit: destroyed, replaced with floor
// - NPC hit: killed, -5 karma, guards activated
// See ref:KARMA-0004.
func (g *GameState) ActionFireSmallMap() bool {
	if g.EventQueueManager.HasWorkflow(smallCannonWorkflowName) {
		return false
	}
	mapType := g.MapState.PlayerLocation.Location.GetMapType()

	// Check if in dungeon - "What?" response
	// Original: Fire command only works in towns (small maps 1-32)
	if mapType == references.DungeonMapType {
		g.SystemCallbacks.Message.AddRowStr("What?")

		return false
	}

	// Find adjacent cannon
	cannonPos, cannonDir := g.findAdjacentCannon()
	if cannonPos == nil {
		g.SystemCallbacks.Message.AppendToCurrentRowStr("What?")

		return false
	}

	// Find target in firing line (door or NPC within range)
	targetPos, hitDoor, hitNPC := g.findSmallMapCannonTarget(*cannonPos, cannonDir)

	// Start cannon fire workflow
	g.StartSmallMapCannonFireWorkflow(*cannonPos, targetPos, cannonDir, hitDoor, hitNPC)

	return true
}

// findAdjacentCannon checks 4 cardinal directions for a cannon tile adjacent to player.
// Returns cannon position and firing direction if found, nil position otherwise.
// Original: original command rules check fire[0..3] for adjacent cannon
// See ref:KARMA-0004.
func (g *GameState) findAdjacentCannon() (*references.Position, references.Direction) {
	playerPos := g.MapState.PlayerLocation.Position
	layeredMap := g.GetLayeredMapByCurrentLocation()
	directions := []references.Direction{references.Up, references.Right, references.Down, references.Left}

	for _, dir := range directions {
		adjacentPos := dir.GetNewPositionInDirection(&playerPos)
		tile := layeredMap.GetTileTopMapOnlyTile(adjacentPos)

		if tile != nil && tile.IsCannon() {
			cannonDir := tile.GetCannonDirection()
			if cannonDir != references.NoneDirection {
				return adjacentPos, cannonDir
			}
		}
	}

	return nil, references.NoneDirection
}

// findSmallMapCannonTarget searches along the cannon's firing direction for targets.
// Stops at first door or NPC within range.
// Returns target position, whether a door was hit, and the NPC if hit.
// Original: original command rules iterates tiles checking for doors/NPCs
// See ref:KARMA-0004.
func (g *GameState) findSmallMapCannonTarget(
	cannonPos references.Position,
	fireDir references.Direction,
) (references.Position, bool, *map_units.MapUnit) {
	layeredMap := g.GetLayeredMapByCurrentLocation()
	npcs := g.CurrentNPCAIController.GetNpcs()
	currentPos := cannonPos

	for range smallMapCannonRange {
		nextPos := fireDir.GetNewPositionInDirection(&currentPos)

		// Check the selected terrain for a door. ref:KARMA-0004
		tile := layeredMap.GetTileTopMapOnlyTile(nextPos)
		if tile != nil && tile.Index.IsDoor() {
			return *nextPos, true, nil
		}
		// REDUX DEVIATION: KARMA-0004: original town fire can remove a selected
		// target after a blocked missile. Redux stops at blocking terrain so
		// cannon fire cannot kill occupants through walls.
		if tile == nil || !sourceMissilePasses(tile.Index) {
			return *nextPos, false, nil
		}

		// Check for NPC at this position
		mapUnit := npcs.GetMapUnitAtPositionAndFloorOrNil(*nextPos, g.MapState.PlayerLocation.Floor)
		if mapUnit != nil {
			if !smallMapCannonCanRemove(*mapUnit) {
				return *nextPos, false, nil
			}

			return *nextPos, false, mapUnit
		}

		currentPos = *nextPos
	}

	// No collision - return max range position
	return currentPos, false, nil
}

func smallMapCannonCanRemove(unit map_units.MapUnit) bool {
	// REDUX DEVIATION: KARMA-0004: the inspected original town-cannon
	// Shadowlord exclusion is ineffective. Redux protects Shadowlords from
	// ordinary cannon removal so their quest defeat remains authoritative.
	switch target := unit.(type) {
	case *map_units.NPCEnemy:
		return target.Variant() != map_units.EnemyVariantShadowlord
	case *map_units.NPCFriendly:
		monster, _ := target.NPCReference.SmallMapCombatMonsterType()

		return monster != references.MonsterShadowlord
	default:
		return true
	}
}

// StartSmallMapCannonFireWorkflow creates the event queue for small map cannon fire.
// Sequence: sound → projectile animation → kapow → effect (door destroy/NPC kill)
// See ref:KARMA-0004.
func (g *GameState) StartSmallMapCannonFireWorkflow(
	cannonPos, targetPos references.Position,
	_ references.Direction,
	hitDoor bool,
	hitNPC *map_units.MapUnit,
) {
	if g.EventQueueManager.HasWorkflow(smallCannonWorkflowName) {
		return
	}
	routeOwnsCycle := g.surfaceCycleOwned
	q := g.EventQueueManager.Workflow(smallCannonWorkflowName).
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput()

	q.First(func() {
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundCannonFire)

		g.SetCriminalStatus(true)
		if _, err := g.ActivateGuards(); err != nil {
			logging.Warnf(logging.CategoryGameState, "Unable to activate guards: %v", err)
		}
	}).ThenSequence(g.ProjectileAnimationSequence(
		cannonPos,
		targetPos,
		references.MissileCannonBall,
		func() {
			g.SystemCallbacks.Visual.PlayKapowAtPosition(targetPos)

			if hitDoor {
				layeredMap := g.GetLayeredMapByCurrentLocation()
				layeredMap.SetTileByLayer(map_state.MapLayer, &targetPos, indexes.BrickFloor)
				g.SystemCallbacks.Message.AddRowStr("Door destroyed!")
			}

			if hitNPC != nil {
				if !smallMapCannonCanRemove(*hitNPC) {
					return
				}
				removed := g.CurrentNPCAIController.GetNpcs().RemoveNPCAtPositionAndFloor(
					targetPos, g.MapState.PlayerLocation.Floor,
				)
				if !removed {
					return
				}

				g.AdjustPartyKarma(-cannonNPCKarmaPenalty)

				g.SystemCallbacks.UI.ShowTimedOverlayMessage("Destroyed!", destroyedMessageDurationMs, false, true)

				g.SystemCallbacks.Visual.SetScreenFlashColor(1.0, 0.0, 0.0)
			}
		},
	)).If(

		func() bool { return hitNPC != nil },
		func(subQ *event_queue.SequentialEventBuilder) {
			subQ.Animate(200, func(progress float32) {
				var intensity float64
				if progress < 0.5 {
					intensity = 0.7 * float64(progress*2.0)
				} else {
					intensity = 0.7 * float64(2.0-progress*2.0)
				}
				g.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)

				dampen := 1.0 - float64(progress)
				offsetX, offsetY := calculateScreenShakeOffset(progress, 6.0*dampen)
				g.SystemCallbacks.Visual.SetScreenShakeOffset(offsetX, offsetY)
			}).Then(func() {
				g.SystemCallbacks.Visual.SetScreenFlashIntensity(0.0)
				g.SystemCallbacks.Visual.SetScreenShakeOffset(0.0, 0.0)
			})
		}).Finally(

		func() {
			g.SystemCallbacks.Visual.ClearProjectile()
			if !routeOwnsCycle {
				g.FinishTurn()
			}
		}).
		Start()
}

// ShipCannonOutcome identifies an authoritative large-map broadside result.
type ShipCannonOutcome uint8

// Ship cannon outcomes cover validation, travel, damage, and removal.
const (
	ShipCannonRejectedVehicle ShipCannonOutcome = iota
	ShipCannonRejectedBroadside
	ShipCannonMissed
	ShipCannonBlocked
	ShipCannonHit
	ShipCannonDestroyed
)

// ShipCannonResult is detached from GameState for ordered presentation.
type ShipCannonResult struct {
	Outcome        ShipCannonOutcome
	Source         references.Position
	ProjectileEnd  references.Position
	TargetPosition references.Position
	HasTarget      bool
	Damage         int
	HPBefore       int
	HPAfter        int
}

func (result ShipCannonResult) accepted() bool {
	return result.Outcome >= ShipCannonMissed
}

func sourceMissilePasses(tile indexes.SpriteIndex) bool {
	value := int(tile) & 0xff

	return sourceMissilePassage[value>>3]&(0x80>>uint(value&7)) != 0
}

func (g *GameState) traceSourceMissile(
	start references.Position,
	direction references.Direction,
	distance int,
) (references.Position, bool) {
	current := start
	for range distance {
		next := direction.GetNewPositionInDirection(&current).
			GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		current = *next
		tile := g.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&current)
		if tile == nil || !sourceMissilePasses(tile.Index) {
			return current, false
		}
	}

	return current, true
}

// ResolveShipCannonFire owns source targeting, LOS, RNG, damage, and removal.
// Presentation must consume only the returned detached result.
func (g *GameState) ResolveShipCannonFire(direction references.Direction) ShipCannonResult {
	result := ShipCannonResult{
		Outcome: ShipCannonRejectedVehicle,
		Source:  g.MapState.PlayerLocation.Position,
	}
	vehicle := g.PartyVehicle.GetVehicleDetails()
	if vehicle.VehicleType != references.FrigateVehicle {
		return result
	}
	if direction < references.Up || direction > references.Right ||
		!g.isValidBroadsideDirection(direction, vehicle.CurrentDirection()) {
		result.Outcome = ShipCannonRejectedBroadside

		return result
	}

	result.Outcome = ShipCannonMissed
	current := result.Source
	var target *map_units.NPCEnemy
	targetDistance := shipCannonRange
	floor := g.MapState.PlayerLocation.Floor
	npcs := g.CurrentNPCAIController.GetNpcs()
	for distance := 1; distance <= shipCannonRange; distance++ {
		next := direction.GetNewPositionInDirection(&current).
			GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		current = *next
		enemy := npcs.GetVisibleEnemyAtPositionAndFloorOrNil(current, floor)
		if enemy == nil || enemy.Variant() == map_units.EnemyVariantWhirlpool {
			continue
		}
		target = enemy
		targetDistance = distance
		result.HasTarget = true
		result.TargetPosition = current

		break
	}
	var pathClear bool
	result.ProjectileEnd, pathClear = g.traceSourceMissile(result.Source, direction, targetDistance)
	if target == nil {
		return result
	}
	if !pathClear {
		result.Outcome = ShipCannonBlocked

		return result
	}

	result.HPBefore = target.CurrentHP()
	result.Damage = g.RollDice(shipCannonDamageMax)
	target.ApplyDamage(result.Damage)
	result.HPAfter = target.CurrentHP()
	result.Outcome = ShipCannonHit
	// REDUX DEVIATION: ship cannon fire removes only on unsigned underflow,
	// leaving exact-zero targets. Redux removes at zero as the resolved bug fix.
	// See ref:SHIP-0005.
	if result.HPAfter <= 0 {
		if !npcs.RemoveEnemy(target) {
			panic("resolved cannon target could not be removed")
		}
		result.Outcome = ShipCannonDestroyed
	}

	return result
}

// ActionFireLargeMap handles the Fire command on the large map
// Original Ultima V: original command rules ship cannon fire
// - Only frigates can fire (not skiffs or on foot)
// - Broadsides only (perpendicular to ship facing)
// - Range: 3 tiles
// - Damage: 1-20 HP to first enemy hit
// - No cannonball consumption (unlimited ammo)
// See ref:SHIP-0005.
func (g *GameState) ActionFireLargeMap(direction references.Direction) bool {
	if g.EventQueueManager.HasWorkflow(shipCannonWorkflowName) {
		return false
	}
	result := g.ResolveShipCannonFire(direction)
	switch result.Outcome {
	case ShipCannonRejectedVehicle:
		g.SystemCallbacks.Message.AddRowStr("What?")

		return false
	case ShipCannonRejectedBroadside:
		g.SystemCallbacks.Message.AddRowStr("Fire broadsides only!")

		return false
	case ShipCannonMissed, ShipCannonBlocked, ShipCannonHit, ShipCannonDestroyed:
	}

	g.StartShipCannonFireWorkflow(result)

	return true
}

// isValidBroadsideDirection accepts a firing axis perpendicular to the ship.
// A north/south-facing ship fires east/west; an east/west-facing ship fires
// north/south. Parallel shots are rejected. ref:SHIP-0005
func (g *GameState) isValidBroadsideDirection(fireDir, shipDir references.Direction) bool {
	// Ship facing N/S (Up/Down) can only fire E/W (Left/Right)
	// Ship facing E/W (Left/Right) can only fire N/S (Up/Down)
	shipVertical := shipDir == references.Up || shipDir == references.Down
	fireVertical := fireDir == references.Up || fireDir == references.Down

	// Valid broadside: perpendicular axes (one vertical, one horizontal)
	return shipVertical != fireVertical
}

// StartShipCannonFireWorkflow creates and starts the ship cannon fire event queue
// Original: original command rules ship cannon fire
// Sequence: play sound, animate projectile, then present the resolved impact.
// See ref:SHIP-0005.
func (g *GameState) StartShipCannonFireWorkflow(result ShipCannonResult) {
	if g.EventQueueManager.HasWorkflow(shipCannonWorkflowName) {
		return
	}
	if !result.accepted() {
		panic("ship cannon workflow requires an accepted result")
	}
	routeOwnsCycle := g.surfaceCycleOwned

	q := g.EventQueueManager.Workflow(shipCannonWorkflowName).
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput()

	q.First(func() {
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundCannonFire)
	}).ThenSequence(g.ProjectileAnimationSequence(
		result.Source,
		result.ProjectileEnd,
		references.MissileCannonBall,
		func() {
			if result.Outcome == ShipCannonHit || result.Outcome == ShipCannonDestroyed {
				g.SystemCallbacks.Visual.PlayKapowAtPosition(result.TargetPosition)
				g.SystemCallbacks.Visual.SetScreenFlashColor(1.0, 0.0, 0.0)
				if result.Outcome == ShipCannonDestroyed {
					g.SystemCallbacks.UI.ShowTimedOverlayMessage(
						"Destroyed!", destroyedMessageDurationMs, false, true,
					)
				}
			}
		},
	)).If(
		func() bool { return result.Outcome == ShipCannonHit || result.Outcome == ShipCannonDestroyed },
		func(subQ *event_queue.SequentialEventBuilder) {
			subQ.Animate(200, func(progress float32) {
				var intensity float64
				if progress < 0.5 {
					intensity = 0.7 * float64(progress*2.0)
				} else {
					intensity = 0.7 * float64(2.0-progress*2.0)
				}
				g.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)

				dampen := 1.0 - float64(progress)
				offsetX, offsetY := calculateScreenShakeOffset(progress, 6.0*dampen)
				g.SystemCallbacks.Visual.SetScreenShakeOffset(offsetX, offsetY)
			}).Then(func() {
				g.SystemCallbacks.Visual.SetScreenFlashIntensity(0.0)
				g.SystemCallbacks.Visual.SetScreenShakeOffset(0.0, 0.0)
			})
		}).Finally(
		func() {
			g.SystemCallbacks.Visual.ClearProjectile()
			g.SystemCallbacks.Visual.SetScreenFlashIntensity(0.0)
			g.SystemCallbacks.Visual.SetScreenShakeOffset(0.0, 0.0)
			if !routeOwnsCycle {
				g.FinishTurn()
			}
		}).
		Start()
}
