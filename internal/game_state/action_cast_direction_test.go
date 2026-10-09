package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCastAnYlemUsesOriginalTownObjectSet(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	theMap := gs.GetLayeredMapByCurrentLocation()
	target := references.Position{X: 10, Y: 10}

	tests := []struct {
		name   string
		tile   indexes.SpriteIndex
		result AnYlemResult
		want   indexes.SpriteIndex
	}{
		{"original removable object", indexes.Plant, AnYlemSuccess, indexes.BrickFloor},
		{"extended pushable excluded", indexes.Chest, AnYlemNoEffect, indexes.Chest},
		{"ordinary terrain excluded", indexes.Grass, AnYlemNoEffect, indexes.Grass},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			theMap.UnSetTileByLayer(map_state.MapOverrideLayer, &target)
			theMap.SetTileByLayer(map_state.MapLayer, &target, test.tile)

			assert.Equal(t, test.result, gs.CastAnYlem(&target))
			assert.Equal(t, test.want, theMap.GetTileTopMapOnlyTile(&target).Index)
		})
	}
}

func TestCastAnSanctUnlocksSurfaceDoorsAndRecognizesChests(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	theMap := gs.GetLayeredMapByCurrentLocation()
	target := references.Position{X: 10, Y: 10}

	tests := []struct {
		name   string
		tile   indexes.SpriteIndex
		result AnSanctResult
		want   indexes.SpriteIndex
	}{
		{"locked door", indexes.LockedDoor, AnSanctDoorUnlocked, indexes.RegularDoor},
		{"windowed locked door", indexes.LockedDoorView, AnSanctDoorUnlocked, indexes.RegularDoorView},
		{"rendered chest without authority", indexes.Chest, AnSanctNoEffect, indexes.Chest},
		{"magic door", indexes.MagicLockDoor, AnSanctNoEffect, indexes.MagicLockDoor},
		{"ordinary terrain", indexes.Grass, AnSanctNoEffect, indexes.Grass},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			theMap.UnSetTileByLayer(map_state.MapOverrideLayer, &target)
			theMap.SetTileByLayer(map_state.MapLayer, &target, test.tile)

			assert.Equal(t, test.result, gs.CastAnSanct(&target))
			require.NotNil(t, theMap.GetTileTopMapOnlyTile(&target))
			assert.Equal(t, test.want, theMap.GetTileTopMapOnlyTile(&target).Index)
		})
	}
}

func TestMagicUnlockDoorRestoresOriginalDoorVariantsAndHandlesNil(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	theMap := gs.GetLayeredMapByCurrentLocation()
	target := references.Position{X: 10, Y: 10}

	assert.Equal(t, MagicUnlockNotADoor, gs.MagicUnlockDoor(nil))

	tests := []struct {
		name string
		tile indexes.SpriteIndex
		want indexes.SpriteIndex
	}{
		{"solid door", indexes.MagicLockDoor, indexes.RegularDoor},
		{"windowed door", indexes.MagicLockDoorWithView, indexes.RegularDoorView},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			theMap.UnSetTileByLayer(map_state.MapOverrideLayer, &target)
			theMap.SetTileByLayer(map_state.MapLayer, &target, test.tile)

			assert.Equal(t, MagicUnlockSuccess, gs.MagicUnlockDoor(&target))
			assert.Equal(t, test.want, theMap.GetTileTopMapOnlyTile(&target).Index)
		})
	}
}

func TestBlinkTeleportSelectsFarthestGrassInWrappedVisibleWindow(t *testing.T) {
	tests := []struct {
		name      string
		start     references.Position
		direction references.Direction
		grass     []references.Position
		want      references.Position
	}{
		{"up", references.Position{X: 100, Y: 100}, references.Up,
			[]references.Position{{X: 100, Y: 99}, {X: 100, Y: 80}}, references.Position{X: 100, Y: 80}},
		{"down", references.Position{X: 100, Y: 100}, references.Down,
			[]references.Position{{X: 100, Y: 101}, {X: 100, Y: 111}}, references.Position{X: 100, Y: 111}},
		{"left", references.Position{X: 100, Y: 100}, references.Left,
			[]references.Position{{X: 99, Y: 100}, {X: 80, Y: 100}}, references.Position{X: 80, Y: 100}},
		{"right", references.Position{X: 100, Y: 100}, references.Right,
			[]references.Position{{X: 101, Y: 100}, {X: 111, Y: 100}}, references.Position{X: 111, Y: 100}},
		{"left wraps at west edge", references.Position{X: 1, Y: 100}, references.Left,
			[]references.Position{{X: 0, Y: 100}, {X: 240, Y: 100}}, references.Position{X: 240, Y: 100}},
		{"up wraps at north edge", references.Position{X: 100, Y: 1}, references.Up,
			[]references.Position{{X: 100, Y: 0}, {X: 100, Y: 240}}, references.Position{X: 100, Y: 240}},
		{"right wraps at east edge", references.Position{X: 254, Y: 100}, references.Right,
			[]references.Position{{X: 255, Y: 100}, {X: 15, Y: 100}}, references.Position{X: 15, Y: 100}},
		{"down wraps at south edge", references.Position{X: 100, Y: 254}, references.Down,
			[]references.Position{{X: 100, Y: 255}, {X: 100, Y: 15}}, references.Position{X: 100, Y: 15}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithPlayerAt(int(test.start.X), int(test.start.Y)).
				WithSystemCallbacks().
				Build()
			theMap := gs.GetLayeredMapByCurrentLocation()
			setBlinkRayTiles(t, theMap, test.start, test.direction, indexes.BrickFloor)
			for _, position := range test.grass {
				position := position
				theMap.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Grass)
			}

			assert.Equal(t, BlinkSuccess, gs.BlinkTeleport(test.direction))
			assert.Equal(t, test.want, gs.MapState.PlayerLocation.Position)
		})
	}
}

func TestBlinkTeleportIsBlockedWithoutEligibleGrass(t *testing.T) {
	start := references.Position{X: 100, Y: 100}
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(int(start.X), int(start.Y)).
		WithSystemCallbacks().
		Build()
	setBlinkRayTiles(t, gs.GetLayeredMapByCurrentLocation(), start, references.Right, indexes.BrickFloor)

	assert.Equal(t, BlinkBlocked, gs.BlinkTeleport(references.Right))
	assert.Equal(t, start, gs.MapState.PlayerLocation.Position)
	assert.Equal(t, BlinkBlocked, gs.BlinkTeleport(references.NoneDirection))
}

func setBlinkRayTiles(
	t *testing.T,
	theMap *map_state.LayeredMap,
	start references.Position,
	direction references.Direction,
	tile indexes.SpriteIndex,
) {
	t.Helper()
	dx, dy := references.Coordinate(0), references.Coordinate(0)
	switch direction {
	case references.Up:
		dy = -1
	case references.Down:
		dy = 1
	case references.Left:
		dx = -1
	case references.Right:
		dx = 1
	default:
		t.Fatal("invalid test direction")
	}
	xOffset, yOffset := blinkWindowOffset(start.X), blinkWindowOffset(start.Y)
	xInWindow := (start.X - xOffset + references.XLargeMapTiles) % references.XLargeMapTiles
	yInWindow := (start.Y - yOffset + references.YLargeMapTiles) % references.YLargeMapTiles
	steps := int(xInWindow)
	if dx > 0 {
		steps = 31 - int(xInWindow)
	} else if dy < 0 {
		steps = int(yInWindow)
	} else if dy > 0 {
		steps = 31 - int(yInWindow)
	}
	for step := 1; step <= steps; step++ {
		position := references.Position{X: start.X + dx*references.Coordinate(step), Y: start.Y + dy*references.Coordinate(step)}
		position.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		theMap.SetTileByLayer(map_state.MapOverrideLayer, &position, tile)
	}
}
