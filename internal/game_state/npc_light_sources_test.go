package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ============================================================================
// IsLightEmittingNPC Tests
// ============================================================================

func TestIsLightEmittingNPC_Seahorse(t *testing.T) {
	// Seahorse sprites are 384-387 (Seahorse1 through Seahorse4)
	// All 4 animation frames should be recognized as light-emitting

	t.Run("Seahorse1", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Seahorse1),
			"Seahorse1 (384) should be light-emitting")
	})

	t.Run("Seahorse2", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Seahorse1+1),
			"Seahorse2 (385) should be light-emitting")
	})

	t.Run("Seahorse3", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Seahorse1+2),
			"Seahorse3 (386) should be light-emitting")
	})

	t.Run("Seahorse4", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Seahorse1+3),
			"Seahorse4 (387) should be light-emitting")
	})

	// Verify sprite index values are what we expect
	t.Run("VerifySeahorseIndexes", func(t *testing.T) {
		assert.Equal(t, 384, indexes.Seahorse1,
			"Seahorse1 should be sprite index 384")
	})
}

func TestIsLightEmittingNPC_Wisp(t *testing.T) {
	// Wisp sprites are 468-471 (Wisp1 through Wisp4)

	t.Run("Wisp1", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Wisp1),
			"Wisp1 (468) should be light-emitting")
	})

	t.Run("Wisp4", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Wisp1+3),
			"Wisp4 (471) should be light-emitting")
	})

	t.Run("VerifyWispIndexes", func(t *testing.T) {
		assert.Equal(t, 468, indexes.Wisp1,
			"Wisp1 should be sprite index 468")
	})
}

func TestIsLightEmittingNPC_Daemon(t *testing.T) {
	// Daemon sprites are 472-475 (Daemon1 through Daemon4)

	t.Run("Daemon1", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Daemon1),
			"Daemon1 (472) should be light-emitting")
	})

	t.Run("Daemon4", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Daemon1+3),
			"Daemon4 (475) should be light-emitting")
	})

	t.Run("VerifyDaemonIndexes", func(t *testing.T) {
		assert.Equal(t, 472, indexes.Daemon1,
			"Daemon1 should be sprite index 472")
	})
}

func TestIsLightEmittingNPC_Ghost(t *testing.T) {
	// Ghost sprites are 412-415 (Ghost1 through Ghost4)

	t.Run("Ghost1", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Ghost1),
			"Ghost1 (412) should be light-emitting")
	})

	t.Run("Ghost4", func(t *testing.T) {
		assert.True(t, IsLightEmittingNPC(indexes.Ghost1+3),
			"Ghost4 (415) should be light-emitting")
	})

	t.Run("VerifyGhostIndexes", func(t *testing.T) {
		assert.Equal(t, 412, indexes.Ghost1,
			"Ghost1 should be sprite index 412")
	})
}

func TestIsLightEmittingNPC_NonLightEmitting(t *testing.T) {
	// These creatures should NOT emit light

	t.Run("Wizard", func(t *testing.T) {
		assert.False(t, IsLightEmittingNPC(indexes.Wizard1),
			"Wizard1 (320) should NOT be light-emitting")
	})

	t.Run("Orc", func(t *testing.T) {
		assert.False(t, IsLightEmittingNPC(indexes.Orc1),
			"Orc1 should NOT be light-emitting")
	})

	t.Run("Skeleton", func(t *testing.T) {
		assert.False(t, IsLightEmittingNPC(indexes.Skeleton1),
			"Skeleton1 should NOT be light-emitting")
	})

	t.Run("Dragon", func(t *testing.T) {
		assert.False(t, IsLightEmittingNPC(indexes.Dragon1),
			"Dragon1 should NOT be light-emitting")
	})

	t.Run("Avatar", func(t *testing.T) {
		assert.False(t, IsLightEmittingNPC(indexes.Avatar1),
			"Avatar should NOT be light-emitting")
	})

	t.Run("ZeroSprite", func(t *testing.T) {
		assert.False(t, IsLightEmittingNPC(0),
			"Sprite 0 should NOT be light-emitting")
	})
}

// ============================================================================
// GetNPCLightMarkerIndex Tests
// ============================================================================

func TestGetNPCLightMarkerIndex_Seahorse(t *testing.T) {
	// All seahorse sprites should return SeahorseMarkerIndex

	t.Run("Seahorse1", func(t *testing.T) {
		marker := GetNPCLightMarkerIndex(indexes.Seahorse1)
		assert.Equal(t, map_state.SeahorseMarkerIndex, marker,
			"Seahorse1 should return SeahorseMarkerIndex")
	})

	t.Run("Seahorse4", func(t *testing.T) {
		marker := GetNPCLightMarkerIndex(indexes.Seahorse1 + 3)
		assert.Equal(t, map_state.SeahorseMarkerIndex, marker,
			"Seahorse4 should return SeahorseMarkerIndex")
	})

	t.Run("VerifyMarkerValue", func(t *testing.T) {
		assert.Equal(t, indexes.SpriteIndex(0xFFFB), map_state.SeahorseMarkerIndex,
			"SeahorseMarkerIndex should be 0xFFFB")
	})
}

func TestGetNPCLightMarkerIndex_Wisp(t *testing.T) {
	marker := GetNPCLightMarkerIndex(indexes.Wisp1)
	assert.Equal(t, map_state.WispMarkerIndex, marker,
		"Wisp1 should return WispMarkerIndex")

	assert.Equal(t, indexes.SpriteIndex(0xFFFC), map_state.WispMarkerIndex,
		"WispMarkerIndex should be 0xFFFC")
}

func TestGetNPCLightMarkerIndex_Daemon(t *testing.T) {
	marker := GetNPCLightMarkerIndex(indexes.Daemon1)
	assert.Equal(t, map_state.DaemonMarkerIndex, marker,
		"Daemon1 should return DaemonMarkerIndex")

	assert.Equal(t, indexes.SpriteIndex(0xFFFA), map_state.DaemonMarkerIndex,
		"DaemonMarkerIndex should be 0xFFFA")
}

func TestGetNPCLightMarkerIndex_Ghost(t *testing.T) {
	marker := GetNPCLightMarkerIndex(indexes.Ghost1)
	assert.Equal(t, map_state.GhostMarkerIndex, marker,
		"Ghost1 should return GhostMarkerIndex")

	assert.Equal(t, indexes.SpriteIndex(0xFFF9), map_state.GhostMarkerIndex,
		"GhostMarkerIndex should be 0xFFF9")
}

func TestGetNPCLightMarkerIndex_NonLightEmitting(t *testing.T) {
	// Non-light-emitting NPCs should return 0

	t.Run("Wizard", func(t *testing.T) {
		marker := GetNPCLightMarkerIndex(indexes.Wizard1)
		assert.Equal(t, indexes.SpriteIndex(0), marker,
			"Wizard should return 0 (no marker)")
	})

	t.Run("Orc", func(t *testing.T) {
		marker := GetNPCLightMarkerIndex(indexes.Orc1)
		assert.Equal(t, indexes.SpriteIndex(0), marker,
			"Orc should return 0 (no marker)")
	})
}

// ============================================================================
// GetLightSourceProperties Tests for NPC Markers
// ============================================================================

func TestGetLightSourceProperties_NPCMarkers(t *testing.T) {
	// Verify LightSourcePropertiesMap contains entries for NPC marker indices

	t.Run("SeahorseMarker", func(t *testing.T) {
		props := map_state.GetLightSourceProperties(map_state.SeahorseMarkerIndex)
		assert.True(t, props.HasBreathing,
			"Seahorse should have breathing animation")
		assert.Greater(t, props.BreathingPeriodMs, 0,
			"Seahorse breathing period should be > 0")
		// Verify it's the seahorse tint color (blue-ish)
		assert.Greater(t, props.TintColor.B, props.TintColor.R,
			"Seahorse glow should be more blue than red")
	})

	t.Run("WispMarker", func(t *testing.T) {
		props := map_state.GetLightSourceProperties(map_state.WispMarkerIndex)
		assert.True(t, props.HasBreathing,
			"Wisp should have breathing animation")
		assert.Greater(t, props.BreathingPeriodMs, 0,
			"Wisp breathing period should be > 0")
	})

	t.Run("DaemonMarker", func(t *testing.T) {
		props := map_state.GetLightSourceProperties(map_state.DaemonMarkerIndex)
		assert.True(t, props.HasBreathing,
			"Daemon should have breathing animation")
		assert.Greater(t, props.BreathingPeriodMs, 0,
			"Daemon breathing period should be > 0")
		// Verify it's the daemon tint color (red-ish)
		assert.Greater(t, props.TintColor.R, props.TintColor.B,
			"Daemon glow should be more red than blue")
	})

	t.Run("GhostMarker", func(t *testing.T) {
		props := map_state.GetLightSourceProperties(map_state.GhostMarkerIndex)
		assert.True(t, props.HasBreathing,
			"Ghost should have breathing animation")
		assert.Greater(t, props.BreathingPeriodMs, 0,
			"Ghost breathing period should be > 0")
	})
}

func TestGetLightSourceProperties_UnknownMarker(t *testing.T) {
	// Unknown sprite should return neutral tint with no breathing
	props := map_state.GetLightSourceProperties(indexes.SpriteIndex(1)) // Grass or similar

	assert.False(t, props.HasBreathing,
		"Unknown sprite should not have breathing")
	assert.Equal(t, float32(1.0), props.TintColor.R,
		"Unknown sprite should have neutral R=1.0")
	assert.Equal(t, float32(1.0), props.TintColor.G,
		"Unknown sprite should have neutral G=1.0")
	assert.Equal(t, float32(1.0), props.TintColor.B,
		"Unknown sprite should have neutral B=1.0")
}

// ============================================================================
// End-to-End Logic Chain Test
// ============================================================================

func TestSeahorseLightEmissionLogicChain(t *testing.T) {
	// This test verifies the complete logic chain from sprite index to light properties
	// Simulates what happens in renderMapUnitSprite()

	seahorseSprite := indexes.SpriteIndex(indexes.Seahorse1) // 384

	// Step 1: Check if light-emitting
	isLightEmitting := IsLightEmittingNPC(seahorseSprite)
	assert.True(t, isLightEmitting, "Step 1: Seahorse should be detected as light-emitting")

	// Step 2: Get marker index
	markerIndex := GetNPCLightMarkerIndex(seahorseSprite)
	assert.NotEqual(t, indexes.SpriteIndex(0), markerIndex,
		"Step 2: Should return non-zero marker index")
	assert.Equal(t, map_state.SeahorseMarkerIndex, markerIndex,
		"Step 2: Should return SeahorseMarkerIndex")

	// Step 3: Get light properties
	props := map_state.GetLightSourceProperties(markerIndex)
	assert.True(t, props.HasBreathing,
		"Step 3: Properties should have breathing enabled")
	assert.Greater(t, props.BreathingPeriodMs, 0,
		"Step 3: Breathing period should be > 0")

	// Log the complete chain for debugging
	t.Logf("Logic chain verified:")
	t.Logf("  Sprite: %d (Seahorse1)", seahorseSprite)
	t.Logf("  IsLightEmittingNPC: %v", isLightEmitting)
	t.Logf("  MarkerIndex: 0x%X", markerIndex)
	t.Logf("  HasBreathing: %v", props.HasBreathing)
	t.Logf("  BreathingPeriodMs: %d", props.BreathingPeriodMs)
	t.Logf("  TintColor: R=%.2f G=%.2f B=%.2f", props.TintColor.R, props.TintColor.G, props.TintColor.B)
}
