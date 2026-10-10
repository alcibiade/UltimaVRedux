package indexes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnimationIdentityGroupsFramesButNotDirectionsOrPixelEffects(t *testing.T) {
	for _, test := range []struct{ member, canonical SpriteIndex }{
		{Fountain1, Fountain1}, {Fountain4, Fountain1}, {Waterfall4, Waterfall1},
		{Clock2, Clock1}, {Guard4, Guard1}, {Daemon4, Daemon1}, {Dragon1, Dragon1},
		{ShadowLord4, ShadowLord1}, {BrickWallArchwayWithPortcullis, BrickWallArchwayWithPortcullis},
		{PoisonField, PoisonField}, {MagicField, MagicField}, {FireField, FireField},
		{ElectricField, ElectricField}, {Shard, Shard}, {Crown, Crown},
		{Water1, Water1}, {Water2, Water2}, {WaterStream2, WaterStream2},
		{ChairBackBack, ChairBackBack}, {-1, -1},
	} {
		require.Equal(t, test.canonical, CanonicalAnimationSprite(test.member), "sprite %d", test.member)
		require.Equal(t, test.canonical, CanonicalAnimationSprite(test.canonical))
	}
	for _, sprite := range []SpriteIndex{Shard, Crown, PoisonField, MagicField, FireField, ElectricField, 512, 516, Water2} {
		_, animated := AnimationFor(sprite)
		require.False(t, animated, "sprite %d is not an atlas-frame family", sprite)
	}
}
