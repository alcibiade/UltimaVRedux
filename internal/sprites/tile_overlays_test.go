package sprites

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestOverlayPipeline_IsEmpty(t *testing.T) {
	t.Run("empty pipeline", func(t *testing.T) {
		pipeline := OverlayPipeline{}
		assert.True(t, pipeline.IsEmpty())
	})

	t.Run("pipeline with effects", func(t *testing.T) {
		pipeline := Overlay().
			WithAdditiveAlphaPulse(color.RGBA{255, 255, 255, 255}, 1500, 0.3, 0.6).
			Build()
		assert.False(t, pipeline.IsEmpty())
	})
}

func TestOverlayBuilder_WithAdditiveAlphaPulse(t *testing.T) {
	glowColor := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	periodMs := int64(2000)
	pulseLow := float32(0.2)
	pulseHigh := float32(0.8)

	pipeline := Overlay().
		WithAdditiveAlphaPulse(glowColor, periodMs, pulseLow, pulseHigh).
		Build()

	require.Len(t, pipeline.Effects, 1)
	effect := pipeline.Effects[0]

	assert.Equal(t, OverlayAdditiveAlphaPulse, effect.Algorithm)
	assert.Equal(t, glowColor, effect.Color)
	assert.Equal(t, periodMs, effect.PeriodMs)
	assert.Equal(t, pulseLow, effect.PulseLow)
	assert.Equal(t, pulseHigh, effect.PulseHigh)
}

func TestOverlayBuilder_MultipleEffects(t *testing.T) {
	pipeline := Overlay().
		WithAdditiveAlphaPulse(color.RGBA{255, 0, 0, 255}, 1000, 0.1, 0.5).
		WithAdditiveAlphaPulse(color.RGBA{0, 255, 0, 255}, 2000, 0.2, 0.6).
		Build()

	assert.Len(t, pipeline.Effects, 2)
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, pipeline.Effects[0].Color)
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, pipeline.Effects[1].Color)
}

func TestGetGlowTexture(t *testing.T) {
	// Reset the global texture for testing
	glowTexture = nil

	texture := getGlowTexture()
	require.NotNil(t, texture, "Glow texture should be created")

	bounds := texture.Bounds()
	assert.Equal(t, TileSize, bounds.Dx(), "Texture width should be TileSize")
	assert.Equal(t, TileSize, bounds.Dy(), "Texture height should be TileSize")

	// Second call should return same texture
	texture2 := getGlowTexture()
	assert.Same(t, texture, texture2, "Should return cached texture")
}

func TestGetSpriteMask(t *testing.T) {
	// Note: Ebitengine's ReadPixels cannot be called before game starts,
	// so we can only test the caching logic and nil handling here.
	// Full mask generation is tested via integration tests during gameplay.

	t.Run("nil sprite returns nil", func(t *testing.T) {
		mask := GetSpriteMask(nil)
		assert.Nil(t, mask)
	})
}

// Note: TestGenerateSpriteMask and full mask tests require Ebitengine game loop.
// These are tested via visual inspection during gameplay or integration tests.

func TestGetNPCGlowOverlay(t *testing.T) {
	tests := []struct {
		name        string
		spriteIndex indexes.SpriteIndex
		wantOverlay bool
		wantType    string
	}{
		{
			name:        "Wisp1 returns wisp overlay",
			spriteIndex: indexes.Wisp1,
			wantOverlay: true,
			wantType:    "wisp",
		},
		{
			name:        "Wisp4 (last animation frame) returns wisp overlay",
			spriteIndex: indexes.Wisp1 + 3,
			wantOverlay: true,
			wantType:    "wisp",
		},
		{
			name:        "Daemon1 returns daemon overlay",
			spriteIndex: indexes.Daemon1,
			wantOverlay: true,
			wantType:    "daemon",
		},
		{
			name:        "Ghost1 returns ghost overlay",
			spriteIndex: indexes.Ghost1,
			wantOverlay: true,
			wantType:    "ghost",
		},
		{
			name:        "Seahorse1 returns seahorse overlay",
			spriteIndex: indexes.Seahorse1,
			wantOverlay: true,
			wantType:    "seahorse",
		},
		{
			name:        "Avatar returns nil (not light-emitting)",
			spriteIndex: indexes.Avatar1,
			wantOverlay: false,
		},
		{
			name:        "Grass returns nil",
			spriteIndex: indexes.Grass,
			wantOverlay: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay := GetNPCGlowOverlay(tt.spriteIndex)

			if tt.wantOverlay {
				require.NotNil(t, overlay, "Expected overlay for %s", tt.name)
				assert.False(t, overlay.IsEmpty(), "Overlay should not be empty")
				assert.Len(t, overlay.Effects, 1, "Overlay should have exactly one effect")
			} else {
				assert.Nil(t, overlay, "Expected nil overlay for %s", tt.name)
			}
		})
	}
}

func TestNPCGlowOverlayPresets(t *testing.T) {
	// Verify preset overlays have expected timing values

	t.Run("wispGlowOverlay has fast period", func(t *testing.T) {
		require.Len(t, wispGlowOverlay.Effects, 1)
		assert.Equal(t, int64(WispGlowPeriodMs), wispGlowOverlay.Effects[0].PeriodMs)
	})

	t.Run("daemonGlowOverlay has fire flicker period", func(t *testing.T) {
		require.Len(t, daemonGlowOverlay.Effects, 1)
		assert.Equal(t, int64(DaemonGlowPeriodMs), daemonGlowOverlay.Effects[0].PeriodMs)
	})

	t.Run("ghostGlowOverlay has slow period", func(t *testing.T) {
		require.Len(t, ghostGlowOverlay.Effects, 1)
		assert.Equal(t, int64(GhostGlowPeriodMs), ghostGlowOverlay.Effects[0].PeriodMs)
	})

	t.Run("seahorseGlowOverlay has gentle period", func(t *testing.T) {
		require.Len(t, seahorseGlowOverlay.Effects, 1)
		assert.Equal(t, int64(SeahorseGlowPeriodMs), seahorseGlowOverlay.Effects[0].PeriodMs)
	})
}

func TestQuestItemGlowOverlays(t *testing.T) {
	t.Run("GetShardGlowOverlay returns shard overlay with separated effects", func(t *testing.T) {
		overlay := GetShardGlowOverlay()
		require.NotNil(t, overlay)
		assert.False(t, overlay.IsEmpty())
		// Quest item overlays now use separated effects: static glow + pulsing alpha
		require.Len(t, overlay.Effects, 2)
		assert.Equal(t, OverlayAdditiveColour, overlay.Effects[0].Algorithm, "First effect should be additive colour")
		assert.Equal(t, OverlayPulsingAlpha, overlay.Effects[1].Algorithm, "Second effect should be pulsing alpha")
		assert.Equal(t, int64(ShardGlowPeriodMs), overlay.Effects[1].PeriodMs, "Shard should have slow ominous period")
	})

	t.Run("GetArtifactGlowOverlay returns artifact overlay with separated effects", func(t *testing.T) {
		overlay := GetArtifactGlowOverlay()
		require.NotNil(t, overlay)
		assert.False(t, overlay.IsEmpty())
		// Quest item overlays now use separated effects: static glow + pulsing alpha
		require.Len(t, overlay.Effects, 2)
		assert.Equal(t, OverlayAdditiveColour, overlay.Effects[0].Algorithm, "First effect should be additive colour")
		assert.Equal(t, OverlayPulsingAlpha, overlay.Effects[1].Algorithm, "Second effect should be pulsing alpha")
		assert.Equal(t, int64(ArtifactGlowPeriodMs), overlay.Effects[1].PeriodMs, "Artifact should have slow majestic period")
	})

	t.Run("Shard and Artifact have different glow colors", func(t *testing.T) {
		shardOverlay := GetShardGlowOverlay()
		artifactOverlay := GetArtifactGlowOverlay()

		// Static glow is the first effect
		shardColor := shardOverlay.Effects[0].Color
		artifactColor := artifactOverlay.Effects[0].Color

		// Verify they have different color profiles
		assert.NotEqual(t, shardColor, artifactColor, "Shard and Artifact should have different glow colors")
	})

	t.Run("GetPulsingAlpha returns correct oscillating values", func(t *testing.T) {
		overlay := GetShardGlowOverlay()

		// Test at different points in the cycle
		// Shard: PeriodMs=2500, PulseLow=0.60, PulseHigh=1.0
		// At time 0, phase is 0, sin(0)=0, so alpha should be center (0.80)
		alpha0 := overlay.GetPulsingAlpha(0)
		assert.InDelta(t, 0.80, float64(alpha0), 0.01, "At phase 0, alpha should be at center")

		// At time PeriodMs/4, phase is π/2, sin(π/2)=1, so alpha should be at max (1.0)
		alpha625 := overlay.GetPulsingAlpha(625) // 2500/4 = 625
		assert.InDelta(t, 1.0, float64(alpha625), 0.01, "At phase π/2, alpha should be at max")

		// At time PeriodMs*3/4, phase is 3π/2, sin(3π/2)=-1, so alpha should be at min (0.60)
		alpha1875 := overlay.GetPulsingAlpha(1875) // 2500*3/4 = 1875
		assert.InDelta(t, 0.60, float64(alpha1875), 0.01, "At phase 3π/2, alpha should be at min")
	})
}
