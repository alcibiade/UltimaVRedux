package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmallLocationReferenceCopiesOnlySemanticDungeonCells(t *testing.T) {
	reference := &SmallLocationReference{
		rawData:  make(map[int]*[XSmallMapTiles][YSmallMapTiles]byte),
		Location: Deceit,
	}
	for floor := range DungeonFloorCount {
		reference.addBlankFloor(floor)
		for y := range int(YDungeonMapTiles) {
			for x := range int(XDungeonMapTiles) {
				reference.rawData[floor][x][y] = byte(floor*64 + y*8 + x)
			}
		}
		reference.rawData[floor][XDungeonMapTiles][YDungeonMapTiles] = 0xff
	}

	cells, ok := reference.CopyDungeonCells()
	require.True(t, ok)
	assert.Equal(t, DungeonCell(0), cells[0][0][0])
	assert.Equal(t, DungeonCell(63), cells[0][7][7])
	assert.Equal(t, DungeonCell(64), cells[1][0][0])
	assert.Equal(t, DungeonCell(255), cells[3][7][7])

	reference.rawData[0][0][0] = 0xfe
	assert.Equal(t, DungeonCell(0), cells[0][0][0], "copy must be detached from immutable source data")
}

func TestSmallLocationReferenceDungeonBounds(t *testing.T) {
	reference := &SmallLocationReference{Location: Deceit}
	assert.Equal(t, Coordinate(7), reference.GetMaxX())
	assert.Equal(t, Coordinate(7), reference.GetMaxY())
}

func TestSmallLocationReferenceRejectsDungeonCellsAsSprites(t *testing.T) {
	reference := &SmallLocationReference{Location: Deceit}
	assert.PanicsWithValue(t, "packed dungeon cells are not sprite indexes", func() {
		reference.GetTileNumber(Position{}, 0)
	})
}
