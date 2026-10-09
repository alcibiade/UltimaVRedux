package grammar

import (
	"testing"
)

// The digit-nine regression covers accepting '9' as the first integer character.
func TestMatchInt_AcceptsLeadingDigitNine(t *testing.T) {
	// Create a teleport command that uses MatchInt with 0-255 range (same as actual teleport command)
	teleportCommand := NewTextCommand([]Match{
		MatchString{
			Str:           "teleport",
			Description:   "Move to an X, Y coordinate on a given map",
			CaseSensitive: false,
		},
		MatchInt{IntMin: 0, IntMax: 255}, // X coordinate
		MatchInt{IntMin: 0, IntMax: 255}, // Y coordinate
	}, func(s string, command *TextCommand) {
		// Empty function for test
	})

	testCases := []struct {
		name        string
		input       string
		expectMatch bool
		description string
	}{
		{
			name:        "teleport command only",
			input:       "teleport",
			expectMatch: true,
			description: "Just the teleport command should match",
		},
		{
			name:        "teleport with space",
			input:       "teleport ",
			expectMatch: true,
			description: "Teleport command with trailing space should match",
		},
		{
			name:        "teleport with digit 0",
			input:       "teleport 0",
			expectMatch: true,
			description: "Teleport with digit 0 should match",
		},
		{
			name:        "teleport with digit 1",
			input:       "teleport 1",
			expectMatch: true,
			description: "Teleport with digit 1 should match",
		},
		{
			name:        "teleport with digit 8",
			input:       "teleport 8",
			expectMatch: true,
			description: "Teleport with digit 8 should match",
		},
		{
			name:        "teleport with leading digit 9",
			input:       "teleport 9",
			expectMatch: true,
			description: "Teleport with leading digit 9 should match",
		},
		{
			name:        "teleport with two-digit number starting with 9",
			input:       "teleport 99",
			expectMatch: true,
			description: "Teleport with 99 should match",
		},
		{
			name:        "teleport with complete coordinates starting with 9",
			input:       "teleport 90 100",
			expectMatch: true,
			description: "Complete teleport command with coordinates starting with 9",
		},
	}

	commands := TextCommands{*teleportCommand}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test OneOrMoreCommandsMatch (this is what TextInput uses for color validation)
			matches := commands.OneOrMoreCommandsMatch(tc.input)

			if matches != tc.expectMatch {
				t.Errorf("%s: OneOrMoreCommandsMatch(%q) = %v, expected %v. %s",
					tc.name, tc.input, matches, tc.expectMatch, tc.description)

				// Additional debugging for the failing case
				if tc.input == "teleport 9" {
					t.Logf("DEBUG: Debugging teleport 9 failure...")

					// Test the MatchInt directly
					matchInt := MatchInt{IntMin: 0, IntMax: 255}
					partialMatch, err := matchInt.PartiallyMatches("9")
					t.Logf("MatchInt.PartiallyMatches('9') = %v, err = %v", partialMatch, err)

					// Test command matching step by step
					cmd := commands[0]
					result := cmd.GetTextCommandIfAtLeastPartialMatch(tc.input)
					t.Logf("GetTextCommandIfAtLeastPartialMatch result: %v", result != nil)
				}
			}
		})
	}
}

// TestMatchInt_DirectValidation tests MatchInt validation directly
func TestMatchInt_DirectValidation(t *testing.T) {
	matchInt := MatchInt{IntMin: 0, IntMax: 255}

	testCases := []struct {
		input    string
		expected bool
		desc     string
	}{
		{"0", true, "Zero should be valid"},
		{"1", true, "One should be valid"},
		{"8", true, "Eight should be valid"},
		{"9", true, "Nine should be valid"},
		{"10", true, "Ten should be valid"},
		{"99", true, "Ninety-nine should be valid"},
		{"255", true, "255 (max) should be valid"},
		{"256", false, "256 (over max) should be invalid"},
		{"-1", false, "Negative should be invalid"},
		{"abc", false, "Non-numeric should be invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := matchInt.PartiallyMatches(tc.input)

			if err != nil && tc.expected {
				t.Errorf("PartiallyMatches(%q) returned error: %v, expected success", tc.input, err)
				return
			}

			if result != tc.expected {
				t.Errorf("PartiallyMatches(%q) = %v, expected %v. %s", tc.input, result, tc.expected, tc.desc)
			}
		})
	}
}

// TestMatchInt_GetString_ShowsCorrectRange verifies GetString returns correct min-max range
// Previously had bug where IntMax was used twice instead of IntMin, IntMax
func TestMatchInt_GetString_ShowsCorrectRange(t *testing.T) {
	testCases := []struct {
		intMin   int
		intMax   int
		expected string
		desc     string
	}{
		{0, 999, "0 to 999", "Standard range"},
		{0, 999999, "0 to 999999", "Large range"},
		{1, 100, "1 to 100", "Non-zero minimum"},
		{-10, 10, "-10 to 10", "Negative to positive range"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			matchInt := MatchInt{IntMin: tc.intMin, IntMax: tc.intMax}
			result := matchInt.GetString()

			if result != tc.expected {
				t.Errorf("GetString() = %q, expected %q (IntMin=%d, IntMax=%d)",
					result, tc.expected, tc.intMin, tc.intMax)
			}
		})
	}
}
