package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestMirageMatchesOriginalTrueRangeTable(t *testing.T) {
	// Squared ranges in the upper-left quadrant, including center; ref:LIGHT-0002.
	table := [6][6]int{
		{50, 41, 34, 29, 26, 25}, {41, 32, 25, 20, 17, 16},
		{34, 25, 18, 13, 10, 9}, {29, 20, 13, 8, 5, 4},
		{26, 17, 10, 5, 2, 1}, {25, 16, 9, 4, 1, 0},
	}
	for y := range 11 {
		for x := range 11 {
			fx, fy := x, y
			if fx > 5 {
				fx = 10 - fx
			}
			if fy > 5 {
				fy = 10 - fy
			}
			want := indexes.SpriteIndex(indexes.Oasis)
			if table[fy][fx] <= 5 {
				want = indexes.Desert3
			}
			position := Position{X: Coordinate(x), Y: Coordinate(y)}
			for _, tile := range []indexes.SpriteIndex{indexes.Oasis, indexes.Desert3} {
				assert.Equal(t, want, MirageTerrainSprite(tile, position, Position{X: 5, Y: 5}, LargeMapType), "viewport %d,%d", x, y)
				assert.Equal(t, tile, MirageTerrainSprite(tile, position, Position{X: 5, Y: 5}, CombatMapType))
			}
		}
	}
	assert.Equal(t, indexes.SpriteIndex(indexes.Desert3), MirageTerrainSprite(indexes.Oasis, Position{X: 255, Y: 0}, Position{}, LargeMapType))
	assert.Equal(t, indexes.SpriteIndex(indexes.Chest), MirageTerrainSprite(indexes.Chest, Position{}, Position{}, SmallMapType))
}
