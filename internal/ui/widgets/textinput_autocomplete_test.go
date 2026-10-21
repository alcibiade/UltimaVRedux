package widgets

import (
	"strings"
	"testing"
)

// TestTextInput_AutoComplete_TrailingSpace tests autocomplete with trailing space
func TestTextInput_AutoComplete_TrailingSpace(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedInMsg []string // Options that MUST appear in message
		description   string
	}{
		{
			name:          "pc with trailing space",
			input:         "pc ",
			expectedInMsg: []string{"stats", "status"},
			description:   "Should show both stats and status options",
		},
		{
			name:          "pc stats with trailing space",
			input:         "pc stats ",
			expectedInMsg: []string{"xp", "hp", "mp"},
			description:   "Should show all stat type options",
		},
		{
			name:          "pc status with trailing space",
			input:         "pc status ",
			expectedInMsg: []string{"good", "poison", "sleep", "charm", "dead"},
			description:   "Should show all status options",
		},
		{
			name:          "pc stats xp with trailing space",
			input:         "pc stats xp ",
			expectedInMsg: []string{"give", "take", "set"},
			description:   "Should show all operation options",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Implement full TextInput test
			// For now, just document expected behavior
			t.Logf("Input: '%s'", tc.input)
			t.Logf("Expected in message: %v", tc.expectedInMsg)
			t.Logf("Description: %s", tc.description)
		})
	}
}

// TestTextInput_AutoComplete_PartialToken tests autocomplete with partial typing
func TestTextInput_AutoComplete_PartialToken(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedInMsg  []string // Options that MUST appear
		shouldComplete bool     // Should auto-complete to single option
		completeText   string   // If shouldComplete, what should it add?
		description    string
	}{
		{
			name:           "pc st partial",
			input:          "pc st",
			expectedInMsg:  []string{"stats", "status"},
			shouldComplete: false,
			description:    "Should show both options matching 'st' prefix",
		},
		{
			name:           "pc sta partial",
			input:          "pc sta",
			shouldComplete: true,
			completeText:   "ts", // Completes "stats"
			description:    "Should auto-complete to 'stats' (unique match)",
		},
		{
			name:           "pc status p partial",
			input:          "pc status p",
			shouldComplete: true,
			completeText:   "oison", // Completes "poison"
			description:    "Should auto-complete to 'poison' (unique match)",
		},
		{
			name:           "pc status s partial",
			input:          "pc status s",
			shouldComplete: true,
			completeText:   "leep", // Completes "sleep"
			description:    "Should auto-complete to 'sleep' (unique match)",
		},
		{
			name:           "pc stats x partial",
			input:          "pc stats x",
			shouldComplete: true,
			completeText:   "p", // Completes "xp"
			description:    "Should auto-complete to 'xp' (unique match)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Implement full TextInput test
			t.Logf("Input: '%s'", tc.input)
			if tc.shouldComplete {
				t.Logf("Should auto-complete with: '%s'", tc.completeText)
			} else {
				t.Logf("Should show options: %v", tc.expectedInMsg)
			}
			t.Logf("Description: %s", tc.description)
		})
	}
}

// TestTextInput_AutoComplete_NoMatches tests autocomplete with no matching commands
func TestTextInput_AutoComplete_NoMatches(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldShow  bool
		description string
	}{
		{
			name:        "xyz no match",
			input:       "xyz",
			shouldShow:  false,
			description: "Should not show autocomplete for non-matching input",
		},
		{
			name:        "pc wrong partial",
			input:       "pc wrong",
			shouldShow:  false,
			description: "Should not show autocomplete when no position matches",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Implement full TextInput test
			t.Logf("Input: '%s'", tc.input)
			t.Logf("Should show autocomplete: %v", tc.shouldShow)
			t.Logf("Description: %s", tc.description)
		})
	}
}

// TestAutocompleteLogic_PositionDetection tests the position detection logic
func TestAutocompleteLogic_PositionDetection(t *testing.T) {
	testCases := []struct {
		input            string
		expectedPosition int
		expectedToken    string
		description      string
	}{
		{
			input:            "pc",
			expectedPosition: 0,
			expectedToken:    "pc",
			description:      "Single token, position 0",
		},
		{
			input:            "pc ",
			expectedPosition: 1,
			expectedToken:    "",
			description:      "Trailing space, position 1 with empty token",
		},
		{
			input:            "pc st",
			expectedPosition: 1,
			expectedToken:    "st",
			description:      "Two tokens, position 1 with partial",
		},
		{
			input:            "pc stats ",
			expectedPosition: 2,
			expectedToken:    "",
			description:      "Two complete tokens + space, position 2 with empty",
		},
		{
			input:            "pc stats xp",
			expectedPosition: 2,
			expectedToken:    "xp",
			description:      "Three tokens, position 2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Proper position detection algorithm
			hasTrailingSpace := len(tc.input) > 0 && tc.input[len(tc.input)-1] == ' '

			// Split and handle trailing space properly
			splitStr := strings.Split(tc.input, " ")

			var currentPosition int
			var currentToken string

			if hasTrailingSpace {
				// Trailing space means starting next position
				// "pc " splits to ["pc", ""] - we want position 1 with empty token
				currentPosition = len(splitStr) - 1 // The empty string position
				currentToken = ""
			} else {
				// No trailing space means we're typing current position
				currentPosition = len(splitStr) - 1
				currentToken = splitStr[currentPosition]
			}

			if currentPosition != tc.expectedPosition {
				t.Errorf("Position: got %d, expected %d", currentPosition, tc.expectedPosition)
			}

			if currentToken != tc.expectedToken {
				t.Errorf("Token: got '%s', expected '%s'", currentToken, tc.expectedToken)
			}
		})
	}
}

// TestLongestCommonPrefix_Integration tests the complete flow with common prefix
// This test documents expected behavior - full integration would require TextInput setup
func TestLongestCommonPrefix_Integration(t *testing.T) {
	testCases := []struct {
		name            string
		options         []string
		currentToken    string
		expectedSuffix  string
		expectedMessage string
	}{
		{
			name:            "pc st → stat (common prefix)",
			options:         []string{"stats", "status"},
			currentToken:    "st",
			expectedSuffix:  "at",
			expectedMessage: "Options: stats, status",
		},
		{
			name:            "pc stat → no suffix (at prefix)",
			options:         []string{"stats", "status"},
			currentToken:    "stat",
			expectedSuffix:  "",
			expectedMessage: "Options: stats, status",
		},
		{
			name:            "pc s → tat (completes to stat)",
			options:         []string{"stats", "status"},
			currentToken:    "s",
			expectedSuffix:  "tat", // Should autocomplete from "s" to "stat"
			expectedMessage: "Options: stats, status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the findLongestCommonPrefix logic
			commonPrefix := findLongestCommonPrefix(tc.options)
			currentTokenLower := strings.ToLower(tc.currentToken)

			var actualSuffix string
			if len(commonPrefix) > len(currentTokenLower) {
				actualSuffix = commonPrefix[len(currentTokenLower):]
			}

			if actualSuffix != tc.expectedSuffix {
				t.Errorf("Expected suffix %q, got %q (commonPrefix=%q, currentToken=%q)",
					tc.expectedSuffix, actualSuffix, commonPrefix, tc.currentToken)
			}

			// Verify the logic would show the message
			t.Logf("Would show: %s", tc.expectedMessage)
		})
	}
}

// TestLongestCommonPrefix_Helper tests the findLongestCommonPrefix helper directly
func TestLongestCommonPrefix_Helper(t *testing.T) {
	testCases := []struct {
		name     string
		options  []string
		expected string
	}{
		{
			name:     "stats and status → stat",
			options:  []string{"stats", "status"},
			expected: "stat",
		},
		{
			name:     "give, take, set → empty (no common prefix)",
			options:  []string{"give", "take", "set"},
			expected: "",
		},
		{
			name:     "fighter, fighter → fighter",
			options:  []string{"fighter", "fighter"},
			expected: "fighter",
		},
		{
			name:     "Single option → returns option",
			options:  []string{"give"},
			expected: "give",
		},
		{
			name:     "Empty list → empty string",
			options:  []string{},
			expected: "",
		},
		{
			name:     "Case insensitive: Stats, STATUS → stat",
			options:  []string{"Stats", "STATUS"},
			expected: "stat",
		},
		{
			name:     "hp, hpmax → hp",
			options:  []string{"hp", "hpmax"},
			expected: "hp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findLongestCommonPrefix(tc.options)
			if result != tc.expected {
				t.Errorf("findLongestCommonPrefix(%v) = %q, expected %q",
					tc.options, result, tc.expected)
			}
		})
	}
}
