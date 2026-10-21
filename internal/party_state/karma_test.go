package party_state

import (
	"testing"
)

func TestNewKarma(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		expected int
	}{
		{"zero karma", 0, 0},
		{"middle karma", 50, 50},
		{"max karma", 99, 99},
		{"byte max", 255, 255}, // Note: exceeds maxKarma, but NewKarma doesn't clamp
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := NewKarma(tt.input)
			if k.Value != tt.expected {
				t.Errorf("NewKarma(%d): got %d, want %d", tt.input, k.Value, tt.expected)
			}
		})
	}
}

func TestKarmaAddDiffClampsToRange(t *testing.T) {
	tests := []struct {
		name     string
		start    int
		diff     int
		expected int
	}{
		{"increase", 50, 10, 60},
		{"decrease", 50, -10, 40},
		{"clamps at zero", 5, -10, 0},
		{"clamps at maximum", 95, 10, maxKarma},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			karma := Karma{Value: tt.start}
			karma.AddDiff(tt.diff)
			if karma.Value != tt.expected {
				t.Errorf("AddDiff(%d) from %d = %d, want %d", tt.diff, tt.start, karma.Value, tt.expected)
			}
		})
	}
}
