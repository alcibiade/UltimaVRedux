package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestSetCalendarTimeSetsExactTimeWithoutAdvancing prevents debug clock edits from consuming a turn or simulating elapsed time.
func TestSetCalendarTimeSetsExactTimeWithoutAdvancing(t *testing.T) {
	initial := datetime.UltimaDate{Year: 3, Month: 4, Day: 5, Hour: 6, Minute: 7, Turn: 99}
	gs := &GameState{DateTime: initial, ElapsedMs: 1234} //nolint:exhaustruct

	change, err := gs.SetCalendarTime(23, 59)
	if err != nil {
		t.Fatalf("SetCalendarTime() error = %v", err)
	}

	expected := initial
	expected.Hour = 23
	expected.Minute = 59
	if change.Before != initial || change.After != expected {
		t.Fatalf("SetCalendarTime() change = %+v, want before %+v after %+v", change, initial, expected)
	}
	if gs.DateTime != expected {
		t.Fatalf("SetCalendarTime() DateTime = %+v, want %+v", gs.DateTime, expected)
	}
	if gs.ElapsedMs != 1234 {
		t.Fatalf("SetCalendarTime() ElapsedMs = %d, want unchanged 1234", gs.ElapsedMs)
	}
}

// TestSetCalendarTimeRejectsInvalidValuesAtomically prevents integer conversion from wrapping invalid clock values into valid bytes.
func TestSetCalendarTimeRejectsInvalidValuesAtomically(t *testing.T) {
	tests := []struct {
		name   string
		hour   int
		minute int
	}{
		{name: "negative hour", hour: -1, minute: 0},
		{name: "hour too large", hour: 24, minute: 0},
		{name: "negative minute", hour: 0, minute: -1},
		{name: "minute too large", hour: 0, minute: 60},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initial := datetime.UltimaDate{Year: 1, Month: 2, Day: 3, Hour: 4, Minute: 5, Turn: 6}
			gs := &GameState{DateTime: initial} //nolint:exhaustruct

			change, err := gs.SetCalendarTime(test.hour, test.minute)
			if err == nil {
				t.Fatal("SetCalendarTime() error = nil, want validation error")
			}
			if gs.DateTime != initial || change.Before != initial || change.After != initial {
				t.Fatalf("SetCalendarTime() mutated state on error: DateTime=%+v change=%+v", gs.DateTime, change)
			}
		})
	}
}

// TestSetCalendarDaySetsExactDayWithoutChangingClockOrTurn prevents a day edit from changing unrelated calendar fields.
func TestSetCalendarDaySetsExactDayWithoutChangingClockOrTurn(t *testing.T) {
	initial := datetime.UltimaDate{Year: 3, Month: 4, Day: 5, Hour: 6, Minute: 7, Turn: 99}
	gs := &GameState{DateTime: initial} //nolint:exhaustruct

	change, err := gs.SetCalendarDay(28)
	if err != nil {
		t.Fatalf("SetCalendarDay() error = %v", err)
	}

	expected := initial
	expected.Day = 28
	if change.Before != initial || change.After != expected || gs.DateTime != expected {
		t.Fatalf("SetCalendarDay() state/change = %+v/%+v, want %+v", gs.DateTime, change, expected)
	}
}

// TestSetCalendarDayRejectsInvalidValuesAtomically prevents invalid Britannian month days from entering game state.
func TestSetCalendarDayRejectsInvalidValuesAtomically(t *testing.T) {
	for _, day := range []int{0, 29} {
		initial := datetime.UltimaDate{Year: 1, Month: 2, Day: 3, Hour: 4, Minute: 5, Turn: 6}
		gs := &GameState{DateTime: initial} //nolint:exhaustruct

		change, err := gs.SetCalendarDay(day)
		if err == nil {
			t.Fatalf("SetCalendarDay(%d) error = nil, want validation error", day)
		}
		if gs.DateTime != initial || change.Before != initial || change.After != initial {
			t.Fatalf("SetCalendarDay(%d) mutated state on error: DateTime=%+v change=%+v", day, gs.DateTime, change)
		}
	}
}

// TestTeleportPlayerOnCurrentMapUsesLogicalMapBounds prevents dungeon coordinates from incorrectly using small-map bounds.
func TestTeleportPlayerOnCurrentMapUsesLogicalMapBounds(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		x        int
		y        int
		mapType  references.GeneralMapType
	}{
		{name: "large map maximum", location: references.BritanniaUnderworld, x: 255, y: 255, mapType: references.LargeMapType},
		{name: "small map maximum", location: references.Britain, x: 31, y: 31, mapType: references.SmallMapType},
		{name: "dungeon maximum", location: references.Deceit, x: 7, y: 7, mapType: references.DungeonMapType},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initialPosition := references.Position{X: 1, Y: 2}
			initialDate := datetime.UltimaDate{Day: 3, Hour: 4, Minute: 5, Turn: 6}
			gs := &GameState{ //nolint:exhaustruct
				MapState: map_state.MapState{ //nolint:exhaustruct
					PlayerLocation: references.PlayerLocation{
						Location: test.location,
						Position: initialPosition,
						Floor:    2,
					},
				},
				DateTime:  initialDate,
				ElapsedMs: 1234,
			}

			change, err := gs.TeleportPlayerOnCurrentMap(test.x, test.y)
			if err != nil {
				t.Fatalf("TeleportPlayerOnCurrentMap() error = %v", err)
			}

			expectedPosition := references.Position{X: references.Coordinate(test.x), Y: references.Coordinate(test.y)}
			if change.MapType != test.mapType || change.Before != initialPosition || change.After != expectedPosition {
				t.Fatalf("TeleportPlayerOnCurrentMap() change = %+v", change)
			}
			if gs.MapState.PlayerLocation.Position != expectedPosition {
				t.Fatalf("TeleportPlayerOnCurrentMap() position = %+v, want %+v", gs.MapState.PlayerLocation.Position, expectedPosition)
			}
			if gs.MapState.PlayerLocation.Location != test.location || gs.MapState.PlayerLocation.Floor != 2 {
				t.Fatalf("TeleportPlayerOnCurrentMap() changed map identity: %+v", gs.MapState.PlayerLocation)
			}
			if gs.DateTime != initialDate || gs.ElapsedMs != 1234 {
				t.Fatalf("TeleportPlayerOnCurrentMap() advanced time: DateTime=%+v ElapsedMs=%d", gs.DateTime, gs.ElapsedMs)
			}
		})
	}
}

// TestTeleportPlayerOnCurrentMapRejectsOutOfBoundsAtomically prevents invalid coordinates from partially changing player state.
func TestTeleportPlayerOnCurrentMapRejectsOutOfBoundsAtomically(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		x        int
		y        int
	}{
		{name: "large negative", location: references.BritanniaUnderworld, x: -1, y: 0},
		{name: "large overflow", location: references.BritanniaUnderworld, x: 256, y: 0},
		{name: "small overflow", location: references.Britain, x: 0, y: 32},
		{name: "dungeon overflow", location: references.Deceit, x: 8, y: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initial := references.Position{X: 3, Y: 4}
			gs := &GameState{ //nolint:exhaustruct
				MapState: map_state.MapState{ //nolint:exhaustruct
					PlayerLocation: references.PlayerLocation{Location: test.location, Position: initial},
				},
			}

			change, err := gs.TeleportPlayerOnCurrentMap(test.x, test.y)
			if err == nil {
				t.Fatal("TeleportPlayerOnCurrentMap() error = nil, want validation error")
			}
			if gs.MapState.PlayerLocation.Position != initial || change.Before != initial || change.After != initial {
				t.Fatalf("TeleportPlayerOnCurrentMap() mutated position on error: position=%+v change=%+v", gs.MapState.PlayerLocation.Position, change)
			}
		})
	}
}

// TestTeleportPlayerOnCurrentMapRejectsNonWorldMap prevents same-map navigation from corrupting specialized combat-map state.
func TestTeleportPlayerOnCurrentMapRejectsNonWorldMap(t *testing.T) {
	initial := references.Position{X: 3, Y: 4}
	gs := &GameState{ //nolint:exhaustruct
		MapState: map_state.MapState{ //nolint:exhaustruct
			PlayerLocation: references.PlayerLocation{Location: references.CombatLocation, Position: initial},
		},
	}

	change, err := gs.TeleportPlayerOnCurrentMap(5, 5)
	if err == nil {
		t.Fatal("TeleportPlayerOnCurrentMap() error = nil, want unsupported map error")
	}
	if gs.MapState.PlayerLocation.Position != initial || change.Before != initial || change.After != initial {
		t.Fatalf("TeleportPlayerOnCurrentMap() mutated position on unsupported map: position=%+v change=%+v", gs.MapState.PlayerLocation.Position, change)
	}
}
