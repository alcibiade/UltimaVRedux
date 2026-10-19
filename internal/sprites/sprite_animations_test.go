package sprites

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REGRESSION TESTS - Prevent non-deterministic animation behavior from regressing
// These tests ensure animations use deterministic timing instead of time.Now()

// Characterize the established frame timing independently of the shared catalog.
const msPerFrameForObjects = 125
const msPerFrameForNPCs = 175

func TestAnimationDeterminism(t *testing.T) {
	// CRITICAL: Same inputs should ALWAYS produce same outputs (deterministic)
	// This prevents regression where animations used time.Now() and were non-reproducible

	testCases := []struct {
		name        string
		spriteIndex indexes.SpriteIndex
		posHash     int32
		elapsedMs   int64
	}{
		{
			name:        "Waterfall animation",
			spriteIndex: indexes.Waterfall_KeyIndex,
			posHash:     123,
			elapsedMs:   1000,
		},
		{
			name:        "Fountain animation",
			spriteIndex: indexes.Fountain_KeyIndex,
			posHash:     456,
			elapsedMs:   2000,
		},
		{
			name:        "Clock animation",
			spriteIndex: indexes.Clock1,
			posHash:     789,
			elapsedMs:   1500,
		},
		{
			name:        "NPC animation",
			spriteIndex: indexes.SitChairDownEat1,
			posHash:     100,
			elapsedMs:   3000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function multiple times with same inputs
			result1 := GetSpriteIndexWithAnimationBySpriteIndexTick(tc.spriteIndex, tc.posHash, tc.elapsedMs)
			result2 := GetSpriteIndexWithAnimationBySpriteIndexTick(tc.spriteIndex, tc.posHash, tc.elapsedMs)
			result3 := GetSpriteIndexWithAnimationBySpriteIndexTick(tc.spriteIndex, tc.posHash, tc.elapsedMs)

			// CRITICAL: Results must be identical (deterministic behavior)
			if result1 != result2 {
				t.Errorf("REGRESSION: Animation not deterministic - first call: %d, second call: %d", result1, result2)
			}
			if result1 != result3 {
				t.Errorf("REGRESSION: Animation not deterministic - first call: %d, third call: %d", result1, result3)
			}

			t.Logf("✅ Deterministic result for %s: %d", tc.name, result1)
		})
	}
}

func TestAnimationProgression(t *testing.T) {
	// Test that animations actually progress over time (not stuck)

	testCases := []struct {
		name        string
		spriteIndex indexes.SpriteIndex
		posHash     int32
		frameMs     int64 // Time for one animation frame
	}{
		{
			name:        "Waterfall progression",
			spriteIndex: indexes.Waterfall_KeyIndex,
			posHash:     0,
			frameMs:     msPerFrameForObjects, // 125ms
		},
		{
			name:        "Fountain progression",
			spriteIndex: indexes.Fountain_KeyIndex,
			posHash:     0,
			frameMs:     msPerFrameForObjects, // 125ms
		},
		{
			name:        "Clock progression",
			spriteIndex: indexes.Clock1,
			posHash:     0,
			frameMs:     msPerFrameForObjects * 2, // 250ms (clocks are 2x slower)
		},
		{
			name:        "NPC progression",
			spriteIndex: indexes.SitChairDownEat1,
			posHash:     0,
			frameMs:     msPerFrameForNPCs, // 175ms
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get animation frame at time 0
			frame1 := GetSpriteIndexWithAnimationBySpriteIndexTick(tc.spriteIndex, tc.posHash, 0)

			// Get animation frame after exactly one frame duration
			frame2 := GetSpriteIndexWithAnimationBySpriteIndexTick(tc.spriteIndex, tc.posHash, tc.frameMs)

			// Animation should progress (different frames)
			if frame1 == frame2 {
				t.Errorf("Animation should progress over time - frame at 0ms: %d, frame at %dms: %d",
					frame1, tc.frameMs, frame2)
			} else {
				t.Logf("✅ Animation progresses: %d → %d over %dms", frame1, frame2, tc.frameMs)
			}
		})
	}
}

func TestAnimationFrameBounds(t *testing.T) {
	// Test that animation frames stay within expected bounds

	t.Run("Waterfall animation bounds", func(t *testing.T) {
		var baseIndex indexes.SpriteIndex = indexes.Waterfall_KeyIndex

		// Test various time points
		for elapsedMs := int64(0); elapsedMs < 1000; elapsedMs += 50 {
			result := GetSpriteIndexWithAnimationBySpriteIndexTick(baseIndex, 0, elapsedMs)

			// Should be within the standard animation frame range
			minIndex := int(baseIndex)
			maxIndex := int(baseIndex) + int(indexes.StandardNumberOfAnimationFrames) - 1

			if int(result) < minIndex || int(result) > maxIndex {
				t.Errorf("Animation frame out of bounds at %dms: got %d, expected range [%d-%d]",
					elapsedMs, result, minIndex, maxIndex)
			}
		}
	})

	t.Run("Clock animation bounds", func(t *testing.T) {
		for _, sourceIndex := range []indexes.SpriteIndex{indexes.Clock1, indexes.Clock2} {
			for elapsedMs := int64(0); elapsedMs < 1000; elapsedMs += 100 {
				result := GetSpriteIndexWithAnimationBySpriteIndexTick(sourceIndex, 0, elapsedMs)
				if result < indexes.Clock1 || result > indexes.Clock2 {
					t.Errorf("Clock source %d animation frame out of bounds at %dms: got %d, expected range [%d-%d]",
						sourceIndex, elapsedMs, result, indexes.Clock1, indexes.Clock2)
				}
			}
		}
	})
}

func TestNPCAnimationNegativePositionHashStaysWithinSpriteGroup(t *testing.T) {
	for elapsedMs := int64(0); elapsedMs < msPerFrameForNPCs*8; elapsedMs += msPerFrameForNPCs {
		got := GetSpriteIndexWithAnimationBySpriteIndexTick(indexes.Orc1, -14331, elapsedMs)
		if got < indexes.Orc1 || got >= indexes.Orc1+indexes.StandardNumberOfAnimationFrames {
			t.Fatalf("Orc frame at %dms = %d, want range [%d,%d]", elapsedMs, got, indexes.Orc1, indexes.Orc1+3)
		}
	}
}

func TestAnimationPositionHashVariation(t *testing.T) {
	// At 1000ms the 125ms waterfall cycle is back at its first frame.
	// Adjacent phases and signed wrap must select exact authored frames.
	for _, tc := range []struct {
		hash  int32
		frame indexes.SpriteIndex
	}{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 0}, {-1, 3}} {
		want := indexes.Waterfall_KeyIndex + tc.frame
		if got := GetSpriteIndexWithAnimationBySpriteIndexTick(indexes.Waterfall_KeyIndex, tc.hash, 1000); got != want {
			t.Errorf("hash %d: got %d, want %d", tc.hash, got, want)
		}
	}
}

func TestAnimationStaticSprites(t *testing.T) {
	// Test that non-animated sprites are returned unchanged

	staticSprites := []indexes.SpriteIndex{
		indexes.BrickFloor,
		indexes.Grass,
		indexes.Shard,           // Shard sprite must NOT animate
		indexes.SpriteIndex(10), // Generic non-animated sprite (avoiding water tiles 1,2,3)
		indexes.SpriteIndex(50), // Another generic non-animated sprite
	}

	for _, sprite := range staticSprites {
		// Static sprites should be returned unchanged regardless of time
		result1 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, 0)
		result2 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, 10000)
		result3 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 100, 5000)

		if result1 != sprite || result2 != sprite || result3 != sprite {
			t.Errorf("Static sprite should remain unchanged: input %d, results %d, %d, %d",
				sprite, result1, result2, result3)
		}
	}
}

func TestWaterTileAnimation(t *testing.T) {
	// Test water tile behavior - CORRECTED based on original Ultima 5 understanding
	// Original water animation: pixel-level scrolling within each tile type
	// SHOALS stay SHOALS, SEA stays SEA, DEEP stays DEEP - only internal pattern scrolls
	// Current implementation: no animation (return tile unchanged) until pixel scrolling implemented

	waterTiles := []indexes.SpriteIndex{
		indexes.Water1,     // DEEP (1)
		indexes.Water2,     // SEA (2)
		indexes.WaterCoast, // SHOALS (3)
	}

	for _, sprite := range waterTiles {
		t.Run(string(rune(sprite+48)), func(t *testing.T) { // Convert index to char for test name
			// Test deterministic behavior
			result1 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, 0)
			result2 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, 0)

			if result1 != result2 {
				t.Errorf("Water tile behavior not deterministic: %d vs %d", result1, result2)
			}

			// Test that water tiles maintain their type (correct behavior)
			frame1 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, 0)
			frame2 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, msPerFrameForObjects)
			frame3 := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, 0, msPerFrameForObjects*2)

			// Water tiles should maintain their type (no tile-swapping animation)
			if frame1 != sprite || frame2 != sprite || frame3 != sprite {
				t.Errorf("Water tile should maintain its type: input %d, results %d, %d, %d",
					sprite, frame1, frame2, frame3)
			} else {
				t.Logf("✅ Water tile maintains type: %s (%d) stays %s",
					getWaterTileName(sprite), sprite, getWaterTileName(sprite))
			}
		})
	}
}

func getWaterTileName(sprite indexes.SpriteIndex) string {
	switch sprite {
	case indexes.Water1:
		return "DEEP"
	case indexes.Water2:
		return "SEA"
	case indexes.WaterCoast:
		return "SHOALS"
	default:
		return "UNKNOWN"
	}
}

func TestWaterTileIntegration(t *testing.T) {
	// Integration test: Water tiles should work seamlessly with existing animation systems
	// CORRECTED: Water tiles now maintain their type (no animation until pixel scrolling implemented)

	t.Run("Water tiles don't affect other animations", func(t *testing.T) {
		elapsedMs := int64(1000)
		posHash := int32(42)

		// Test that non-water animations still work correctly
		waterfallResult := GetSpriteIndexWithAnimationBySpriteIndexTick(indexes.Waterfall_KeyIndex, posHash, elapsedMs)
		fountainResult := GetSpriteIndexWithAnimationBySpriteIndexTick(indexes.Fountain_KeyIndex, posHash, elapsedMs)
		clockResult := GetSpriteIndexWithAnimationBySpriteIndexTick(indexes.Clock1, posHash, elapsedMs)

		// Verify animations are within expected ranges
		if waterfallResult < indexes.Waterfall_KeyIndex || waterfallResult >= indexes.Waterfall_KeyIndex+indexes.StandardNumberOfAnimationFrames {
			t.Errorf("Waterfall animation out of range: %d", waterfallResult)
		}

		if fountainResult < indexes.Fountain_KeyIndex || fountainResult >= indexes.Fountain_KeyIndex+indexes.StandardNumberOfAnimationFrames {
			t.Errorf("Fountain animation out of range: %d", fountainResult)
		}

		if clockResult < indexes.Clock1 || clockResult > indexes.Clock2 {
			t.Errorf("Clock animation out of range: %d", clockResult)
		}

		t.Logf("✅ Other animations unaffected: waterfall=%d, fountain=%d, clock=%d", waterfallResult, fountainResult, clockResult)
	})

	t.Run("Water tiles maintain type consistency", func(t *testing.T) {
		// Test that water tiles maintain their identity across different conditions
		elapsedMs := int64(2000)

		waterTiles := []indexes.SpriteIndex{indexes.Water1, indexes.Water2, indexes.WaterCoast}

		for _, tile := range waterTiles {
			// Test with different position hashes - all should return same tile type
			for posHash := int32(0); posHash < 5; posHash++ {
				result := GetSpriteIndexWithAnimationBySpriteIndexTick(tile, posHash, elapsedMs)
				if result != tile {
					t.Errorf("Water tile changed type: %s (%d) became %d",
						getWaterTileName(tile), tile, result)
				}
			}
		}

		t.Logf("✅ All water tiles maintain their type consistently")
	})

}
