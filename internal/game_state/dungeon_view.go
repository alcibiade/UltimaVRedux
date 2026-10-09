package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	// DungeonViewSize is the width and height of the original symbolic display.
	DungeonViewSize = 22
	// DungeonViewCenter is the display coordinate occupied by the party.
	DungeonViewCenter = 11
)

var errDungeonViewUnavailable = errors.New("dungeon view is unavailable")

// DungeonViewCell is one display-space cell in the original 22x22 symbolic view.
type DungeonViewCell struct {
	Visible        bool
	Player         bool
	SourcePosition references.Position
	SourceCell     references.DungeonCell
}

// DungeonViewModel is a detached, UI-independent symbolic projection of one dungeon floor.
type DungeonViewModel struct {
	Location     references.Location
	Floor        references.FloorNumber
	Facing       references.DungeonFacing
	Revision     uint64
	VisibleCount int
	Cells        [DungeonViewSize][DungeonViewSize]DungeonViewCell
}

// BuildDungeonViewModel reproduces original dungeon map-view rules's eight-connected flood over a wrapped 8x8 floor.
// See ref:VIEW-0003.
func BuildDungeonViewModel(
	snapshot DungeonStateSnapshot,
	player references.PlayerLocation,
) (DungeonViewModel, error) {
	if snapshot.Location.GetMapType() != references.DungeonMapType {
		return DungeonViewModel{}, fmt.Errorf("%w: snapshot location %s is not a dungeon", errDungeonViewUnavailable, snapshot.Location)
	}
	if player.Location != snapshot.Location {
		return DungeonViewModel{}, fmt.Errorf(
			"%w: player location %s does not match snapshot location %s",
			errDungeonViewUnavailable,
			player.Location,
			snapshot.Location,
		)
	}
	if !references.IsDungeonCoordinate(player.Floor, player.Position) {
		return DungeonViewModel{}, fmt.Errorf(
			"%w: invalid floor %d or player position %v",
			errDungeonViewUnavailable,
			player.Floor,
			player.Position,
		)
	}

	model := DungeonViewModel{
		Location: snapshot.Location,
		Floor:    player.Floor,
		Facing:   snapshot.Facing,
		Revision: snapshot.Revision,
	}
	type displayPosition struct {
		x int
		y int
	}
	var visited [DungeonViewSize][DungeonViewSize]bool
	var queue [DungeonViewSize * DungeonViewSize]displayPosition
	head, tail := 0, 0

	reveal := func(position displayPosition, playerCell bool) bool {
		visited[position.y][position.x] = true
		sourcePosition := references.Position{
			X: wrappedDungeonViewCoordinate(position.x - DungeonViewCenter + int(player.Position.X)),
			Y: wrappedDungeonViewCoordinate(position.y - DungeonViewCenter + int(player.Position.Y)),
		}
		sourceCell, ok := snapshot.Cells.Cell(player.Floor, sourcePosition)
		if !ok {
			panic("validated dungeon view source coordinate became invalid")
		}
		model.Cells[position.y][position.x] = DungeonViewCell{
			Visible:        true,
			Player:         playerCell,
			SourcePosition: sourcePosition,
			SourceCell:     sourceCell,
		}
		model.VisibleCount++

		return !dungeonViewCellBlocksPropagation(sourceCell)
	}

	center := displayPosition{x: DungeonViewCenter, y: DungeonViewCenter}
	reveal(center, true)
	queue[tail] = center
	tail++

	neighbors := [...]displayPosition{
		{x: -1, y: -1},
		{x: 0, y: -1},
		{x: 1, y: -1},
		{x: -1, y: 0},
		{x: 1, y: 0},
		{x: -1, y: 1},
		{x: 0, y: 1},
		{x: 1, y: 1},
	}
	for head < tail {
		current := queue[head]
		head++
		for _, offset := range neighbors {
			next := displayPosition{x: current.x + offset.x, y: current.y + offset.y}
			if next.x < 0 || next.x >= DungeonViewSize || next.y < 0 || next.y >= DungeonViewSize ||
				visited[next.y][next.x] {
				continue
			}
			if reveal(next, false) {
				queue[tail] = next
				tail++
			}
		}
	}

	return model, nil
}

// CurrentDungeonViewModel builds the symbolic view from current authoritative state.
func (g *GameState) CurrentDungeonViewModel() (DungeonViewModel, error) {
	snapshot, ok := g.DungeonStateSnapshot()
	if !ok {
		return DungeonViewModel{}, errDungeonViewUnavailable
	}

	return BuildDungeonViewModel(snapshot, g.MapState.PlayerLocation)
}

func wrappedDungeonViewCoordinate(value int) references.Coordinate {
	size := int(references.XDungeonMapTiles)
	value %= size
	if value < 0 {
		value += size
	}

	return references.Coordinate(value)
}

func dungeonViewCellBlocksPropagation(cell references.DungeonCell) bool {
	//nolint:exhaustive // Only the three original dungeon map-view rules blocker kinds stop propagation.
	// See ref:VIEW-0003.
	switch cell.Kind() {
	case references.DungeonWall, references.DungeonSpecialWall, references.DungeonHiddenDoor:
		return true
	default:
		return false
	}
}
