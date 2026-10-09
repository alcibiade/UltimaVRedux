package party_state

import (
	"testing"
)

func TestKarma_IncreaseKarma(t *testing.T) {
	tests := []struct {
		name       string
		initial    int
		increaseBy int
		expected   int
	}{
		{"increase by 1", 50, 1, 51},
		{"increase by 3", 50, 3, 53},
		{"increase from 0", 0, 5, 5},
		{"cap at 99", 98, 5, 99},
		{"already at max", 99, 1, 99},
		{"increase by 0", 50, 0, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := Karma{Value: tt.initial}
			k.IncreaseKarma(tt.increaseBy)
			if k.Value != tt.expected {
				t.Errorf("IncreaseKarma(%d): got %d, want %d", tt.increaseBy, k.Value, tt.expected)
			}
		})
	}
}

func TestKarma_DecreaseKarma(t *testing.T) {
	tests := []struct {
		name       string
		initial    int
		decreaseBy int
		expected   int
	}{
		{"decrease by 1", 50, 1, 49},
		{"decrease by 2", 50, 2, 48},
		{"floor at 0", 3, 5, 0},
		{"already at zero", 0, 1, 0},
		{"decrease from max", 99, 10, 89},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := Karma{Value: tt.initial}
			k.DecreaseKarma(tt.decreaseBy)
			if k.Value != tt.expected {
				t.Errorf("DecreaseKarma(%d): got %d, want %d", tt.decreaseBy, k.Value, tt.expected)
			}
		})
	}
}

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
