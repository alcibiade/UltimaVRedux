package sprites

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// MsPerFrameForWater is the presentation interval for scrolling water pixels.
const MsPerFrameForWater = 82

// GetSpriteIndexWithAnimationBySpriteIndexTick is a deterministic animation function.
// Use elapsedMs from your game clock so animation timing is in sync with gameplay timing.
func GetSpriteIndexWithAnimationBySpriteIndexTick(
	spriteIndex indexes.SpriteIndex, posHash int32, elapsedMs int64,
) indexes.SpriteIndex {
	family, ok := indexes.AnimationFor(spriteIndex)
	if !ok || (spriteIndex != family.Base && !family.NormalizeInput) {
		return spriteIndex
	}
	rotation := cycleIndex(family.Frames, elapsedMs/family.IntervalMs, int64(posHash))

	return family.Base + indexes.SpriteIndex(rotation)
}
