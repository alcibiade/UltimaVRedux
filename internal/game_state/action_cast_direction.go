package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// AnYlemResult represents the result of attempting to destroy adjacent matter.
type AnYlemResult int

const (
	// AnYlemSuccess indicates that eligible scenery was destroyed.
	AnYlemSuccess AnYlemResult = iota
	// AnYlemNoEffect indicates that the target was not removable.
	AnYlemNoEffect
)

// CastAnYlem removes one of the original eligible scenery tiles.
func (g *GameState) CastAnYlem(pos *references.Position) AnYlemResult {
	// A piece the party built on its deed stays where its placement records it (ref:CRAFT-0009).
	if pos == nil || g.isBuiltDeedPiece(*pos) {
		return AnYlemNoEffect
	}
	theMap := g.GetLayeredMapByCurrentLocation()
	targetTile := theMap.GetTileTopMapOnlyTile(pos)
	if targetTile == nil || !isAnYlemObject(targetTile.Index) {
		return AnYlemNoEffect
	}

	theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.BrickFloor)

	return AnYlemSuccess
}

// AnSanctResult represents the result of attempting to unlock or disarm an adjacent target.
type AnSanctResult int

const (
	// AnSanctDoorUnlocked indicates that a mundane door was unlocked.
	AnSanctDoorUnlocked AnSanctResult = iota
	// AnSanctChestDisarmed indicates that a surface chest was recognized and disarmed.
	AnSanctChestDisarmed
	// AnSanctNoEffect indicates that the target was neither a mundane locked door nor a chest.
	AnSanctNoEffect
)

// CastAnSanct unlocks a mundane surface door or disarms a surface chest.
func (g *GameState) CastAnSanct(pos *references.Position) AnSanctResult {
	theMap := g.GetLayeredMapByCurrentLocation()
	targetTile := theMap.GetTileTopMapOnlyTile(pos)
	if targetTile == nil {
		return AnSanctNoEffect
	}

	switch targetTile.Index { //nolint:exhaustive // Only mundane locked doors change.
	case indexes.LockedDoor:
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.RegularDoor)

		return AnSanctDoorUnlocked
	case indexes.LockedDoorView:
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.RegularDoorView)

		return AnSanctDoorUnlocked
	}

	// Magical disarming clears the actual object chest's shared trap bit.
	// A rendered chest tile alone is not authoritative chest state.
	// See ref:SPELL-0001.
	if chest := g.surfaceChestAt(pos); chest != nil {
		chest.UnlockSurfaceChest()

		return AnSanctChestDisarmed
	}

	return AnSanctNoEffect
}

// =============================================================================
// Magic Unlock Door (In Ex Por)
// =============================================================================

// MagicUnlockResult represents the result of attempting to magic-unlock a door.
type MagicUnlockResult int

// Magic-unlock outcomes distinguish magical doors, ordinary doors, and other tiles.
const (
	MagicUnlockSuccess MagicUnlockResult = iota
	MagicUnlockNotMagicDoor
	MagicUnlockNotADoor
)

// MagicUnlockDoor attempts to unlock a magically locked door at the given position.
// Returns the result of the attempt.
func (g *GameState) MagicUnlockDoor(pos *references.Position) MagicUnlockResult {
	if pos == nil {
		return MagicUnlockNotADoor
	}
	theMap := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, g.MapState.PlayerLocation.Floor)
	targetTile := theMap.GetTileTopMapOnlyTile(pos)
	if targetTile == nil {
		return MagicUnlockNotADoor
	}

	switch targetTile.Index { //nolint:exhaustive // Only door tiles have unlock outcomes; other tiles are not doors.
	case indexes.MagicLockDoor:
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.RegularDoor)

		return MagicUnlockSuccess
	case indexes.MagicLockDoorWithView:
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.RegularDoorView)

		return MagicUnlockSuccess
	case indexes.LockedDoor, indexes.LockedDoorView, indexes.RegularDoor, indexes.RegularDoorView:
		// Not a magic door
		return MagicUnlockNotMagicDoor
	default:
		return MagicUnlockNotADoor
	}
}

// =============================================================================
// Magic Lock Door (An Ex Por)
// =============================================================================

// MagicLockResult represents the result of attempting to magic-lock a door.
type MagicLockResult int

// Magic-lock outcomes distinguish newly locked doors, existing locks, and other tiles.
const (
	MagicLockSuccess MagicLockResult = iota
	MagicLockAlreadyMagic
	MagicLockNotADoor
)

// MagicLockDoor attempts to magically lock a door at the given position.
// Returns the result of the attempt.
func (g *GameState) MagicLockDoor(pos *references.Position) MagicLockResult {
	theMap := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, g.MapState.PlayerLocation.Floor)
	if pos == nil {
		return MagicLockNotADoor
	}
	targetTile := theMap.GetTileTopMapOnlyTile(pos)
	if targetTile == nil {
		// A square beyond the map's edge holds no door.
		return MagicLockNotADoor
	}

	switch targetTile.Index { //nolint:exhaustive // Only door tiles can be locked; other tiles are not doors.
	case indexes.MagicLockDoor, indexes.MagicLockDoorWithView:
		// Already magic locked
		return MagicLockAlreadyMagic
	case indexes.LockedDoor, indexes.RegularDoor:
		// Lock with magic
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.MagicLockDoor)

		return MagicLockSuccess
	case indexes.LockedDoorView, indexes.RegularDoorView:
		// Lock with magic (windowed door variant)
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.MagicLockDoorWithView)

		return MagicLockSuccess
	default:
		return MagicLockNotADoor
	}
}

// =============================================================================
// Dispel Field (An Grav)
// =============================================================================

// DispelFieldResult represents the result of attempting to dispel a field.
type DispelFieldResult int

// Dispel outcomes report whether an eligible field was removed.
const (
	DispelFieldSuccess DispelFieldResult = iota
	DispelFieldNotAField
)

// DispelField attempts to dispel a magic field at the given position.
// Removes poison fields, magic fields, and fire fields.
// Returns the result of the attempt.
func (g *GameState) DispelField(pos *references.Position) DispelFieldResult {
	theMap := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, g.MapState.PlayerLocation.Floor)
	if pos == nil {
		return DispelFieldNotAField
	}
	targetTile := theMap.GetTileTopMapOnlyTile(pos)
	if targetTile == nil {
		return DispelFieldNotAField
	}

	if targetTile.Index == indexes.PoisonField || targetTile.Index == indexes.MagicField || targetTile.Index == indexes.FireField {
		// Dispel the field - replace with floor
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.BrickFloor)

		return DispelFieldSuccess
	}

	return DispelFieldNotAField
}

// =============================================================================
// Blink Teleport (In Por)
// =============================================================================

// BlinkResult represents the result of attempting to blink teleport.
type BlinkResult int

// Blink outcomes distinguish successful travel, blocked travel, and combat routing.
const (
	BlinkSuccess            BlinkResult = iota
	BlinkBlocked                        // No eligible grass in the retained outdoor window
	BlinkCombatNotSupported             // Combat uses its independent automatic targeting route
)

// BlinkTeleport moves the party to the farthest grass tile in the selected
// direction within the retained 32x32 outdoor window. See ref:SPELL-0002.
// Returns the result of the attempt.
func (g *GameState) BlinkTeleport(direction references.Direction) BlinkResult {
	mapType := g.MapState.PlayerLocation.Location.GetMapType()

	// Surface combat has a separate random destination effect.
	if mapType == references.CombatMapType {
		return BlinkCombatNotSupported
	}
	if mapType != references.LargeMapType {
		return BlinkBlocked
	}

	dx, dy := references.Coordinate(0), references.Coordinate(0)
	switch direction {
	case references.NoneDirection:
		return BlinkBlocked
	case references.Up:
		dy = -1
	case references.Down:
		dy = 1
	case references.Left:
		dx = -1
	case references.Right:
		dx = 1
	default:
		return BlinkBlocked
	}

	playerPos := g.MapState.PlayerLocation.Position
	window := g.requireOutdoorWindow()
	xLimit := min(window.origin.X+outdoorWindowSize, references.XLargeMapTiles)
	yLimit := min(window.origin.Y+outdoorWindowSize, references.YLargeMapTiles)
	theMap := g.GetLayeredMapByCurrentLocation()
	var destination *references.Position
	// Only the initial adjacent candidate wraps. Both axes must then satisfy
	// numeric bounds, even the perpendicular axis. Obstacles do not stop the ray.
	candidate := references.Position{X: (playerPos.X + dx) & outdoorWorldMask, Y: (playerPos.Y + dy) & outdoorWorldMask}
	for candidate.X >= window.origin.X && candidate.X < xLimit &&
		candidate.Y >= window.origin.Y && candidate.Y < yLimit {
		tile := theMap.GetTileTopMapOnlyTile(&candidate)
		if tile != nil && tile.Is(indexes.Grass) {
			candidateCopy := candidate
			destination = &candidateCopy
		}
		candidate.X += dx
		candidate.Y += dy
	}
	if destination == nil {
		return BlinkBlocked
	}

	g.MapState.PlayerLocation.Position = *destination
	g.InitializeOutdoorWindow()

	return BlinkSuccess
}
