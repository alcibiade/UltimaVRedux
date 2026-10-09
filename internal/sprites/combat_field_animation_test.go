package sprites

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatFieldSpritesDoNotCrossAnimate(t *testing.T) {
	for _, sprite := range []indexes.SpriteIndex{
		indexes.PoisonField,
		indexes.MagicField,
		indexes.ElectricField,
	} {
		assert.Equal(t, sprite, GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, 10_000))
	}
}
