package references

import (
	"fmt"
	"image"

	pathfind "github.com/bradhannah/berad-sdk/modules/berad-path"
)

// FindRoute returns the cheapest orthogonal route from start to goal inside
// bounds, including both ends, or nil when there is none or either end lies
// outside bounds. entryCost prices entering a position: at least 1, or
// negative when it cannot be entered, as tile walkable weights do. Equal-cost
// choices repeat for equal inputs.
func FindRoute(start, goal Position, bounds image.Rectangle, entryCost func(Position) int) []Position {
	startPoint, goalPoint := image.Pt(int(start.X), int(start.Y)), image.Pt(int(goal.X), int(goal.Y))
	if !startPoint.In(bounds) || !goalPoint.In(bounds) {
		return nil
	}
	result, err := pathfind.Find(pathfind.Query{
		Start:      startPoint,
		Goal:       goalPoint,
		Bounds:     bounds,
		Moves:      pathfind.Four,
		MinStep:    1,
		MaxVisited: 0,
		Cost: func(_, to image.Point) (int, bool) {
			cost := entryCost(Position{X: Coordinate(to.X), Y: Coordinate(to.Y)})

			return cost, cost >= 0
		},
	})
	if err != nil {
		panic(fmt.Sprintf("invalid route search: %v", err))
	}
	if result.Outcome != pathfind.Found {
		return nil
	}

	route := make([]Position, len(result.Route))
	for i, point := range result.Route {
		route[i] = Position{X: Coordinate(point.X), Y: Coordinate(point.Y)}
	}

	return route
}
