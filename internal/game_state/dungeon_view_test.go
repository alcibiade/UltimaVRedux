package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDungeonViewModelRevealsOpenDisplay(t *testing.T) {
	model, err := BuildDungeonViewModel(
		DungeonStateSnapshot{Location: references.Deceit, Revision: 12},
		references.PlayerLocation{
			Location: references.Deceit,
			Position: references.Position{X: 3, Y: 4},
			Floor:    2,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, references.Deceit, model.Location)
	assert.Equal(t, references.FloorNumber(2), model.Floor)
	assert.Equal(t, references.DungeonFacingNorth, model.Facing)
	assert.Equal(t, uint64(12), model.Revision)
	assert.Equal(t, DungeonViewSize*DungeonViewSize, model.VisibleCount)
	assert.True(t, model.Cells[DungeonViewCenter][DungeonViewCenter].Player)
	assert.Equal(t,
		references.Position{X: 3, Y: 4},
		model.Cells[DungeonViewCenter][DungeonViewCenter].SourcePosition,
	)
}

func TestBuildDungeonViewModelWrapsSourceCoordinatesRepeatedly(t *testing.T) {
	model, err := BuildDungeonViewModel(
		DungeonStateSnapshot{Location: references.Deceit},
		references.PlayerLocation{Location: references.Deceit},
	)
	require.NoError(t, err)

	assert.Equal(t, references.Position{X: 5, Y: 5}, model.Cells[0][0].SourcePosition)
	assert.Equal(t, references.Position{X: 7, Y: 7}, model.Cells[10][10].SourcePosition)
	assert.Equal(t, references.Position{X: 2, Y: 2}, model.Cells[21][21].SourcePosition)
	assert.Equal(t, model.Cells[0][0].SourcePosition, model.Cells[8][8].SourcePosition)
}

func TestBuildDungeonViewModelShowsBlockingBoundary(t *testing.T) {
	var cells references.DungeonCells
	player := references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 4, Y: 4},
	}
	blockers := []struct {
		position references.Position
		cell     references.DungeonCell
	}{
		{position: references.Position{X: 3, Y: 3}, cell: 0xb0},
		{position: references.Position{X: 4, Y: 3}, cell: 0xc0},
		{position: references.Position{X: 5, Y: 3}, cell: 0xd0},
		{position: references.Position{X: 3, Y: 4}, cell: 0xb1},
		{position: references.Position{X: 5, Y: 4}, cell: 0xc8},
		{position: references.Position{X: 3, Y: 5}, cell: 0xdf},
		{position: references.Position{X: 4, Y: 5}, cell: 0xb8},
		{position: references.Position{X: 5, Y: 5}, cell: 0xc1},
	}
	for _, blocker := range blockers {
		require.True(t, cells.SetCell(0, blocker.position, blocker.cell))
	}

	model, err := BuildDungeonViewModel(DungeonStateSnapshot{Location: references.Deceit, Cells: cells}, player)
	require.NoError(t, err)

	assert.Equal(t, 9, model.VisibleCount)
	for y := DungeonViewCenter - 1; y <= DungeonViewCenter+1; y++ {
		for x := DungeonViewCenter - 1; x <= DungeonViewCenter+1; x++ {
			assert.True(t, model.Cells[y][x].Visible, "cell (%d,%d)", x, y)
		}
	}
	assert.False(t, model.Cells[DungeonViewCenter-2][DungeonViewCenter].Visible)
}

func TestBuildDungeonViewModelAllowsDiagonalPropagation(t *testing.T) {
	var cells references.DungeonCells
	player := references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 4, Y: 4},
	}
	require.True(t, cells.SetCell(0, references.Position{X: 4, Y: 3}, 0xb0))
	require.True(t, cells.SetCell(0, references.Position{X: 3, Y: 4}, 0xb0))

	model, err := BuildDungeonViewModel(DungeonStateSnapshot{Location: references.Deceit, Cells: cells}, player)
	require.NoError(t, err)

	assert.True(t, model.Cells[DungeonViewCenter-1][DungeonViewCenter-1].Visible)
	assert.True(t, model.Cells[DungeonViewCenter-2][DungeonViewCenter-2].Visible)
}

func TestBuildDungeonViewModelCenterAlwaysSeedsFlood(t *testing.T) {
	var cells references.DungeonCells
	for y := range references.YDungeonMapTiles {
		for x := range references.XDungeonMapTiles {
			cells[0][y][x] = 0xb0
		}
	}

	model, err := BuildDungeonViewModel(
		DungeonStateSnapshot{Location: references.Deceit, Cells: cells},
		references.PlayerLocation{Location: references.Deceit},
	)
	require.NoError(t, err)

	assert.Equal(t, 9, model.VisibleCount)
	assert.True(t, model.Cells[DungeonViewCenter][DungeonViewCenter].Player)
}

func TestBuildDungeonViewModelCarriesFacingAndIgnoresWanderer(t *testing.T) {
	snapshot := DungeonStateSnapshot{
		Location: references.Deceit,
		Facing:   references.DungeonFacingNorth,
		Wanderer: &DungeonWanderer{Position: references.Position{X: 1, Y: 1}},
	}
	player := references.PlayerLocation{Location: references.Deceit}
	first, err := BuildDungeonViewModel(snapshot, player)
	require.NoError(t, err)

	snapshot.Facing = references.DungeonFacingSouth
	snapshot.Wanderer.Position = references.Position{X: 7, Y: 7}
	second, err := BuildDungeonViewModel(snapshot, player)
	require.NoError(t, err)

	assert.Equal(t, references.DungeonFacingNorth, first.Facing)
	assert.Equal(t, references.DungeonFacingSouth, second.Facing)
	first.Facing = second.Facing
	assert.Equal(t, first, second)
}

func TestBuildDungeonViewModelRejectsInvalidContext(t *testing.T) {
	tests := []struct {
		name     string
		snapshot DungeonStateSnapshot
		player   references.PlayerLocation
	}{
		{
			name:     "non-dungeon snapshot",
			snapshot: DungeonStateSnapshot{Location: references.BritanniaUnderworld},
			player:   references.PlayerLocation{Location: references.BritanniaUnderworld},
		},
		{
			name:     "location mismatch",
			snapshot: DungeonStateSnapshot{Location: references.Deceit},
			player:   references.PlayerLocation{Location: references.Doom},
		},
		{
			name:     "invalid floor",
			snapshot: DungeonStateSnapshot{Location: references.Deceit},
			player: references.PlayerLocation{
				Location: references.Deceit,
				Floor:    references.DungeonFloorCount,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildDungeonViewModel(test.snapshot, test.player)
			assert.ErrorIs(t, err, errDungeonViewUnavailable)
		})
	}
}

func TestDungeonViewCellBlocksPropagation(t *testing.T) {
	for _, cell := range []references.DungeonCell{0xb0, 0xbf, 0xc0, 0xcf, 0xd0, 0xdf} {
		assert.True(t, dungeonViewCellBlocksPropagation(cell), "cell %02x", cell.Raw())
	}
	for _, cell := range []references.DungeonCell{0x00, 0x61, 0x80, 0xa0, 0xe0, 0xf0} {
		assert.False(t, dungeonViewCellBlocksPropagation(cell), "cell %02x", cell.Raw())
	}
}
