package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Whirlpool spawning constants based on original Ultima V
// See ref:SHIP-0003.
const (
	WhirlpoolSpawnProbability = 8   // 1 in 8 chance on deep water
	minDistanceFromPlayer     = 7   // Must be >6 tiles from player (validloc check)
	maxSpawnAttempts          = 128 // Max attempts to find valid location (0x80 in original)
	spawnAreaSize             = 32  // Random position range (32x32 area around offset)
)

// isWhirlpoolPredicate returns a predicate function that matches whirlpool enemies
func isWhirlpoolPredicate() map_units.EnemyMatchFunc {
	return func(enemy *map_units.NPCEnemy) bool {
		return enemy.Variant() == map_units.EnemyVariantWhirlpool
	}
}

// isWhirlpoolAtPosPredicate returns a predicate function that matches whirlpools at a specific position
func isWhirlpoolAtPosPredicate(pos references.Position) map_units.EnemyMatchFunc {
	return func(enemy *map_units.NPCEnemy) bool {
		return enemy.Pos() == pos && enemy.Variant() == map_units.EnemyVariantWhirlpool
	}
}

// TrySpawnWhirlpool is the probabilistic single-whirlpool integration fixture
// helper. Ordinary world generation is owned by the large-map AI controller;
// its source terrain/region gates and multiple-whirlpool support are independent.
func (g *GameState) TrySpawnWhirlpool(pos references.Position) bool {
	// Only spawn on large maps
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return false
	}

	// Only allow one whirlpool at a time on the map
	if g.HasAnyWhirlpool() {
		return false
	}

	// Check if position already has a whirlpool (redundant with above but kept for safety)
	if g.hasWhirlpoolAt(pos) {
		return false
	}

	// Get tile at position
	layeredMap := g.GetCurrentLayeredMap()
	tile := layeredMap.GetTopTile(&pos)

	// Whirlpool generation requires deep-water terrain (raw tile 1) and
	// the upper-boundary result of the 0..7 spawn draw. ref:SHIP-0006
	if !tile.Is(indexes.Water1) {
		return false
	}

	// 1 in 8 spawn probability using GameState RNG for determinism
	if !g.OneInXOdds(WhirlpoolSpawnProbability) {
		return false
	}

	// Create whirlpool enemy at position
	return g.spawnWhirlpoolAt(pos)
}

// findValidDeepWaterSpawnLocation searches for a valid deep water spawn location
// matching original Ultima V behavior (original world-creature rules validloc/pickmon):
//   - Random position within 32x32 area around player
//   - Must be >6 tiles from player on both axes (off-screen)
//   - Must be on deep water tile
//
// Returns the position and true if found, or zero position and false if not found
// See ref:SHIP-0003.
func (g *GameState) findValidDeepWaterSpawnLocation() (references.Position, bool) {
	playerPos := g.MapState.PlayerLocation.Position
	layeredMap := g.GetCurrentLayeredMap()

	// Try up to maxSpawnAttempts times to find a valid location (like original's 0x80 loop)
	for range maxSpawnAttempts {
		// Sample the spawn-area coordinate in 0..31, add the viewport offset,
		// and wrap the world coordinate to one byte. ref:ENEMY-0004
		randX := references.Coordinate(g.GetRNG().Intn(spawnAreaSize))
		randY := references.Coordinate(g.GetRNG().Intn(spawnAreaSize))

		// Calculate position relative to player (using player as center, similar to xoffset/yoffset)
		candidateX := (playerPos.X - spawnAreaSize/2 + randX) & references.MapCoordinateWrapMask
		candidateY := (playerPos.Y - spawnAreaSize/2 + randY) & references.MapCoordinateWrapMask
		candidatePos := references.Position{X: candidateX, Y: candidateY}

		// Require the candidate to differ by more than six tiles from the
		// party on both axes. ref:ENEMY-0004
		distX, distY := playerPos.GetWrappedAxisDistances(candidatePos)
		if distX < minDistanceFromPlayer || distY < minDistanceFromPlayer {
			continue // Too close to player
		}

		// Check if tile is deep water (like pickmon's DEEP check)
		tile := layeredMap.GetTopTile(&candidatePos)
		if !tile.Is(indexes.Water1) {
			continue // Not deep water
		}

		return candidatePos, true
	}

	// Never found a valid location after maxSpawnAttempts
	return references.Position{}, false
}

// ForceSpawnWhirlpoolAtRandomLocation spawns a whirlpool at a random off-screen
// deep water location, bypassing probability checks. Used for debug commands.
//
// Returns true if whirlpool was spawned, false if no valid location found
func (g *GameState) ForceSpawnWhirlpoolAtRandomLocation() bool {
	_, spawned := g.ForceSpawnWhirlpoolAtRandomLocationWithPosition()

	return spawned
}

// ForceSpawnWhirlpoolAtRandomLocationWithPosition returns the inserted position on success.
func (g *GameState) ForceSpawnWhirlpoolAtRandomLocationWithPosition() (references.Position, bool) {
	// Only spawn on large maps
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return references.Position{X: 0, Y: 0}, false
	}

	// Only allow one whirlpool at a time on the map
	if g.HasAnyWhirlpool() {
		return references.Position{X: 0, Y: 0}, false
	}

	// Find valid spawn location (no probability check)
	candidatePos, found := g.findValidDeepWaterSpawnLocation()
	if !found {
		return references.Position{X: 0, Y: 0}, false
	}

	if !g.spawnWhirlpoolAt(candidatePos) {
		return references.Position{X: 0, Y: 0}, false
	}

	return candidatePos, true
}

// spawnWhirlpoolAt creates a whirlpool enemy at the specified position
func (g *GameState) spawnWhirlpoolAt(pos references.Position) bool {
	// Create whirlpool enemy reference
	whirlpoolTile := g.GameReferences.TileReferences.GetTile(indexes.Whirlpool_KeyIndex)
	whirlpoolEnemy := references.EnemyReference{
		MonsterType:  references.MonsterWhirlpool,
		Name:         references.MonsterWhirlpool.Name(),
		NamePlural:   references.MonsterWhirlpool.Name(),
		KeyFrameTile: whirlpoolTile,
		// Whirlpools have minimal stats - they're environmental hazards not combat enemies
		Armour:         0,
		Damage:         0,
		Dexterity:      0,
		HitPoints:      1, // Minimal HP so they can exist
		Intelligence:   0,
		MaxPerMap:      0, // No limit on whirlpools
		Strength:       0,
		TreasureNumber: 0,
		EnemyAbilities: map[references.EnemyAbility]bool{
			references.NoCorpse:          true, // Don't leave corpses
			references.DisappearsOnDeath: true, // Disappear when removed
		},
		HasWhirlpoolMovement: true, // Enable whirlpool movement AI (50% skip, 25% random, 25% seek)
		IsWaterEnemy:         true, // Whirlpools are water entities
		AttackRange:          0,    // Whirlpools don't attack in traditional sense
	}

	// Create NPC enemy instance
	npcEnemy := map_units.NewEnemyNPC(whirlpoolEnemy, references.DynamicSpawnIndex)
	npcEnemy.SetPos(pos)
	npcEnemy.SetFloor(g.MapState.PlayerLocation.Floor)
	npcEnemy.SetVisible(true)

	// Add to current large map NPC controller
	return g.CurrentNPCAIController.GetNpcs().AddEnemy(npcEnemy)
}

// HasAnyWhirlpool returns true if any whirlpool exists anywhere on the current map
func (g *GameState) HasAnyWhirlpool() bool {
	return g.CurrentNPCAIController.GetNpcs().HasEnemyMatching(isWhirlpoolPredicate())
}

// hasWhirlpoolAt checks if a whirlpool already exists at the given position
func (g *GameState) hasWhirlpoolAt(pos references.Position) bool {
	return g.CurrentNPCAIController.GetNpcs().HasEnemyMatching(isWhirlpoolAtPosPredicate(pos))
}

// GetWhirlpoolAt returns the whirlpool at the given position, or nil if none exists
func (g *GameState) GetWhirlpoolAt(pos references.Position) *map_units.NPCEnemy {
	return g.CurrentNPCAIController.GetNpcs().GetFirstEnemyMatching(isWhirlpoolAtPosPredicate(pos))
}

// RemoveWhirlpoolAt removes a whirlpool at the given position
func (g *GameState) RemoveWhirlpoolAt(pos references.Position) bool {
	npcs := g.CurrentNPCAIController.GetNpcs()

	// Remove by position since GetWhirlpoolAt returns a copy
	return npcs.RemoveNPCAtPosition(pos)
}

// GetFirstWhirlpool returns the first whirlpool in controller order, or nil.
func (g *GameState) GetFirstWhirlpool() *map_units.NPCEnemy {
	return g.CurrentNPCAIController.GetNpcs().GetFirstEnemyMatching(isWhirlpoolPredicate())
}

// CheckWhirlpoolAdjacency checks if whirlpool is adjacent to player and triggers collision
// Called before monster movement in turn.go.
//
// Original Ultima V : outdoor contact and ranged attack checks adjacency BEFORE world creature movement
// outdoor contact and ranged attack at original world-creature rules check: (dis_x == 1 && dis_y == 0) || (dis_x == 0 && dis_y == 1)
// If adjacent, outdoor contact resolution is called which triggers whirlpool transportation
//
// Returns true if the whirlpool collision workflow was triggered.
// See ref:SHIP-0003.
func (g *GameState) CheckWhirlpoolAdjacency() bool {
	// Only check on large maps where whirlpools exist
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return false
	}

	if g.EventQueueManager.HasWorkflow("whirlpool_collision") {
		return true
	}
	playerPos := g.MapState.PlayerLocation.Position
	playerFloor := g.MapState.PlayerLocation.Floor
	// Canonical monster generation can create multiple whirlpools. Scan in
	// stable controller order rather than letting a distant first one mask contact.
	whirlpool := g.CurrentNPCAIController.GetNpcs().GetFirstEnemyMatching(func(enemy *map_units.NPCEnemy) bool {
		return enemy.Variant() == map_units.EnemyVariantWhirlpool && enemy.IsVisible() &&
			playerPos.IsNextTo(enemy.Pos(), playerFloor, enemy.Floor())
	})
	if whirlpool == nil {
		return false
	}

	return g.checkWhirlpoolActorContact(whirlpool)
}

func (g *GameState) checkWhirlpoolActorContact(whirlpool *map_units.NPCEnemy) bool {
	if g.EventQueueManager.HasWorkflow("whirlpool_collision") {
		return true
	}
	whirlpoolPos := whirlpool.Pos()

	// Original outdoor contact resolution leaves an adjacent whirlpool in place when it
	// hits a party on foot. Every vehicle path consumes it before transport.
	// See ref:SHIP-0003.
	if g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
		g.RemoveWhirlpoolAt(whirlpoolPos)
	}

	// Build and start transportation queue
	q := g.EventQueueManager.Workflow("whirlpool_collision").
		Priority(event_queue.PriorityHigh).
		Sequential()
	g.BuildWhirlpoolTransportationQueue(q)
	q.Start()

	return true
}

// IsPlayerOnWhirlpool tests a co-located whirlpool and a vessel capable
// of moving onto it. Frigates and skiffs qualify for transport. Contact
// on foot instead damages the party without transporting it. ref:SHIP-0003
func (g *GameState) IsPlayerOnWhirlpool() bool {
	vehicleType := g.PartyVehicle.GetVehicleDetails().VehicleType
	if vehicleType != references.FrigateVehicle && vehicleType != references.SkiffVehicle {
		return false
	}

	// Check if current position has a whirlpool
	currentPos := g.MapState.PlayerLocation.Position
	whirlpool := g.GetWhirlpoolAt(currentPos)

	return whirlpool != nil
}

// IsPlayerCurrentlyOnWhirlpool returns true if player is currently on a whirlpool
// Alias for IsPlayerOnWhirlpool() with a name that clarifies intent at detection sites
// Used in movement handlers to detect whirlpools before triggering transportation sequences
func (g *GameState) IsPlayerCurrentlyOnWhirlpool() bool {
	return g.IsPlayerOnWhirlpool()
}

// WhirlpoolHitResult is detached authoritative state for presentation.
type WhirlpoolHitResult struct {
	Hit                 SourceHitResult
	Teleported          bool
	DestinationLocation references.Location
	DestinationPosition references.Position
	DestinationFloor    references.FloorNumber
}

const (
	whirlpoolDestinationX     = 34
	whirlpoolDestinationY     = 18
	whirlpoolDestinationFloor = -1
)

// ResolveWhirlpoolHit resolves hit2 before teleporting its result to the source destination.
func (g *GameState) ResolveWhirlpoolHit() WhirlpoolHitResult {
	destination := references.Position{X: whirlpoolDestinationX, Y: whirlpoolDestinationY}
	vehicleBefore := g.PartyVehicle.GetVehicleDetails().VehicleType
	result := WhirlpoolHitResult{
		Hit:                 g.ResolveSourceHit(),
		DestinationLocation: references.BritanniaUnderworld,
		DestinationPosition: destination,
		DestinationFloor:    whirlpoolDestinationFloor,
	}
	if vehicleBefore == references.NoPartyVehicle {
		return result
	}
	result.Teleported = true
	g.TransitionToLocation(
		result.DestinationLocation,
		result.DestinationFloor,
		result.DestinationPosition,
		TransitionFullLocation,
		false,
	)
	if g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
		g.PartyVehicle.MapUnitDetails().Position = result.DestinationPosition
		g.PartyVehicle.MapUnitDetails().Floor = result.DestinationFloor
	}

	return result
}

// BuildWhirlpoolTransportationQueue presents a committed hit result
// and transports to Underworld (34,18), floor -1. The owning workflow orders
// message, shake, fade-out, damage/teleport, then fade-in, using a handoff
// to preserve automatic-sailing ownership. ref:SHIP-0003
func (g *GameState) BuildWhirlpoolTransportationQueue(q *event_queue.SequentialEventBuilder) {
	g.buildWhirlpoolTransportationQueue(q, g.resolveWhirlpoolTransportation)
}

func (g *GameState) resolveWhirlpoolTransportation() WhirlpoolHitResult {
	if g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
		whirlpool := g.GetWhirlpoolAt(g.MapState.PlayerLocation.Position)
		if whirlpool != nil {
			g.RemoveWhirlpoolAt(whirlpool.Pos())
		}
	}

	return g.ResolveWhirlpoolHit()
}

// resolve supplies either the workflow-owned transport mutation or a captured
// on-foot result already committed at the outdoor actor boundary.
func (g *GameState) buildWhirlpoolTransportationQueue(q *event_queue.SequentialEventBuilder, resolve func() WhirlpoolHitResult) {
	g.buildWhirlpoolTransportationSteps(q, resolve)
	q.Finally(g.cleanupWhirlpoolTransportation)
}

func (g *GameState) cleanupWhirlpoolTransportation() {
	g.SystemCallbacks.Audio.Stop(SoundWhirlpoolVortex).Now()
	g.SystemCallbacks.Visual.SetMapFadeLevel(0.0)
}

// Steps do not install a finalizer: the owning workflow composes presentation
// cleanup with any retained gameplay continuation in its single Finally.
func (g *GameState) buildWhirlpoolTransportationSteps(q *event_queue.SequentialEventBuilder, resolve func() WhirlpoolHitResult) {
	// Visual effect timing
	const (
		shakeDurationMs = 500
		shakePixels     = 4.0
		fadeDurationMs  = 500
		flashDurationMs = 200
	)
	var result WhirlpoolHitResult

	q.BlockInput().
		First(func() {
			g.SystemCallbacks.Message.AddRowStr("WHIRLPOOL!")
			g.SystemCallbacks.Audio.Play(SoundWhirlpoolVortex).Loop().Start()
		}).ThenSequence(

		NewPartyStateBuilder(g).
			ToAllParty().
			WithShake(shakeDurationMs, shakePixels).
			FlashWater().
			Build()).
		Animate(uint64(fadeDurationMs), func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
		}).
		Then(func() {
			g.SystemCallbacks.Audio.PlaySoundEffect(SoundWhirlpoolSplash)
			result = resolve()
			if result.Hit.VehicleBefore == references.FrigateVehicle {
				g.SystemCallbacks.Audio.PlaySoundEffect(SoundHullCrunch)
			}
			switch result.Hit.Outcome {
			case SourceHitFrigateDamaged, SourceHitPartyDamaged:
			case SourceHitEscapedBySkiff, SourceHitEscapedByCarpet:
				g.SystemCallbacks.Message.AddRowStr("Ship sunk!")
				g.SystemCallbacks.Message.AddRowStr("Abandon ship!")
			case SourceHitDrowned:
				g.SystemCallbacks.Message.AddRowStr("Ship sunk!")
				g.SystemCallbacks.Message.AddRowStr("DROWNING!!!")
			}
		}).
		Animate(uint64(fadeDurationMs), func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(float64(1.0 - progress))
		}).Then(func() {
		if result.Hit.PartyWiped {
			g.SystemCallbacks.Flow.Consequences.TriggerDeathSequence()
		}
	})
}
