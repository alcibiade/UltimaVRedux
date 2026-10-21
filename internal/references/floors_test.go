package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestGetLadderOfStairsTypeFollowsTargetFloor(t *testing.T) {
	assert.Equal(t, NotLadderOrStair, GetLadderOfStairsType(1, 1))
	assert.Equal(t, LadderOrStairUp, GetLadderOfStairsType(0, 1))
	assert.Equal(t, LadderOrStairDown, GetLadderOfStairsType(1, 0))
}

func TestIsSpecificLadderOrStairsMatchesDirection(t *testing.T) {
	for _, up := range []indexes.SpriteIndex{indexes.LadderUp, indexes.StairsNorth, indexes.StairsEast} {
		assert.True(t, IsSpecificLadderOrStairs(up, LadderOrStairUp))
		assert.False(t, IsSpecificLadderOrStairs(up, LadderOrStairDown))
	}
	for _, down := range []indexes.SpriteIndex{indexes.LadderDown, indexes.StairsSouth, indexes.StairsWest} {
		assert.True(t, IsSpecificLadderOrStairs(down, LadderOrStairDown))
		assert.False(t, IsSpecificLadderOrStairs(down, LadderOrStairUp))
	}
	assert.False(t, IsSpecificLadderOrStairs(indexes.LadderUp, NotLadderOrStair))
}
