// Package sprites provides preset overlay pipelines for common visual configurations.
package sprites

import (
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ============================================================================
// Moongate Overlay Constants
// ============================================================================

const (
	MoongateGlowPeriodMs  = 1500 // Full pulse cycle duration in milliseconds
	MoongateGlowPulseLow  = 0.30 // Minimum alpha (30%)
	MoongateGlowPulseHigh = 0.60 // Maximum alpha (60%)
)

// ============================================================================
// NPC Overlay Constants
// Timing values sourced from LightSourcePropertiesMap in light_source_tinting.go
// ============================================================================

const (
	// Wisp - ethereal energy beings with fast flickering blue-white glow
	WispGlowPeriodMs  = 1200 // Fast flicker for energy beings
	WispGlowPulseLow  = 0.25 // Minimum alpha
	WispGlowPulseHigh = 0.55 // Maximum alpha

	// Daemon - demonic fire creatures with flickering red glow
	DaemonGlowPeriodMs  = 1400 // Fire flicker
	DaemonGlowPulseLow  = 0.20 // Minimum alpha
	DaemonGlowPulseHigh = 0.70 // Maximum alpha (very visible)

	// Ghost - spectral beings with slow wavering pale blue glow
	GhostGlowPeriodMs  = 2000 // Slow spectral waver
	GhostGlowPulseLow  = 0.15 // Minimum alpha (subtle)
	GhostGlowPulseHigh = 0.35 // Maximum alpha

	// Seahorse - mystical sea creatures with gentle magical pulse
	SeahorseGlowPeriodMs  = 1800 // Gentle magical pulse
	SeahorseGlowPulseLow  = 0.20 // Minimum alpha
	SeahorseGlowPulseHigh = 0.45 // Maximum alpha
)

// ============================================================================
// Quest Item Overlay Constants
// ============================================================================

const (
	// Shard - corrupted virtue gems with slow ominous pulse
	ShardGlowPeriodMs  = 2500 // Slow ominous pulse
	ShardGlowPulseLow  = 0.60 // Minimum alpha (sprite fades to 60%)
	ShardGlowPulseHigh = 1.0  // Maximum alpha (sprite fully visible)
	ShardGlowAlpha     = 0.35 // Static glow intensity

	// Artifact - Lord British's artifacts with slow majestic pulse
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
var (
	// MoongateGlowOverlay - pulsing blue glow for active moongates.
	// Oscillates between 30% and 60% alpha over 1.5 seconds.
	// Uses additive blending for mystical glow effect.
	MoongateGlowOverlay = Overlay().
				WithPulsingGlow(pkgcolor.MoongateGlow.RGBA(), MoongateGlowPeriodMs, MoongateGlowPulseLow, MoongateGlowPulseHigh).
				Build()

	// ============================================================================
	// NPC Glow Overlays (internal - accessed via GetNPCGlowOverlay)
	// ============================================================================

	// wispGlowOverlay - fast flickering ethereal blue-white glow for energy beings.
	wispGlowOverlay = Overlay().
			WithPulsingGlow(pkgcolor.WispGlow.RGBA(), WispGlowPeriodMs, WispGlowPulseLow, WispGlowPulseHigh).
			Build()

	// daemonGlowOverlay - flickering red demonic glow for fire creatures.
	daemonGlowOverlay = Overlay().
				WithPulsingGlow(pkgcolor.DaemonGlow.RGBA(), DaemonGlowPeriodMs, DaemonGlowPulseLow, DaemonGlowPulseHigh).
				Build()

	// ghostGlowOverlay - slow wavering pale spectral glow for undead spirits.
	ghostGlowOverlay = Overlay().
				WithPulsingGlow(pkgcolor.GhostGlow.RGBA(), GhostGlowPeriodMs, GhostGlowPulseLow, GhostGlowPulseHigh).
				Build()

	// seahorseGlowOverlay - gentle pulsing magical glow for mystical sea creatures.
	seahorseGlowOverlay = Overlay().
				WithPulsingGlow(pkgcolor.SeahorseGlow.RGBA(), SeahorseGlowPeriodMs, SeahorseGlowPulseLow, SeahorseGlowPulseHigh).
				Build()

	// ============================================================================
	// Quest Item Glow Overlays (internal - accessed via GetShardGlowOverlay/GetArtifactGlowOverlay)
	// These use separated effects: static glow (additive) + pulsing alpha (base sprite)
	// ============================================================================

	// shardGlowOverlay - slow ominous red-purple glow for corrupted Shards of Evil.
	// Uses static additive glow for constant light emission, plus pulsing alpha
	// applied to the base sprite for visibility modulation.
	shardGlowOverlay = Overlay().
				WithStaticGlow(pkgcolor.ShardGlow.RGBA(), ShardGlowAlpha).
				WithPulsingAlpha(ShardGlowPeriodMs, ShardGlowPulseLow, ShardGlowPulseHigh).
				Build()

	// artifactGlowOverlay - slow majestic golden-white glow for Lord British's Artifacts.
	// Uses static additive glow for constant light emission, plus pulsing alpha
	// applied to the base sprite for visibility modulation.
	artifactGlowOverlay = Overlay().
				WithStaticGlow(pkgcolor.ArtifactGlow.RGBA(), ArtifactGlowAlpha).
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
