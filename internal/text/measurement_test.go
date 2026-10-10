package text

import (
	"testing"
)

func TestMeasureText_ExplicitNewlines(t *testing.T) {
	// Test messages from death sequence
	tests := []struct {
		name          string
		message       string
		maxChars      int
		expectedLines int
	}{
		{
			name:          "Simple two-line message",
			message:       "An unending darkness\nengulfs thee...",
			maxChars:      40,
			expectedLines: 2,
		},
		{
			name:          "Three-line message",
			message:       "No evil lives here,\nonly peace and darkness.",
			maxChars:      40,
			expectedLines: 2, // Wait, this is only 2 lines!
		},
		{
			name:          "Synthetic report with 3 lines",
			message:       "Synthetic first line\nsynthetic second line\nsynthetic third line",
			maxChars:      40,
			expectedLines: 3,
		},
		{
			name:          "Four lines with empty line",
			message:       "Someone shouts\n\n\"FORTIS FORTUNA\nAVENTARI\"",
			maxChars:      40,
			expectedLines: 4,
		},
		{
			name:          "Single line no newline",
			message:       "Vertigo...",
			maxChars:      40,
			expectedLines: 1,
		},
		{
			name:          "Exact-width line remains intact",
			message:       "nothing of note.",
			maxChars:      16,
			expectedLines: 1,
		},
		{
			name:          "Trailing newline does not add split artifact",
			message:       "First\nSecond\n",
			maxChars:      40,
			expectedLines: 2,
		},
	}

	// Create a simple font for testing
	font := NewUltimaFont(20)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			measurement := MeasureText(tt.message, font, 26, tt.maxChars)

			if measurement.Lines != tt.expectedLines {
				t.Errorf("MeasureText(%q) = %d lines, want %d lines",
					tt.message, measurement.Lines, tt.expectedLines)
			}

			t.Logf("Message: %q", tt.message)
			t.Logf("  Lines: %d, MaxChars: %d", measurement.Lines, measurement.MaxLineChars)
		})
	}
}

func TestMeasureText_WrappingBehavior(t *testing.T) {
	font := NewUltimaFont(20)

	// Test that long lines get wrapped
	longMessage := "This is a very long line that should wrap because it exceeds the maximum characters per line"
	measurement := MeasureText(longMessage, font, 26, 30)

	t.Logf("Long message: %q", longMessage)
	t.Logf("  With maxChars=30: Lines=%d, MaxLineChars=%d", measurement.Lines, measurement.MaxLineChars)

	if measurement.Lines < 2 {
		t.Errorf("Expected long message to wrap into multiple lines, got %d", measurement.Lines)
	}
}
