package sprites

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestDecorativeBackgroundVariantsPreserveAnimationAndDefault(t *testing.T) {
	sheet := NewSpriteSheet()
	for _, decoration := range []indexes.SpriteIndex{indexes.LampPost, indexes.CampFire} {
		pipeline := TileEffectRegistry[decoration]
		for _, enabled := range []bool{true, false} {
			sheet.TileEffectsEnabled = enabled
			for _, elapsed := range []int64{0, 100, 500} {
				got := sheet.GetAnimatedSpriteWithBackground(decoration, indexes.BrickFloor, 0, elapsed)
				require.Same(t, got, sheet.GetAnimatedSpriteWithBackground(decoration, indexes.BrickFloor, 0, elapsed))
				require.Equal(t, image.Rect(0, 0, 18, 18), got.Bounds())
				require.Equal(t, pipeline, TileEffectRegistry[decoration])
				// Compare with independent alpha composition of the same effect over brick.
				foreground := pipeline
				foreground.Background = -1
				frameCount, frame := 1, 0
				if enabled {
					frameCount = sheet.tileEffectProcessor.getFrameCountForPipeline(pipeline)
					interval := pipeline.IntervalMs
					if interval <= 0 {
						interval = defaultEffectIntervalMs
					}
					frame = positiveModulo(elapsed/int64(interval), frameCount)
				} else {
					foreground = TileEffectPipeline{Background: -1, BaseBackground: -1}
				}
				layer := sheet.tileEffectProcessor.generateSingleFrame(sheet.decorationWithoutBakedBackground(decoration, pipeline.Background), frame, foreground, frameCount)
				expected := ebiten.NewImage(TileSize, TileSize)
				expected.DrawImage(sheet.GetRawSprite(indexes.BrickFloor), nil)
				expected.DrawImage(layer, nil)
				want := WrapSpriteWith18x18Border(expected)
				a, b := make([]byte, 18*18*4), make([]byte, 18*18*4)
				got.ReadPixels(a)
				want.ReadPixels(b)
				require.Equal(t, b, a)
			}
		}
	}
}

func TestCoveLampVariantRemovesBakedGrassWithoutChangingDefault(t *testing.T) {
	sheet := NewSpriteSheet()
	greenPixels := func(sprite *ebiten.Image) int {
		pixels := make([]byte, sprite.Bounds().Dx()*sprite.Bounds().Dy()*4)
		sprite.ReadPixels(pixels)
		count := 0
		for offset := 0; offset < len(pixels); offset += 4 {
			if pixels[offset+3] > 0 && pixels[offset+1] > pixels[offset] && pixels[offset+1] > pixels[offset+2] {
				count++
			}
		}
		return count
	}
	for _, enabled := range []bool{true, false} {
		sheet.TileEffectsEnabled = enabled
		before := greenPixels(sheet.GetAnimatedSprite(indexes.LampPost, 0, 0))
		require.Positive(t, before, "fixture includes baked original grass")
		require.Zero(t, greenPixels(sheet.GetAnimatedSpriteWithBackground(indexes.LampPost, indexes.BrickFloor, 0, 0)))
		require.Equal(t, before, greenPixels(sheet.GetAnimatedSprite(indexes.LampPost, 0, 0)))
	}
}
