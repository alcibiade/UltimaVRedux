package references

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const generatedDungeonCombatMapIndex = -1

// ErrInvalidGeneratedDungeonCombatMap indicates invalid synthetic map inputs.
var ErrInvalidGeneratedDungeonCombatMap = errors.New("invalid generated dungeon combat map")

// Center tiles for each generated dungeon cell family; ref:DNG-0010.
//
//nolint:gochecknoglobals // Immutable original data indexed by dungeon-cell high nibble.
var generatedDungeonCenterTiles = [8]byte{0, 0xc8, 0xc9, 0xc8, 0xdc, 0xd8, 0, 0}

// Corridor encounter party coordinates, transformed by facing; ref:DNG-0010.
// Indexed by DungeonFacing.
//
//nolint:gochecknoglobals // Immutable original formation data.
var generatedDungeonPartyPositions = [4][MaxPlayersPerMap]Position{
	{{X: 5, Y: 6}, {X: 4, Y: 7}, {X: 6, Y: 7}, {X: 3, Y: 8}, {X: 5, Y: 8}, {X: 7, Y: 8}},
	{{X: 4, Y: 5}, {X: 3, Y: 4}, {X: 3, Y: 6}, {X: 2, Y: 3}, {X: 2, Y: 5}, {X: 2, Y: 7}},
	{{X: 5, Y: 4}, {X: 4, Y: 3}, {X: 6, Y: 3}, {X: 3, Y: 2}, {X: 5, Y: 2}, {X: 7, Y: 2}},
	{{X: 6, Y: 5}, {X: 7, Y: 4}, {X: 7, Y: 6}, {X: 8, Y: 3}, {X: 8, Y: 5}, {X: 8, Y: 7}},
}

// Corridor encounter monster coordinates, transformed by facing; ref:DNG-0010.
// Indexed by DungeonFacing.
//
//nolint:gochecknoglobals // Immutable original formation data.
var generatedDungeonMonsterPositions = [4][MaxMonstersPerMap]Position{
	{
		{X: 5, Y: 2}, {X: 4, Y: 2}, {X: 6, Y: 2}, {X: 3, Y: 3},
		{X: 7, Y: 3}, {X: 2, Y: 4}, {X: 8, Y: 4}, {X: 5, Y: 1},
		{X: 2, Y: 2}, {X: 8, Y: 2}, {X: 3, Y: 1}, {X: 7, Y: 1},
		{X: 2, Y: 0}, {X: 4, Y: 0}, {X: 6, Y: 0}, {X: 8, Y: 0},
	},
	{
		{X: 8, Y: 5}, {X: 8, Y: 4}, {X: 8, Y: 6}, {X: 7, Y: 3},
		{X: 7, Y: 7}, {X: 6, Y: 2}, {X: 6, Y: 8}, {X: 9, Y: 5},
		{X: 8, Y: 2}, {X: 8, Y: 8}, {X: 9, Y: 7}, {X: 9, Y: 3},
		{X: 10, Y: 2}, {X: 10, Y: 4}, {X: 10, Y: 6}, {X: 10, Y: 8},
	},
	{
		{X: 5, Y: 8}, {X: 4, Y: 8}, {X: 6, Y: 8}, {X: 3, Y: 7},
		{X: 7, Y: 7}, {X: 2, Y: 6}, {X: 8, Y: 6}, {X: 5, Y: 9},
		{X: 2, Y: 8}, {X: 8, Y: 8}, {X: 7, Y: 9}, {X: 3, Y: 9},
		{X: 2, Y: 10}, {X: 4, Y: 10}, {X: 6, Y: 10}, {X: 8, Y: 10},
	},
	{
		{X: 2, Y: 5}, {X: 2, Y: 4}, {X: 2, Y: 6}, {X: 3, Y: 3},
		{X: 3, Y: 7}, {X: 4, Y: 2}, {X: 4, Y: 8}, {X: 1, Y: 5},
		{X: 2, Y: 2}, {X: 2, Y: 8}, {X: 1, Y: 3}, {X: 1, Y: 7},
		{X: 0, Y: 2}, {X: 0, Y: 4}, {X: 0, Y: 6}, {X: 0, Y: 8},
	},
}

// Central party positions for generated camping encounters; ref:DNG-0010.
//
//nolint:gochecknoglobals // Immutable source formation shared by generated maps and tests.
var generatedDungeonCampingPartyPositions = [MaxPlayersPerMap]Position{
	{X: 5, Y: 4}, {X: 6, Y: 5}, {X: 5, Y: 6},
	{X: 4, Y: 5}, {X: 4, Y: 4}, {X: 6, Y: 6},
}

// Surrounding monster positions for generated camping encounters; ref:DNG-0010.
//
//nolint:gochecknoglobals // Immutable source formation shared by generated maps and tests.
var generatedDungeonCampingMonsterPositions = [MaxMonstersPerMap]Position{
	{X: 2, Y: 4}, {X: 2, Y: 5}, {X: 2, Y: 7}, {X: 3, Y: 2},
	{X: 3, Y: 3}, {X: 3, Y: 8}, {X: 4, Y: 8}, {X: 5, Y: 3},
	{X: 5, Y: 8}, {X: 6, Y: 3}, {X: 7, Y: 4}, {X: 7, Y: 7},
	{X: 7, Y: 8}, {X: 8, Y: 3}, {X: 8, Y: 6}, {X: 8, Y: 7},
}

// BuildGeneratedDungeonCombatMap builds encounter terrain and walls (ref:DNG-0010) from
// detached dungeon cells. It does not allocate DUNGEON.CBT trigger slots.
func BuildGeneratedDungeonCombatMap(
	location Location,
	cells *DungeonCells,
	floor FloorNumber,
	position Position,
	facing DungeonFacing,
) (*CombatMapReference, error) {
	if location < Deceit || location > Doom {
		return nil, fmt.Errorf("%w: location %s is not a dungeon", ErrInvalidGeneratedDungeonCombatMap, location)
	}
	if cells == nil || !IsDungeonCoordinate(floor, position) {
		return nil, fmt.Errorf("%w: source cell is unavailable", ErrInvalidGeneratedDungeonCombatMap)
	}
	if !facing.IsValid() {
		return nil, fmt.Errorf("%w: facing %d", ErrInvalidGeneratedDungeonCombatMap, facing)
	}

	center, _ := cells.Cell(floor, position)
	wallTile, floorTile := generatedDungeonCombatStyle(location)
	rawData := &[XCombatMapTiles][YCombatMapTiles]byte{}
	for y := range YCombatMapTiles {
		for x := range XCombatMapTiles {
			rawData[x][y] = floorTile
		}
	}

	generatedDungeonBaseWalls(rawData, wallTile)
	centerKind := center.Raw() & 0xf0
	if centerKind != 0 && centerKind < 0x80 {
		centerTile := generatedDungeonCenterTiles[centerKind>>4]
		if centerTile != 0 {
			rawData[5][5] = centerTile
		}
	}

	directions := [...]struct{ dx, dy int }{{dy: -1}, {dx: 1}, {dy: 1}, {dx: -1}}
	for direction, delta := range directions {
		neighborPosition := Position{
			X: Coordinate((int(position.X) + delta.dx + int(XDungeonMapTiles)) % int(XDungeonMapTiles)),
			Y: Coordinate((int(position.Y) + delta.dy + int(YDungeonMapTiles)) % int(YDungeonMapTiles)),
		}
		neighbor, _ := cells.Cell(floor, neighborPosition)
		generatedDungeonApplyNeighbor(rawData, wallTile, floorTile, centerKind, neighbor.Raw()&0xf0, direction)
	}

	combatMap := NewCombatMapReference(generatedDungeonCombatMapIndex, DungeonCombatTerritory)
	combatMap.MapName = "Dungeon Wanderer Combat"
	combatMap.SetRawData(rawData)
	if centerKind == 0x40 && generatedDungeonCenterTiles[centerKind>>4] == 0xdc {
		combatMap.generatedDungeonChest = &GeneratedDungeonChest{
			Position: Position{X: 5, Y: 5}, SourceFloor: floor,
			FloorTile: indexes.SpriteIndex(floorTile),
		}
	}
	if center.Kind() == DungeonLadderBoth {
		combatMap.dungeonDualLadderPosition = Position{X: 5, Y: 5}
		combatMap.hasDungeonDualLadder = true
	}
	partyDirectionIndexes := [...]int{2, 1, 3, 0} // CBT rows for N, E, S, W.
	for dungeonFacing, positions := range generatedDungeonPartyPositions {
		combatMap.PlayerPositions[partyDirectionIndexes[dungeonFacing]] = positions
	}
	combatMap.MonsterPositions = generatedDungeonMonsterPositions[facing]

	return combatMap, nil
}

// BuildGeneratedDungeonCampingMap builds the source-defined dungeon camp arena
// without allocating fixed-room trigger slots.
func BuildGeneratedDungeonCampingMap(
	location Location,
	cells *DungeonCells,
	floor FloorNumber,
	position Position,
	facing DungeonFacing,
) (*CombatMapReference, error) {
	combatMap, err := BuildGeneratedDungeonCombatMap(location, cells, floor, position, facing)
	if err != nil {
		return nil, err
	}
	center, _ := cells.Cell(floor, position)
	centerKind := center.Raw() & 0xf0
	if centerKind == 0 || centerKind >= 0x80 || generatedDungeonCenterTiles[centerKind>>4] == 0 {
		combatMap.SetTileNumber(Position{X: 5, Y: 5}, indexes.CampFire)
	}
	combatMap.MapName = "Dungeon Camping"
	for direction := range NumDirections {
		combatMap.PlayerPositions[direction] = generatedDungeonCampingPartyPositions
	}
	combatMap.MonsterPositions = generatedDungeonCampingMonsterPositions

	return combatMap, nil
}

func generatedDungeonCombatStyle(location Location) (byte, byte) {
	//nolint:exhaustive // The builder validates a dungeon location; all unlisted dungeons share one style.
	switch location {
	case Deceit, Wrong, Covetous:
		return 0x4f, 0x45
	default:
		return 0x4d, 0x05
	}
}

func generatedDungeonBaseWalls(data *[XCombatMapTiles][YCombatMapTiles]byte, wallTile byte) {
	for coordinate := range XCombatMapTiles {
		data[coordinate][1] = wallTile
		data[coordinate][9] = wallTile
		data[1][coordinate] = wallTile
		data[9][coordinate] = wallTile
	}
	data[0][0] = 0xff
	data[10][0] = 0xff
	data[0][10] = 0xff
	data[10][10] = 0xff
}

func generatedDungeonApplyNeighbor(
	data *[XCombatMapTiles][YCombatMapTiles]byte,
	wallTile, floorTile, centerKind, neighborKind byte,
	direction int,
) {
	if neighborKind < 0xa0 {
		generatedDungeonMakeOpening(data, floorTile, direction, 2, 7)
	}
	if neighborKind == 0xb0 || neighborKind == 0xc0 || neighborKind == 0xd0 {
		generatedDungeonMakeWall(data, wallTile, centerKind, direction)

		return
	}
	generatedDungeonMakeOpening(data, floorTile, direction, 3, 5)
}

func generatedDungeonMakeOpening(
	data *[XCombatMapTiles][YCombatMapTiles]byte,
	floorTile byte,
	direction, start, count int,
) {
	for offset := range count {
		coordinate := start + offset
		switch direction {
		case 0:
			data[coordinate][1] = floorTile
		case 1:
			data[9][coordinate] = floorTile
		case 2:
			data[coordinate][9] = floorTile
		case 3:
			data[1][coordinate] = floorTile
		default:
			panic("invalid generated dungeon combat direction")
		}
	}
}

func generatedDungeonMakeWall(
	data *[XCombatMapTiles][YCombatMapTiles]byte,
	wallTile, centerKind byte,
	direction int,
) {
	for coordinate := range XCombatMapTiles {
		switch direction {
		case 0:
			data[coordinate][0] = 0xff
		case 1:
			data[10][coordinate] = 0xff
		case 2:
			data[coordinate][10] = 0xff
		case 3:
			data[0][coordinate] = 0xff
		default:
			panic("invalid generated dungeon combat direction")
		}
	}
	if direction == 0 && centerKind == 0xe0 {
		data[5][2] = wallTile
		data[5][8] = wallTile
	}
	if direction == 3 && centerKind == 0xe0 {
		data[2][5] = wallTile
		data[8][5] = wallTile
	}
}
