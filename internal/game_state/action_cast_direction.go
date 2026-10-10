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

	switch targetTile.Index {
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

const (
	MagicLockSuccess MagicLockResult = iota
	MagicLockAlreadyMagic
	MagicLockNotADoor
)

// MagicLockDoor attempts to magically lock a door at the given position.
// Returns the result of the attempt.
func (g *GameState) MagicLockDoor(pos *references.Position) MagicLockResult {
	theMap := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, g.MapState.PlayerLocation.Floor)
	targetTile := theMap.GetTileTopMapOnlyTile(pos)

	switch targetTile.Index {
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

const (
	DispelFieldSuccess DispelFieldResult = iota
	DispelFieldNotAField
)

// DispelField attempts to dispel a magic field at the given position.
// Removes poison fields, magic fields, and fire fields.
// Returns the result of the attempt.
func (g *GameState) DispelField(pos *references.Position) DispelFieldResult {
	theMap := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, g.MapState.PlayerLocation.Floor)
	targetTile := theMap.GetTileTopMapOnlyTile(pos)

	switch targetTile.Index {
	case indexes.PoisonField, indexes.MagicField, indexes.FireField:
		// Dispel the field - replace with floor
		theMap.SetTileByLayer(map_state.MapOverrideLayer, pos, indexes.BrickFloor)

		return DispelFieldSuccess
	default:
		return DispelFieldNotAField
	}
}

// =============================================================================
// Blink Teleport (In Por)
// =============================================================================

// BlinkResult represents the result of attempting to blink teleport.
type BlinkResult int

const (
	BlinkSuccess            BlinkResult = iota
	BlinkBlocked                        // Destination is not passable
	BlinkCombatNotSupported             // Combat mode not implemented yet
)

// BlinkTeleport moves the party to the farthest grass tile in the selected
// direction within the original 32x32 overworld search window.
// Returns the result of the attempt.
func (g *GameState) BlinkTeleport(direction references.Direction) BlinkResult {
	mapType := g.MapState.PlayerLocation.Location.GetMapType()

	// Combat mode not implemented yet
	if mapType == references.CombatMapType {
		return BlinkCombatNotSupported
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
	xOffset := blinkWindowOffset(playerPos.X)
	yOffset := blinkWindowOffset(playerPos.Y)
	xInWindow := (playerPos.X - xOffset + references.XLargeMapTiles) % references.XLargeMapTiles
	yInWindow := (playerPos.Y - yOffset + references.YLargeMapTiles) % references.YLargeMapTiles
	steps := int(xInWindow)
	switch {
	case dx > 0:
		steps = 31 - int(xInWindow)
	case dy < 0:
		steps = int(yInWindow)
	case dy > 0:
		steps = 31 - int(yInWindow)
	}

	theMap := g.GetLayeredMapByCurrentLocation()
	var destination *references.Position
	for step := 1; step <= steps; step++ {
		candidate := references.Position{
			X: playerPos.X + dx*references.Coordinate(step),
			Y: playerPos.Y + dy*references.Coordinate(step),
		}
		candidate.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		tile := theMap.GetTileTopMapOnlyTile(&candidate)
		if tile != nil && tile.Is(indexes.Grass) && !g.ObjectPresentAt(&candidate) {
			candidateCopy := candidate
			destination = &candidateCopy
		}
	}
	if destination == nil {
		return BlinkBlocked
	}

	g.MapState.PlayerLocation.Position = *destination

	return BlinkSuccess
}

func blinkWindowOffset(coordinate references.Coordinate) references.Coordinate {
	offset := coordinate & 0xf0
	if coordinate&0x0f < 8 {
		offset = (offset - 0x10) & 0xf0
	}

	return offset
}
