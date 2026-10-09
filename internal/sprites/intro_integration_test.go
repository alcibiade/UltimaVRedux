package sprites

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationLoadIntroSprites(t *testing.T) {
	dataDir := originaldata.Directory(t)

	sprites, err := LoadIntroSprites(dataDir)

	require.NoError(t, err)
	t.Cleanup(func() {
		owned := make(map[*ebiten.Image]bool)
		for _, frames := range [][]*ebiten.Image{sprites.OriginFrames, sprites.FlameAnimation.images,
			{sprites.Presents, sprites.ArticleA, sprites.Production, sprites.BritishSignature,
				sprites.Ultima16Logo, sprites.FlameFizzBase, sprites.ReduxLogo}} {
			for _, frame := range frames {
				owned[frame] = true
			}
		}
		for frame := range owned {
			frame.Dispose()
		}
	})
	require.Len(t, sprites.OriginFrames, 7)
	require.Len(t, sprites.BritishPaths, 4)
	assert.Equal(t, 288, sprites.FlameFizzBase.Bounds().Dx())
	assert.Equal(t, 49, sprites.FlameFizzBase.Bounds().Dy())
	assert.Equal(t, 319, sprites.Ultima16Logo.Bounds().Dx())
	assert.Equal(t, 61, sprites.Ultima16Logo.Bounds().Dy())
	assert.Equal(t, 288, sprites.FlameAnimation.GetCurrentImageAtTime(0).Bounds().Dx())
	assert.Equal(t, 49, sprites.FlameAnimation.GetCurrentImageAtTime(0).Bounds().Dy())
}
