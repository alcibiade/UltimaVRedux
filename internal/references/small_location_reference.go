package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

type SmallLocationReference struct {
	rawData              map[int]*[XSmallMapTiles][YSmallMapTiles]byte
	Location             Location
	FriendlyLocationName string
	EnteringText         string
	SmallMapType         SmallMapMasterTypes
	ListOfFloors         []FloorNumber

	npcRefs *[]NPCReference
}

func NewSingleSmallMapReference(
	location Location,
	npcRefs *[]NPCReference,
	dataOvl *DataOvl,
) *SmallLocationReference {
	smr := &SmallLocationReference{}
	smr.Location = location
	smr.rawData = make(map[int]*[XSmallMapTiles][YSmallMapTiles]byte)

	// NOTE: this needs to be moved to a higher level
	smr.FriendlyLocationName = dataOvl.LocationNames[location]
	smr.SmallMapType = GetMapMasterFromLocation(location)
	smr.EnteringText = smr.getEnteringText()

	smr.npcRefs = npcRefs

	return smr
}

func (s *SmallLocationReference) addBlankFloor(index int) {
	// Initialize the array
	tileData := &[XSmallMapTiles][YSmallMapTiles]byte{}

	// Add the initialized array to the map at the given index
	s.rawData[index] = tileData

	s.ListOfFloors = s.getListOfFloors()
}

func (s *SmallLocationReference) GetFloorMinMax() (FloorNumber, FloorNumber) {
	if s.HasBasement() {
		return -1, FloorNumber(len(s.rawData) - 2)
	}

	return 0, FloorNumber(len(s.rawData) - 1)
}

func (s *SmallLocationReference) HasBasement() bool {
	_, ok := s.rawData[-1]

	return ok
}

func (s *SmallLocationReference) GetTileNumber(xy Position, nFloor FloorNumber) indexes.SpriteIndex {
	if s.Location.GetMapType() == DungeonMapType {
		panic("packed dungeon cells are not sprite indexes")
	}

	return indexes.SpriteIndex(s.rawData[int(nFloor)][xy.X][xy.Y])
}

// GetDungeonCell returns one immutable packed source cell.
func (s *SmallLocationReference) GetDungeonCell(floor FloorNumber, position Position) (DungeonCell, bool) {
	if s == nil || s.Location.GetMapType() != DungeonMapType || !IsDungeonCoordinate(floor, position) {
		return 0, false
	}
	floorData, ok := s.rawData[int(floor)]
	if !ok || floorData == nil {
		return 0, false
	}

	return DungeonCell(floorData[position.X][position.Y]), true
}

// CopyDungeonCells returns a detached semantic copy of all eight source floors.
func (s *SmallLocationReference) CopyDungeonCells() (DungeonCells, bool) {
	if s == nil || s.Location.GetMapType() != DungeonMapType || len(s.rawData) != DungeonFloorCount {
		return DungeonCells{}, false
	}

	var cells DungeonCells
	for floor := FloorNumber(0); floor < DungeonFloorCount; floor++ {
		for y := Coordinate(0); y < YDungeonMapTiles; y++ {
			for x := Coordinate(0); x < XDungeonMapTiles; x++ {
				position := Position{X: x, Y: y}
				cell, ok := s.GetDungeonCell(floor, position)
				if !ok {
					return DungeonCells{}, false
				}
				cells[floor][y][x] = cell
			}
		}
	}

	return cells, true
}

// func (s *SmallLocationReference) GetTileNumber(nFloor FloorNumber, x Coordinate, y Coordinate) indexes.SpriteIndex {
// 	return indexes.SpriteIndex(s.rawData[int(nFloor)][x][y])
// }

func (s *SmallLocationReference) GetEnteringText() string {
	return s.Location.String()
}

// GetExitingText returns the text to display when leaving a location
// Mirrors getEnteringText() pattern with "Leaving" instead of "Enter"
//
//nolint:cyclop // Many location cases require high complexity
func (s *SmallLocationReference) GetExitingText() string {
	switch s.Location {
	case LordBritishsCastle:
		return "Leaving the Castle of Lord British"
	case PalaceOfBlackthorn:
		return "Leaving the Palace of Lord Blackthorn"
	case Fogsbane, Stormcrow, Waveguide, Greyhaven:
		return "Leaving Lighthouse\n\n" + s.FriendlyLocationName
	case WestBritanny, EastBritanny, NorthBritanny, Paws, Cove:
		return "Leaving Village\n\n" + s.FriendlyLocationName
	case Moonglow, Britain, Jhelom, Yew, Minoc, Trinsic, SkaraBrae, NewMagincia:
		return "Leaving Towne\n\n" + s.FriendlyLocationName
	case IolosHut, GrendelsHut, SinVraalsHut, SuteksHut:
		return "Leaving Hut\n\n" + s.FriendlyLocationName
	case Ararat:
		return "Leaving Ruins\n\n" + s.FriendlyLocationName
	case Bordermarch, Farthing, Windemere, Stonegate, Lycaeum, EmpathAbbey, SerpentsHold, BuccaneersDen:
		return "Leaving Keep\n\n" + s.FriendlyLocationName
	case Deceit, Despise, Destard, Wrong, Covetous, Shame, Hythloth, Doom:
		return "Leaving Dungeon\n\n" + s.FriendlyLocationName
	case EmptyLocation, CampingLocation, BritanniaUnderworld, CombatRestingShrine:
		// Non-interactive locations use generic message
		return "Leaving..."
	}

	return "Leaving..."
}

func (s *SmallLocationReference) getEnteringText() string {
	switch s.Location {
	case LordBritishsCastle:
		return "Enter the Castle of Lord British!"
	case PalaceOfBlackthorn:
		return "Enter the Palace of Lord Blackthorn"
	case Fogsbane, Stormcrow, Waveguide, Greyhaven:
		return "Enter Lighthouse\n\n" + s.FriendlyLocationName
	case WestBritanny, EastBritanny, NorthBritanny, Paws, Cove:
		return "Enter Village\n\n" + s.FriendlyLocationName
	case Moonglow, Britain, Jhelom, Yew, Minoc, Trinsic, SkaraBrae, NewMagincia:
		return "Enter Towne\n\n" + s.FriendlyLocationName
	case IolosHut, GrendelsHut, SinVraalsHut, SuteksHut:
		return "Enter Hut\n\n" + s.FriendlyLocationName
	case Ararat:
		return "Enter Ruins\n\n" + s.FriendlyLocationName
	case Bordermarch, Farthing, Windemere, Stonegate, Lycaeum, EmpathAbbey, SerpentsHold, BuccaneersDen:
		return "Enter Keep\n\n" + s.FriendlyLocationName
	case Deceit, Despise, Destard, Wrong, Covetous, Shame, Hythloth, Doom:
		return "Enter Dungeon\n\n" + s.FriendlyLocationName
	}

	return "NOT IMPLEMENTED"
}

func (s *SmallLocationReference) GetOuterTile() indexes.SpriteIndex {
	switch s.Location {
	case SinVraalsHut:
		return indexes.Desert1
	case GrendelsHut:
		return indexes.Swamp
	default:
		return indexes.Grass
	}
}

func (s *SmallLocationReference) GetNumberOfFloors() int {
	return len(s.rawData)
}

func (s *SmallLocationReference) GetMaxY() Coordinate {
	return s.GetMaxX()
}

func (s *SmallLocationReference) GetMaxX() Coordinate {
	switch s.Location.GetMapType() {
	case LargeMapType:
		return XLargeMapTiles - 1
	case SmallMapType:
		return XSmallMapTiles - 1
	case DungeonMapType:
		return XDungeonMapTiles - 1
	case CombatMapType, CampingMapType:
		// SmallLocationReference does not represent these runtime-only maps.
	}

	panic("missing max tiles")
}

func (s *SmallLocationReference) getListOfFloors() []FloorNumber {
	numFloors := s.GetNumberOfFloors()
	startIndex := FloorNumber(0)

	if s.HasBasement() {
		startIndex = -1
	}

	// Initialize the floors slice with values directly
	floors := make([]FloorNumber, numFloors)
	for i := FloorNumber(0); i < FloorNumber(numFloors); i++ {
		floors[i] = startIndex + i
	}

	return floors
}

func (s *SmallLocationReference) CanGoUpOneFloor(currentFloor FloorNumber) bool {
	floorIndex := 0
	if s.HasBasement() {
		floorIndex = -1
	}
	nextFloor := currentFloor + 1

	return int(nextFloor) < s.GetNumberOfFloors()+floorIndex
}

func (s *SmallLocationReference) CanGoDownOneFloor(currentFloor FloorNumber) bool {
	if currentFloor < 0 {
		return s.HasBasement()
	}

	return true
}

func (s *SmallLocationReference) GetNPCReferences() *[]NPCReference {
	return s.npcRefs
}

func (s *SmallLocationReference) GetClosestLadder(npcCurrentPosition Position, nCurrentFloor, nTargetFloor FloorNumber) Position {
	ladderOrStairType := LadderOrStairDown
	if nCurrentFloor < nTargetFloor {
		ladderOrStairType = LadderOrStairUp
	}
	ls := s.getListOfAllLaddersAndStairs(nCurrentFloor, ladderOrStairType)

	bestPosition := Position{}
	for _, pos := range ls {
		if bestPosition.IsZeros() {
			bestPosition = pos

			continue
		}
		if npcCurrentPosition.HeuristicTileDistance(pos) < npcCurrentPosition.HeuristicTileDistance(bestPosition) {
			bestPosition = pos
		}
	}

	if bestPosition.IsZeros() {
		panic("every NPC should have a ladder or stair close to them")
	}

	return bestPosition
}

func (s *SmallLocationReference) getListOfAllLaddersAndStairs(nFloor FloorNumber, ladderOrStairType LadderOrStairType) []Position {
	positions := make([]Position, 0)

	for x := Coordinate(0); x < s.GetMaxX(); x++ {
		for y := Coordinate(0); y < s.GetMaxY(); y++ {
			pos := Position{X: x, Y: y}
			isLadderOrStair := IsSpecificLadderOrStairs(s.GetTileNumber(pos, nFloor), ladderOrStairType)
			if isLadderOrStair {
				positions = append(positions, pos)
			}
		}
	}

	return positions
}
