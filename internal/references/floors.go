package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

type FloorNumber int8

//goland:noinspection GoUnusedConst
const (
	Basement  FloorNumber = -1
	MainFloor FloorNumber = 0
)

type LadderOrStairType int

const (
	NotLadderOrStair LadderOrStairType = iota
	LadderOrStairUp
	LadderOrStairDown
)

func GetLadderOfStairsType(currentFloor, targetFloor FloorNumber) LadderOrStairType {
	if currentFloor == targetFloor {
		return NotLadderOrStair
	}

	if targetFloor > currentFloor {
		return LadderOrStairUp
	}

	return LadderOrStairDown
}

func IsSpecificLadderOrStairs(s indexes.SpriteIndex, ladderOrStairType LadderOrStairType) bool {
	switch ladderOrStairType {
	case LadderOrStairUp:
		return s == indexes.LadderUp || s == indexes.StairsNorth || s == indexes.StairsEast
	case LadderOrStairDown:
		return s == indexes.LadderDown || s == indexes.StairsSouth || s == indexes.StairsWest
	}

	return false
}
