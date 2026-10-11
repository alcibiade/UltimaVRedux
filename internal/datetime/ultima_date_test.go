package datetime

import "testing"

func TestFindingsFullDaylightUsesOriginalAmbientThreshold(t *testing.T) {
	for hour := range byte(24) {
		for _, minute := range []byte{0, 9, 10, 29, 30, 59} {
			date := UltimaDate{Hour: hour, Minute: minute}
			if got, want := date.IsFullDaylight(), hour >= 6 && hour < 19; got != want {
				t.Errorf("%02d:%02d full daylight = %t, want %t", hour, minute, got, want)
			}
		}
	}
}

func TestAdvanceCalendarMonthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date UltimaDate
		want UltimaDate
	}{
		{
			name: "month 12 advances to month 13",
			date: UltimaDate{Year: 1, Month: 12, Day: 28, Hour: 23, Minute: 59},
			want: UltimaDate{Year: 1, Month: 13, Day: 1, Turn: 1},
		},
		{
			name: "month 13 advances to next year",
			date: UltimaDate{Year: 1, Month: 13, Day: 28, Hour: 23, Minute: 59},
			want: UltimaDate{Year: 2, Month: 1, Day: 1, Turn: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			date := test.date
			date.Advance(1)
			if date != test.want {
				t.Fatalf("Advance(1) = %#v, want %#v", date, test.want)
			}
		})
	}
}

func TestAdvanceUsesExactElapsedHours(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		date    UltimaDate
		minutes int
		want    UltimaDate
	}{
		{
			name:    "whole hour preserves minute",
			date:    UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 12, Minute: 30, Turn: 7},
			minutes: 60,
			want:    UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 13, Minute: 30, Turn: 8},
		},
		{
			name:    "multi-hour advancement crosses midnight once",
			date:    UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 20, Minute: 45},
			minutes: 300,
			want:    UltimaDate{Year: 1, Month: 1, Day: 2, Hour: 1, Minute: 45, Turn: 1},
		},
		{
			name:    "exact minute boundary",
			date:    UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 12, Minute: 59},
			minutes: 1,
			want:    UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 13, Turn: 1},
		},
		{
			name:    "request over nine hours remains clamped",
			date:    UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 12, Minute: 30},
			minutes: 600,
			want:    UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 21, Minute: 30, Turn: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			date := test.date
			date.Advance(test.minutes)
			if date != test.want {
				t.Fatalf("Advance(%d) = %#v, want %#v", test.minutes, date, test.want)
			}
		})
	}
}

func TestGetTotalDaysUsesThirteenMonthYear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date UltimaDate
		want int
	}{
		{name: "epoch", date: UltimaDate{Year: 1, Month: 1, Day: 1}, want: 0},
		{name: "last day of month 12", date: UltimaDate{Year: 1, Month: 12, Day: 28}, want: 335},
		{name: "first day of month 13", date: UltimaDate{Year: 1, Month: 13, Day: 1}, want: 336},
		{name: "last day of year", date: UltimaDate{Year: 1, Month: 13, Day: 28}, want: 363},
		{name: "first day of next year", date: UltimaDate{Year: 2, Month: 1, Day: 1}, want: 364},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.date.GetTotalDays(); got != test.want {
				t.Fatalf("GetTotalDays() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestGetAmbientTintDarknessFadesAtDawnAndDusk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		hour   uint8
		minute uint8
		want   float32
	}{
		{name: "night before dawn", hour: 4, minute: 59, want: 1},
		{name: "dawn begins fully dark", hour: 5, minute: 0, want: 1},
		{name: "dawn midpoint", hour: 5, minute: 15, want: 0.5},
		{name: "dawn completes", hour: 5, minute: 30, want: 0},
		{name: "day before dusk", hour: 19, minute: 29, want: 0},
		{name: "dusk begins fully bright", hour: 19, minute: 30, want: 0},
		{name: "dusk midpoint", hour: 19, minute: 45, want: 0.5},
		{name: "night begins fully dark", hour: 20, minute: 0, want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			date := UltimaDate{Hour: test.hour, Minute: test.minute}
			if got := date.GetAmbientTintDarkness(); got != test.want {
				t.Fatalf("GetAmbientTintDarkness() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDateOnlyReturnsValidBritannianDateWithoutClockState(t *testing.T) {
	t.Parallel()

	dateTime := UltimaDate{Year: 7, Month: 13, Day: 28, Hour: 23, Minute: 59, Turn: 123}
	date := dateTime.DateOnly()
	if date != (BritannianDate{Year: 7, Month: 13, Day: 28}) || !date.IsValid() {
		t.Fatalf("DateOnly() = %#v, want valid date-only value", date)
	}
	for _, invalid := range []BritannianDate{
		{},
		{Year: 1, Month: 14, Day: 1},
		{Year: 1, Month: 1, Day: 29},
	} {
		if invalid.IsValid() {
			t.Fatalf("BritannianDate.IsValid() accepted %#v", invalid)
		}
	}
}
