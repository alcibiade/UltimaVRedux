package astar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestInitializeByMovementCost_FindsRouteAroundBlockedPositions(t *testing.T) {
	pathfinder := NewAStarMap()
	blocked := map[references.Position]bool{
		{X: 1, Y: 0}: true,
		{X: 1, Y: 1}: true,
	}
	pathfinder.InitializeByMovementCost(
		references.Position{X: 0, Y: 0},
		3,
		3,
		func(position references.Position) int {
			if blocked[position] {
				return -1
			}

			return 1
		},
	)

	path := pathfinder.AStar(references.Position{X: 2, Y: 0})

	require.Len(t, path, 7)
	assert.Equal(t, references.Position{X: 0, Y: 0}, path[0])
	assert.Equal(t, references.Position{X: 2, Y: 0}, path[len(path)-1])
	for _, position := range path {
		assert.False(t, blocked[position])
	}
}

func TestInitializeByMovementCost_ReturnsNilWithoutRoute(t *testing.T) {
	pathfinder := NewAStarMap()
	pathfinder.InitializeByMovementCost(
		references.Position{X: 0, Y: 0},
		3,
		3,
		func(position references.Position) int {
			if position.X == 1 {
				return -1
			}

			return 1
		},
	)

	assert.Nil(t, pathfinder.AStar(references.Position{X: 2, Y: 0}))
}

func TestAStar_ExistingMapUnitInitializationUsesCurrentPosition(t *testing.T) {
	unit := map_units.NewNPCFriendly(references.NPCReference{}, 0)
	unit.SetPos(references.Position{X: 0, Y: 0})
	pathfinder := NewAStarMap()
	pathfinder.mapUnit = unit
	pathfinder.start = unit.Pos()
	pathfinder.walkableMap = map[references.Position]int{
		{X: 0, Y: 0}: 1,
		{X: 1, Y: 0}: 1,
		{X: 2, Y: 0}: 1,
	}
	unit.SetPos(references.Position{X: 1, Y: 0})

	path := pathfinder.AStar(references.Position{X: 2, Y: 0})

	require.Len(t, path, 2)
	assert.Equal(t, references.Position{X: 1, Y: 0}, path[0])
}
