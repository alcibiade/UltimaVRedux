package sprites

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// MsPerFrameForWater is the presentation interval for scrolling water pixels.
const MsPerFrameForWater = 82

// GetSpriteIndexWithAnimationBySpriteIndexTick is a deterministic animation function.
// Use elapsedMs from your game clock so animation timing is in sync with gameplay timing.
func GetSpriteIndexWithAnimationBySpriteIndexTick(spriteIndex indexes.SpriteIndex, posHash int32, elapsedMs int64) indexes.SpriteIndex {
	family, ok := indexes.AnimationFor(spriteIndex)
	if !ok || (spriteIndex != family.Base && !family.NormalizeInput) {
		return spriteIndex
	}
	rotation := positiveModulo(elapsedMs/family.IntervalMs+int64(posHash), family.Frames)

	return family.Base + indexes.SpriteIndex(rotation)
}

func positiveModulo(value int64, modulus int) int {
	remainder := int(value % int64(modulus))
	if remainder < 0 {
		return remainder + modulus
	}

	return remainder
}
