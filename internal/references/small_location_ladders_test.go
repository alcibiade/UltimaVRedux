package references

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestClosestLadderIncludesEdgesAndMissingConnections(t *testing.T) {
	location := &SmallLocationReference{
		Location: Trinsic,
		rawData:  map[int]*[XSmallMapTiles][YSmallMapTiles]byte{0: {}},
	}
	require.Equal(t, Position{X: -1, Y: -1}, location.GetClosestLadder(Position{}, 0, 1))
	require.Equal(t, Position{X: -1, Y: -1}, location.GetClosestLadder(Position{}, 1, 0))
	for _, position := range []Position{{X: 0, Y: 0}, {X: 31, Y: 6}, {X: 6, Y: 31}, {X: 31, Y: 31}} {
		location.rawData[0][position.X][position.Y] = byte(indexes.LadderUp)
		require.Equal(t, position, location.GetClosestLadder(position, 0, 1))
	}
	// The origin remains a valid closest candidate after later entries are found.
	require.Equal(t, Position{}, location.GetClosestLadder(Position{}, 0, 1))
}

func TestClosestLadderUsesRequestedFloorAndDirection(t *testing.T) {
	location := &SmallLocationReference{
		Location: Trinsic,
		rawData:  map[int]*[XSmallMapTiles][YSmallMapTiles]byte{-1: {}, 0: {}, 1: {}},
	}
	up, down, upper, basement := Position{X: 10, Y: 10}, Position{X: 20, Y: 20}, Position{X: 5, Y: 5}, Position{X: 6, Y: 6}
	location.rawData[0][up.X][up.Y] = byte(indexes.StairsNorth)
	location.rawData[0][down.X][down.Y] = byte(indexes.StairsSouth)
	location.rawData[1][upper.X][upper.Y] = byte(indexes.LadderDown)
	location.rawData[-1][basement.X][basement.Y] = byte(indexes.LadderUp)
	require.Equal(t, up, location.GetClosestLadder(down, 0, 1), "receiving from above searches ground-floor up connections")
	require.Equal(t, down, location.GetClosestLadder(up, 0, -1), "receiving from below searches ground-floor down connections")
	require.Equal(t, upper, location.GetClosestLadder(up, 1, 0))
	require.Equal(t, basement, location.GetClosestLadder(down, -1, 0))
	require.Equal(t, Position{X: -1, Y: -1}, location.GetClosestLadder(up, 0, 0))
}
