// Package astar implements the A* pathfinding algorithm for NPC movement in Ultima 5 Redux.
//
// The A* algorithm finds the shortest path between two points on a map while
// considering terrain walkability and movement costs. This implementation supports:
//
//   - Configurable walkability based on unit type (land, water, air units)
//   - Weighted terrain costs for different tile types
//   - Map wrapping for world map navigation
//   - Limited search radius for performance optimization
//
// Usage:
//
//	astarMap := NewAStarMap()
//	astarMap.InitializeByLayeredMap(npcUnit, gameMap, blockedTiles)
//	path := astarMap.AStar(targetPosition)
//
// The algorithm uses Manhattan distance as the heuristic function and maintains
// open and closed sets for efficient pathfinding. Performance is O((V + E) log V)
// where V is the number of tiles and E is the number of connections.
package astar

import (
	"container/heap"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type Map struct {
	walkableMap map[references.Position]int
	bWrap       bool
	maxX, maxY  references.Coordinate
	mapUnit     map_units.MapUnit
	start       references.Position
}

// NewAStarMap creates a new A* pathfinding map instance.
// The map must be initialized with InitializeByLayeredMap before pathfinding.
func NewAStarMap() *Map {
	a := &Map{}

	return a
}

// AStar performs A* pathfinding from the map unit's current position to the goal.
// The algorithm finds the shortest path considering tile walkability and weights.
//
// The pathfinding uses Manhattan distance as the heuristic and considers:
//   - Tile walkability (based on the map unit's movement capabilities)
//   - Tile weights (movement costs for different terrain types)
//   - Map wrapping for world maps
//
// Returns a slice of positions representing the path from start to goal,
// including both endpoints, or nil if no path exists.
func (m *Map) AStar(goal references.Position) []references.Position {
	openSet := &priorityQueue{}
	heap.Init(openSet)
	start := m.start
	if m.mapUnit != nil {
		start = m.mapUnit.Pos()
	}

	startNode := &Node{
		Position: start,
		GScore:   0,
		FScore:   start.HeuristicTileDistance(goal),
	}
	heap.Push(openSet, startNode)

	closedSet := make(map[references.Position]bool)
	gScore := map[references.Position]int{
		start: 0,
	}

	for openSet.Len() > 0 {
		// Get the node with the lowest FScore
		current, ok := heap.Pop(openSet).(*Node)
		if !ok {
			panic("A* priority queue contains a non-node entry")
		}

		// If we've reached the goal, reconstruct the path
		if current.Position == goal {
			return m.reconstructPath(current)
		}

		closedSet[current.Position] = true

		// Explore neighbors
		for _, neighbor := range current.Position.Neighbors() {
			// Check if the neighbor is walkable
			weight, exists := m.walkableMap[neighbor]
			if !exists || weight < 0 { // Impassable tile
				continue
			}

			// Skip if already in the closed set
			if closedSet[neighbor] {
				continue
			}

			// Calculate tentative GScore
			tentativeGScore := gScore[current.Position] + weight

			// Update scores and add to the open set
			if _, seen := gScore[neighbor]; !seen || tentativeGScore < gScore[neighbor] {
				gScore[neighbor] = tentativeGScore
				heap.Push(openSet, &Node{
					Position: neighbor,
					GScore:   tentativeGScore,
					FScore:   tentativeGScore + neighbor.HeuristicTileDistance(goal),
					Parent:   current,
				})
			}
		}
	}

	// No path found
	return nil
}

func (m *Map) reconstructPath(node *Node) []references.Position {
	var path []references.Position
	for node != nil {
		if m.bWrap {
			// if it is wrapped, then we check for wrapping at this last possible moment
			node.Position = *node.Position.GetWrapped(m.maxX, m.maxY)
		}

		path = append([]references.Position{node.Position}, path...)
		node = node.Parent
	}

	return path
}

// InitializeByLayeredMap prepares the A* map with walkability data from a game map.
// This must be called before performing pathfinding operations.
//
// Parameters:
//   - mapUnit: The unit that will be pathfinding (determines movement capabilities)
//   - lMap: The layered map containing terrain and obstacle data
//   - extraBlockTiles: Additional positions to treat as blocked (e.g., other units)
//
// The method builds a walkability map based on the unit's movement type and terrain.
func (m *Map) InitializeByLayeredMap(
	mapUnit map_units.MapUnit,
	lMap *map_state.LayeredMap,
	extraBlockTiles []references.Position,
) {
	m.mapUnit = mapUnit
	m.start = mapUnit.Pos()
	m.bWrap = false
	m.walkableMap = make(map[references.Position]int)
	for x := references.Coordinate(0); x < lMap.XMaxTilesPerMap; x++ {
		for y := references.Coordinate(0); y < lMap.YMaxTilesPerMap; y++ {
			pos := references.Position{
				X: x,
				Y: y,
			}
			topTile := lMap.GetTopTile(&pos)
			// Use NPC-specific pathfinding weight for NPCs (they can't walk on fireplaces, etc.)
			if _, ok := m.mapUnit.(*map_units.NPCFriendly); ok {
				m.walkableMap[pos] = topTile.GetNPCWalkableWeight()
			} else {
				m.walkableMap[pos] = topTile.GetWalkableWeight()
			}
		}
	}
	for i := 0; i < len(extraBlockTiles); i++ {
		m.walkableMap[extraBlockTiles[i]] = -1
	}
}

// InitializeByMovementCost prepares an unwrapped pathfinding map using rules
// supplied by the caller. Negative costs mark positions as impassable.
func (m *Map) InitializeByMovementCost(
	start references.Position,
	maxX, maxY references.Coordinate,
	movementCost func(references.Position) int,
) {
	if movementCost == nil {
		panic("astar.Map.InitializeByMovementCost called without movement cost function")
	}

	m.mapUnit = nil
	m.start = start
	m.bWrap = false
	m.maxX = maxX
	m.maxY = maxY
	m.walkableMap = make(map[references.Position]int, int(maxX)*int(maxY))
	for x := references.Coordinate(0); x < maxX; x++ {
		for y := references.Coordinate(0); y < maxY; y++ {
			position := references.Position{X: x, Y: y}
			m.walkableMap[position] = movementCost(position)
		}
	}
}

func (m *Map) InitializeByLayeredMapWithLimit(
	mapUnit map_units.MapUnit,
	lMap *map_state.LayeredMap,
	extraBlockTiles []references.Position,
	bWrap bool,
	centerPos references.Position,
	nMaxRadius int,
	maxX, maxY references.Coordinate,
) {
	if maxX <= 0 || maxY <= 0 || nMaxRadius < 0 || nMaxRadius > int(maxX) || nMaxRadius > int(maxY) {
		panic("A* window radius must fit within positive map dimensions")
	}
	m.mapUnit = mapUnit
	m.start = mapUnit.Pos()
	m.bWrap = bWrap
	m.maxX, m.maxY = maxX, maxY
	m.walkableMap = make(map[references.Position]int)

	for worldX := int(centerPos.X) - nMaxRadius; worldX < int(centerPos.X)+nMaxRadius; worldX++ {
		for worldY := int(centerPos.Y) - nMaxRadius; worldY < int(centerPos.Y)+nMaxRadius; worldY++ {
			pos := references.Position{
				X: mapWindowRemainder(worldX, maxX),
				Y: mapWindowRemainder(worldY, maxY),
			}
			if m.bWrap {
				pos = *pos.GetWrapped(m.maxX, m.maxY)
			}
			topTile := lMap.GetTopTile(&pos)
			if enemy, ok := m.mapUnit.(*map_units.NPCEnemy); ok {
				if enemy.EnemyReference.CanMoveToTile(topTile) {
					m.walkableMap[pos] = 1
				} else {
					m.walkableMap[pos] = -1
				}
			} else if _, ok := m.mapUnit.(*map_units.NPCFriendly); ok {
				// Use NPC-specific pathfinding weight for friendly NPCs
				m.walkableMap[pos] = topTile.GetNPCWalkableWeight()
			} else {
				m.walkableMap[pos] = topTile.GetWalkableWeight()
			}
		}
	}
	for i := 0; i < len(extraBlockTiles); i++ {
		m.walkableMap[extraBlockTiles[i]] = -1
	}
}

func mapWindowRemainder(worldCoordinate int, dimension references.Coordinate) references.Coordinate {
	//nolint:gosec // Remainder magnitude is below the positive int16 map dimension.
	return references.Coordinate(worldCoordinate % int(dimension))
}
