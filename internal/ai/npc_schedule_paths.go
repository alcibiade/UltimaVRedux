package ai

import (
	"cmp"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/astar"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// scheduleLadders reads the observed floor's persistent terrain, including
// overrides. Visual objects never create or conceal a connection. Equal-distance
// candidates retain coordinate order, including (0,0) and the final row/column.
func (n *NPCAIControllerSmallMap) scheduleLadders(
	anchor references.Position, otherFloor references.FloorNumber,
) []references.Position {
	direction := references.GetLadderOfStairsType(n.mapState.PlayerLocation.Floor, otherFloor)
	var candidates []references.Position
	for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
		for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
			position := references.Position{X: x, Y: y}
			if references.IsSpecificLadderOrStairs(n.npcTerrainAt(position).Index, direction) {
				candidates = append(candidates, position)
			}
		}
	}
	slices.SortStableFunc(candidates, func(left, right references.Position) int {
		return cmp.Compare(anchor.HeuristicTileDistance(left), anchor.HeuristicTileDistance(right))
	})

	return candidates
}

// schedulePositionOccupied tests authoritative units on the observed floor.
// In particular, an incoming actor's old X/Y cannot exempt a receiving occupant.
func (n *NPCAIControllerSmallMap) schedulePositionOccupied(
	friendly *map_units.NPCFriendly, position references.Position,
) bool {
	if position == n.mapState.PlayerLocation.Position {
		return true
	}
	for _, unit := range n.mapUnits {
		if unit != friendly && unit.Floor() == n.mapState.PlayerLocation.Floor && unit.Pos() == position {
			return true
		}
	}

	return false
}

// findScheduledPath plans on the observed floor without moving the actor or
// player. The returned path includes both endpoints; nil means defer this route.
func (n *NPCAIControllerSmallMap) findScheduledPath(
	friendly *map_units.NPCFriendly, start, goal references.Position,
) []references.Position {
	if !isSmallMapPosition(start) || !isSmallMapPosition(goal) || n.schedulePositionOccupied(friendly, goal) {
		return nil
	}
	behavior := friendly.NPCReference.Schedule.GetIndividualNPCBehaviourByUltimaDate(*n.dateTime)
	aStarMap := astar.NewAStarMap()
	aStarMap.InitializeByMovementCost(start, references.XSmallMapTiles, references.YSmallMapTiles,
		func(position references.Position) int {
			// REDUX DEVIATION: NPC-0002: original planning marks limited nearby
			// objects. Redux excludes all authoritative occupants, including goals,
			// to avoid repeatedly planning routes with known collisions.
			if n.schedulePositionOccupied(friendly, position) {
				return -1
			}
			// NPC-0004: planning has no beach/Shadowlord exception, but permits
			// the exact scheduled destination on this floor (including beds).
			if behavior.Floor == n.mapState.PlayerLocation.Floor && behavior.Position == position {
				return 1
			}
			terrain := n.npcTerrainAt(position)
			// REDUX DEVIATION: NPC-0002: exclude chairs on departure routes,
			// avoiding a plan that actual ladder-travel steps must reject.
			if behavior.Floor != n.mapState.PlayerLocation.Floor && terrain.IsChair() {
				return -1
			}

			return terrain.GetNPCWalkableWeight()
		})

	return aStarMap.AStar(goal)
}
