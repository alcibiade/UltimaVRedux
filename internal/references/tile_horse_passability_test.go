package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestHorsePassabilityIncludesOriginalPathAndShorelineFamilies(t *testing.T) {
	for _, index := range []indexes.SpriteIndex{
		indexes.OutsidePath1,
		indexes.OutsidePath2,
		indexes.OutsidePath3,
		indexes.OutsidePath4,
		indexes.OutsideWaterCoast1,
		indexes.OutsideWaterCoast2,
		indexes.OutsideWaterCoast3,
		indexes.OutsideWaterCoast4,
	} {
		tile := Tile{Index: index}
		assert.True(t, tile.IsHorsePassable(), "tile %d should be horse-passable", index)
	}
}

func TestHorsePassabilityAppliesOriginalHorseExclusions(t *testing.T) {
	for _, index := range []indexes.SpriteIndex{
		indexes.Swamp,
		indexes.Lava,
		indexes.ChairBackForward,
		indexes.ChairBackLeft,
		indexes.ChairBackBack,
		indexes.ChairBackRight,
	} {
		tile := Tile{Index: index}
		assert.False(t, tile.IsHorsePassable(), "tile %d should block horses", index)
	}
}

func TestHorsePassabilityMatchesOriginalOnFootRule(t *testing.T) {
	for index := range 256 {
		tile := Tile{Index: indexes.SpriteIndex(index)}
		want := tile.IsWalkingPassable() &&
			!tile.Is(indexes.Swamp) &&
			!tile.Is(indexes.Lava) &&
			!tile.IsChair()

		assert.Equal(t, want, tile.IsHorsePassable(), "horse passability mismatch for tile %d", index)
	}
}
