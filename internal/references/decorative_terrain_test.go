package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestDecorativeTerrainOverridesAreLocationFloorPositionAndTileSpecific(t *testing.T) {
	for _, x := range []Coordinate{14, 16} {
		base, found := DecorativeTerrainBase(Cove, MainFloor, Position{X: x, Y: 17}, indexes.LampPost)
		assert.True(t, found)
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), base)
	}
	for _, placement := range []decorativeTerrainPlacement{
		{Britain, MainFloor, Position{X: 14, Y: 17}, indexes.LampPost},
		{Cove, Basement, Position{X: 14, Y: 17}, indexes.LampPost},
		{Cove, MainFloor, Position{X: 15, Y: 17}, indexes.LampPost},
		{Cove, MainFloor, Position{X: 14, Y: 17}, indexes.Grass},
	} {
		_, found := DecorativeTerrainBase(placement.location, placement.floor, placement.position, placement.decoration)
		assert.False(t, found)
	}
}
