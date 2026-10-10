package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// DungeonChestAssessments is a detached [floor][y][x] view of one active visit's
// closed-chest knowledge. Zero entries denote uninspected cells. It is not a
// playthrough-wide discovery registry and must be restored with the visit cells.
type DungeonChestAssessments [references.DungeonFloorCount][references.YDungeonMapTiles][references.XDungeonMapTiles]references.ChestAssessment

// ChestAssessment reads active-visit knowledge at a validated cell coordinate.
func (d *DungeonState) ChestAssessment(floor references.FloorNumber, position references.Position) references.ChestAssessment {
	if _, ok := d.Cell(floor, position); !ok {
		panic("invalid dungeon chest assessment coordinate")
	}

	return d.chestAssessments[floor][position.Y][position.X]
}

// SetChestAssessment records a completed inspection or disarming operation.
func (d *DungeonState) SetChestAssessment(floor references.FloorNumber, position references.Position, assessment references.ChestAssessment) {
	cell, ok := d.Cell(floor, position)
	if !ok || !validDungeonChestAssessment(cell, assessment) {
		panic("invalid dungeon chest assessment")
	}
	if d.chestAssessments[floor][position.Y][position.X] == assessment {
		return
	}
	d.chestAssessments[floor][position.Y][position.X] = assessment
	d.revision++
}

// RestoreChestAssessments validates the whole detached array before restoring
// native-save knowledge. Physical cells must already be installed. No RNG or
// physical mutation occurs, and invalid input leaves the current array intact.
func (d *DungeonState) RestoreChestAssessments(assessments DungeonChestAssessments) error {
	for floor := range assessments {
		for y := range assessments[floor] {
			for x, assessment := range assessments[floor][y] {
				cell, ok := d.Cell(references.FloorNumber(floor), references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)})
				if !ok || !validDungeonChestAssessment(cell, assessment) {
					return fmt.Errorf("invalid dungeon chest assessment %d at floor %d (%d,%d)", assessment, floor, x, y)
				}
			}
		}
	}
	d.chestAssessments = assessments

	return nil
}

func validDungeonChestAssessment(cell references.DungeonCell, assessment references.ChestAssessment) bool {
	if cell.Kind() != references.DungeonClosedChest {
		return assessment == references.ChestUninspected
	}

	return assessment.ValidForChest(cell.Raw()&0x07 != 0)
}
