package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestBuildGeneratedDungeonCombatMapExactGeometryWithWrappedNeighbors(t *testing.T) {
	t.Parallel()

	var cells DungeonCells
	position := Position{X: 0, Y: 0}
	cells.SetCell(0, position, 0xe0)
	cells.SetCell(0, Position{X: 0, Y: 7}, 0xb0) // Wrapped north: wall.
	cells.SetCell(0, Position{X: 1, Y: 0}, 0xa0) // East: door.
	cells.SetCell(0, Position{X: 0, Y: 1}, 0x00) // South: hall then door.
	cells.SetCell(0, Position{X: 7, Y: 0}, 0xd0) // Wrapped west: wall.

	combatMap, err := BuildGeneratedDungeonCombatMap(Deceit, &cells, 0, position, DungeonFacingNorth)
	require.NoError(t, err)

	const (
		floor = byte(0x45)
		wall  = byte(0x4f)
		none  = byte(0xff)
	)
	wantRows := [YCombatMapTiles][XCombatMapTiles]byte{
		{none, none, none, none, none, none, none, none, none, none, none},
		{none, wall, wall, wall, wall, wall, wall, wall, wall, wall, wall},
		{none, wall, floor, floor, floor, wall, floor, floor, floor, wall, floor},
		{none, wall, floor, floor, floor, floor, floor, floor, floor, floor, floor},
		{none, wall, floor, floor, floor, floor, floor, floor, floor, floor, floor},
		{none, wall, wall, floor, floor, floor, floor, floor, wall, floor, floor},
		{none, wall, floor, floor, floor, floor, floor, floor, floor, floor, floor},
		{none, wall, floor, floor, floor, floor, floor, floor, floor, floor, floor},
		{none, wall, floor, floor, floor, wall, floor, floor, floor, wall, floor},
		{none, wall, floor, floor, floor, floor, floor, floor, floor, wall, wall},
		{none, wall, floor, floor, floor, floor, floor, floor, floor, wall, none},
	}
	for y, row := range wantRows {
		for x, want := range row {
			assert.Equal(t, want, byte(combatMap.GetTileNumber(Position{X: Coordinate(x), Y: Coordinate(y)})),
				"tile (%d,%d)", x, y)
		}
	}
	_, triggerSlotsPresent := combatMap.DungeonTriggerSlots()
	assert.False(t, triggerSlotsPresent, "generated maps have no DUNGEON.CBT trigger slots")
}

func TestBuildGeneratedDungeonCombatMapStylesAndCenterFeatures(t *testing.T) {
	t.Parallel()

	styles := []struct {
		location Location
		wall     byte
		floor    byte
	}{
		{location: Deceit, wall: 0x4f, floor: 0x45},
		{location: Despise, wall: 0x4d, floor: 0x05},
		{location: Destard, wall: 0x4d, floor: 0x05},
		{location: Wrong, wall: 0x4f, floor: 0x45},
		{location: Covetous, wall: 0x4f, floor: 0x45},
		{location: Shame, wall: 0x4d, floor: 0x05},
		{location: Hythloth, wall: 0x4d, floor: 0x05},
		{location: Doom, wall: 0x4d, floor: 0x05},
	}
	for _, style := range styles {
		var cells DungeonCells
		combatMap, err := BuildGeneratedDungeonCombatMap(
			style.location, &cells, 0, Position{X: 4, Y: 4}, DungeonFacingNorth,
		)
		require.NoError(t, err)
		assert.Equal(t, style.wall, byte(combatMap.GetTileNumber(Position{X: 1, Y: 1})), style.location.String())
		assert.Equal(t, style.floor, byte(combatMap.GetTileNumber(Position{X: 5, Y: 5})), style.location.String())
	}

	wantCenters := [...]byte{0x05, 0xc8, 0xc9, 0xc8, 0xdc, 0xd8, 0x05, 0x05}
	for nibble, want := range wantCenters {
		var cells DungeonCells
		position := Position{X: 4, Y: 4}
		cells.SetCell(0, position, DungeonCell(nibble<<4))
		combatMap, err := BuildGeneratedDungeonCombatMap(Destard, &cells, 0, position, DungeonFacingNorth)
		require.NoError(t, err)
		assert.Equal(t, want, byte(combatMap.GetTileNumber(Position{X: 5, Y: 5})), "center nibble %d", nibble)
		assert.Equal(t, nibble == int(DungeonLadderBoth), combatMap.IsDungeonDualLadder(Position{X: 5, Y: 5}),
			"center nibble %d dual ladder", nibble)
	}
}

func TestBuildGeneratedDungeonCombatMapExactFacingFormations(t *testing.T) {
	t.Parallel()

	partyPositions := [4][MaxPlayersPerMap]Position{
		{{X: 5, Y: 6}, {X: 4, Y: 7}, {X: 6, Y: 7}, {X: 3, Y: 8}, {X: 5, Y: 8}, {X: 7, Y: 8}},
		{{X: 4, Y: 5}, {X: 3, Y: 4}, {X: 3, Y: 6}, {X: 2, Y: 3}, {X: 2, Y: 5}, {X: 2, Y: 7}},
		{{X: 5, Y: 4}, {X: 4, Y: 3}, {X: 6, Y: 3}, {X: 3, Y: 2}, {X: 5, Y: 2}, {X: 7, Y: 2}},
		{{X: 6, Y: 5}, {X: 7, Y: 4}, {X: 7, Y: 6}, {X: 8, Y: 3}, {X: 8, Y: 5}, {X: 8, Y: 7}},
	}
	monsterPositions := [4][MaxMonstersPerMap]Position{
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
	partyDirectionIndexes := [...]int{2, 1, 3, 0}
	var cells DungeonCells
	for facing := DungeonFacingNorth; facing <= DungeonFacingWest; facing++ {
		combatMap, err := BuildGeneratedDungeonCombatMap(Deceit, &cells, 0, Position{X: 4, Y: 4}, facing)
		require.NoError(t, err)
		for partyFacing, directionIndex := range partyDirectionIndexes {
			assert.Equal(t, partyPositions[partyFacing], combatMap.PlayerPositions[directionIndex])
		}
		assert.Equal(t, monsterPositions[facing], combatMap.MonsterPositions, "facing %d", facing)
	}
}

func TestBuildGeneratedDungeonCampingMapUsesCampFormationsAndNoTriggers(t *testing.T) {
	cells := &DungeonCells{}
	position := Position{X: 2, Y: 3}
	cells.SetCell(0, position, 0x00)

	combatMap, err := BuildGeneratedDungeonCampingMap(
		Deceit,
		cells,
		0,
		position,
		DungeonFacingWest,
	)

	require.NoError(t, err)
	assert.Equal(t, "Dungeon Camping", combatMap.MapName)
	assert.Equal(t, byte(indexes.CampFire), byte(combatMap.GetTileNumber(Position{X: 5, Y: 5})))
	for direction := range NumDirections {
		assert.Equal(t, generatedDungeonCampingPartyPositions, combatMap.PlayerPositions[direction])
	}
	assert.Equal(t, generatedDungeonCampingMonsterPositions, combatMap.MonsterPositions)
	_, hasTriggers := combatMap.DungeonTriggerSlots()
	assert.False(t, hasTriggers)
}

func TestBuildGeneratedDungeonCampingMapPreservesCenterFeatureOverFire(t *testing.T) {
	cells := &DungeonCells{}
	position := Position{X: 2, Y: 3}
	cells.SetCell(0, position, 0x20)

	combatMap, err := BuildGeneratedDungeonCampingMap(
		Deceit,
		cells,
		0,
		position,
		DungeonFacingNorth,
	)

	require.NoError(t, err)
	assert.Equal(t, generatedDungeonCenterTiles[2], byte(combatMap.GetTileNumber(Position{X: 5, Y: 5})))
}

func TestGeneratedDungeonClosedChestRetainsSourceDepthAndMaterializesFloorOnce(t *testing.T) {
	tests := []struct {
		name      string
		location  Location
		floor     FloorNumber
		wantFloor indexes.SpriteIndex
	}{
		{name: "level one stone style", location: Deceit, floor: 0, wantFloor: 0x45},
		{name: "level eight grass style", location: Doom, floor: 7, wantFloor: 0x05},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cells DungeonCells
			position := Position{X: 4, Y: 4}
			cells.SetCell(test.floor, position, 0x40)

			combatMap, err := BuildGeneratedDungeonCombatMap(
				test.location, &cells, test.floor, position, DungeonFacingNorth,
			)
			require.NoError(t, err)
			center := Position{X: 5, Y: 5}
			assert.Equal(t, indexes.SpriteIndex(0xdc), combatMap.GetTileNumber(center),
				"the intermediate map retains the display marker")
			chest, found := combatMap.GeneratedDungeonChest()
			require.True(t, found)
			assert.Equal(t, center, chest.Position)
			assert.Equal(t, test.floor, chest.SourceFloor)
			assert.Equal(t, test.wantFloor, chest.FloorTile)

			materialized, found := combatMap.MaterializeGeneratedDungeonChest()
			require.True(t, found)
			assert.Equal(t, chest, materialized)
			assert.Equal(t, test.wantFloor, combatMap.GetTileNumber(center))
			_, found = combatMap.MaterializeGeneratedDungeonChest()
			assert.False(t, found, "one generated marker must produce one logical chest")
		})
	}
}
