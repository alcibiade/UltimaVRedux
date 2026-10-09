package references

import (
	"testing"
)

// TestLocationStringBug demonstrates BUG23 - Location.String() corruption
func TestLocationStringBug_ReproducesCorruption(t *testing.T) {
	tests := []struct {
		location Location
		expected string
		bugDesc  string
	}{
		{Moonglow, "Moonglow", "Moonglow should display correctly (not 'oonglowB')"},
		{Britain, "Britain", "Britain should display correctly (not 'ritainJ')"},
		{Jhelom, "Jhelom", "Jhelom should display correctly"},
		{Yew, "Yew", "Yew should display correctly"},
		{Trinsic, "Trinsic", "Trinsic should display correctly"},
		{BritanniaUnderworld, "BritanniaUnderworld", "BritanniaUnderworld should display correctly"},
	}

	t.Logf("=== BUG23 REPRODUCTION TEST ===")
	t.Logf("Testing Location.String() method for character trimming corruption")
	t.Logf("Expected: All location names display correctly")
	t.Logf("Bug Behavior: First character trimmed + extra character appended")

	for i, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			actual := test.location.String()

			t.Logf("Test %d: %s", i+1, test.bugDesc)
			t.Logf("  Location enum value: %d", int(test.location))
			t.Logf("  Expected: '%s'", test.expected)
			t.Logf("  Actual:   '%s'", actual)

			if actual != test.expected {
				t.Logf("  🐛 BUG REPRODUCED: Corruption detected!")
				t.Logf("  Analysis:")
				if len(actual) > 0 && len(test.expected) > 0 {
					if len(actual) == len(test.expected) {
						t.Logf("    - Same length, different content")
					} else {
						t.Logf("    - Length mismatch: expected %d, got %d", len(test.expected), len(actual))
					}

					// Analyze character differences
					expectedRunes := []rune(test.expected)
					actualRunes := []rune(actual)
					if len(expectedRunes) > 0 && len(actualRunes) > 0 {
						t.Logf("    - Expected first char: '%c' (0x%X)", expectedRunes[0], expectedRunes[0])
						t.Logf("    - Actual first char:   '%c' (0x%X)", actualRunes[0], actualRunes[0])
						if len(actualRunes) > 1 && len(expectedRunes) > 1 {
							t.Logf("    - Expected last char:  '%c' (0x%X)", expectedRunes[len(expectedRunes)-1], expectedRunes[len(expectedRunes)-1])
							t.Logf("    - Actual last char:    '%c' (0x%X)", actualRunes[len(actualRunes)-1], actualRunes[len(actualRunes)-1])
						}
					}
				}

				// This test should FAIL initially to demonstrate the bug
				t.Errorf("❌ BUG23 REPRODUCED: Location.String() returned '%s', expected '%s'", actual, test.expected)
			} else {
				t.Logf("  ✅ Location name correct")
			}
		})
	}
}

// TestLocationStringIndexAnalysis helps debug the string generation logic
func TestLocationStringIndexAnalysis_DebugInfo(t *testing.T) {
	t.Logf("=== LOCATION STRING GENERATION ANALYSIS ===")
	t.Logf("Analyzing the generated string constants and indexes")

	t.Logf("_Location_name_0 string length: %d", len(_Location_name_0))
	t.Logf("_Location_index_0 array length: %d", len(_Location_index_0))

	// Analyze first few locations to understand the pattern
	locations := []Location{BritanniaUnderworld, Moonglow, Britain, Jhelom, Yew}
	for i, loc := range locations {
		if int(loc) < len(_Location_index_0)-1 {
			startIdx := _Location_index_0[loc]
			endIdx := _Location_index_0[loc+1]

			t.Logf("Location %d (%s enum value):", i, loc.String())
			t.Logf("  Enum value: %d", int(loc))
			t.Logf("  Start index: %d", startIdx)
			t.Logf("  End index: %d", endIdx)
			t.Logf("  Calculated length: %d", endIdx-startIdx)
			t.Logf("  Substring: '%s'", _Location_name_0[startIdx:endIdx])
		}
	}

	// Show part of the concatenated string
	t.Logf("First 100 characters of _Location_name_0:")
	if len(_Location_name_0) >= 100 {
		t.Logf("  '%s'", _Location_name_0[:100])
	} else {
		t.Logf("  '%s'", _Location_name_0)
	}
}
