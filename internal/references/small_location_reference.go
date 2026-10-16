package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// SmallLocationReference holds authored map floors, location metadata and NPC references.
type SmallLocationReference struct {
	rawData              map[int]*[XSmallMapTiles][YSmallMapTiles]byte
	Location             Location
	FriendlyLocationName string
	EnteringText         string
	SmallMapType         SmallMapMasterTypes
	ListOfFloors         []FloorNumber

	npcRefs *[]NPCReference
}

// NewSingleSmallMapReference initializes location metadata before floor data is loaded.
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

// GetFloorMinMax returns the inclusive floor range, accounting for a basement.
func (s *SmallLocationReference) GetFloorMinMax() (FloorNumber, FloorNumber) {
	if s.HasBasement() {
		return -1, FloorNumber(len(s.rawData) - 2)
	}

	return 0, FloorNumber(len(s.rawData) - 1)
}

// HasBasement reports whether floor -1 has authored map data.
func (s *SmallLocationReference) HasBasement() bool {
	_, ok := s.rawData[-1]

	return ok
}

// GetTileNumber returns an authored sprite index; packed dungeon cells use GetDungeonCell.
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

// GetEnteringText returns the location's display name.
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
	case EmptyLocation, CampingLocation, BritanniaUnderworld, CombatRestingShrine,
		ShrineLocation, CodexLocation, CombatLocation, BlackthornPrison, FinaleLocation:
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
	case EmptyLocation, CampingLocation, ShrineLocation, CodexLocation, CombatLocation,
		BlackthornPrison, FinaleLocation, BritanniaUnderworld, CombatRestingShrine:
		// These locations have no authored small-map entry message.
	}

	return "NOT IMPLEMENTED"
}

// GetOuterTile returns the terrain surrounding the small map.
func (s *SmallLocationReference) GetOuterTile() indexes.SpriteIndex {
	//nolint:exhaustive // Only these two locations override the default grass surround.
	switch s.Location {
	case SinVraalsHut:
		return indexes.Desert1
	case GrendelsHut:
		return indexes.Swamp
	default:
		return indexes.Grass
	}
}

// GetNumberOfFloors returns the number of loaded authored floors.
func (s *SmallLocationReference) GetNumberOfFloors() int {
	return len(s.rawData)
}

// GetMaxY returns the inclusive vertical bound of the square map.
func (s *SmallLocationReference) GetMaxY() Coordinate {
	return s.GetMaxX()
}

// GetMaxX returns the inclusive horizontal bound for the location's map type.
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

// CanGoUpOneFloor reports whether the next higher floor is below the loaded upper bound.
func (s *SmallLocationReference) CanGoUpOneFloor(currentFloor FloorNumber) bool {
	floorIndex := 0
	if s.HasBasement() {
		floorIndex = -1
	}
	nextFloor := currentFloor + 1

	return int(nextFloor) < s.GetNumberOfFloors()+floorIndex
}

// CanGoDownOneFloor permits nonnegative floors and requires a basement for negative floors.
func (s *SmallLocationReference) CanGoDownOneFloor(currentFloor FloorNumber) bool {
	if currentFloor < 0 {
		return s.HasBasement()
	}

	return true
}

// GetNPCReferences returns the location's authored NPC reference collection.
func (s *SmallLocationReference) GetNPCReferences() *[]NPCReference {
	return s.npcRefs
}

// GetClosestLadder returns the closest authored directional connection, or
// (-1,-1) when none exists. Runtime route selection must additionally check
// persistent terrain overrides, occupancy and reachability.
func (s *SmallLocationReference) GetClosestLadder(npcCurrentPosition Position, nCurrentFloor, nTargetFloor FloorNumber) Position {
	ladderOrStairType := GetLadderOfStairsType(nCurrentFloor, nTargetFloor)
	ls := s.getListOfAllLaddersAndStairs(nCurrentFloor, ladderOrStairType)

	if len(ls) == 0 {
		// Runtime routes must handle missing connections as pending travel.
		return Position{X: -1, Y: -1}
	}
	bestPosition := ls[0]
	for _, pos := range ls[1:] {
		if npcCurrentPosition.HeuristicTileDistance(pos) < npcCurrentPosition.HeuristicTileDistance(bestPosition) {
			bestPosition = pos
		}
	}

	return bestPosition
}

func (s *SmallLocationReference) getListOfAllLaddersAndStairs(nFloor FloorNumber, ladderOrStairType LadderOrStairType) []Position {
	positions := make([]Position, 0)
	if s.rawData[int(nFloor)] == nil {
		return positions
	}

	for x := Coordinate(0); x <= s.GetMaxX(); x++ {
		for y := Coordinate(0); y <= s.GetMaxY(); y++ {
			pos := Position{X: x, Y: y}
			isLadderOrStair := IsSpecificLadderOrStairs(s.GetTileNumber(pos, nFloor), ladderOrStairType)
			if isLadderOrStair {
				positions = append(positions, pos)
			}
		}
	}

	return positions
}
