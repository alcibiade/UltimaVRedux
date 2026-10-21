package references

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestLargeMapReferenceGetSpriteIndexPreservesInitialAnimationFrameBehavior(t *testing.T) {
	mapReference := &LargeMapReference{}
	mapReference.rawData[0][0] = byte(indexes.WaterfallKeyIndex + 2)
	mapReference.rawData[1][0] = byte(indexes.Clock2)
	mapReference.rawData[2][0] = byte(indexes.Grass)

	assert.Equal(t, indexes.WaterfallKeyIndex, mapReference.GetSpriteIndex(0, 0))
	assert.Equal(t, indexes.SpriteIndex(indexes.Clock1), mapReference.GetSpriteIndex(1, 0))
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), mapReference.GetSpriteIndex(2, 0))
}

func TestLargeMapReferenceGetSpriteIndexWrapsCoordinates(t *testing.T) {
	mapReference := &LargeMapReference{}
	mapReference.rawData[XLargeMapTiles-1][YLargeMapTiles-1] = byte(indexes.Grass)

	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), mapReference.GetSpriteIndex(-1, -1))
}
