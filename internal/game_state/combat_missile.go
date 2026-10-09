package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	originalMissilePathCapacity = 330
	originalMissileSampleStep   = 13
	originalRockSampleStep      = 8
	originalArrowSampleOffset   = 4
	originalCombatTilePixels    = 16
	originalCombatWindowInset   = 8
	originalCombatPixelMinimum  = 8
	originalCombatPixelMaximum  = 183
)

// Original missile terrain validation bitmap. A set bit allows missiles through.
//
// See ref:TILE-0004.
//
//nolint:gochecknoglobals // Immutable source-authentic lookup table.
var combatMissilePassability = [32]byte{
	0xff, 0xf3, 0xc3, 0x8f, 0xff, 0xff, 0xff, 0xc0,
	0xdd, 0xf8, 0x03, 0xdf, 0xff, 0xff, 0x00, 0x00,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f,
	0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xff, 0xff,
}

// CombatProjectileLaunch describes an authoritative projectile path without presentation lifecycle concerns.
type CombatProjectileLaunch struct {
	From    references.Position
	Aim     references.Position
	Impact  references.Position
	Missile references.MissileType
	Blocked bool
	Animate bool
}

func isCombatMissilePassable(sprite indexes.SpriteIndex) bool {
	index := int(sprite)
	if index < 0 || index >= len(combatMissilePassability)*8 {
		return false
	}

	return combatMissilePassability[index>>3]&(0x80>>uint(index&7)) != 0
}

type combatMissilePixel struct {
	x int
	y int
}

// combatMissileImpactPosition ports projectile sample path and projectile travel, including
// the Arrow offset and Rock stride before each missile terrain validation sample.
// See ref:TILE-0004.
func combatMissileImpactPosition(
	terrainAt func(references.Position) indexes.SpriteIndex,
	from, to references.Position,
	missile references.MissileType,
) (references.Position, bool) {
	if terrainAt == nil {
		return from, false
	}

	path := originalCombatMissilePath(from, to)
	start, step := originalMissileSampling(missile)
	for pathIndex := start; pathIndex < len(path); pathIndex += step {
		pixel := path[pathIndex]
		if pixel.x < originalCombatPixelMinimum || pixel.x > originalCombatPixelMaximum ||
			pixel.y < originalCombatPixelMinimum || pixel.y > originalCombatPixelMaximum {
			break
		}
		// Original projectile rules advance to the next sample and checks the terminator before
		// testing missile terrain validation, so the final displayed sample is never collision-tested.
		// See ref:TILE-0004.
		if pathIndex+step >= len(path) {
			break
		}
		position := references.Position{
			X: references.Coordinate((pixel.x - originalCombatWindowInset) / originalCombatTilePixels), //nolint:gosec // Combat pixels map to 0..10.
			Y: references.Coordinate((pixel.y - originalCombatWindowInset) / originalCombatTilePixels), //nolint:gosec // Combat pixels map to 0..10.
		}
		if position != from && !isCombatMissilePassable(terrainAt(position)) {
			return position, false
		}
	}

	return to, true
}

func originalMissileSampling(missile references.MissileType) (int, int) {
	switch missile {
	case references.MissileArrow:
		return originalArrowSampleOffset, originalMissileSampleStep
	case references.MissileRock:
		return 0, originalRockSampleStep
	case references.MissileNone,
		references.MissileCannonBall,
		references.MissileAxe,
		references.MissileRed,
		references.MissileBlue,
		references.MissileGreen,
		references.MissileViolet:
		return 0, originalMissileSampleStep
	default:
		return 0, originalMissileSampleStep
	}
}

func traceCombatProjectileLaunch(
	terrainAt func(references.Position) indexes.SpriteIndex,
	from, aim references.Position,
	missile references.MissileType,
) CombatProjectileLaunch {
	impact, reachedAim := combatMissileImpactPosition(terrainAt, from, aim, missile)

	return CombatProjectileLaunch{
		From:    from,
		Aim:     aim,
		Impact:  impact,
		Missile: missile,
		Blocked: !reachedAim,
		Animate: true,
	}
}

// combatMissileTerrainAt supplies missile terrain validation with the collision-relevant
// part of object display projection/terrain underlay projection: a displayed object clears its terrain projection cell
// to zero. Invisible monsters are omitted, whereas invisible PCs is still overlaid.
// Keep this authoritative; collision must not depend on whether a frame drew.
// See ref:TILE-0004.
func (g *GameState) combatMissileTerrainAt(position references.Position) indexes.SpriteIndex {
	combat := g.CombatScene
	terrain := combat.CombatMapRef.GetTileNumber(position)
	if isCombatMissilePassable(terrain) || terrain == indexes.BrickWallArchway {
		return terrain
	}
	if combat.HasObjectAt(position) || combat.hasFieldAt(position) {
		return 0
	}
	for sourceSlot := range sourceCombatQueueSlots {
		entry := combat.TurnQueue.getActiveEntryAtSourceSlot(sourceSlot)
		if entry == nil {
			continue
		}
		identity := turnQueueIdentity(entry)
		if combat.GetCombatantPosition(identity) != position {
			continue
		}
		if identity.IsPartyMember {
			if combat.isCombatPCActive(identity) && !combat.isCombatPCUnderground(identity) {
				return 0
			}

			continue
		}
		monster := combat.MonsterManager.GetMonster(identity.EntityIndex)
		if monster != nil && monster.IsActive() && !monster.IsDead() && !monster.IsInvisible() {
			return 0
		}
	}

	return terrain
}

// combatMissileImpactOccupantAt mirrors impact occupant selection rather than player
// aiming: an active invisible combatant can receive a redirected projectile.
// See ref:TILE-0004.
func (g *GameState) combatMissileImpactOccupantAt(position references.Position) (CombatTurnIdentity, bool) {
	for sourceSlot := range sourceCombatQueueSlots {
		entry := g.CombatScene.TurnQueue.getActiveEntryAtSourceSlot(sourceSlot)
		if entry == nil {
			continue
		}
		identity := turnQueueIdentity(entry)
		if g.CombatScene.GetCombatantPosition(identity) != position {
			continue
		}
		if identity.IsPartyMember {
			if g.CombatScene.isCombatPCActive(identity) && !g.CombatScene.isCombatPCUnderground(identity) {
				return identity, true
			}

			continue
		}
		monster := g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex)
		if monster != nil && monster.IsActive() && !monster.IsDead() && !monster.IsCorpserHidden() {
			return identity, true
		}
	}

	return CombatTurnIdentity{}, false
}

func originalCombatMissilePath(from, to references.Position) []combatMissilePixel {
	x0 := int(from.X)*originalCombatTilePixels + originalCombatTilePixels
	y0 := int(from.Y)*originalCombatTilePixels + originalCombatTilePixels
	x1 := int(to.X)*originalCombatTilePixels + originalCombatTilePixels
	y1 := int(to.Y)*originalCombatTilePixels + originalCombatTilePixels
	slope := originalCombatTilePixels * 12 * 100
	if x1 != x0 {
		slope = ((y1 - y0) * 100) / (x1 - x0)
	}
	if slope < 0 {
		slope = -slope
	}
	stepX, stepY := 1, 1
	if x1 < x0 {
		stepX = -1
	}
	if y1 < y0 {
		stepY = -1
	}
	xRemaining := absoluteInt(x1 - x0)
	yRemaining := absoluteInt(y1 - y0)
	temporarySlope := slope
	path := make([]combatMissilePixel, 0, originalMissilePathCapacity)
	path = append(path, combatMissilePixel{x: x0, y: y0})

	for (xRemaining > 0 || yRemaining > 0) && len(path) < originalMissilePathCapacity-1 {
		for temporarySlope > 0 && yRemaining > 0 && len(path) < originalMissilePathCapacity-1 {
			temporarySlope -= 100
			y0 += stepY
			yRemaining--
			path = append(path, combatMissilePixel{x: x0, y: y0})
		}
		if len(path) >= originalMissilePathCapacity-1 {
			break
		}
		xRemaining--
		temporarySlope += slope
		x0 += stepX
		path = append(path, combatMissilePixel{x: x0, y: y0})
	}

	return path
}

func absoluteInt(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
