package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestTileIsPushableForMovableFurnitureOnly(t *testing.T) {
	for _, sprite := range []indexes.SpriteIndex{
		indexes.ChairBackForward, indexes.CannonUp, indexes.Barrel, indexes.TableMiddle,
		indexes.Chest, indexes.EndTable, indexes.Plant,
	} {
		assert.True(t, (&Tile{Index: sprite}).IsPushable(), "sprite %d", sprite)
	}
	for _, sprite := range []indexes.SpriteIndex{indexes.BrickFloor, indexes.StoneBrickWall} {
		assert.False(t, (&Tile{Index: sprite}).IsPushable(), "sprite %d", sprite)
	}
}
