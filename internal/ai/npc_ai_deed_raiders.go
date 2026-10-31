package ai

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// REDUX DEVIATION: CRAFT-0021: the original has no deed maps and no small-map monsters that
// hunt the party apart from Shadowlords. Optional crafting raiders walk toward the party on the
// party's own deed map, around walls and closed doors, and never change a tile.

// IsDeedRaiderTerrainPassable reports whether a raider may step onto a tile (ref:CRAFT-0021):
// ordinary town NPC terrain, except doors. Raiders cannot open doors, so a closed door is a wall
// to them; an open door shows its floor and lets them through.
func IsDeedRaiderTerrainPassable(tile *references.Tile) bool {
	return tile != nil && tile.IsNPCTerrainPassable() && !tile.Index.IsDoor()
}

// moveDeedRaider takes one step of a raider toward the party. A raider already next to the party
// waits for the encounter handoff. It follows the shortest route around walls and closed doors,
// and around other units when one is open; with the way blocked by units it steps greedily, and
// with the party walled in it closes in as far as the walls allow. Movement draws no randomness.
func (n *NPCAIControllerSmallMap) moveDeedRaider(raider *map_units.NPCEnemy) {
	floor := n.mapState.PlayerLocation.Floor
	if raider.Floor() != floor {
		return
	}
	avatar := n.mapState.PlayerLocation.Position
	position := raider.Pos()
	if position.IsNextTo(avatar, raider.Floor(), floor) {
		return
	}
	if step, ok := n.deedRaiderRouteStep(raider, position, avatar, true); ok {
		raider.SetPos(step)

		return
	}
	n.moveDeedRaiderGreedy(raider, position, avatar)
}

func (n *NPCAIControllerSmallMap) deedRaiderRouteStep(
	raider *map_units.NPCEnemy, start, goal references.Position, avoidUnits bool,
) (references.Position, bool) {
	bounds := image.Rect(0, 0, int(references.XSmallMapTiles), int(references.YSmallMapTiles))
	route := references.FindRoute(start, goal, bounds, func(position references.Position) int {
		if position == goal {
			return 1
		}
		if avoidUnits && n.deedRaiderSquareOccupied(raider, position) {
			return -1
		}
		if !IsDeedRaiderTerrainPassable(n.npcTerrainAt(position)) {
			return -1
		}

		return 1
	})
	if len(route) < 2 || route[1] == goal || n.deedRaiderSquareOccupied(raider, route[1]) {
		return references.Position{}, false
	}

	return route[1], true
}

// moveDeedRaiderGreedy steps to the first passable, free neighbour (east, north, west, south)
// that brings the raider closer to the party.
func (n *NPCAIControllerSmallMap) moveDeedRaiderGreedy(
	raider *map_units.NPCEnemy, position, avatar references.Position,
) {
	current := n.manhattanDistance(position, avatar)
	for _, direction := range originalNPCDirections {
		next := references.Position{X: position.X + direction.X, Y: position.Y + direction.Y}
		if !isSmallMapPosition(next) || next == avatar || n.manhattanDistance(next, avatar) >= current ||
			n.deedRaiderSquareOccupied(raider, next) || !IsDeedRaiderTerrainPassable(n.npcTerrainAt(next)) {
			continue
		}
		raider.SetPos(next)

		return
	}
}

func (n *NPCAIControllerSmallMap) deedRaiderSquareOccupied(
	raider *map_units.NPCEnemy, position references.Position,
) bool {
	if position == n.mapState.PlayerLocation.Position {
		return true
	}
	for _, unit := range n.mapUnits {
		if unit != raider && unit.IsVisible() && unit.Floor() == raider.Floor() && unit.Pos() == position {
			return true
		}
	}

	return false
}
