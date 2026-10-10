package game_state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestDungeonReferencesMatchOriginalDataOrdering(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		Build()
	dataPath := filepath.Join(
		state.GameConfig.SavedConfigData.DataFilePath,
		references.GetSmallMapFile(references.Dungeon),
	)
	raw, err := os.ReadFile(dataPath)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), 8*references.DungeonCellCount)

	locations := [...]references.Location{
		references.Deceit,
		references.Despise,
		references.Destard,
		references.Wrong,
		references.Covetous,
		references.Shame,
		references.Hythloth,
		references.Doom,
	}
	for dungeonIndex, location := range locations {
		reference := state.GameReferences.LocationReferences.GetLocationReference(location)
		for floor := references.FloorNumber(0); floor < references.DungeonFloorCount; floor++ {
			for y := references.Coordinate(0); y < references.YDungeonMapTiles; y++ {
				for x := references.Coordinate(0); x < references.XDungeonMapTiles; x++ {
					position := references.Position{X: x, Y: y}
					cell, ok := reference.GetDungeonCell(floor, position)
					require.True(t, ok)
					offset := dungeonIndex*references.DungeonCellCount +
						int(floor)*64 + int(y)*8 + int(x)
					require.Equal(t, raw[offset], cell.Raw(),
						"%s floor=%d x=%d y=%d", location, floor, x, y)
				}
			}
		}
	}
}
