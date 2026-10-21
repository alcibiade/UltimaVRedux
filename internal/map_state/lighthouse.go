// Package map_state provides lighthouse rotation system faithful to original Ultima V.
//
// The lighthouse system creates a rotating beam of light that sweeps across the map
// during nighttime hours. Sixteen predefined
// cone shapes that cycle to create the rotation effect.
//
// Beam behavior:
//   - 16 cone shapes, each with storage for 16 offsets
//   - Each shape has up to 16 x,y offset pairs defining lit tiles
//   - Beam shows 3 consecutive segments at any time
//   - Rotation removes trailing edge, advances angle, adds leading edge
//   - Skipped when sight >= 50 (full daylight; ref:LIGHT-0001) or no lighthouse on map
//   - Original display updates used approximately 18Hz timing
package map_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Lighthouse timing and rotation constants from original Ultima V.
const (
	// LighthouseUpdateIntervalMs controls rotation speed.
	// Original display timing was approximately 18Hz (55ms);
	// Redux uses 110ms for smoother visuals.
	LighthouseUpdateIntervalMs = 110

	// MaxLighthouseAngle is the maximum rotation angle (0-15 for 16 positions).
	MaxLighthouseAngle = 15

	// ConeSegmentsLit is the number of consecutive cone segments visible at once.
	// Original shows segments at currentAngle, currentAngle+1, currentAngle+2.
	ConeSegmentsLit = 3

	// MaxLighthousesPerMap limits lighthouses per map (per original game).
	MaxLighthousesPerMap = 2

	// MaxOffsetsPerShape is the maximum offsets in a single cone shape.
	MaxOffsetsPerShape = 16

	// UninitializedAngle indicates lighthouse hasn't been initialized yet.
	// Represents an unset rotation direction, encoded originally as byte 0xff.
	UninitializedAngle = -1
)

// LighthouseOffset defines a single x,y offset for the light cone.
// These offsets are relative to the lighthouse beacon position.
type LighthouseOffset struct {
	X, Y int8
}

// isTerminator returns true if this offset is the (0,0) terminator.
// Original uses (0,0) to mark end of offset list for shapes with fewer than 16 tiles.
func (lo LighthouseOffset) isTerminator() bool {
	return lo.X == 0 && lo.Y == 0
}

// lighthouseConeShapes contains the 16 directional beam footprints.
// Each shape represents one of 16 rotation angles (22.5 degrees apart).
// Shapes are indexed 0-15, each with up to 16 offset pairs.
// (0,0) acts as terminator for shapes with fewer than 16 lit tiles.
//
//nolint:gochecknoglobals // Data table from original game - immutable reference data
var lighthouseConeShapes = [16][MaxOffsetsPerShape]LighthouseOffset{
	// Shape 0 (pointing roughly North-Northwest)
	{{-1, -2}, {-1, -3}, {-2, -3}, {-2, -4}, {-2, -5}, {-2, -6}, {-2, -7}, {-3, -5},
		{-3, -6}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 1 (pointing roughly North)
	{{0, -1}, {0, -2}, {0, -3}, {0, -4}, {0, -5}, {0, -6}, {0, -7}, {-1, -4},
		{-1, -5}, {-1, -6}, {-1, -7}, {1, -4}, {1, -5}, {1, -6}, {1, -7}, {0, 0}},

	// Shape 2 (pointing roughly North-Northeast)
	{{1, -2}, {1, -3}, {2, -3}, {2, -4}, {2, -5}, {2, -6}, {2, -7}, {3, -5},
		{3, -6}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 3 (pointing roughly Northeast)
	{{1, -1}, {2, -2}, {3, -3}, {3, -4}, {4, -3}, {4, -4}, {4, -5}, {4, -6},
		{5, -4}, {5, -5}, {6, -4}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 4 (pointing roughly East-Northeast)
	{{2, -1}, {3, -1}, {3, -2}, {4, -2}, {5, -2}, {5, -3}, {6, -2}, {6, -3},
		{7, -2}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 5 (pointing roughly East)
	{{1, 0}, {2, 0}, {3, 0}, {4, 0}, {5, 0}, {6, 0}, {7, 0}, {4, -1},
		{5, -1}, {6, -1}, {7, -1}, {4, 1}, {5, 1}, {6, 1}, {7, 1}, {0, 0}},

	// Shape 6 (pointing roughly East-Southeast)
	{{2, 1}, {3, 1}, {3, 2}, {4, 2}, {5, 2}, {5, 3}, {6, 2}, {6, 3},
		{7, 2}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 7 (pointing roughly Southeast)
	{{1, 1}, {2, 2}, {3, 3}, {4, 3}, {4, 4}, {4, 5}, {5, 4}, {5, 5},
		{6, 4}, {3, 4}, {4, 6}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 8 (pointing roughly South-Southeast)
	{{1, 2}, {1, 3}, {2, 3}, {2, 4}, {2, 5}, {2, 6}, {2, 7}, {3, 5},
		{3, 6}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 9 (pointing roughly South)
	{{0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}, {1, 4},
		{1, 5}, {1, 6}, {1, 7}, {-1, 4}, {-1, 5}, {-1, 6}, {-1, 7}, {0, 0}},

	// Shape 10 (pointing roughly South-Southwest) - 0xA
	{{-1, 2}, {-1, 3}, {-2, 3}, {-2, 4}, {-2, 5}, {-2, 6}, {-2, 7}, {-3, 5},
		{-3, 6}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 11 (pointing roughly Southwest) - 0xB
	{{-1, 1}, {-2, 2}, {-3, 3}, {-3, 4}, {-4, 3}, {-4, 4}, {-4, 5}, {-4, 6},
		{-5, 4}, {-5, 5}, {-6, 4}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 12 (pointing roughly West-Southwest) - 0xC
	{{-2, 1}, {-3, 1}, {-3, 2}, {-4, 2}, {-5, 2}, {-6, 2}, {-7, 2}, {-5, 3},
		{-6, 3}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 13 (pointing roughly West) - 0xD
	{{-1, 0}, {-2, 0}, {-3, 0}, {-4, 0}, {-5, 0}, {-6, 0}, {-7, 0}, {-4, 1},
		{-5, 1}, {-6, 1}, {-7, 1}, {-4, -1}, {-5, -1}, {-6, -1}, {-7, -1}, {0, 0}},

	// Shape 14 (pointing roughly West-Northwest) - 0xE
	{{-2, -1}, {-3, -1}, {-3, -2}, {-4, -2}, {-5, -2}, {-6, -2}, {-7, -2}, {-5, -3},
		{-6, -3}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},

	// Shape 15 (pointing roughly Northwest) - 0xF
	{{-1, -1}, {-2, -2}, {-3, -3}, {-3, -4}, {-4, -3}, {-4, -4}, {-4, -5}, {-4, -6},
		{-5, -4}, {-5, -5}, {-6, -4}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},
}

// LighthouseState tracks lighthouse rotation state for a map.
// Manages up to 2 lighthouses per map, matching original game limit.
type LighthouseState struct {
	// positions holds lighthouse beacon positions (up to MaxLighthousesPerMap).
	positions []references.Position

	// currentAngle is the rotation angle (0-15), or UninitializedAngle if not yet started.
	currentAngle int

	// litTilesMap tracks which tiles are currently lit by lighthouse cones.
	// This is kept separate from other light sources to enable "lighthouse-only" detection.
	litTilesMap map[references.Position]bool

	// lastUpdateMs tracks last rotation time for timing control.
	lastUpdateMs int64

	// isLargeMap indicates if operating on a wrapping large map (256x256).
	// When true, lit tile positions are wrapped to handle map edge cases.
	isLargeMap bool
}

// NewLighthouseState creates a new lighthouse state in uninitialized state.
func NewLighthouseState() *LighthouseState {
	return &LighthouseState{
		positions:    make([]references.Position, 0, MaxLighthousesPerMap),
		currentAngle: UninitializedAngle,
		litTilesMap:  make(map[references.Position]bool),
		lastUpdateMs: 0,
		isLargeMap:   false,
	}
}

// SetPositions sets the lighthouse beacon positions.
// Accepts up to MaxLighthousesPerMap positions; extras are ignored.
// Only resets rotation state if positions actually changed, preserving
// smooth rotation when player moves within range of same lighthouses.
func (ls *LighthouseState) SetPositions(positions []references.Position) {
	// Build new positions list (limited to MaxLighthousesPerMap)
	newPositions := make([]references.Position, 0, MaxLighthousesPerMap)
	for i, pos := range positions {
		if i >= MaxLighthousesPerMap {
			break
		}
		newPositions = append(newPositions, pos)
	}

	// Check if positions actually changed
	positionsChanged := len(newPositions) != len(ls.positions)
	if !positionsChanged {
		for i, pos := range newPositions {
			if pos != ls.positions[i] {
				positionsChanged = true

				break
			}
		}
	}

	ls.positions = newPositions

	// Only reset rotation state if positions actually changed
	if positionsChanged {
		ls.currentAngle = UninitializedAngle
		ls.litTilesMap = make(map[references.Position]bool)
	}

	// Debug logging to verify lighthouse detection
	logging.Debug(logging.CategoryLighthouse, "SetPositions: %d lighthouse(s) set", len(ls.positions))
	for i, pos := range ls.positions {
		logging.Debug(logging.CategoryLighthouse, "  Lighthouse %d at (%d, %d)", i+1, pos.X, pos.Y)
	}
}

// HasLighthouses returns true if there are any lighthouses on the map.
func (ls *LighthouseState) HasLighthouses() bool {
	return len(ls.positions) > 0
}

// SetLargeMapMode sets whether the lighthouse operates on a wrapping large map.
// When true, lit tile positions are wrapped to handle coordinates near map edges.
// Should be called after SetPositions when operating on overworld (256x256 map).
func (ls *LighthouseState) SetLargeMapMode(isLargeMap bool) {
	ls.isLargeMap = isLargeMap
}

// Update checks if rotation should occur based on elapsed time and daylight.
// Returns true if rotation occurred (and lit tiles changed).
//
// Parameters:
//   - elapsedMs: Current game time in milliseconds
//   - isFullDaylight: If true, lighthouse is disabled (matches original sight >= 50 check)
func (ls *LighthouseState) Update(elapsedMs int64, isFullDaylight bool) bool {
	// Skip if no lighthouses on map
	if !ls.HasLighthouses() {
		return false
	}

	// Skip during full daylight (original: if sight >= 50)
	if isFullDaylight {
		// Reset to uninitialized so beam restarts when darkness returns
		if ls.currentAngle != UninitializedAngle {
			ls.currentAngle = UninitializedAngle
			ls.litTilesMap = make(map[references.Position]bool)

			return true // Tiles changed (cleared)
		}

		return false
	}

	// Check timing - rotate at ~18Hz intervals
	if elapsedMs-ls.lastUpdateMs < LighthouseUpdateIntervalMs {
		return false
	}
	ls.lastUpdateMs = elapsedMs

	// Perform rotation
	ls.rotate()

	return true
}

// rotate advances the lighthouse beam by one step.
// Removes the trailing segment, advances the angle, and lights the leading segment.
func (ls *LighthouseState) rotate() {
	if ls.currentAngle == UninitializedAngle {
		// First rotation: initialize by lighting segments 0, 1, 2
		ls.currentAngle = 0
		ls.litTilesMap = make(map[references.Position]bool)

		for i := range ConeSegmentsLit {
			ls.lightShape(i)
		}
	} else {
		// Normal rotation: remove trailing edge, advance, add leading edge
		// Remove the old trailing segment
		ls.unlightShape(ls.currentAngle)

		// Advance angle
		ls.currentAngle++
		if ls.currentAngle > MaxLighthouseAngle {
			ls.currentAngle = 0
		}

		// Add the new leading segment (currentAngle + ConeSegmentsLit - 1)
		leadingSegment := (ls.currentAngle + ConeSegmentsLit - 1) % (MaxLighthouseAngle + 1)
		ls.lightShape(leadingSegment)
	}
}

// lightShape adds all tiles from the given shape index to litTilesMap.
func (ls *LighthouseState) lightShape(shapeIndex int) {
	// Wrap shape index to valid range
	shapeIndex %= MaxLighthouseAngle + 1

	shape := &lighthouseConeShapes[shapeIndex]

	for _, pos := range ls.positions {
		for _, offset := range shape {
			if offset.isTerminator() {
				break
			}
			tilePos := references.Position{
				X: pos.X + references.Coordinate(offset.X),
				Y: pos.Y + references.Coordinate(offset.Y),
			}
			// Wrap coordinates for large maps (handles positions near map edges)
			if ls.isLargeMap {
				tilePos = *tilePos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}
			ls.litTilesMap[tilePos] = true
		}
	}
}

// unlightShape removes all tiles from the given shape index from litTilesMap.
func (ls *LighthouseState) unlightShape(shapeIndex int) {
	// Wrap shape index to valid range
	shapeIndex %= MaxLighthouseAngle + 1

	shape := &lighthouseConeShapes[shapeIndex]

	for _, pos := range ls.positions {
		for _, offset := range shape {
			if offset.isTerminator() {
				break
			}
			tilePos := references.Position{
				X: pos.X + references.Coordinate(offset.X),
				Y: pos.Y + references.Coordinate(offset.Y),
			}
			// Wrap coordinates for large maps (handles positions near map edges)
			if ls.isLargeMap {
				tilePos = *tilePos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}
			delete(ls.litTilesMap, tilePos)
		}
	}
}

// IsTileLit returns true if the given position is lit by a lighthouse cone
// or is the lighthouse beacon position itself.
func (ls *LighthouseState) IsTileLit(pos references.Position) bool {
	// Check if it's the lighthouse beacon position itself
	for _, beaconPos := range ls.positions {
		if pos == beaconPos {
			return true
		}
	}
	// Check if it's in the rotating cone
	return ls.litTilesMap[pos]
}

// GetLitTiles returns a copy of the lit tiles map for external use.
func (ls *LighthouseState) GetLitTiles() map[references.Position]bool {
	result := make(map[references.Position]bool, len(ls.litTilesMap))
	for k, v := range ls.litTilesMap {
		result[k] = v
	}

	return result
}

// GetPositions returns the lighthouse beacon positions.
// Useful for debugging and testing.
func (ls *LighthouseState) GetPositions() []references.Position {
	return ls.positions
}
