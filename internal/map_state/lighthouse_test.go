package map_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestLighthouseConeDataValidity verifies all 16 cone shapes have valid data.
func TestLighthouseConeDataValidity(t *testing.T) {
	t.Parallel()

	for shapeIdx := 0; shapeIdx <= MaxLighthouseAngle; shapeIdx++ {
		shape := lighthouseConeShapes[shapeIdx]

		// Count non-terminator offsets
		offsetCount := 0
		for _, offset := range shape {
			if offset.isTerminator() {
				break
			}
			offsetCount++
		}

		// Each shape should have at least 1 offset
		if offsetCount == 0 {
			t.Errorf("Shape %d has no offsets (all terminators)", shapeIdx)
		}

		// No shape should have more than MaxOffsetsPerShape offsets
		if offsetCount > MaxOffsetsPerShape {
			t.Errorf("Shape %d has %d offsets, exceeds max %d", shapeIdx, offsetCount, MaxOffsetsPerShape)
		}

		t.Logf("Shape %d has %d offsets", shapeIdx, offsetCount)
	}
}

// TestLighthouseOffsetTerminator verifies the terminator check works correctly.
func TestLighthouseOffsetTerminator(t *testing.T) {
	t.Parallel()

	terminator := LighthouseOffset{X: 0, Y: 0}
	if !terminator.isTerminator() {
		t.Error("(0,0) should be detected as terminator")
	}

	nonTerminator := LighthouseOffset{X: 1, Y: -2}
	if nonTerminator.isTerminator() {
		t.Error("(1,-2) should not be detected as terminator")
	}
}

// TestNewLighthouseState verifies initial state is correct.
func TestNewLighthouseState(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()

	if ls.currentAngle != UninitializedAngle {
		t.Errorf("Initial angle should be %d, got %d", UninitializedAngle, ls.currentAngle)
	}

	if ls.HasLighthouses() {
		t.Error("New state should have no lighthouses")
	}

	if len(ls.litTilesMap) != 0 {
		t.Error("New state should have no lit tiles")
	}
}

// TestLighthouseSetPositions verifies position setting works correctly.
func TestLighthouseSetPositions(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()

	// Set single lighthouse
	pos1 := references.Position{X: 10, Y: 15}
	ls.SetPositions([]references.Position{pos1})

	if !ls.HasLighthouses() {
		t.Error("Should have lighthouses after SetPositions")
	}

	positions := ls.GetPositions()
	if len(positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(positions))
	}
	if positions[0] != pos1 {
		t.Errorf("Expected position %v, got %v", pos1, positions[0])
	}

	// Set two lighthouses
	pos2 := references.Position{X: 20, Y: 25}
	ls.SetPositions([]references.Position{pos1, pos2})

	positions = ls.GetPositions()
	if len(positions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(positions))
	}

	// Try to set more than max - should be capped
	pos3 := references.Position{X: 30, Y: 35}
	ls.SetPositions([]references.Position{pos1, pos2, pos3})

	positions = ls.GetPositions()
	if len(positions) != MaxLighthousesPerMap {
		t.Errorf("Expected max %d positions, got %d", MaxLighthousesPerMap, len(positions))
	}
}

// TestLighthouseInitialRotation verifies first rotation lights 3 segments.
func TestLighthouseInitialRotation(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// First update should initialize
	changed := ls.Update(LighthouseUpdateIntervalMs, false)

	if !changed {
		t.Error("First update should indicate change")
	}

	if ls.GetCurrentAngle() != 0 {
		t.Errorf("After first rotation, angle should be 0, got %d", ls.GetCurrentAngle())
	}

	// Should have lit tiles from shapes 0, 1, 2
	litTiles := ls.GetLitTiles()
	if len(litTiles) == 0 {
		t.Error("Should have lit tiles after initial rotation")
	}

	t.Logf("Initial rotation lit %d tiles", len(litTiles))
}

// TestLighthouseRotationCycle verifies rotation cycles through all angles.
func TestLighthouseRotationCycle(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// Force initialization
	ls.Update(LighthouseUpdateIntervalMs, false)
	if ls.GetCurrentAngle() != 0 {
		t.Fatalf("Expected angle 0 after init, got %d", ls.GetCurrentAngle())
	}

	// Rotate through all 16 positions
	expectedAngles := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 0}

	for i, expected := range expectedAngles {
		elapsedMs := int64((i + 2) * LighthouseUpdateIntervalMs)
		ls.Update(elapsedMs, false)

		if ls.GetCurrentAngle() != expected {
			t.Errorf("Rotation %d: expected angle %d, got %d", i+1, expected, ls.GetCurrentAngle())
		}
	}
}

// TestLighthouseTimingControl verifies rotation respects timing interval.
func TestLighthouseTimingControl(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// First update at t=55ms
	ls.Update(LighthouseUpdateIntervalMs, false)
	initialAngle := ls.GetCurrentAngle()

	// Update at t=60ms - not enough time passed, should not rotate
	changed := ls.Update(60, false)
	if changed {
		t.Error("Should not rotate before interval elapsed")
	}
	if ls.GetCurrentAngle() != initialAngle {
		t.Errorf("Angle changed before interval: %d -> %d", initialAngle, ls.GetCurrentAngle())
	}

	// Update at t=110ms - enough time passed
	changed = ls.Update(LighthouseUpdateIntervalMs*2, false)
	if !changed {
		t.Error("Should rotate after interval elapsed")
	}
	if ls.GetCurrentAngle() == initialAngle {
		t.Error("Angle should have changed after interval")
	}
}

// TestLighthouseDaylightSkip verifies lighthouse disabled during daylight.
func TestLighthouseDaylightSkip(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// Update during daylight - should not activate
	changed := ls.Update(LighthouseUpdateIntervalMs, true)

	if changed {
		t.Error("Should not change during daylight on uninitialized state")
	}

	if ls.GetCurrentAngle() != UninitializedAngle {
		t.Errorf("Should remain uninitialized during daylight, got angle %d", ls.GetCurrentAngle())
	}

	if len(ls.GetLitTiles()) != 0 {
		t.Error("Should have no lit tiles during daylight")
	}
}

// TestLighthouseDaylightReset verifies lighthouse resets when daylight arrives.
func TestLighthouseDaylightReset(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// Initialize at night
	ls.Update(LighthouseUpdateIntervalMs, false)
	if ls.GetCurrentAngle() == UninitializedAngle {
		t.Fatal("Should be initialized after night update")
	}

	initialLitCount := len(ls.GetLitTiles())
	if initialLitCount == 0 {
		t.Fatal("Should have lit tiles after initialization")
	}

	// Daylight arrives - should reset
	changed := ls.Update(LighthouseUpdateIntervalMs*2, true)

	if !changed {
		t.Error("Should indicate change when clearing for daylight")
	}

	if ls.GetCurrentAngle() != UninitializedAngle {
		t.Errorf("Should reset to uninitialized on daylight, got angle %d", ls.GetCurrentAngle())
	}

	if len(ls.GetLitTiles()) != 0 {
		t.Error("Should clear lit tiles on daylight")
	}
}

// TestLighthouseNoLighthouses verifies behavior with no lighthouses.
func TestLighthouseNoLighthouses(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()

	// Update with no lighthouses - should not change
	changed := ls.Update(LighthouseUpdateIntervalMs, false)

	if changed {
		t.Error("Should not change with no lighthouses")
	}

	if ls.GetCurrentAngle() != UninitializedAngle {
		t.Errorf("Should remain uninitialized with no lighthouses, got %d", ls.GetCurrentAngle())
	}
}

// TestLighthouseIsTileLit verifies individual tile lit checking.
func TestLighthouseIsTileLit(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// Initialize
	ls.Update(LighthouseUpdateIntervalMs, false)

	// Check a tile that should be lit (based on shape 0: first offset is -1,-2)
	expectedLit := references.Position{X: 15, Y: 14} // 16 + (-1), 16 + (-2)
	if !ls.IsTileLit(expectedLit) {
		t.Errorf("Tile %v should be lit", expectedLit)
	}

	// Check a tile that should not be lit (far from lighthouse)
	notLit := references.Position{X: 0, Y: 0}
	if ls.IsTileLit(notLit) {
		t.Errorf("Tile %v should not be lit", notLit)
	}
}

// TestLighthouseBeaconItselfIsLit verifies the beacon position itself is always lit.
func TestLighthouseBeaconItselfIsLit(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// Even before initialization, the beacon position should be lit
	// (the beacon is the light source)
	if !ls.IsTileLit(lighthousePos) {
		t.Error("Lighthouse beacon position itself should always be lit")
	}

	// Initialize and rotate
	ls.Update(LighthouseUpdateIntervalMs, false)

	// Still should be lit after rotation
	if !ls.IsTileLit(lighthousePos) {
		t.Error("Lighthouse beacon position should remain lit after rotation")
	}

	// Rotate many times and verify beacon stays lit
	for i := 0; i < 20; i++ {
		ls.Update(int64(i+2)*LighthouseUpdateIntervalMs, false)
		if !ls.IsTileLit(lighthousePos) {
			t.Errorf("Lighthouse beacon should be lit at rotation %d", i)
		}
	}
}

// TestLighthouseTwoLighthouses verifies behavior with two lighthouses.
func TestLighthouseTwoLighthouses(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	pos1 := references.Position{X: 10, Y: 10}
	pos2 := references.Position{X: 20, Y: 20}
	ls.SetPositions([]references.Position{pos1, pos2})

	// Initialize
	ls.Update(LighthouseUpdateIntervalMs, false)

	litTiles := ls.GetLitTiles()

	// Should have tiles from both lighthouses
	// Check tiles relative to first lighthouse (shape 0, offset -1,-2)
	tile1 := references.Position{X: 9, Y: 8} // 10 + (-1), 10 + (-2)
	if !ls.IsTileLit(tile1) {
		t.Errorf("Tile %v (from lighthouse 1) should be lit", tile1)
	}

	// Check tiles relative to second lighthouse (shape 0, offset -1,-2)
	tile2 := references.Position{X: 19, Y: 18} // 20 + (-1), 20 + (-2)
	if !ls.IsTileLit(tile2) {
		t.Errorf("Tile %v (from lighthouse 2) should be lit", tile2)
	}

	t.Logf("Two lighthouses lit %d tiles total", len(litTiles))
}

// TestLighthouseShapeWrapAround verifies shape index wrapping works correctly.
func TestLighthouseShapeWrapAround(t *testing.T) {
	t.Parallel()

	ls := NewLighthouseState()
	lighthousePos := references.Position{X: 16, Y: 16}
	ls.SetPositions([]references.Position{lighthousePos})

	// Initialize and rotate to angle 15
	elapsedMs := int64(LighthouseUpdateIntervalMs)
	ls.Update(elapsedMs, false) // angle 0

	for i := 1; i <= 15; i++ {
		elapsedMs += LighthouseUpdateIntervalMs
		ls.Update(elapsedMs, false)
	}

	if ls.GetCurrentAngle() != 15 {
		t.Fatalf("Expected angle 15, got %d", ls.GetCurrentAngle())
	}

	// Next rotation should wrap to 0
	elapsedMs += LighthouseUpdateIntervalMs
	ls.Update(elapsedMs, false)

	if ls.GetCurrentAngle() != 0 {
		t.Errorf("Expected angle to wrap to 0, got %d", ls.GetCurrentAngle())
	}
}

// TestFindLighthousePositionsInViewport verifies viewport scanning for lighthouse tiles.
// This function is used on large maps (overworld) to detect lighthouses in visible area.
func TestFindLighthousePositionsInViewport(t *testing.T) {
	t.Parallel()

	// Create a test LayeredMap (256x256 wrapping map like overworld)
	layeredMap := newLayeredMap(
		references.XLargeMapTiles, // 256
		references.YLargeMapTiles, // 256
		nil,                       // tileRefs not needed for this test
		32, 32,                    // visible tiles
		true, // wrapping map
	)

	// Place lighthouse tiles at known positions
	lighthousePos1 := references.Position{X: 88, Y: 120}  // Britain lighthouse
	lighthousePos2 := references.Position{X: 150, Y: 180} // Another lighthouse

	layeredMap.SetTileByLayer(MapLayer, &lighthousePos1, indexes.Lighthouse)
	layeredMap.SetTileByLayer(MapLayer, &lighthousePos2, indexes.Lighthouse)

	// Test 1: Player near first lighthouse
	playerPos := references.Position{X: 90, Y: 122}
	positions := layeredMap.FindLighthousePositionsInViewport(playerPos, 16, indexes.Lighthouse)

	if len(positions) != 1 {
		t.Errorf("Expected 1 lighthouse near player, got %d", len(positions))
	}
	if len(positions) > 0 && positions[0] != lighthousePos1 {
		t.Errorf("Expected position %v, got %v", lighthousePos1, positions[0])
	}

	// Test 2: Player far from any lighthouse
	farPlayerPos := references.Position{X: 10, Y: 10}
	positions = layeredMap.FindLighthousePositionsInViewport(farPlayerPos, 16, indexes.Lighthouse)

	if len(positions) != 0 {
		t.Errorf("Expected 0 lighthouses far from player, got %d", len(positions))
	}

	// Test 3: Player position where both lighthouses are visible (if close enough)
	// Place second lighthouse closer to first for this test
	closePos2 := references.Position{X: 100, Y: 130}
	layeredMap.SetTileByLayer(MapLayer, &closePos2, indexes.Lighthouse)

	playerBetween := references.Position{X: 94, Y: 125}
	positions = layeredMap.FindLighthousePositionsInViewport(playerBetween, 16, indexes.Lighthouse)

	if len(positions) != 2 {
		t.Errorf("Expected 2 lighthouses visible, got %d", len(positions))
	}

	t.Logf("Found %d lighthouses in viewport", len(positions))
}

// TestFindLighthousePositionsInViewportWrapping verifies coordinate wrapping at map edges.
func TestFindLighthousePositionsInViewportWrapping(t *testing.T) {
	t.Parallel()

	// Create a wrapping map
	layeredMap := newLayeredMap(
		references.XLargeMapTiles,
		references.YLargeMapTiles,
		nil,
		32, 32,
		true, // wrapping map
	)

	// Place lighthouse near map edge
	lighthouseNearEdge := references.Position{X: 5, Y: 5}
	layeredMap.SetTileByLayer(MapLayer, &lighthouseNearEdge, indexes.Lighthouse)

	// Player on opposite side of map edge (should wrap around and find lighthouse)
	playerNearEdge := references.Position{X: 250, Y: 250} // Near 256 edge, radius 16 should wrap to find (5,5)

	// The viewport scan wraps, so (250,250) with radius 16 includes x: 234-266(wrapped 0-10), y: 234-266(0-10)
	positions := layeredMap.FindLighthousePositionsInViewport(playerNearEdge, 16, indexes.Lighthouse)

	if len(positions) != 1 {
		t.Errorf("Expected 1 lighthouse via wrapping, got %d", len(positions))
	}

	if len(positions) > 0 {
		// The returned position should be wrapped to valid coordinates
		if positions[0].X < 0 || positions[0].X >= references.XLargeMapTiles {
			t.Errorf("Position X out of bounds: %d", positions[0].X)
		}
		if positions[0].Y < 0 || positions[0].Y >= references.YLargeMapTiles {
			t.Errorf("Position Y out of bounds: %d", positions[0].Y)
		}
		t.Logf("Found lighthouse at wrapped position: %v", positions[0])
	}
}

// TestFindLighthousePositionsMaxLimit verifies max lighthouse limit is enforced.
func TestFindLighthousePositionsMaxLimit(t *testing.T) {
	t.Parallel()

	layeredMap := newLayeredMap(
		references.XLargeMapTiles,
		references.YLargeMapTiles,
		nil,
		32, 32,
		true,
	)

	// Place 4 lighthouses (more than MaxLighthousesPerMap which is 2)
	for i := 0; i < 4; i++ {
		pos := references.Position{X: references.Coordinate(100 + i*2), Y: 100}
		layeredMap.SetTileByLayer(MapLayer, &pos, indexes.Lighthouse)
	}

	// Search with player in center
	playerPos := references.Position{X: 103, Y: 100}
	positions := layeredMap.FindLighthousePositionsInViewport(playerPos, 16, indexes.Lighthouse)

	if len(positions) > MaxLighthousesPerMap {
		t.Errorf("Expected max %d lighthouses, got %d", MaxLighthousesPerMap, len(positions))
	}

	t.Logf("Found %d lighthouses (max %d)", len(positions), MaxLighthousesPerMap)
}
