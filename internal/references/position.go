package references

import (
	"fmt"
	"math"

	"github.com/bradhannah/UltimaVRedux/pkg/helpers"
)

// Coordinate is one axis of a tile position; it is signed so that off-map moves can be wrapped.
type Coordinate int16

// Position is a tile on a map.
type Position struct {
	X Coordinate `json:"x" yaml:"x"`
	Y Coordinate `json:"y" yaml:"y"`
}

// FormatBritannianCoordinates encodes Y then X using the original runic A-P nibbles.
func FormatBritannianCoordinates(x, y Coordinate) string {
	encode := func(value Coordinate) string {
		wrapped := byte(value)

		return fmt.Sprintf("%c'%c\"", 'A'+wrapped>>4, 'A'+wrapped&0x0f)
	}

	return encode(y) + ", " + encode(x)
}

func move(val, mapSize Coordinate, decrease, bWrap bool) Coordinate {
	if decrease {
		if val == 0 {
			if bWrap {
				return mapSize - 1
			}

			return 0
		}

		return val - 1
	}

	if val+1 >= mapSize {
		if bWrap {
			return 0
		}

		return val
	}

	return val + 1
}

// GetPositionToLeft returns a new position one tile to the left, without wrapping.
func (p *Position) GetPositionToLeft() *Position {
	return &Position{p.X - 1, p.Y}
}

// GetPositionToRight returns a new position one tile to the right, without wrapping.
func (p *Position) GetPositionToRight() *Position {
	return &Position{p.X + 1, p.Y}
}

// GetPositionUp returns a new position one tile up, without wrapping.
func (p *Position) GetPositionUp() *Position {
	return &Position{p.X, p.Y - 1}
}

// GetPositionDown returns a new position one tile down, without wrapping.
func (p *Position) GetPositionDown() *Position {
	return &Position{p.X, p.Y + 1}
}

// Left returns a new Position one tile to the left (X-1).
// Does not modify the original Position.
func (p Position) Left() Position {
	return Position{X: p.X - 1, Y: p.Y}
}

// Right returns a new Position one tile to the right (X+1).
// Does not modify the original Position.
func (p Position) Right() Position {
	return Position{X: p.X + 1, Y: p.Y}
}

// Up returns a new Position one tile up (Y-1).
// Does not modify the original Position.
func (p Position) Up() Position {
	return Position{X: p.X, Y: p.Y - 1}
}

// Down returns a new Position one tile down (Y+1).
// Does not modify the original Position.
func (p Position) Down() Position {
	return Position{X: p.X, Y: p.Y + 1}
}

// NOTE: These position calculation methods can experience integer overflow at extreme coordinate values
// (int16 max ±32767). However, this is not a practical concern since game coordinates are typically 0-255.
// The methods are designed for simplicity and performance within normal gameplay boundaries.

// GetFourDirectionsWrapped returns the left, right, up and down neighbours, wrapped to a maxX by maxY map.
func (p *Position) GetFourDirectionsWrapped(maxX, maxY Coordinate) []Position {
	return []Position{
		{X: move(p.X, maxX, true, true), Y: p.Y},  // Left
		{X: move(p.X, maxX, false, true), Y: p.Y}, // Right
		{X: p.X, Y: move(p.Y, maxY, true, true)},  // Up
		{X: p.X, Y: move(p.Y, maxY, false, true)}, // Down
	}
}

// GetWrappedDistanceBetweenWrapped returns the straight-line distance to p2 on a maxX by maxY map that wraps at its edges.
func (p *Position) GetWrappedDistanceBetweenWrapped(p2 *Position, maxX, maxY Coordinate) float64 {
	dxRaw := float64(helpers.AbsInt(int(p.X - p2.X)))
	dyRaw := float64(helpers.AbsInt(int(p.Y - p2.Y)))
	dx := math.Min(dxRaw, float64(maxX)-dxRaw)
	dy := math.Min(dyRaw, float64(maxY)-dyRaw)

	return math.Sqrt(dx*dx + dy*dy)
}

// GoLeft moves one tile left in place, wrapping on the large map or stopping at the edge on small maps.
func (p *Position) GoLeft(bLarge bool) {
	mapSize := XSmallMapTiles
	if bLarge {
		mapSize = XLargeMapTiles
	}
	p.X = move(p.X, mapSize, true, bLarge)
}

// GoRight moves one tile right in place, wrapping on the large map or stopping at the edge on small maps.
func (p *Position) GoRight(bLarge bool) {
	mapSize := XSmallMapTiles
	if bLarge {
		mapSize = XLargeMapTiles
	}
	p.X = move(p.X, mapSize, false, bLarge)
}

// GoUp moves one tile up in place, wrapping on the large map or stopping at the edge on small maps.
func (p *Position) GoUp(bLarge bool) {
	mapSize := YSmallMapTiles
	if bLarge {
		mapSize = YLargeMapTiles
	}
	p.Y = move(p.Y, mapSize, true, bLarge)
}

// GoDown moves one tile down in place, wrapping on the large map or stopping at the edge on small maps.
func (p *Position) GoDown(bLarge bool) {
	mapSize := YSmallMapTiles
	if bLarge {
		mapSize = YLargeMapTiles
	}
	p.Y = move(p.Y, mapSize, false, bLarge)
}

// Equals reports whether both positions have the same X and Y.
func (p *Position) Equals(position *Position) bool {
	return p.X == position.X && p.Y == position.Y
}

// GetWrapped wraps the position in place into a maxX by maxY map and returns it.
func (p *Position) GetWrapped(maxX, maxY Coordinate) *Position {
	if p.X < 0 {
		p.X += maxX
	} else if p.X >= maxX {
		p.X %= maxX
	}

	if p.Y < 0 {
		p.Y += maxY
	} else if p.Y >= maxY {
		p.Y %= maxY
	}

	return p
}

// AnimationHash packs an X/Y position without overflowing Coordinate's int16
// representation. Animation code decodes X by dividing the result by 256.
func (p Position) AnimationHash() int32 {
	return int32(p.X)*256 + int32(p.Y)
}

// IsWithinN reports whether targetPosition is within n tiles on both axes.
func (p *Position) IsWithinN(targetPosition *Position, n int) bool {
	return helpers.AbsInt(int(targetPosition.X-p.X)) <= n && helpers.AbsInt(int(targetPosition.Y-p.Y)) <= n
}

// Neighbors returns the left, right, up and down positions without wrapping.
func (p *Position) Neighbors() []Position {
	return []Position{
		{X: p.X - 1, Y: p.Y}, // Left
		{X: p.X + 1, Y: p.Y}, // Right
		{X: p.X, Y: p.Y - 1}, // Up
		{X: p.X, Y: p.Y + 1}, // Down
	}
}

// HeuristicTileDistance returns the Manhattan distance to b, for path finding.
func (p *Position) HeuristicTileDistance(b Position) int {
	return helpers.AbsInt(int(p.X-b.X)) + helpers.AbsInt(int(p.Y-b.Y))
}

// IsNextTo reports whether both positions are orthogonally adjacent on the same floor.
func (p *Position) IsNextTo(position Position, floor1 FloorNumber, floor2 FloorNumber) bool {
	// Must be on same floor AND orthogonally adjacent (not diagonal)
	// Orthogonal adjacency: either same X with Y±1, or same Y with X±1
	return floor1 == floor2 &&
		((p.X == position.X && helpers.AbsInt(int(p.Y-position.Y)) == 1) ||
			(p.Y == position.Y && helpers.AbsInt(int(p.X-position.X)) == 1))
}

// GetDominateDirection determines the dominant directional axis (horizontal or vertical) from the
// current position to another.
func (p *Position) GetDominateDirection(to Position) Direction {
	deltaX := int(to.X - p.X)
	deltaY := int(to.Y - p.Y)

	if helpers.AbsInt(deltaX) > helpers.AbsInt(deltaY) {
		if deltaX < 0 {
			return Left
		}

		return Right
	}

	if deltaY < 0 {
		return Up
	}

	return Down
}

// GetWrappedAxisDistances returns the wrapped X and Y distances on a large map (256x256).
// Handles wraparound so distances > 128 are converted to their shorter path.
// Used for spawn distance checks where each axis must be validated separately.
func (p *Position) GetWrappedAxisDistances(other Position) (int, int) {
	distX := helpers.AbsInt(int(p.X - other.X))
	distY := helpers.AbsInt(int(p.Y - other.Y))

	// Handle wraparound for large map (256x256)
	if distX > 128 {
		distX = 256 - distX
	}

	if distY > 128 {
		distY = 256 - distY
	}

	return distX, distY
}
