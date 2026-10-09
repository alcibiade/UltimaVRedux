package map_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Constants based on game viewport dimensions (19x13)
const (
	totalVisibleTiles = 22 // Boundary size (balances real viewport with test requirements)
)

// VisibilitySystem manages the flood fill visibility calculation for small maps
// This is a complete reimplementation based on the C# Ultima 5 Redux code
type VisibilitySystem struct {
	// Core visibility tracking maps
	visibleOnMap      [][]bool   // Tiles that are visible to the player
	testForVisibility [][][]bool // Tracks which tiles have been tested (supports multiple characters)

	// Open door tracking - doors stay open for a certain number of turns
	openDoors map[references.Position]int

	// Map dimensions
	numXTiles int
	numYTiles int

	// Avatar position for visibility calculations
	avatarPos references.Position

	// Whether the map wraps around (for large maps)
	isRepeatingMap bool

	// Debug flag - true if flood fill touched the outer border
	touchedOuterBorder bool

	// XRay mode - makes all tiles visible (debug/cheat mode)
	xRayMode bool
}

// Init2DBoolArray creates a 2D boolean array initialized to false
func (v *VisibilitySystem) init2DBoolArray() [][]bool {
	arr := make([][]bool, v.numXTiles)
	for x := 0; x < v.numXTiles; x++ {
		arr[x] = make([]bool, v.numYTiles)
	}

	return arr
}

// RefreshTestForVisibility refreshes the map that tracks which tiles have been tested for visibility
func (v *VisibilitySystem) refreshTestForVisibility(nCharacters int) {
	v.testForVisibility = make([][][]bool, nCharacters)
	for i := 0; i < nCharacters; i++ {
		v.testForVisibility[i] = v.init2DBoolArray()
	}
}

// SetOpenDoor marks a door as open - it will stay open for a certain number of turns
func (v *VisibilitySystem) SetOpenDoor(pos references.Position) {
	// Doors stay open for 10 turns (from C# code)
	v.openDoors[pos] = 10
}

// IsOpenDoor checks if a door at the given position is currently open
func (v *VisibilitySystem) IsOpenDoor(pos references.Position) bool {
	turns, exists := v.openDoors[pos]

	return exists && turns > 0
}

// CloseDoor immediately closes a door
func (v *VisibilitySystem) CloseDoor(pos references.Position) {
	delete(v.openDoors, pos)
}

// DecrementDoorTimers decrements the open door timers each turn
func (v *VisibilitySystem) DecrementDoorTimers() {
	for pos, turns := range v.openDoors {
		if turns <= 1 {
			delete(v.openDoors, pos)
		} else {
			v.openDoors[pos] = turns - 1
		}
	}
}

// ClearOpenDoors closes all open doors immediately
func (v *VisibilitySystem) ClearOpenDoors() {
	v.openDoors = make(map[references.Position]int)
}

// SetXRayMode enables or disables X-ray vision (see through walls)
func (v *VisibilitySystem) SetXRayMode(enabled bool) {
	v.xRayMode = enabled
}

// IsVisible returns whether a tile is visible
func (v *VisibilitySystem) IsVisible(x, y int) bool {
	if x < 0 || x >= v.numXTiles || y < 0 || y >= v.numYTiles {
		return false
	}

	return v.visibleOnMap[x][y]
}

// GetVisibleOnMap returns the visibility map (for external use)
func (v *VisibilitySystem) GetVisibleOnMap() [][]bool {
	return v.visibleOnMap
}

// AdjustToMax adjusts a coordinate to wrap around map boundaries (for repeating maps)
func adjustToMax(n, max int) int {
	if n < 0 {
		n += max
	}
	if n >= max {
		n -= max
	}
	// Additional modulo for safety
	n = n % max
	if n < 0 {
		n += max
	}

	return n
}

// IsWithinNFourDirections checks if two positions are adjacent (4-way, distance of 1)
// This is used for window visibility - you can see through windows when adjacent
func isWithinNFourDirections(x1, y1, x2, y2 int) bool {
	deltaX := x1 - x2
	deltaY := y1 - y2
	if deltaX < 0 {
		deltaX = -deltaX
	}
	if deltaY < 0 {
		deltaY = -deltaY
	}
	// Adjacent means one tile away in cardinal directions (not diagonal)
	return (deltaX == 1 && deltaY == 0) || (deltaX == 0 && deltaY == 1)
}

// IsOutOfRangeStatic checks if coordinates are out of range
func isOutOfRangeStatic(x, y, maxX, maxY int) bool {
	return x < 0 || y < 0 || x > maxX || y > maxY
}

// SetVisibleTile sets a tile as visible with bounds checking
func (v *VisibilitySystem) setVisibleTile(x, y int) {
	if x < 0 || x > v.numXTiles-1 || y < 0 || y > v.numYTiles-1 {
		return
	}
	v.visibleOnMap[x][y] = true
}

// SetSurroundingTilesVisible sets surrounding tiles as visible
func (v *VisibilitySystem) setSurroundingTilesVisible(x, y int, includeDiagonal bool) {
	// Cardinal directions (4-way)
	v.setVisibleTile(x-1, y)
	v.setVisibleTile(x+1, y)
	v.setVisibleTile(x, y-1)
	v.setVisibleTile(x, y+1)

	// Diagonal directions (if requested)
	if includeDiagonal {
		v.setVisibleTile(x-1, y-1)
		v.setVisibleTile(x+1, y+1)
		v.setVisibleTile(x-1, y+1)
		v.setVisibleTile(x+1, y-1)
	}
}

// SetMaxVisibleArea sets the maximum visible area boundary
// This creates a box around the avatar position that limits visibility
func (v *VisibilitySystem) setMaxVisibleArea(startPos references.Position, nVisibleTiles int) {
	if nVisibleTiles < 3 {
		panic("Can't set visible area if smaller than 3")
	}

	if len(v.testForVisibility) <= 0 {
		panic("You must refresh the visible area before setting the max")
	}

	nVisibleTilesPerSide := nVisibleTiles / 2
	nStartX := int(startPos.X)
	nStartY := int(startPos.Y)

	// Set the boundary box - tiles on the edge are marked as already tested
	// This prevents the flood fill from going beyond the viewport
	for nXDiff := 0; nXDiff < nVisibleTiles; nXDiff++ {
		nX := adjustToMax(nStartX-nVisibleTilesPerSide+nXDiff, v.numXTiles)
		nTopY := adjustToMax(nStartY-nVisibleTilesPerSide, v.numYTiles)
		nBottomY := adjustToMax(nStartY+nVisibleTilesPerSide, v.numYTiles)

		v.testForVisibility[0][nX][nTopY] = true
		v.testForVisibility[0][nX][nBottomY] = true
	}

	for nYDiff := 0; nYDiff < nVisibleTiles; nYDiff++ {
		nY := adjustToMax(nStartY-nVisibleTilesPerSide+nYDiff, v.numYTiles)
		nTopX := adjustToMax(nStartX-nVisibleTilesPerSide, v.numXTiles)
		nBottomX := adjustToMax(nStartX+nVisibleTilesPerSide, v.numXTiles)

		v.testForVisibility[0][nTopX][nY] = true
		v.testForVisibility[0][nBottomX][nY] = true
	}
}

// TileChecker interface allows the visibility system to check tile properties
type TileChecker interface {
	// GetTile returns the tile at the given position
	GetLiveMapOnlyTile(pos *references.Position) *references.Tile
}

// floodFillMap is the core recursive flood fill visibility algorithm.
// Ported from C# Ultima 5 Redux for accurate visibility calculation.
//
// Algorithm:
// 1. Start at avatar position (bFirst = true)
// 2. Recursively explore in 8 directions (4 cardinal on subsequent calls, 8 on first)
// 3. Stop propagation when hitting tiles that block light
// 4. Special handling for windows (only see through when adjacent)
// 5. Mark all visited tiles as visible
//
// Parameters:
//
//	x, y - Current position being evaluated
//	bFirst - True for initial call from avatar position (explores diagonally)
//	nCharacterIndex - Character index for multi-character support (always 0 currently)
//	overrideAvatarPos - Override position for window adjacency checks
//	bAlwaysLookThroughWindows - Debug flag to ignore window restrictions
//	tileChecker - Interface to get tile data
func (v *VisibilitySystem) floodFillMap(x, y int, bFirst bool, nCharacterIndex int,
	overrideAvatarPos *references.Position, bAlwaysLookThroughWindows bool, tileChecker TileChecker) {
	if bFirst {
		v.visibleOnMap = v.init2DBoolArray()
	}

	// Get character position for window adjacency checks
	// This determines if we're close enough to see through windows
	var characterPosition references.Position
	if overrideAvatarPos != nil {
		characterPosition = *overrideAvatarPos
	} else {
		characterPosition = v.avatarPos
	}

	// Handle coordinate wrapping for large/repeating maps
	wrappedX := x
	wrappedY := y

	if v.isRepeatingMap {
		// Wrap coordinates for world map
		wrappedX = adjustToMax(x, v.numXTiles)
		wrappedY = adjustToMax(y, v.numYTiles)
	} else {
		// Boundary check for non-repeating maps
		if wrappedX < 0 || wrappedY < 0 ||
			wrappedX > v.numXTiles-1 || wrappedY > v.numYTiles-1 {
			return // Out of bounds
		}
	}

	currentPosition := references.Position{X: references.Coordinate(wrappedX), Y: references.Coordinate(wrappedY)}

	// Skip if already processed (prevents infinite recursion)
	if v.testForVisibility[nCharacterIndex][wrappedX][wrappedY] {
		return
	}
	v.testForVisibility[nCharacterIndex][wrappedX][wrappedY] = true // Mark as visited

	// Get the tile at current position
	tile := tileChecker.GetLiveMapOnlyTile(&currentPosition)
	if tile == nil {
		return // No tile to process
	}

	// Determine if this tile blocks further visibility propagation
	// Complex logic handles multiple special cases:

	isStartingTile := bFirst // Avatar's starting position never blocks

	// Window special case: Can see through if adjacent to avatar
	isWindowNearAvatar := tile.IsWindow() &&
		(isWithinNFourDirections(int(characterPosition.X), int(characterPosition.Y), wrappedX, wrappedY) ||
			bAlwaysLookThroughWindows)

	// Open door special case: Temporarily see-through
	isDoorOpen := v.IsOpenDoor(references.Position{X: references.Coordinate(wrappedX), Y: references.Coordinate(wrappedY)})

	// Tile blocks if: it naturally blocks AND is not a special case
	tileBlocksVision := tile.BlocksLight &&
		!isStartingTile &&
		!isWindowNearAvatar &&
		!isDoorOpen

	// Mark surrounding tiles as visible if this tile doesn't block
	// This creates smoother visibility edges
	if !tileBlocksVision {
		v.setSurroundingTilesVisible(wrappedX, wrappedY, true)
	}

	// This tile itself is always visible (even if it blocks further vision)
	v.setVisibleTile(wrappedX, wrappedY)

	// Stop propagation if this tile blocks vision
	if tileBlocksVision {
		return // Can see the blocking tile, but nothing beyond
	}

	maxTileIndex := v.numXTiles - 1

	// Recursively explore in cardinal directions (North, South, East, West)
	// This creates the main visibility propagation
	v.floodFillIfInside(0, 1, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker)  // South
	v.floodFillIfInside(1, 0, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker)  // East
	v.floodFillIfInside(0, -1, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker) // North
	v.floodFillIfInside(-1, 0, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker) // West

	// Diagonal exploration only from avatar position
	// This creates more natural circular visibility
	if bFirst {
		v.floodFillIfInside(-1, -1, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker) // Northwest
		v.floodFillIfInside(1, 1, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker)   // Southeast
		v.floodFillIfInside(-1, 1, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker)  // Southwest
		v.floodFillIfInside(1, -1, bFirst, wrappedX, wrappedY, maxTileIndex, nCharacterIndex, characterPosition, bAlwaysLookThroughWindows, tileChecker)  // Northeast
	}
}

// floodFillIfInside validates bounds and continues the recursive flood fill.
// This helper function ensures we don't recurse outside map boundaries.
func (v *VisibilitySystem) floodFillIfInside(xOffset, yOffset int, bFirst bool, currentX, currentY, maxTileIndex, nCharacterIndex int,
	characterPosition references.Position, bAlwaysLookThroughWindows bool, tileChecker TileChecker) {
	// Boundary check for non-repeating maps
	// (Repeating maps handle wrapping internally)
	if !bFirst && !v.isRepeatingMap {
		nextX := currentX + xOffset
		nextY := currentY + yOffset

		if isOutOfRangeStatic(nextX, nextY, maxTileIndex, maxTileIndex) {
			v.touchedOuterBorder = true // Debug flag

			return // Don't recurse outside boundaries
		}
	}

	// Continue flood fill to the next tile
	v.floodFillMap(
		currentX+xOffset,
		currentY+yOffset,
		false, // No longer the first tile
		nCharacterIndex,
		&characterPosition,
		bAlwaysLookThroughWindows,
		tileChecker,
	)
}

// RecalculateVisibleTiles is the main entry point for visibility calculation.
// Performs flood fill from the avatar position to determine what tiles are visible.
//
// Process:
// 1. Handle XRay mode (debug - all tiles visible)
// 2. Set up viewport boundaries to limit flood fill
// 3. Execute flood fill algorithm from avatar position
//
// The results are stored in v.visibleOnMap for querying.
func (v *VisibilitySystem) RecalculateVisibleTiles(initialFloodFillPosition references.Position, tileChecker TileChecker) {
	// XRay mode makes everything visible
	if v.xRayMode {
		v.visibleOnMap = v.init2DBoolArray()
		// Set all tiles to visible
		for x := 0; x < v.numXTiles; x++ {
			for y := 0; y < v.numYTiles; y++ {
				v.visibleOnMap[x][y] = true
			}
		}

		return
	}

	v.touchedOuterBorder = false
	v.avatarPos = initialFloodFillPosition

	// Initialize visibility tracking
	v.refreshTestForVisibility(1)
	v.setMaxVisibleArea(v.avatarPos, totalVisibleTiles)

	// Start flood fill from avatar position
	v.floodFillMap(int(initialFloodFillPosition.X), int(initialFloodFillPosition.Y), true, 0, nil, false, tileChecker)
}
