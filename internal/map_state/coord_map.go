package map_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// VisibilityCoords flags which map coordinates are visible, indexed [x][y].
type VisibilityCoords map[references.Coordinate]map[references.Coordinate]bool

// ResetVisibilityCoords sets every existing flag to bDefault.
func (v *VisibilityCoords) ResetVisibilityCoords(bDefault bool) {
	for row := range *v {
		for col := range (*v)[row] {
			(*v)[row][col] = bDefault
		}
	}
}

// setVisibilityCoordsRectangle marks the half-open rectangle [topLeft, bottomRight) visible,
// wrapping coordinates at xMax and yMax when bWrap is set.
func (v *VisibilityCoords) setVisibilityCoordsRectangle(
	topLeft, bottomRight *references.Position,
	xMax, yMax references.Coordinate,
	bWrap bool,
) {
	for coordX := topLeft.X; coordX < bottomRight.X; coordX++ {
		for coordY := topLeft.Y; coordY < bottomRight.Y; coordY++ {
			if bWrap {
				xy := (&references.Position{X: coordX, Y: coordY}).GetWrapped(xMax, yMax)
				(*v)[xy.X][xy.Y] = true
			} else {
				(*v)[coordX][coordY] = true
			}
		}
	}
}
