package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	sourceOutdoorViewSize   = 11
	sourceOutdoorViewCenter = 5
)

// traceOutdoorMissile reuses original projectile rules's pixel path and 13-pixel samples in
// player-centred source screen coordinates. Neither Bresenham tile LOS nor a
// renderer visibility cache is equivalent to missile terrain validation.
// See ref:ENEMY-0007.
func (g *GameState) traceOutdoorMissile(from, target references.Position, missile references.MissileType) (references.Position, bool) {
	center := references.Position{X: sourceOutdoorViewCenter, Y: sourceOutdoorViewCenter}
	start := references.Position{
		X: center.X + outdoorWrappedDelta(target.X, from.X),
		Y: center.Y + outdoorWrappedDelta(target.Y, from.Y),
	}
	buffer := g.outdoorMissileBuffer(target)
	impact, reached := combatMissileImpactPosition(func(pos references.Position) indexes.SpriteIndex {
		return buffer[pos.Y][pos.X]
	}, start, center, missile)

	return outdoorScreenPosition(target, impact), reached
}

func outdoorWrappedDelta(from, to references.Coordinate) references.Coordinate {
	delta := to - from
	if delta > references.MapWrapThreshold {
		delta -= references.MapCoordinateSpace
	} else if delta < -references.MapWrapThreshold {
		delta += references.MapCoordinateSpace
	}

	return delta
}

func outdoorScreenPosition(center, screen references.Position) references.Position {
	position := references.Position{
		X: center.X + screen.X - sourceOutdoorViewCenter,
		Y: center.Y + screen.Y - sourceOutdoorViewCenter,
	}

	return *position.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
}

type outdoorMissileView [sourceOutdoorViewSize][sourceOutdoorViewSize]indexes.SpriteIndex

// outdoorMissileBuffer reconstructs the collision-relevant DOS display buffer
// from authoritative terrain, lighting and objects. visibility flood leaves
// unseen cells at NOTHING (0xff), which missile passage table permits; object display projection
// clears displayed object cells to zero, which also permits missiles.
// See ref:ENEMY-0007.
func (g *GameState) outdoorMissileBuffer(center references.Position) outdoorMissileView {
	terrainAt := func(screen references.Position) indexes.SpriteIndex {
		world := outdoorScreenPosition(center, screen)

		return g.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&world).Index
	}
	var buffer outdoorMissileView
	for y := range buffer {
		for x := range buffer[y] {
			buffer[y][x] = indexes.BlackSquare
		}
	}
	sight := g.outdoorMissileSight()
	if terrainAt(references.Position{X: sourceOutdoorViewCenter, Y: sourceOutdoorViewCenter}) == indexes.BlackSquare &&
		!g.IsAmuletActive() {
		return buffer // Unmapped-terrain handling at NOTHING terrain.
		// See ref:ENEMY-0007.
	}
	lit := g.outdoorMissileFixedLights(center)
	// Source flood visits W, SW, S, SE, E, NE, N, NW.
	// See ref:ENEMY-0007.
	directions := [...]references.Position{
		{X: -1}, {X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1},
		{X: 1}, {X: 1, Y: -1}, {Y: -1}, {X: -1, Y: -1},
	}
	origin := references.Position{X: sourceOutdoorViewCenter, Y: sourceOutdoorViewCenter}
	queue := []references.Position{origin}
	var visited [sourceOutdoorViewSize][sourceOutdoorViewSize]bool
	visited[origin.Y][origin.X] = true
	buffer[origin.Y][origin.X] = terrainAt(origin)
	for front := 0; front < len(queue); front++ {
		parent := queue[front]
		for _, direction := range directions {
			pos := references.Position{X: parent.X + direction.X, Y: parent.Y + direction.Y}
			if pos.X < 0 || pos.X >= sourceOutdoorViewSize || pos.Y < 0 || pos.Y >= sourceOutdoorViewSize || visited[pos.Y][pos.X] {
				continue
			}
			tile := terrainAt(pos)
			dx, dy := int(pos.X-origin.X), int(pos.Y-origin.Y)
			distance := dx*dx + dy*dy // Original squared-distance lookup's table is squared distance.
			// See ref:ENEMY-0007.
			transparent := outdoorSourceSeeThrough(tile, distance)
			illuminated := distance <= sight || (lit[pos.Y][pos.X] && (transparent || lit[parent.Y][parent.X]))
			if illuminated {
				buffer[pos.Y][pos.X] = tile
			}
			// Unlit opaque cells may later be reached from a lit parent.
			visited[pos.Y][pos.X] = illuminated || transparent
			if transparent {
				queue = append(queue, pos)
			}
		}
	}
	for y := range buffer {
		for x, tile := range buffer[y] {
			if isCombatMissilePassable(tile) || tile == indexes.BrickWallArchway {
				continue
			}
			// Screen coordinates are bounded to the original 11x11 view.
			pos := outdoorScreenPosition(center, references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)})
			if g.outdoorMissileObjectClearsTerrain(pos, tile) {
				buffer[y][x] = 0
			}
		}
	}

	return buffer
}

func (g *GameState) outdoorMissileObjectClearsTerrain(pos references.Position, tile indexes.SpriteIndex) bool {
	// Tile-context sprite projection does not overlay creatures on tropical forest or flags.
	// See ref:ENEMY-0007.
	if tile == indexes.Forest4 || tile == indexes.SnakeSign1 {
		return false
	}
	if pos == g.MapState.PlayerLocation.Position || g.ItemStacksMap.HasItemStackAtPosition(&pos) {
		return true
	}
	unit := g.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(pos, g.MapState.PlayerLocation.Floor)
	if unit != nil && (*unit).IsVisible() && !g.isQuestItemNPCHidden(*unit) {
		return true
	}
	hasItem, item := g.IsQuestItemAtPosition(pos)

	return hasItem && g.IsQuestItemVisible(pos, item)
}

// Ambient sight update advances source sight independently of Redux's cosmetic tint
// radius. Debug X-ray and renderer refreshes must not change hit outcomes.
// See ref:ENEMY-0007.
func (g *GameState) outdoorMissileSight() int {
	const night, daylight, torch, magic = 2, 50, 10, 18
	sight := night
	if !g.MapState.IsUnderworld() {
		dawn := [...]int{2, 5, 10, 20, 34, 49}
		switch {
		case g.DateTime.Hour == 5:
			sight = dawn[g.DateTime.Minute/10]
		case g.DateTime.Hour == 19:
			sight = dawn[(59-g.DateTime.Minute)/10]
		case g.DateTime.Hour > 5 && g.DateTime.Hour < 19:
			sight = daylight
		}
	}
	if g.MapState.Lighting.HasTorchLit() {
		sight = max(sight, torch)
	}
	if g.MapState.Lighting.HasMagicLight() {
		sight = max(sight, magic)
	}

	return sight
}

func outdoorSourceSeeThrough(tile indexes.SpriteIndex, distance int) bool {
	switch tile { //nolint:exhaustive // sight-passability test's terrain exceptions.
	// See ref:ENEMY-0007.
	case indexes.StoneGlassWindow, indexes.StoneCrossWindow, indexes.RegularDoorView,
		indexes.LockedDoorView, indexes.MagicLockDoorWithView:
		return distance == 1
	case indexes.Forest3, indexes.Forest4, indexes.SmallMountains, indexes.TallMountains,
		indexes.LargeRockWall, indexes.StoneBrickWallSecret, indexes.StoneBrickWall,
		indexes.Bookcase, indexes.MagicLockDoor, indexes.RegularDoor, indexes.LockedDoor,
		indexes.Fireplace, indexes.CornerStructure1, indexes.CornerStructure2,
		indexes.CornerStructure3, indexes.CornerStructure4, indexes.SignWarning,
		indexes.WhiteSquare, indexes.BlackSquare:
		return false
	default:
		return true
	}
}

// Fixed light-source scan uses radius-squared 10 for its fixed glowing terrain.
// Scan three tiles beyond the original view to include off-screen sources.
// See ref:ENEMY-0007.
func (g *GameState) outdoorMissileFixedLights(center references.Position) [sourceOutdoorViewSize][sourceOutdoorViewSize]bool {
	const lightRadius = 3
	var lit [sourceOutdoorViewSize][sourceOutdoorViewSize]bool
	for y := references.Coordinate(-lightRadius); y < sourceOutdoorViewSize+lightRadius; y++ {
		for x := references.Coordinate(-lightRadius); x < sourceOutdoorViewSize+lightRadius; x++ {
			light := references.Position{X: x, Y: y}
			world := outdoorScreenPosition(center, light)
			tile := g.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&world).Index
			switch tile { //nolint:exhaustive // Original glowing[] table.
			case indexes.Moongate, indexes.LampPost, indexes.CandleOnTable, indexes.Brazier,
				indexes.BlueFlame, indexes.CookStove, indexes.RightSconce, indexes.LeftSconce,
				indexes.CampFire, indexes.Fireplace:
				g.fillOutdoorMissileLight(center, light, &lit)
			default:
			}
		}
	}

	return lit
}

func (g *GameState) fillOutdoorMissileLight(center, light references.Position, lit *[sourceOutdoorViewSize][sourceOutdoorViewSize]bool) {
	const squaredRadius = 10
	queue := []references.Position{light}
	seen := map[references.Position]bool{light: true}
	for front := 0; front < len(queue); front++ {
		pos := queue[front]
		if pos.X >= 0 && pos.X < sourceOutdoorViewSize && pos.Y >= 0 && pos.Y < sourceOutdoorViewSize {
			lit[pos.Y][pos.X] = true
		}
		for dy := references.Coordinate(-1); dy <= 1; dy++ {
			for dx := references.Coordinate(-1); dx <= 1; dx++ {
				next := references.Position{X: pos.X + dx, Y: pos.Y + dy}
				x, y := int(next.X-light.X), int(next.Y-light.Y)
				if seen[next] || x*x+y*y > squaredRadius {
					continue
				}
				seen[next] = true
				if next.X >= 0 && next.X < sourceOutdoorViewSize && next.Y >= 0 && next.Y < sourceOutdoorViewSize {
					lit[next.Y][next.X] = true
				}
				world := outdoorScreenPosition(center, next)
				if outdoorSourceSeeThrough(g.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&world).Index, x*x+y*y) {
					queue = append(queue, next)
				}
			}
		}
	}
}
