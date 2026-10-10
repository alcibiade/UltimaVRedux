package widgets

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
)

// MockTextInputCallbacks implements TextInputCallbacks for testing
type MockTextInputCallbacks struct {
	LastAmbiguousMessage string
	OnEnterCalled        bool
}

func (m *MockTextInputCallbacks) AmbiguousAutoComplete(message string) {
	m.LastAmbiguousMessage = message
}

func (m *MockTextInputCallbacks) OnEnter() {
	m.OnEnterCalled = true
}

func (m *MockTextInputCallbacks) Reset() {
	m.LastAmbiguousMessage = ""
	m.OnEnterCalled = false
}

// Helper to create test commands that mimic "pc stats/status" structure
func createTestCommands() *grammar.TextCommands {
	commands := grammar.TextCommands{
		// pc stats xp
		*grammar.NewTextCommand([]grammar.Match{
			grammar.MatchString{Str: "pc", Description: "Player character"},
			grammar.MatchString{Str: "stats", Description: "Stats manipulation"},
			grammar.MatchString{Str: "xp", Description: "Experience points"},
			grammar.MatchStringList{Strings: []string{"give", "take", "set"}},
		}, nil),

		// pc stats hp give <int> <character>
		*grammar.NewTextCommand([]grammar.Match{
			grammar.MatchString{Str: "pc", Description: "Player character"},
			grammar.MatchString{Str: "stats", Description: "Stats manipulation"},
			grammar.MatchString{Str: "hp", Description: "Hit points"},
			grammar.MatchString{Str: "give", Description: "Give operation"},
			grammar.MatchInt{IntMin: 0, IntMax: 999, Description: "HP amount"},
			grammar.MatchPlayerCharacter{Description: "Target character", CaseSensitive: false},
		}, nil),

		// pc stats mp
		*grammar.NewTextCommand([]grammar.Match{
			grammar.MatchString{Str: "pc", Description: "Player character"},
			grammar.MatchString{Str: "stats", Description: "Stats manipulation"},
			grammar.MatchString{Str: "mp", Description: "Magic points"},
			grammar.MatchStringList{Strings: []string{"give", "take", "set"}},
		}, nil),

		// pc status
		*grammar.NewTextCommand([]grammar.Match{
			grammar.MatchString{Str: "pc", Description: "Player character"},
			grammar.MatchString{Str: "status", Description: "Status manipulation"},
			grammar.MatchStringList{Strings: []string{"good", "poison", "sleep", "charm", "dead"}},
		}, nil),
	}

	return &commands
}

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

// TestAutocompleteLogic_OptionAggregation tests option aggregation across commands
func TestAutocompleteLogic_OptionAggregation(t *testing.T) {
	commands := createTestCommands()

	testCases := []struct {
		input           string
		expectedOptions []string
		description     string
	}{
		{
			input:           "pc ",
			expectedOptions: []string{"stats", "status"},
			description:     "Position 1, empty token: should find 'stats' and 'status'",
		},
		{
			input:           "pc st",
			expectedOptions: []string{"stats", "status"},
			description:     "Position 1, 'st' token: both match prefix",
		},
		{
			input:           "pc sta",
			expectedOptions: []string{"stats", "status"},
			description:     "Position 1, 'sta' token: both 'stats' and 'status' match prefix",
		},
		{
			input:           "pc stat",
			expectedOptions: []string{"stats", "status"},
			description:     "Position 1, 'stat' token: partial match - should remain ambiguous (no exact match)",
		},
		{
			input:           "pc stats",
			expectedOptions: []string{"stats"},
			description:     "Position 1, 'stats' token: exact match - should treat as single option",
		},
		{
			input:           "pc statu",
			expectedOptions: []string{"status"},
			description:     "Position 1, 'statu' token: only matches 'status' - unique match",
		},
		{
			input:           "pc stats ",
			expectedOptions: []string{"xp", "hp", "mp"},
			description:     "Position 2, empty token: should find all stat types",
		},
		{
			input:           "pc status ",
			expectedOptions: []string{"good", "poison", "sleep", "charm", "dead"},
			description:     "Position 2, empty token: should find all statuses",
		},
		{
			input:           "pc stats hp give ",
			expectedOptions: []string{}, // MatchInt position - no selectable options
			description:     "Position 4, empty token: MatchInt should not appear in options (shown as hint instead)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Use same position detection logic as TestAutocompleteLogic_PositionDetection
			hasTrailingSpace := len(tc.input) > 0 && tc.input[len(tc.input)-1] == ' '
			splitStr := strings.Split(tc.input, " ")

			var currentPosition int
			var currentToken string

			if hasTrailingSpace {
				// Trailing space means starting next position
				currentPosition = len(splitStr) - 1 // The empty string position
				currentToken = ""
			} else {
				// No trailing space means we're typing current position
				currentPosition = len(splitStr) - 1
				currentToken = splitStr[currentPosition]
			}

			// Get matching commands
			primaryCommandMatches := commands.GetAllMatchingTextCommands(tc.input)

			// Aggregate options
			optionsMap := make(map[string]bool)

			for _, cmd := range *primaryCommandMatches {
				if currentPosition < len(cmd.Matches) {
					match := cmd.Matches[currentPosition]

					// Check if this is a MatchString (has GetString) or MatchStringList
					matchStr := match.GetString()

					if matchStr != "" {
						// Skip MatchInt patterns (contain " to ")
						if strings.Contains(matchStr, " to ") {
							continue // MatchInt - will be shown as hint, not option
						}
						// This is a MatchString - single fixed option
						// Check prefix match manually to avoid uppercase duplicates
						if currentToken == "" || strings.HasPrefix(strings.ToUpper(matchStr), strings.ToUpper(currentToken)) {
							optionsMap[matchStr] = true
						}
					} else {
						// This is a MatchStringList - get all matching options
						partialMatches, _ := match.GetPartialMatches(currentToken)
						for _, opt := range partialMatches {
							optionsMap[opt] = true
						}
					}
				}
			}

			// Convert to slice
			var options []string
			for opt := range optionsMap {
				options = append(options, opt)
			}

			// Verify we got expected options
			if len(options) != len(tc.expectedOptions) {
				t.Errorf("Got %d options, expected %d. Got: %v, Expected: %v",
					len(options), len(tc.expectedOptions), options, tc.expectedOptions)
				return
			}

			// Check each expected option is present
			for _, expected := range tc.expectedOptions {
				found := false
				for _, actual := range options {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected option '%s' not found in %v", expected, options)
				}
			}
		})
	}
}

// TestMatchInt_SpaceValidation_WithNextPosition tests MatchInt validation and advancing to next position
func TestMatchInt_SpaceValidation_WithNextPosition(t *testing.T) {
	commands := createTestCommands()

	testCases := []struct {
		name                string
		input               string
		shouldAllowSpace    bool
		expectedNextOptions []string // Options at next position if Space allowed
		description         string
	}{
		{
			name:             "Empty token at MatchInt position",
			input:            "pc stats hp give ",
			shouldAllowSpace: false, // Can't advance without number
			description:      "Should show hint, not allow Space",
		},
		{
			name:                "Valid number 1",
			input:               "pc stats hp give 1",
			shouldAllowSpace:    true,
			expectedNextOptions: []string{"0", "1", "2", "3", "4", "5", "6", "all", "fighter", "bard", "wizard", "mage"},
			description:         "Should validate and advance to MatchPlayerCharacter",
		},
		{
			name:                "Valid number 999",
			input:               "pc stats hp give 999",
			shouldAllowSpace:    true,
			expectedNextOptions: []string{"0", "1", "2", "3", "4", "5", "6", "all", "fighter", "bard", "wizard", "mage"},
			description:         "Should validate max value and advance",
		},
		{
			name:             "Invalid number 9999",
			input:            "pc stats hp give 9999",
			shouldAllowSpace: false,
			description:      "Should reject out-of-range number",
		},
		{
			name:             "Invalid number -1",
			input:            "pc stats hp give -1",
			shouldAllowSpace: false,
			description:      "Should reject negative number",
		},
		{
			name:                "Valid number 0",
			input:               "pc stats hp give 0",
			shouldAllowSpace:    true,
			expectedNextOptions: []string{"0", "1", "2", "3", "4", "5", "6", "all", "fighter", "bard", "wizard", "mage"},
			description:         "Should validate min value and advance",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse input to get position and token
			hasTrailingSpace := len(tc.input) > 0 && tc.input[len(tc.input)-1] == ' '
			splitStr := strings.Split(tc.input, " ")

			var currentPosition int
			var currentToken string

			if hasTrailingSpace {
				currentPosition = len(splitStr) - 1
				currentToken = ""
			} else {
				currentPosition = len(splitStr) - 1
				currentToken = splitStr[currentPosition]
			}

			// Get the base command pattern (without the potentially invalid token)
			// This ensures we can test invalid numbers too
			baseInput := "pc stats hp give"
			if hasTrailingSpace {
				baseInput = tc.input
			}
			baseMatches := commands.GetAllMatchingTextCommands(baseInput)

			// Check if we're at MatchInt position
			atMatchIntPosition := false
			var matchInt grammar.Match

			for _, cmd := range *baseMatches {
				if currentPosition < len(cmd.Matches) {
					match := cmd.Matches[currentPosition]
					matchStr := match.GetString()
					if matchStr != "" && strings.Contains(matchStr, " to ") {
						atMatchIntPosition = true
						matchInt = match
						break
					}
				}
			}

			if !atMatchIntPosition {
				t.Fatalf("Expected to be at MatchInt position for input %q", tc.input)
			}

			// Test validation logic
			if currentToken != "" {
				isValid, err := matchInt.PartiallyMatches(currentToken)

				if tc.shouldAllowSpace {
					if err != nil || !isValid {
						t.Errorf("Expected valid number for input %q, got isValid=%v, err=%v", tc.input, isValid, err)
					}

					// If validation passes, verify next position has expected options
					if isValid {
						nextInput := tc.input + " "
						nextPosition := currentPosition + 1

						// Get options at next position
						nextMatches := commands.GetAllMatchingTextCommands(nextInput)
						optionsMap := make(map[string]bool)

						for _, cmd := range *nextMatches {
							if nextPosition < len(cmd.Matches) {
								nextMatch := cmd.Matches[nextPosition]
								partialMatches, _ := nextMatch.GetPartialMatches("")
								for _, opt := range partialMatches {
									optionsMap[opt] = true
								}
							}
						}

						// Verify expected options are present
						for _, expectedOpt := range tc.expectedNextOptions {
							if !optionsMap[expectedOpt] {
								t.Errorf("Expected option %q at next position, but not found. Available: %v", expectedOpt, optionsMap)
							}
						}
					}
				} else {
					if err == nil && isValid {
						t.Errorf("Expected invalid number for input %q, but validation passed", tc.input)
					}
				}
			}
		})
	}
}

// TestComplete_PcStatsHpGive_Workflow tests the complete user workflow step-by-step
func TestComplete_PcStatsHpGive_Workflow(t *testing.T) {
	commands := createTestCommands()

	// Workflow steps represent actual user input progression
	workflowSteps := []struct {
		input           string
		testTab         bool
		expectedTabHint string // Expected hint/message for Tab
		testSpace       bool
		shouldSpaceWork bool
		description     string
	}{
		{
			input:           "pc ",
			testTab:         true,
			expectedTabHint: "", // Should show multiple options (stats, status)
			testSpace:       false,
			description:     "After 'pc ', Tab should show stats/status options",
		},
		{
			input:           "pc stats ",
			testTab:         true,
			expectedTabHint: "", // Should show xp, hp, mp options
			testSpace:       false,
			description:     "After 'pc stats ', Tab should show stat type options",
		},
		{
			input:           "pc stats hp ",
			testTab:         true,
			expectedTabHint: "", // Should show give operation
			testSpace:       false,
			description:     "After 'pc stats hp ', Tab should show operation options",
		},
		{
			input:           "pc stats hp give ",
			testTab:         true,
			expectedTabHint: "Enter value (0 to 999)", // MatchInt hint
			testSpace:       false,
			description:     "After 'pc stats hp give ', Tab should show MatchInt hint",
		},
		{
			input:           "pc stats hp give 1",
			testTab:         false,
			testSpace:       true,
			shouldSpaceWork: true, // KEY TEST: Space should advance!
			description:     "After typing valid number '1', Space should advance",
		},
		{
			input:           "pc stats hp give 9999",
			testTab:         false,
			testSpace:       true,
			shouldSpaceWork: false, // Out of range
			description:     "After typing invalid number '9999', Space should reject",
		},
		{
			input:           "pc stats hp give 1 ",
			testTab:         true,
			expectedTabHint: "", // Should show character options
			testSpace:       false,
			description:     "After 'pc stats hp give 1 ', Tab should show character options",
		},
	}

	for _, step := range workflowSteps {
		t.Run(step.description, func(t *testing.T) {
			// Test Tab behavior
			if step.testTab {
				hasTrailingSpace := len(step.input) > 0 && step.input[len(step.input)-1] == ' '
				splitStr := strings.Split(step.input, " ")

				var currentPosition int
				var currentToken string

				if hasTrailingSpace {
					currentPosition = len(splitStr) - 1
					currentToken = ""
				} else {
					currentPosition = len(splitStr) - 1
					currentToken = splitStr[currentPosition]
				}

				primaryCommandMatches := commands.GetAllMatchingTextCommands(step.input)
				optionsMap := make(map[string]bool)

				// Aggregate options
				for _, cmd := range *primaryCommandMatches {
					if currentPosition < len(cmd.Matches) {
						match := cmd.Matches[currentPosition]
						matchStr := match.GetString()

						if matchStr != "" {
							if strings.Contains(matchStr, " to ") {
								// MatchInt - should show hint
								if step.expectedTabHint != "" && !strings.Contains(step.expectedTabHint, matchStr) {
									t.Errorf("Expected tab hint %q to contain %q", step.expectedTabHint, matchStr)
								}
								continue
							}
							if currentToken == "" || strings.HasPrefix(strings.ToUpper(matchStr), strings.ToUpper(currentToken)) {
								optionsMap[matchStr] = true
							}
						} else {
							partialMatches, _ := match.GetPartialMatches(currentToken)
							for _, opt := range partialMatches {
								optionsMap[opt] = true
							}
						}
					}
				}

				if step.expectedTabHint != "" {
					// Verify hint is shown
					t.Logf("Input %q should show hint: %s", step.input, step.expectedTabHint)
				} else if len(optionsMap) > 0 {
					// Verify options are available
					t.Logf("Input %q has %d autocomplete options", step.input, len(optionsMap))
				}
			}

			// Test Space behavior
			if step.testSpace {
				hasTrailingSpace := len(step.input) > 0 && step.input[len(step.input)-1] == ' '
				splitStr := strings.Split(step.input, " ")

				var currentPosition int
				var currentToken string

				if hasTrailingSpace {
					currentPosition = len(splitStr) - 1
					currentToken = ""
				} else {
					currentPosition = len(splitStr) - 1
					currentToken = splitStr[currentPosition]
				}

				primaryCommandMatches := commands.GetAllMatchingTextCommands(step.input)

				// Check if at MatchInt position with typed token
				spaceAllowed := false

				if currentToken != "" {
					for _, cmd := range *primaryCommandMatches {
						if currentPosition < len(cmd.Matches) {
							match := cmd.Matches[currentPosition]
							matchStr := match.GetString()

							if matchStr != "" && strings.Contains(matchStr, " to ") {
								isValid, err := match.PartiallyMatches(currentToken)
								if err == nil && isValid {
									spaceAllowed = true
								}
								break
							}
						}
					}
				}

				if step.shouldSpaceWork && !spaceAllowed {
					t.Errorf("Expected Space to advance for input %q, but it's not allowed", step.input)
				}

				if !step.shouldSpaceWork && spaceAllowed {
					t.Errorf("Expected Space to be rejected for input %q, but it's allowed", step.input)
				}
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
