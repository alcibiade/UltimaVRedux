package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDungeonCellPreservesEveryPackedValue(t *testing.T) {
	for raw := range 256 {
		cell := DungeonCell(raw)
		assert.Equal(t, byte(raw), cell.Raw())
		assert.Equal(t, DungeonCellKind(raw>>4), cell.Kind())
		assert.Equal(t, byte(raw&0x0f), cell.Payload())
		assert.Equal(t, raw&0x08 != 0, cell.HasBit08())
	}
}

func TestDungeonCellsUseOriginalFloorYAndXOrdering(t *testing.T) {
	var cells DungeonCells
	position := Position{X: 6, Y: 3}
	require.True(t, cells.SetCell(5, position, DungeonCell(0xc8)))

	cell, ok := cells.Cell(5, position)
	require.True(t, ok)
	assert.Equal(t, byte(0xc8), cell.Raw())
	assert.Equal(t, DungeonCell(0), cells[5][6][3])
}

func TestDungeonCellsRejectInvalidCoordinates(t *testing.T) {
	var cells DungeonCells
	invalid := []struct {
		floor    FloorNumber
		position Position
	}{
		{floor: -1, position: Position{}},
		{floor: DungeonFloorCount, position: Position{}},
		{floor: 0, position: Position{X: -1}},
		{floor: 0, position: Position{X: XDungeonMapTiles}},
		{floor: 0, position: Position{Y: YDungeonMapTiles}},
	}

	for _, testCase := range invalid {
		_, ok := cells.Cell(testCase.floor, testCase.position)
		assert.False(t, ok)
		assert.False(t, cells.SetCell(testCase.floor, testCase.position, DungeonCell(1)))
	}
}

func TestDungeonFacingUsesOriginalClockwiseValues(t *testing.T) {
	assert.Equal(t, DungeonFacingEast, DungeonFacingNorth.TurnRight())
	assert.Equal(t, DungeonFacingWest, DungeonFacingNorth.TurnLeft())
	assert.Equal(t, DungeonFacingSouth, DungeonFacingNorth.TurnAround())
	assert.Equal(t, Right, DungeonFacingEast.Direction())
}
