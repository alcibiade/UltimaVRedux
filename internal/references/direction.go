package references

// Direction is a four-way cardinal facing, or NoneDirection when there is no facing.
type Direction int

// Directions; NoneDirection is the zero value and means no facing.
const (
	NoneDirection Direction = iota
	Up
	Down
	Left
	Right
)

// GetNewPositionInDirection returns the position one tile from position in this direction, or nil for NoneDirection.
func (d Direction) GetNewPositionInDirection(position *Position) *Position {
	var newPosition *Position
	switch d {
	case Up:
		newPosition = position.GetPositionUp()
	case Down:
		newPosition = position.GetPositionDown()
	case Left:
		newPosition = position.GetPositionToLeft()
	case Right:
		newPosition = position.GetPositionToRight()
	case NoneDirection:
		newPosition = nil
	}

	return newPosition
}

// GetDirectionCompassName returns the compass word for this direction (Up is North); it panics for NoneDirection.
func (d Direction) GetDirectionCompassName() string {
	switch d {
	case Up:
		return "North"
	case Down:
		return "South"
	case Left:
		return "West"
	case Right:
		return "East"
	case NoneDirection:
	}

	panic("unhandled default case")
}

// GetOppositeDirection returns the reverse direction; NoneDirection reverses to itself.
func (d Direction) GetOppositeDirection() Direction {
	switch d {
	case Up:
		return Down
	case Down:
		return Up
	case Left:
		return Right
	case Right:
		return Left
	case NoneDirection:
		return NoneDirection
	default:
		panic("unhandled default case")
	}
}
