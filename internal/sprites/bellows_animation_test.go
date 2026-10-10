package sprites

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestBellowsAnimationCyclesBothAuthoredFrames(t *testing.T) {
	for _, source := range []indexes.SpriteIndex{indexes.Bellows1, indexes.Bellows2} {
		for _, test := range []struct {
			elapsed int64
			want    indexes.SpriteIndex
		}{
			{0, indexes.Bellows1},
			{249, indexes.Bellows1},
			{250, indexes.Bellows2},
			{499, indexes.Bellows2},
			{500, indexes.Bellows1},
			{750, indexes.Bellows2},
		} {
			got := GetSpriteIndexWithAnimationBySpriteIndexTick(source, 0, test.elapsed)
			require.Equal(t, test.want, got, "source %d at %dms", source, test.elapsed)
		}
		for _, hash := range []int32{-14331, -1, 0, 1, 14331} {
			for elapsed := int64(0); elapsed < 2000; elapsed += 125 {
				got := GetSpriteIndexWithAnimationBySpriteIndexTick(source, hash, elapsed)
				require.Contains(t, []indexes.SpriteIndex{indexes.Bellows1, indexes.Bellows2}, got)
				require.Equal(t, got, GetSpriteIndexWithAnimationBySpriteIndexTick(source, hash, elapsed))
				clock := GetSpriteIndexWithAnimationBySpriteIndexTick(indexes.Clock1, hash, elapsed)
				require.Equal(t, clock-indexes.Clock1, got-indexes.Bellows1)
			}
		}
	}
}
