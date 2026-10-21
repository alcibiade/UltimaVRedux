package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// FloorNumber is a floor of a map: negative below ground, 0 on the surface, positive above.
type FloorNumber int8

// Named floors: Basement is below the MainFloor.
//
//goland:noinspection GoUnusedConst
const (
	Basement  FloorNumber = -1
	MainFloor FloorNumber = 0
)

// LadderOrStairType is the direction a ladder or staircase is taken in.
type LadderOrStairType int

// Ladder and stair directions; NotLadderOrStair means no floor change is being made.
const (
	NotLadderOrStair LadderOrStairType = iota
	LadderOrStairUp
	LadderOrStairDown
)

// GetLadderOfStairsType returns the ladder or stair direction needed to go from currentFloor to targetFloor.
func GetLadderOfStairsType(currentFloor, targetFloor FloorNumber) LadderOrStairType {
	if currentFloor == targetFloor {
		return NotLadderOrStair
	}

	if targetFloor > currentFloor {
		return LadderOrStairUp
	}

	return LadderOrStairDown
}

// IsSpecificLadderOrStairs reports whether the sprite is a ladder or staircase that goes the given way.
func IsSpecificLadderOrStairs(s indexes.SpriteIndex, ladderOrStairType LadderOrStairType) bool {
	switch ladderOrStairType {
	case LadderOrStairUp:
		return s == indexes.LadderUp || s == indexes.StairsNorth || s == indexes.StairsEast
	case LadderOrStairDown:
		return s == indexes.LadderDown || s == indexes.StairsSouth || s == indexes.StairsWest
	case NotLadderOrStair:
	}

	return false
}
