package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

const (
	outdoorWindowSize         = 32
	outdoorQuadrantSize       = 16
	outdoorWorldMask          = 255
	outdoorWindowMask         = outdoorWindowSize - 1
	outdoorWindowInitialInset = 8
	outdoorWindowOriginMask   = 0xf0
	outdoorWindowSafeMinimum  = 5
	outdoorWindowSafeMaximum  = 26
)

// outdoorWindow is gameplay authority, independent of the camera. It is suspended
// during combat/camping/shrine visits and rebuilt on hydration. See ref:SPELL-0002.
type outdoorWindow struct {
	origin      references.Position
	floor       references.FloorNumber
	initialized bool
}

// InitializeOutdoorWindow begins a fresh outdoor visit at the final position.
// Even identical origins reload all four logical quadrants: only owned temporary
// Push writes expire, never unrelated persistent terrain or actors.
func (g *GameState) InitializeOutdoorWindow() {
	location := g.MapState.PlayerLocation
	if location.Location.GetMapType() != references.LargeMapType {
		return
	}
	g.expireOutdoorPushTiles(func(transientLargeMapPushTileKey) bool { return true })
	g.outdoorWindow = outdoorWindow{
		origin: references.Position{
			X: (location.Position.X - outdoorWindowInitialInset) & outdoorWindowOriginMask,
			Y: (location.Position.Y - outdoorWindowInitialInset) & outdoorWindowOriginMask,
		},
		floor:       location.Floor,
		initialized: true,
	}
}

func (g *GameState) requireOutdoorWindow() outdoorWindow {
	window := g.outdoorWindow
	if !window.initialized || window.floor != g.MapState.PlayerLocation.Floor ||
		g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		panic("outdoor window must be initialized at world entry before movement or Blink")
	}

	return window
}

// MovePlayerOneStep commits one already-accepted cardinal displacement. Collision,
// command time, vehicle cadence and feedback belong to the caller. Forced steps
// use this same boundary individually; Push and guardian corrections do not.
func (g *GameState) MovePlayerOneStep(direction references.Direction) {
	before := g.MapState.PlayerLocation.Position
	next := direction.GetNewPositionInDirection(&before)
	if next == nil {
		return
	}
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		// Retain the existing non-wrapping small-map boundary behavior.
		switch direction {
		case references.Up:
			g.MapState.PlayerLocation.Position.GoUp(false)
		case references.Down:
			g.MapState.PlayerLocation.Position.GoDown(false)
		case references.Left:
			g.MapState.PlayerLocation.Position.GoLeft(false)
		case references.Right:
			g.MapState.PlayerLocation.Position.GoRight(false)
		case references.NoneDirection:
			// Rejected by the adjacent-position lookup above.
		}

		return
	}
	window := g.requireOutdoorWindow()
	dx, dy := next.X-before.X, next.Y-before.Y
	next.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	g.SetPlayerPosition(*next)
	rx := (next.X - window.origin.X) & outdoorWindowMask
	ry := (next.Y - window.origin.Y) & outdoorWindowMask
	if rx >= outdoorWindowSafeMinimum && rx <= outdoorWindowSafeMaximum &&
		ry >= outdoorWindowSafeMinimum && ry <= outdoorWindowSafeMaximum {
		return
	}

	// Shift exactly once using the actual direction, including 255 -> 0 steps.
	// Q0=NW, Q1=NE, Q2=SW, Q3=SE. The copied half survives; the outgoing
	// half is discarded and later re-entry reads fresh persistent terrain.
	retained := retainedOutdoorQuadrants(direction)
	g.expireOutdoorPushTiles(func(key transientLargeMapPushTileKey) bool {
		if key.floor != window.floor {
			return false
		}
		quadrant, inside := window.quadrant(key.position)

		return !inside || !retained[quadrant]
	})
	g.outdoorWindow.origin = references.Position{
		X: (window.origin.X + dx*outdoorQuadrantSize) & outdoorWorldMask,
		Y: (window.origin.Y + dy*outdoorQuadrantSize) & outdoorWorldMask,
	}
}

func retainedOutdoorQuadrants(direction references.Direction) [4]bool {
	switch direction {
	case references.Right:
		return [4]bool{false, true, false, true} // old Q1 -> Q0, Q3 -> Q2
	case references.Left:
		return [4]bool{true, false, true, false} // old Q0 -> Q1, Q2 -> Q3
	case references.Down:
		return [4]bool{false, false, true, true} // old Q2 -> Q0, Q3 -> Q1
	case references.Up:
		return [4]bool{true, true, false, false} // old Q0 -> Q2, Q1 -> Q3
	case references.NoneDirection:
		return [4]bool{}
	default:
		panic("outdoor scrolling requires a cardinal direction")
	}
}

func (window outdoorWindow) quadrant(position references.Position) (int, bool) {
	x := (position.X - window.origin.X) & outdoorWorldMask
	y := (position.Y - window.origin.Y) & outdoorWorldMask
	if x >= outdoorWindowSize || y >= outdoorWindowSize {
		return 0, false
	}

	return int(x/outdoorQuadrantSize + 2*(y/outdoorQuadrantSize)), true
}
