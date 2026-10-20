package sprites

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// ============================================================================
// BUILDER TESTS
// ============================================================================

func TestEffect_BuilderCreatesEmptyPipeline(t *testing.T) {
	builder := Effect()
	require.NotNil(t, builder)
	assert.Equal(t, indexes.SpriteIndex(-1), builder.pipeline.Background)
	assert.Empty(t, builder.pipeline.Effects)
	assert.True(t, builder.pipeline.Bounds.Empty())
	assert.Equal(t, 0, builder.pipeline.IntervalMs)
}

func TestEffect_WithBackground(t *testing.T) {
	pipeline := Effect().
		WithBackground(indexes.BrickFloor).
		AddVerticalRise(FilterAll).
		Build()

	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), pipeline.Background)
}

func TestEffect_WithBounds_InclusiveCoordinates(t *testing.T) {
	// WithBounds(6, 1, 12, 4) should affect pixels 6-12 on x, 1-4 on y (inclusive)
	pipeline := Effect().
		WithBounds(6, 1, 12, 4).
		AddVerticalRise(FilterAll).
		Build()

	// Internally converted to exclusive Rectangle
	expected := image.Rect(6, 1, 13, 5)
	assert.Equal(t, expected, pipeline.Bounds)

	// Verify the bounds contain the expected inclusive pixels
	assert.True(t, image.Pt(6, 1).In(pipeline.Bounds), "should contain top-left")
	assert.True(t, image.Pt(12, 4).In(pipeline.Bounds), "should contain bottom-right")
	assert.False(t, image.Pt(5, 1).In(pipeline.Bounds), "should not contain x=5")
	assert.False(t, image.Pt(13, 4).In(pipeline.Bounds), "should not contain x=13")
}

func TestEffect_WithInterval(t *testing.T) {
	pipeline := Effect().
		WithInterval(100).
		AddVerticalRise(FilterAll).
		Build()

	assert.Equal(t, 100, pipeline.IntervalMs)
}

func TestEffect_AddVerticalRise(t *testing.T) {
	pipeline := Effect().
		AddVerticalRise(FilterRedOnly).
		Build()

	require.Len(t, pipeline.Effects, 1)
	assert.Equal(t, EffectVerticalRise, pipeline.Effects[0].Algorithm)
	assert.Equal(t, FilterRedOnly, pipeline.Effects[0].Filter)
}

func TestEffect_AddWaveDisplace(t *testing.T) {
	pipeline := Effect().
		AddWaveDisplace(FilterAll).
		Build()

	require.Len(t, pipeline.Effects, 1)
	assert.Equal(t, EffectWaveDisplace, pipeline.Effects[0].Algorithm)
	assert.Equal(t, FilterAll, pipeline.Effects[0].Filter)
}

func TestEffect_StackedEffects_PreservesOrder(t *testing.T) {
	pipeline := Effect().
		AddVerticalRise(FilterRedOnly).
		AddWaveDisplace(FilterAll).
		Build()

	require.Len(t, pipeline.Effects, 2)
	assert.Equal(t, EffectVerticalRise, pipeline.Effects[0].Algorithm)
	assert.Equal(t, EffectWaveDisplace, pipeline.Effects[1].Algorithm)
}

func TestEffect_FluentChaining(t *testing.T) {
	// Test that all methods return the builder for chaining
	pipeline := Effect().
		WithBackground(indexes.BrickFloor).
		WithBounds(0, 0, 15, 15).
		WithInterval(200).
		AddVerticalRise(FilterRedOnly).
		AddWaveDisplace(FilterYellowOnly).
		Build()

	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), pipeline.Background)
	assert.Equal(t, image.Rect(0, 0, 16, 16), pipeline.Bounds)
	assert.Equal(t, 200, pipeline.IntervalMs)
	assert.Len(t, pipeline.Effects, 2)
}

// ============================================================================
// BUILDER VALIDATION TESTS
// ============================================================================

func TestEffect_Build_PanicsWithNoEffects(t *testing.T) {
	assert.Panics(t, func() {
		Effect().Build()
	}, "Build() should panic when no effects are added")
}

func TestEffect_Build_PanicsWithNegativeBounds(t *testing.T) {
	assert.Panics(t, func() {
		Effect().
			WithBounds(-1, 0, 10, 10).
			AddVerticalRise(FilterAll).
			Build()
	}, "Build() should panic with negative x1")

	assert.Panics(t, func() {
		Effect().
			WithBounds(0, -1, 10, 10).
			AddVerticalRise(FilterAll).
			Build()
	}, "Build() should panic with negative y1")
}

func TestEffect_Build_PanicsWithBoundsExceedingTileSize(t *testing.T) {
	assert.Panics(t, func() {
		Effect().
			WithBounds(0, 0, 16, 15). // x2=16 -> max.X=17 > TileSize(16)
			AddVerticalRise(FilterAll).
			Build()
	}, "Build() should panic when bounds exceed tile size")

	assert.Panics(t, func() {
		Effect().
			WithBounds(0, 0, 15, 16). // y2=16 -> max.Y=17 > TileSize(16)
			AddVerticalRise(FilterAll).
			Build()
	}, "Build() should panic when bounds exceed tile size")
}

func TestEffect_Build_AcceptsMaxValidBounds(t *testing.T) {
	// WithBounds(0, 0, 15, 15) should be valid (full tile)
	assert.NotPanics(t, func() {
		Effect().
			WithBounds(0, 0, 15, 15).
			AddVerticalRise(FilterAll).
			Build()
	}, "Build() should accept max valid bounds (0,0,15,15)")
}

func TestEffect_Build_AcceptsNoBounds(t *testing.T) {
	// No bounds should default to full tile (empty Rectangle)
	assert.NotPanics(t, func() {
		pipeline := Effect().
			AddVerticalRise(FilterAll).
			Build()
		assert.True(t, pipeline.Bounds.Empty())
	}, "Build() should accept no bounds")
}

// ============================================================================
// REGISTRY TESTS
// ============================================================================

func TestTileEffectRegistry_AllEntriesValid(t *testing.T) {
	// Verify all registry entries built without panicking
	// (they would have panicked at package init if invalid)
	assert.NotEmpty(t, TileEffectRegistry, "Registry should not be empty")

	for spriteIndex, pipeline := range TileEffectRegistry {
		assert.NotEmpty(t, pipeline.Effects,
			"Sprite %d should have at least one effect", spriteIndex)
	}
}

func TestTileEffectRegistry_ContainsExpectedTiles(t *testing.T) {
	expectedTiles := []indexes.SpriteIndex{
		indexes.RightSconce,
		indexes.LeftSconce,
		indexes.Brazier,
		indexes.Fireplace,
		indexes.CandleOnTable,
		indexes.CampFire,
		indexes.FireLogs,
		indexes.PoisonField,
		indexes.MagicField,
		indexes.FireField,
		indexes.ElectricField,
		indexes.ItemTorch,
		indexes.Lava,
	}

	for _, tile := range expectedTiles {
		_, exists := TileEffectRegistry[tile]
		assert.True(t, exists, "Registry should contain %v", tile)
	}
}

func TestTileEffectRegistry_CombatFieldsUseDistinctEffects(t *testing.T) {
	tests := []struct {
		sprite    indexes.SpriteIndex
		algorithm EffectAlgorithm
		filter    ColorFilter
		effects   int
	}{
		{indexes.PoisonField, EffectVerticalRise, FilterPoison, 1},
		{indexes.MagicField, EffectVerticalRise, FilterSleep, 2},
		{indexes.FireField, EffectVerticalRise, FilterFire, 1},
		{indexes.ElectricField, EffectBrightnessPulse, FilterProtection, 1},
	}

	for _, test := range tests {
		pipeline, exists := TileEffectRegistry[test.sprite]
		require.True(t, exists, "field sprite %v should be animated", test.sprite)
		require.Len(t, pipeline.Effects, test.effects)
		assert.Equal(t, test.algorithm, pipeline.Effects[0].Algorithm)
		assert.Equal(t, test.filter, pipeline.Effects[0].Filter)
	}
}

func TestApplyPulseChangesOnlyFilteredPixels(t *testing.T) {
	processor := &TileEffectProcessor{}
	input := make([]byte, TileSize*TileSize*4)
	poisonPixel := 0
	input[poisonPixel] = ucolor.EGAGreen.R
	input[poisonPixel+1] = ucolor.EGAGreen.G
	input[poisonPixel+2] = ucolor.EGAGreen.B
	input[poisonPixel+3] = ucolor.EGAGreen.A
	blackPixel := 4
	input[blackPixel] = ucolor.EGABlack.R
	input[blackPixel+1] = ucolor.EGABlack.G
	input[blackPixel+2] = ucolor.EGABlack.B
	input[blackPixel+3] = ucolor.EGABlack.A

	output := processor.applyEffect(input, 0, image.Rect(0, 0, 16, 16),
		TileEffect{Algorithm: EffectPulse, Filter: FilterPoison}, 8)

	assert.Less(t, output[poisonPixel+3], input[poisonPixel+3])
	assert.Equal(t, input[blackPixel:blackPixel+4], output[blackPixel:blackPixel+4])
}

func TestFilterProtectionMatchesLightBlueFieldPixels(t *testing.T) {
	processor := &TileEffectProcessor{}
	pixels := []byte{
		ucolor.EGALightBlue.R,
		ucolor.EGALightBlue.G,
		ucolor.EGALightBlue.B,
		ucolor.EGALightBlue.A,
	}

	assert.True(t, processor.passesFilter(pixels, 0, FilterProtection))
}

func TestApplyBrightnessPulseChangesColorNotOpacity(t *testing.T) {
	processor := &TileEffectProcessor{}
	input := make([]byte, TileSize*TileSize*4)
	copy(input, []byte{
		ucolor.EGALightBlue.R,
		ucolor.EGALightBlue.G,
		ucolor.EGALightBlue.B,
		ucolor.EGALightBlue.A,
	})
	output := processor.applyEffect(input, 0, image.Rect(0, 0, 16, 16),
		TileEffect{Algorithm: EffectBrightnessPulse, Filter: FilterProtection}, 8)

	assert.NotEqual(t, input[:3], output[:3])
	assert.Equal(t, input[3], output[3])
}

func TestSleepFieldKeepsIndependentAlphaAppearance(t *testing.T) {
	processor := &TileEffectProcessor{}
	input := make([]byte, TileSize*TileSize*4)
	copy(input, []byte{170, 0, 170, 128, 0, 0, 0, 255})
	// This is the authored sleep palette, including a partially transparent pixel.
	// The retained appearance changes alpha alone, not conventional RGBA opacity.
	for _, sample := range []struct {
		frame int
		alpha byte
	}{{0, 70}, {2, 121}, {6, 19}} {
		output := processor.applyEffect(input, sample.frame, image.Rectangle{},
			TileEffect{Algorithm: EffectPulse, Filter: FilterSleep}, 8)
		require.Equal(t, []byte{170, 0, 170, sample.alpha, 0, 0, 0, 255}, output[:8])
	}
	require.Equal(t, []byte{170, 0, 170, 128, 0, 0, 0, 255}, input[:8])
}

func TestFlameAdapterBoundsSelectDestinationsNotSampleColours(t *testing.T) {
	processor := &TileEffectProcessor{}
	input := make([]byte, TileSize*TileSize*4)
	put := func(x, y int, rgba []byte) { copy(input[(y*TileSize+x)*4:], rgba) }
	red, green := []byte{170, 0, 0, 255}, []byte{0, 170, 0, 255}
	put(5, 1, red) // Outside the right sconce's authored bounds.
	put(6, 1, red) // Frame zero samples (7,2), even though its colour is not selected.
	put(7, 1, green)
	put(7, 2, green)
	pipeline := TileEffectRegistry[indexes.RightSconce]
	output := processor.applyEffect(input, 0, pipeline.Bounds, pipeline.Effects[0], 8)
	require.Equal(t, red, output[(TileSize+5)*4:][:4])
	require.Equal(t, green, output[(TileSize+6)*4:][:4])
	require.Equal(t, green, output[(TileSize+7)*4:][:4])
	require.Equal(t, red, input[(TileSize+6)*4:][:4], "source pixels remain immutable")
}

func TestTileEffectRegistry_FlamesTilesUseVerticalRise(t *testing.T) {
	flameTiles := []indexes.SpriteIndex{
		indexes.RightSconce,
		indexes.LeftSconce,
		indexes.Brazier,
		indexes.Fireplace,
		indexes.CampFire,
		indexes.FireLogs,
		indexes.FireField,
		indexes.ItemTorch,
	}

	for _, tile := range flameTiles {
		pipeline, exists := TileEffectRegistry[tile]
		require.True(t, exists, "Registry should contain %v", tile)
		require.NotEmpty(t, pipeline.Effects, "Pipeline should have effects for %v", tile)
		assert.Equal(t, EffectVerticalRise, pipeline.Effects[0].Algorithm,
			"Flame tile %v should use VerticalRise effect", tile)
	}
}

func TestTileEffectRegistry_LavaUsesWaveDisplace(t *testing.T) {
	pipeline, exists := TileEffectRegistry[indexes.Lava]
	require.True(t, exists, "Registry should contain Lava")
	require.NotEmpty(t, pipeline.Effects)

	// Lava uses VerticalScroll + WaveDisplace pipeline
	// Find the WaveDisplace effect (may not be first in pipeline)
	var foundWaveDisplace bool
	for _, effect := range pipeline.Effects {
		if effect.Algorithm == EffectWaveDisplace {
			foundWaveDisplace = true
			assert.Equal(t, FilterAll, effect.Filter)
			break
		}
	}
	assert.True(t, foundWaveDisplace, "Lava should have WaveDisplace effect in its pipeline")
}

// ============================================================================
// FRAME TIMING AND SELECTION TESTS
// ============================================================================

func TestTileEffectRegistry_WallMountedFlamesHaveBrickBackground(t *testing.T) {
	wallMounted := []indexes.SpriteIndex{
		indexes.RightSconce,
		indexes.LeftSconce,
		indexes.Fireplace,
		indexes.CandleOnTable,
	}

	for _, tile := range wallMounted {
		pipeline := TileEffectRegistry[tile]
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), pipeline.Background,
			"Wall-mounted flame %v should have BrickFloor background", tile)
	}
}

func TestTileEffectRegistry_OutdoorFlamesHaveGrassBackground(t *testing.T) {
	outdoor := []indexes.SpriteIndex{
		indexes.CampFire,
		indexes.LampPost,
		indexes.CookStove,
	}

	for _, tile := range outdoor {
		pipeline := TileEffectRegistry[tile]
		assert.Equal(t, indexes.SpriteIndex(indexes.Grass), pipeline.Background,
			"Outdoor flame %v should have Grass background", tile)
	}
}

func TestTileEffectRegistry_OverlayFlamesHaveNoBackground(t *testing.T) {
	overlays := []indexes.SpriteIndex{
		indexes.FireField,
		indexes.ItemTorch,
	}

	for _, tile := range overlays {
		pipeline := TileEffectRegistry[tile]
		assert.Equal(t, indexes.SpriteIndex(-1), pipeline.Background,
			"Overlay flame %v should have no background", tile)
	}
}

// ============================================================================
// COLOR FILTER TESTS
// ============================================================================

func TestIsEGARed_ExactMatch(t *testing.T) {
	assert.True(t, isEGARed(170, 0, 0))
}

func TestIsEGARed_WithinTolerance(t *testing.T) {
	assert.True(t, isEGARed(175, 5, 5))   // +5 on each channel
	assert.True(t, isEGARed(165, 0, 0))   // -5 on R
	assert.True(t, isEGARed(170, 10, 10)) // at tolerance limit
}

func TestIsEGARed_OutsideTolerance(t *testing.T) {
	assert.False(t, isEGARed(181, 0, 0))  // R too high (diff = 11)
	assert.False(t, isEGARed(158, 0, 0))  // R too low (diff = 12)
	assert.False(t, isEGARed(170, 12, 0)) // G too high (diff = 12)
	assert.False(t, isEGARed(170, 0, 12)) // B too high (diff = 12)
}

func TestIsEGARed_NotRed(t *testing.T) {
	assert.False(t, isEGARed(255, 255, 85))  // Yellow
	assert.False(t, isEGARed(0, 0, 0))       // Black
	assert.False(t, isEGARed(255, 255, 255)) // White
}

func TestIsEGAYellow_ExactMatch(t *testing.T) {
	assert.True(t, isEGAYellow(255, 255, 85))
}

func TestIsEGAYellow_WithinTolerance(t *testing.T) {
	assert.True(t, isEGAYellow(250, 250, 80)) // -5 on each
	assert.True(t, isEGAYellow(255, 255, 90)) // +5 on B
}

func TestIsEGAYellow_OutsideTolerance(t *testing.T) {
	assert.False(t, isEGAYellow(244, 255, 85)) // R too low
	assert.False(t, isEGAYellow(255, 244, 85)) // G too low
	assert.False(t, isEGAYellow(255, 255, 74)) // B too low
}

func TestIsEGAYellow_NotYellow(t *testing.T) {
	assert.False(t, isEGAYellow(170, 0, 0))     // Red
	assert.False(t, isEGAYellow(0, 0, 0))       // Black
	assert.False(t, isEGAYellow(255, 255, 255)) // White
}

// ============================================================================
// HELPER FUNCTION TESTS
// ============================================================================

// ============================================================================
// PROCESSOR TESTS (without full sprite sheet - limited scope)
// ============================================================================

func TestNewTileEffectProcessor(t *testing.T) {
	// Can't fully test without sprite sheet, but verify creation
	processor := NewTileEffectProcessor(nil)
	require.NotNil(t, processor)
	assert.False(t, processor.initialized)
	assert.NotNil(t, processor.effectFrames)
}

func TestTileEffectProcessor_HasEffect_BeforeInit(t *testing.T) {
	// HasEffect checks the registry directly (no lazy init needed)
	// This allows it to be called before Ebitengine is running
	processor := &TileEffectProcessor{
		effectFrames: make(map[indexes.SpriteIndex][]*ebiten.Image),
		initialized:  false,
	}

	// Grass is not in registry
	assert.False(t, processor.HasEffect(indexes.Grass))

	// RightSconce IS in registry (defined in tile_effects_registry.go)
	assert.True(t, processor.HasEffect(indexes.RightSconce))

	// Verify initialization was NOT triggered (deferred until GetEffectSprite is called)
	assert.False(t, processor.initialized)
}

// ============================================================================
// NEW ALGORITHM TESTS (Phase 1)
// ============================================================================

func TestEffectAlgorithm_PreferredFrameCount(t *testing.T) {
	tests := []struct {
		algorithm EffectAlgorithm
		expected  int
		name      string
	}{
		{EffectVerticalRise, 8, "VerticalRise uses 8 frames"},
		{EffectWaveDisplace, 8, "WaveDisplace uses 8 frames"},
		{EffectVerticalScroll, 16, "VerticalScroll uses 16 frames"},
		{EffectTransparencyOverlay, 16, "TransparencyOverlay uses 16 frames"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.algorithm.PreferredFrameCount())
		})
	}
}

func TestEffect_WithEvenXOffset(t *testing.T) {
	pipeline := Effect().
		WithEvenXOffset(6).
		AddVerticalScroll().
		Build()

	assert.Equal(t, PositionOffsetEvenX, pipeline.OffsetMode)
	assert.Equal(t, 6, pipeline.EvenXOffset)
}

func TestEffect_AddVerticalScroll(t *testing.T) {
	pipeline := Effect().
		AddVerticalScroll().
		Build()

	require.Len(t, pipeline.Effects, 1)
	assert.Equal(t, EffectVerticalScroll, pipeline.Effects[0].Algorithm)
	assert.Equal(t, FilterAll, pipeline.Effects[0].Filter)
}

func TestEffect_AddTransparencyOverlay(t *testing.T) {
	pipeline := Effect().
		AddTransparencyOverlay(indexes.WaterCoast).
		Build()

	require.Len(t, pipeline.Effects, 1)
	assert.Equal(t, EffectTransparencyOverlay, pipeline.Effects[0].Algorithm)
	assert.Equal(t, indexes.SpriteIndex(indexes.WaterCoast), pipeline.Effects[0].Background)
}

func TestEffect_WaterStyle_FluentChaining(t *testing.T) {
	// Test building a water-style pipeline with all new features
	pipeline := Effect().
		WithInterval(82).
		WithEvenXOffset(6).
		AddVerticalScroll().
		Build()

	assert.Equal(t, 82, pipeline.IntervalMs)
	assert.Equal(t, PositionOffsetEvenX, pipeline.OffsetMode)
	assert.Equal(t, 6, pipeline.EvenXOffset)
	require.Len(t, pipeline.Effects, 1)
	assert.Equal(t, EffectVerticalScroll, pipeline.Effects[0].Algorithm)
}

func TestEffect_MixedTileStyle_FluentChaining(t *testing.T) {
	// Test building a mixed tile pipeline (transparency overlay + water animation)
	pipeline := Effect().
		WithInterval(82).
		WithEvenXOffset(6).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build()

	assert.Equal(t, 82, pipeline.IntervalMs)
	assert.Equal(t, PositionOffsetEvenX, pipeline.OffsetMode)
	require.Len(t, pipeline.Effects, 1)
	assert.Equal(t, EffectTransparencyOverlay, pipeline.Effects[0].Algorithm)
	assert.Equal(t, indexes.SpriteIndex(indexes.WaterCoast), pipeline.Effects[0].Background)
}

func TestEffect_LavaStyle_CombinedEffects(t *testing.T) {
	// Lava uses VerticalScroll + WaveDisplace combined
	pipeline := Effect().
		WithInterval(82).
		WithEvenXOffset(6).
		AddVerticalScroll().
		AddWaveDisplace(FilterAll).
		Build()

	require.Len(t, pipeline.Effects, 2)
	assert.Equal(t, EffectVerticalScroll, pipeline.Effects[0].Algorithm)
	assert.Equal(t, EffectWaveDisplace, pipeline.Effects[1].Algorithm)
}

func TestTileEffectProcessor_GetFrameCountForPipeline(t *testing.T) {
	processor := NewTileEffectProcessor(nil)

	tests := []struct {
		name     string
		pipeline TileEffectPipeline
		expected int
	}{
		{
			name: "Fire effect uses 8 frames",
			pipeline: Effect().
				AddVerticalRise(FilterRedOnly).
				Build(),
			expected: 8,
		},
		{
			name: "Water effect uses 16 frames",
			pipeline: Effect().
				AddVerticalScroll().
				Build(),
			expected: 16,
		},
		{
			name: "Mixed fire+water uses max (16 frames)",
			pipeline: Effect().
				AddVerticalRise(FilterRedOnly).
				AddVerticalScroll().
				Build(),
			expected: 16,
		},
		{
			name: "Explicit frame count overrides",
			pipeline: TileEffectPipeline{
				FrameCount: 12,
				Effects: []TileEffect{
					{Algorithm: EffectVerticalScroll},
				},
			},
			expected: 12,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.getFrameCountForPipeline(tc.pipeline)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestScrollAdapterUsesPipelineFrameCount(t *testing.T) {
	processor := &TileEffectProcessor{}
	input := make([]byte, TileSize*TileSize*4)
	copy(input, []byte{80, 40, 20, 128})
	// A short authored cycle must wrap before the SDK's image-height cycle.
	output := processor.applyEffect(input, 5, image.Rectangle{}, TileEffect{Algorithm: EffectVerticalScroll}, 4)
	require.Equal(t, []byte{80, 40, 20, 128}, output[TileSize*4:][:4])
	require.Equal(t, []byte{0, 0, 0, 0}, output[5*TileSize*4:][:4])
}
