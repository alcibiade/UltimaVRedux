package indexes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBellowsAnimationFamily(t *testing.T) {
	clock, ok := AnimationFor(Clock1)
	require.True(t, ok)
	for _, sprite := range []SpriteIndex{Bellows1, Bellows2} {
		family, animated := AnimationFor(sprite)
		require.True(t, animated)
		require.Equal(t, AnimationFamily{Bellows1, 2, clock.IntervalMs, true}, family)
		require.Equal(t, Bellows1, int(CanonicalAnimationSprite(sprite)))
	}
	for _, sprite := range []SpriteIndex{WhiteSquare, BlackSquare} {
		_, animated := AnimationFor(sprite)
		require.False(t, animated, "neighboring sprite %d must stay static", sprite)
	}
}
