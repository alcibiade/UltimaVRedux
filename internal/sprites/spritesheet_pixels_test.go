package sprites

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestStaticSpritePixelsMatchAtlas(t *testing.T) {
	sheet := NewSpriteSheet()
	t.Cleanup(func() {
		for _, sprite := range sheet.staticBorderedCache {
			if sprite != nil {
				sprite.Dispose()
			}
		}
		sheet.SpriteImage.Dispose()
	})
	source, err := png.Decode(bytes.NewReader(transparentTiles))
	require.NoError(t, err)
	actual, expected := make([]byte, 18*18*4), make([]byte, 18*18*4)
	for index := indexes.SpriteIndex(0); index < totalSprites; index++ {
		if sheet.tileEffectProcessor.HasEffect(index) {
			require.Nil(t, sheet.staticBorderedCache[index], "effect tile %d must remain deferred", index)
			continue
		}
		require.NotNil(t, sheet.staticBorderedCache[index], "static tile %d must be prepared by the constructor", index)
		sprite := sheet.GetSprite(index)
		require.Equal(t, image.Rect(0, 0, 18, 18), sprite.Bounds())
		require.Same(t, sprite, sheet.GetSprite(index))
		bounds := sheet.getSpriteImageRectangle(index)
		require.Equal(t, image.Rect(int(index)%32*16, int(index)/32*16, (int(index)%32+1)*16, (int(index)/32+1)*16), bounds)
		require.True(t, bounds.In(source.Bounds()))
		// Exhaustive cache shape/identity and CPU crop geometry are cheap;
		// native pixels need only corners, a row transition, and interior artwork.
		if !representativeSpritePixelIndex(index) {
			continue
		}
		clear(expected)
		for y := range TileSize {
			for x := range TileSize {
				r, g, b, a := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				target := ((y+1)*18 + x + 1) * 4
				copy(expected[target:target+4], []byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)})
			}
		}
		sprite.ReadPixels(actual)
		if !bytes.Equal(expected, actual) {
			t.Fatalf("static tile %d differs from atlas pixels or its transparent border", index)
		}
	}
	require.False(t, sheet.tileEffectProcessor.initialized, "static preparation must not generate effect frames")
}

func representativeSpritePixelIndex(index indexes.SpriteIndex) bool {
	switch index {
	case 0, 31, 32, 247, 480, 511:
		return true
	default:
		return false
	}
}

func TestWrapAtlasSubimagePreservesPartialAlphaAndOrigin(t *testing.T) {
	atlas := ebiten.NewImage(64, 64)
	t.Cleanup(atlas.Dispose)
	pixels := make([]byte, 64*64*4)
	for y := range 64 {
		for x := range 64 {
			i := (y*64 + x) * 4
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = byte(x), byte(y), byte(x+y), 128
		}
	}
	atlas.WritePixels(pixels)
	for _, origin := range []image.Point{{0, 0}, {16, 32}, {48, 48}} {
		crop := atlas.SubImage(image.Rectangle{Min: origin, Max: origin.Add(image.Pt(16, 16))}).(*ebiten.Image)
		oldCopy := ebiten.NewImageFromImage(crop)
		oldBorder := WrapSpriteWith18x18Border(oldCopy)
		direct := WrapSpriteWith18x18Border(crop)
		before, after := make([]byte, 18*18*4), make([]byte, 18*18*4)
		oldBorder.ReadPixels(before)
		direct.ReadPixels(after)
		require.Equal(t, image.Rect(0, 0, 18, 18), direct.Bounds())
		equal := bytes.Equal(before, after)
		oldCopy.Dispose()
		oldBorder.Dispose()
		direct.Dispose()
		require.True(t, equal, "cropped atlas origin %v must preserve premultiplied pixels", origin)
	}
}

func TestRawSpriteCachePixelsIdentityAndBounds(t *testing.T) {
	atlas := ebiten.NewImage(origTilesWidth, origTilesHeight)
	t.Cleanup(atlas.Dispose)
	pixels := make([]byte, origTilesWidth*origTilesHeight*4)
	for y := range origTilesHeight {
		for x := range origTilesWidth {
			offset := (y*origTilesWidth + x) * 4
			pixels[offset], pixels[offset+1], pixels[offset+2], pixels[offset+3] = byte(x%128), byte(y%128), byte((x+y)%128), 128
		}
	}
	atlas.WritePixels(pixels)
	// Atlas-only fixtures must retain lazy cache support without constructors.
	sheet := &SpriteSheet{SpriteImage: atlas}
	owned := make(map[*ebiten.Image]bool)
	t.Cleanup(func() {
		for sprite := range owned {
			sprite.Dispose()
		}
	})
	actual, expected := make([]byte, 16*16*4), make([]byte, 16*16*4)
	for index := indexes.SpriteIndex(0); index < totalSprites; index++ {
		bounds := sheet.getSpriteImageRectangle(index)
		require.Equal(t, image.Pt(int(index)%32*16, int(index)/32*16), bounds.Min)
		require.Equal(t, image.Pt(TileSize, TileSize), bounds.Size())
		for y := range TileSize {
			source := ((bounds.Min.Y+y)*origTilesWidth + bounds.Min.X) * 4
			copy(expected[y*TileSize*4:(y+1)*TileSize*4], pixels[source:source+TileSize*4])
		}
		if representativeSpritePixelIndex(index) {
			sprite := sheet.GetRawSprite(index)
			require.False(t, owned[sprite], "different tiles must not alias one cached image")
			owned[sprite] = true
			require.Same(t, sprite, sheet.GetRawSprite(index))
			require.Equal(t, image.Rect(0, 0, 16, 16), sprite.Bounds())
			sprite.ReadPixels(actual)
			if !bytes.Equal(expected, actual) {
				t.Fatalf("raw sprite %d differs from source atlas pixels", index)
			}
		}
	}
	for _, index := range []indexes.SpriteIndex{-1, 512, 1024, 32000} {
		sprite := sheet.GetRawSprite(index)
		owned[sprite] = true
		require.Equal(t, image.Rect(0, 0, 16, 16), sprite.Bounds())
		sprite.ReadPixels(actual)
		clear(expected)
		require.True(t, bytes.Equal(expected, actual), "invalid tile %d stays transparent", index)
	}
	other := &SpriteSheet{SpriteImage: atlas}
	borrowed := sheet.GetRawSprite(indexes.Grass)
	owned[borrowed] = true
	independent := other.GetRawSprite(indexes.Grass)
	owned[independent] = true
	require.NotSame(t, borrowed, independent, "raw cache belongs to the sheet, not a global singleton")
	editable := ebiten.NewImageFromImage(borrowed)
	owned[editable] = true
	borrowed.ReadPixels(expected)
	editable.Clear()
	borrowed.ReadPixels(actual)
	require.True(t, bytes.Equal(expected, actual), "editing an owned copy must not mutate its borrowed source")
}

func TestRawAnimatedSpriteReusesResolvedTile(t *testing.T) {
	atlas := ebiten.NewImage(origTilesWidth, origTilesHeight)
	t.Cleanup(atlas.Dispose)
	sheet := &SpriteSheet{SpriteImage: atlas}
	owned := make(map[*ebiten.Image]bool)
	t.Cleanup(func() {
		for sprite := range owned {
			sprite.Dispose()
		}
	})
	for _, hash := range []int32{-37, 0, 1, 17} {
		for _, elapsed := range []int64{0, 99, 100, 999, 1000, 10000} {
			index := GetSpriteIndexWithAnimationBySpriteIndexTick(indexes.Water1, hash, elapsed)
			want := sheet.GetRawSprite(index)
			owned[want] = true
			require.Same(t, want, sheet.GetRawAnimatedSprite(indexes.Water1, hash, elapsed))
		}
	}
}

func TestRawEffectLookupRetainsGeneratedFramesAndSourcePixels(t *testing.T) {
	sheet := smallResolverSpriteSheet(t)
	base := sheet.GetRawSprite(indexes.Water1)
	before, after := make([]byte, 16*16*4), make([]byte, 16*16*4)
	base.ReadPixels(before)
	for offset := 0; offset < len(before); offset += 4 {
		require.Equal(t, []byte{255, 255, 255, 255}, before[offset:offset+4], "source preservation must not pass on an empty tile")
	}
	processor := sheet.GetTileEffectProcessor()
	for _, hash := range []int32{0, 261, 517, -17} {
		for _, elapsed := range []int64{0, 81, 82, 1230, 1311, 1312, 1313, 1394} {
			frame, _ := processor.frameIndices(indexes.Water1, hash, elapsed)
			want := processor.effectFrames[indexes.Water1][frame]
			require.NotNil(t, want)
			require.Same(t, want, sheet.GetRawAnimatedSprite(indexes.Water1, hash, elapsed))
		}
	}
	base.ReadPixels(after)
	require.True(t, bytes.Equal(before, after), "effect lookup cannot modify its borrowed atlas tile")
	require.Same(t, base, sheet.GetRawSprite(indexes.Water1))
	// Literal boundaries exercise production timing, not a local copy of it.
	for _, tc := range []struct {
		elapsed   int64
		odd, even int
	}{{0, 0, 6}, {81, 0, 6}, {82, 1, 7}, {1230, 15, 5}, {1311, 15, 5}, {1312, 0, 6}, {1313, 0, 6}, {1394, 1, 7}} {
		require.Same(t, processor.effectFrames[indexes.Water1][tc.odd],
			sheet.GetRawAnimatedSprite(indexes.Water1, 261, tc.elapsed))
		require.Same(t, processor.effectFrames[indexes.Water1][tc.even],
			sheet.GetRawAnimatedSprite(indexes.Water1, 517, tc.elapsed))
	}
}

func BenchmarkRawSpriteLookup(b *testing.B) {
	atlas := ebiten.NewImage(origTilesWidth, origTilesHeight)
	b.Cleanup(atlas.Dispose)
	sheet := &SpriteSheet{SpriteImage: atlas}
	sheet.GetRawSprite(indexes.Grass)
	b.ReportAllocs()
	for b.Loop() {
		sheet.GetRawSprite(indexes.Grass)
	}
}

func TestAnimatedSpriteReusesResolvedArtworkAcrossPositions(t *testing.T) {
	sheet := smallResolverSpriteSheet(t)
	for _, enabled := range []bool{true, false, true} {
		sheet.TileEffectsEnabled = enabled
		for _, sprite := range []indexes.SpriteIndex{indexes.Grass, indexes.Orc1, indexes.Water1, indexes.NoSprites} {
			for _, hash := range []int32{-14331, -1, 0, 1, 247, 4096} {
				for _, elapsed := range []int64{0, 81, 82, 99, 100, 999, 1000, 10000} {
					var want *ebiten.Image
					if enabled && sheet.tileEffectProcessor.HasEffect(sprite) {
						processor := sheet.tileEffectProcessor
						_, frame := processor.frameIndices(sprite, hash, elapsed)
						want = processor.borderedFrames[sprite][frame]
					} else {
						resolved := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, hash, elapsed)
						want = sheet.GetSprite(resolved)
					}
					require.Same(t, want, sheet.GetAnimatedSprite(sprite, hash, elapsed),
						"effects=%v sprite=%d hash=%d elapsed=%d", enabled, sprite, hash, elapsed)
				}
			}
		}
	}
}

// Seed only water's real cache layout. The production resolvers still select
// both raw and bordered frames; identity tests need no registry-wide generation.
func smallResolverSpriteSheet(t *testing.T) *SpriteSheet {
	t.Helper()
	atlas := ebiten.NewImage(origTilesWidth, origTilesHeight)
	atlas.Fill(image.White)
	sheet := &SpriteSheet{SpriteImage: atlas, TileEffectsEnabled: true}
	processor := NewTileEffectProcessor(sheet)
	sheet.tileEffectProcessor = processor
	pipeline := TileEffectRegistry[indexes.Water1]
	count := processor.getFrameCountForPipeline(pipeline)
	processor.effectFrames[indexes.Water1] = make([]*ebiten.Image, count)
	processor.borderedFrames[indexes.Water1] = make([]*ebiten.Image, count*2)
	for frame := range count {
		processor.effectFrames[indexes.Water1][frame] = ebiten.NewImage(TileSize, TileSize)
		processor.borderedFrames[indexes.Water1][frame] = ebiten.NewImage(ExpandedTileSize, ExpandedTileSize)
		processor.borderedFrames[indexes.Water1][count+frame] = ebiten.NewImage(ExpandedTileSize, ExpandedTileSize)
	}
	processor.initialized = true
	t.Cleanup(func() {
		owned := make(map[*ebiten.Image]bool)
		owned[atlas] = true
		for _, cache := range [][1024]*ebiten.Image{sheet.staticBorderedCache, sheet.spriteImageCache} {
			for _, sprite := range cache {
				owned[sprite] = true
			}
		}
		for _, sprite := range sheet.rawSpriteCache {
			owned[sprite] = true
		}
		owned[sheet.transparentRaw], owned[sheet.transparentBordered] = true, true
		for _, cache := range []map[indexes.SpriteIndex][]*ebiten.Image{processor.effectFrames, processor.borderedFrames} {
			for _, frames := range cache {
				for _, sprite := range frames {
					owned[sprite] = true
				}
			}
		}
		for sprite := range owned {
			if sprite != nil {
				sprite.Dispose()
			}
		}
	})
	return sheet
}

func TestTransparentFallbacksAreBorrowedAndSizeSpecific(t *testing.T) {
	sheet := &SpriteSheet{}
	bordered, raw := sheet.GetSprite(indexes.NoSprites), sheet.GetRawSprite(indexes.NoSprites)
	t.Cleanup(bordered.Dispose)
	t.Cleanup(raw.Dispose)
	require.NotSame(t, bordered, raw)
	for _, invalid := range []indexes.SpriteIndex{-1, 512, 1024, 32000} {
		require.Same(t, bordered, sheet.GetSprite(invalid))
		require.Same(t, raw, sheet.GetRawSprite(invalid))
	}
	for sprite, size := range map[*ebiten.Image]int{bordered: ExpandedTileSize, raw: TileSize} {
		require.Equal(t, image.Rect(0, 0, size, size), sprite.Bounds())
		pixels := make([]byte, size*size*4)
		sprite.ReadPixels(pixels)
		require.Equal(t, make([]byte, len(pixels)), pixels)
		editable := ebiten.NewImageFromImage(sprite)
		editable.Fill(image.White)
		editable.Dispose()
		sprite.ReadPixels(pixels)
		require.Equal(t, make([]byte, len(pixels)), pixels, "owned copies cannot mutate the fallback")
	}
}

func BenchmarkAnimatedSpriteResolvedLookup(b *testing.B) {
	atlas := ebiten.NewImage(origTilesWidth, origTilesHeight)
	b.Cleanup(atlas.Dispose)
	sheet := &SpriteSheet{SpriteImage: atlas}
	for hash := range int32(4) {
		sheet.GetAnimatedSprite(indexes.Orc1, hash, 0)
	}
	b.Cleanup(func() {
		for _, sprite := range sheet.staticBorderedCache {
			if sprite != nil {
				sprite.Dispose()
			}
		}
	})
	b.ReportAllocs()
	var hash int32
	for b.Loop() {
		sheet.GetAnimatedSprite(indexes.Orc1, hash, 1234)
		hash++
	}
}
