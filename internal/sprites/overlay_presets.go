// Package sprites provides preset overlay pipelines for common visual configurations.
package sprites

import (
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ============================================================================
// Moongate Overlay Constants
// ============================================================================

// Moongate additive-layer alpha timing; RGB contribution remains constant.
const (
	MoongateGlowPeriodMs  = 1500 // Full pulse cycle duration in milliseconds
	MoongateGlowPulseLow  = 0.30 // Minimum alpha (30%)
	MoongateGlowPulseHigh = 0.60 // Maximum alpha (60%)
)

// ============================================================================
// NPC Overlay Constants
// Timing values sourced from LightSourcePropertiesMap in light_source_tinting.go
// ============================================================================

// Creature additive-layer phase and alpha settings retain the existing presentation.
const (
	// WispGlowPeriodMs is the wisp's cosmetic alpha cycle.
	WispGlowPeriodMs  = 1200 // Fast flicker for energy beings
	WispGlowPulseLow  = 0.25 // Minimum alpha
	WispGlowPulseHigh = 0.55 // Maximum alpha

	// DaemonGlowPeriodMs is the daemon's cosmetic alpha cycle.
	DaemonGlowPeriodMs  = 1400 // Fire flicker
	DaemonGlowPulseLow  = 0.20 // Minimum alpha
	DaemonGlowPulseHigh = 0.70 // Maximum alpha (very visible)

	// GhostGlowPeriodMs is the ghost's cosmetic alpha cycle.
	GhostGlowPeriodMs  = 2000 // Slow spectral waver
	GhostGlowPulseLow  = 0.15 // Minimum alpha (subtle)
	GhostGlowPulseHigh = 0.35 // Maximum alpha

	// SeahorseGlowPeriodMs is the seahorse's cosmetic alpha cycle.
	SeahorseGlowPeriodMs  = 1800 // Gentle magical pulse
	SeahorseGlowPulseLow  = 0.20 // Minimum alpha
	SeahorseGlowPulseHigh = 0.45 // Maximum alpha
)

// ============================================================================
// Quest Item Overlay Constants
// ============================================================================

// Quest-item base opacity pulses independently of their constant additive colour.
const (
	// ShardGlowPeriodMs controls the shard's base-sprite opacity cycle.
	ShardGlowPeriodMs  = 2500 // Slow ominous pulse
	ShardGlowPulseLow  = 0.60 // Minimum alpha (sprite fades to 60%)
	ShardGlowPulseHigh = 1.0  // Maximum alpha (sprite fully visible)
	ShardGlowAlpha     = 0.35 // Static glow intensity

	// ArtifactGlowPeriodMs controls the artifact's base-sprite opacity cycle.
	ArtifactGlowPeriodMs  = 2000 // Slow majestic pulse
	ArtifactGlowPulseLow  = 0.80 // Minimum alpha (sprite fades to 70%)
	ArtifactGlowPulseHigh = 1.0  // Maximum alpha (sprite fully visible)
	ArtifactGlowAlpha     = 0.45 // Static glow intensity
)

// ============================================================================
// Combat Outline Pulse Constants
// ============================================================================

const (
	// ActivePCOutlinePulsePeriodMs is the slower green border pulse for the acting PC.
	ActivePCOutlinePulsePeriodMs = 1200
	// ActivePCOutlinePulseLow is the dimmest border intensity.
	ActivePCOutlinePulseLow = 0.67
	// ActivePCOutlinePulseHigh is the brightest border intensity.
	ActivePCOutlinePulseHigh = 1.0
	// ActivePCOutlinePulseSteps bounds dynamic outline cache entries per sprite.
	ActivePCOutlinePulseSteps = 8

	// ActiveMonsterOutlinePulsePeriodMs is the faster yellow border pulse for an acting monster.
	ActiveMonsterOutlinePulsePeriodMs = 800
	// ActiveMonsterOutlinePulseLow is the dimmest acting-monster border intensity.
	ActiveMonsterOutlinePulseLow = 0.67
	// ActiveMonsterOutlinePulseHigh is the brightest acting-monster border intensity.
	ActiveMonsterOutlinePulseHigh = 1.0
	// ActiveMonsterOutlinePulseSteps bounds dynamic outline cache entries per sprite.
	ActiveMonsterOutlinePulseSteps = 8
)

// ============================================================================
// Preset Overlay Pipelines
// ============================================================================

// Preset overlay pipelines for common visual configurations.
// These provide consistent appearance and can be reused across different entities.
//
//nolint:gochecknoglobals // Immutable game palette/timing presets initialized once and shared by renderers.
var (
	// MoongateGlowOverlay - pulsing blue glow for active moongates.
	// Alpha cycles between 30% and 60%; the added RGB does not pulse under BlendLighter.
	// The moongate's base sprite has its own independent pulse scale.
	MoongateGlowOverlay = Overlay().
				WithAdditiveAlphaPulse(pkgcolor.MoongateGlow.RGBA(),
			MoongateGlowPeriodMs, MoongateGlowPulseLow, MoongateGlowPulseHigh).
		Build()

	// ============================================================================
	// NPC Glow Overlays (internal - accessed via GetNPCGlowOverlay)
	// ============================================================================

	// wispGlowOverlay - fast flickering ethereal blue-white glow for energy beings.
	wispGlowOverlay = Overlay().
			WithAdditiveAlphaPulse(pkgcolor.WispGlow.RGBA(), WispGlowPeriodMs, WispGlowPulseLow, WispGlowPulseHigh).
			Build()

	// daemonGlowOverlay - flickering red demonic glow for fire creatures.
	daemonGlowOverlay = Overlay().
				WithAdditiveAlphaPulse(pkgcolor.DaemonGlow.RGBA(), DaemonGlowPeriodMs, DaemonGlowPulseLow, DaemonGlowPulseHigh).
				Build()

	// ghostGlowOverlay - slow wavering pale spectral glow for undead spirits.
	ghostGlowOverlay = Overlay().
				WithAdditiveAlphaPulse(pkgcolor.GhostGlow.RGBA(), GhostGlowPeriodMs, GhostGlowPulseLow, GhostGlowPulseHigh).
				Build()

	// seahorseGlowOverlay - gentle pulsing magical glow for mystical sea creatures.
	seahorseGlowOverlay = Overlay().
				WithAdditiveAlphaPulse(pkgcolor.SeahorseGlow.RGBA(),
			SeahorseGlowPeriodMs, SeahorseGlowPulseLow, SeahorseGlowPulseHigh).
		Build()

	// ============================================================================
	// Quest Item Glow Overlays (internal - accessed via GetShardGlowOverlay/GetArtifactGlowOverlay)
	// These use separated effects: static glow (additive) + pulsing alpha (base sprite)
	// ============================================================================

	// shardGlowOverlay - slow ominous red-purple glow for corrupted Shards of Evil.
	// Uses static additive glow for constant light emission, plus pulsing alpha
	// applied to the base sprite for visibility modulation.
	shardGlowOverlay = Overlay().
				WithAdditiveColour(pkgcolor.ShardGlow.RGBA(), ShardGlowAlpha).
				WithPulsingAlpha(ShardGlowPeriodMs, ShardGlowPulseLow, ShardGlowPulseHigh).
				Build()

	// artifactGlowOverlay - slow majestic golden-white glow for Lord British's Artifacts.
	// Uses static additive glow for constant light emission, plus pulsing alpha
	// applied to the base sprite for visibility modulation.
	artifactGlowOverlay = Overlay().
				WithAdditiveColour(pkgcolor.ArtifactGlow.RGBA(), ArtifactGlowAlpha).
				WithPulsingAlpha(ArtifactGlowPeriodMs, ArtifactGlowPulseLow, ArtifactGlowPulseHigh).
				Build()
)

// ============================================================================
// NPC Overlay Lookup
// ============================================================================

// Sprite range constants for light-emitting creatures (4 animation frames each).
const (
	wispStartSprite     = indexes.Wisp1
	wispEndSprite       = indexes.Wisp1 + 3
	seahorseStartSprite = indexes.Seahorse1
	seahorseEndSprite   = indexes.Seahorse1 + 3
	daemonStartSprite   = indexes.Daemon1
	daemonEndSprite     = indexes.Daemon1 + 3
	ghostStartSprite    = indexes.Ghost1
	ghostEndSprite      = indexes.Ghost1 + 3
)

// GetNPCGlowOverlay returns the appropriate glow overlay for a light-emitting NPC sprite.
// Returns nil if the sprite does not have a glow effect.
func GetNPCGlowOverlay(spriteIndex indexes.SpriteIndex) *OverlayPipeline {
	switch {
	case spriteIndex >= wispStartSprite && spriteIndex <= wispEndSprite:
		return &wispGlowOverlay
	case spriteIndex >= seahorseStartSprite && spriteIndex <= seahorseEndSprite:
		return &seahorseGlowOverlay
	case spriteIndex >= daemonStartSprite && spriteIndex <= daemonEndSprite:
		return &daemonGlowOverlay
	case spriteIndex >= ghostStartSprite && spriteIndex <= ghostEndSprite:
		return &ghostGlowOverlay
	default:
		return nil
	}
}

// GetShardGlowOverlay returns the glow overlay for corrupted Shards of Evil.
func GetShardGlowOverlay() *OverlayPipeline {
	return &shardGlowOverlay
}

// GetArtifactGlowOverlay returns the glow overlay for Lord British's Artifacts.
func GetArtifactGlowOverlay() *OverlayPipeline {
	return &artifactGlowOverlay
}
