package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	errInvalidCalendarHour   = errors.New("hour must be between 0 and 23")
	errInvalidCalendarMinute = errors.New("minute must be between 0 and 59")
	errInvalidCalendarDay    = errors.New("day must be between 1 and 28")
	errUnsupportedMapType    = errors.New("current map type does not support same-map teleportation")
	errPositionOutOfBounds   = errors.New("position")
)

// CalendarChange describes an exact calendar mutation.
type CalendarChange struct {
	Before datetime.UltimaDate
	After  datetime.UltimaDate
}

// PlayerPositionChange describes a position mutation within the active map.
type PlayerPositionChange struct {
	MapType references.GeneralMapType
	Before  references.Position
	After   references.Position
}

// SetCalendarTime sets the exact hour and minute without advancing game time.
func (g *GameState) SetCalendarTime(hour, minute int) (CalendarChange, error) {
	change := CalendarChange{Before: g.DateTime, After: g.DateTime}
	if hour < 0 || hour >= datetime.HoursPerDay {
		return change, fmt.Errorf("%w: %d", errInvalidCalendarHour, hour)
	}
	if minute < 0 || minute >= datetime.MinutesPerHour {
		return change, fmt.Errorf("%w: %d", errInvalidCalendarMinute, minute)
	}

	g.DateTime.Hour = byte(hour)
	g.DateTime.Minute = byte(minute)
	change.After = g.DateTime

	return change, nil
}

// SetCalendarDay sets the day of the current month without advancing game time.
func (g *GameState) SetCalendarDay(day int) (CalendarChange, error) {
	change := CalendarChange{Before: g.DateTime, After: g.DateTime}
	if day < 1 || day > datetime.DaysInMonth {
		return change, fmt.Errorf("%w: %d", errInvalidCalendarDay, day)
	}

	g.DateTime.Day = byte(day)
	change.After = g.DateTime

	return change, nil
}

// TeleportPlayerOnCurrentMap sets the player position without changing maps or advancing time.
func (g *GameState) TeleportPlayerOnCurrentMap(xCoordinate, yCoordinate int) (PlayerPositionChange, error) {
	mapType := g.MapState.PlayerLocation.Location.GetMapType()
	change := PlayerPositionChange{
		MapType: mapType,
		Before:  g.MapState.PlayerLocation.Position,
		After:   g.MapState.PlayerLocation.Position,
	}

	var width, height references.Coordinate
	switch mapType {
	case references.LargeMapType:
		width, height = references.XLargeMapTiles, references.YLargeMapTiles
	case references.SmallMapType:
		width, height = references.XSmallMapTiles, references.YSmallMapTiles
	case references.DungeonMapType:
		width, height = references.XDungeonMapTiles, references.YDungeonMapTiles
	case references.CombatMapType, references.CampingMapType:
		return change, fmt.Errorf("%w: %d", errUnsupportedMapType, mapType)
	default:
		return change, fmt.Errorf("%w: %d", errUnsupportedMapType, mapType)
	}

	if xCoordinate < 0 || xCoordinate >= int(width) || yCoordinate < 0 || yCoordinate >= int(height) {
		return change, fmt.Errorf(
			"%w (%d,%d) is outside current map bounds 0..%d,0..%d",
			errPositionOutOfBounds,
			xCoordinate,
			yCoordinate,
			width-1,
			height-1,
		)
	}

	// The map-bound checks above constrain both values to 0..255, within Coordinate's int16 range.
	change.After = references.Position{
		X: references.Coordinate(xCoordinate), //nolint:gosec // G115: bounded by the active map dimensions.
		Y: references.Coordinate(yCoordinate), //nolint:gosec // G115: bounded by the active map dimensions.
	}
	g.SetPlayerPosition(change.After)

	return change, nil
}
