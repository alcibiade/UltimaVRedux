package grammar

import (
	"strconv"
	"strings"
)

// MatchPlayerCharacter matches player character targeting strings:
// - Numeric: 0 (all), 1-6 (character indices)
// - Keyword: "all"
// - Class names: "fighter", "bard", "wizard", "mage"
type MatchPlayerCharacter struct {
	Description   string
	CaseSensitive bool
}

// ShouldAutofillWithFirstCharacter returns false - we want to show all options
func (m MatchPlayerCharacter) ShouldAutofillWithFirstCharacter() bool {
	return false
}

// GetDescription returns the matcher description
func (m MatchPlayerCharacter) GetDescription() string {
	return m.Description
}

// GetString returns empty string (no single fixed value)
func (m MatchPlayerCharacter) GetString() string {
	return ""
}

// GetPartialMatches returns all valid inputs that match the partial string
func (m MatchPlayerCharacter) GetPartialMatches(s string) ([]string, error) {
	matches := make([]string, 0)

	if s == "" {
		// Empty input - show all options
		return m.getAllOptions(), nil
	}

	// Normalize input for comparison
	input := s
	if !m.CaseSensitive {
		input = strings.ToLower(s)
	}

	// Try numeric match (0-6)
	if n, err := strconv.Atoi(s); err == nil {
		if n >= 0 && n <= 6 {
			matches = append(matches, s)
		}

		return matches, nil
	}

	// Try keyword match
	if m.matchesPrefix("all", input) {
		matches = append(matches, "all")
	}

	// Try class name matches
	classes := []string{"fighter", "bard", "wizard", "mage"}
	for _, class := range classes {
		if m.matchesPrefix(class, input) {
			matches = append(matches, class)
		}
	}

	return matches, nil
}

// PartiallyMatches checks if the string is a valid partial or complete match
func (m MatchPlayerCharacter) PartiallyMatches(str string) (bool, error) {
	matches, err := m.GetPartialMatches(str)

	return len(matches) > 0, err
}

// GetSuffixHint returns autocomplete hint for current input
func (m MatchPlayerCharacter) GetSuffixHint(currentStr string) string {
	matches, err := m.GetPartialMatches(currentStr)
	if err != nil {
		return ""
	}

	// Only show hint if exactly one match
	if len(matches) != 1 {
		return ""
	}

	match := matches[0]
	if m.CaseSensitive {
		return strings.TrimPrefix(match, currentStr)
	}

	return strings.TrimPrefix(strings.ToLower(match), strings.ToLower(currentStr))
}

// matchesPrefix checks if target starts with prefix (case-insensitive if configured)
func (m MatchPlayerCharacter) matchesPrefix(target, prefix string) bool {
	if !m.CaseSensitive {
		return strings.HasPrefix(strings.ToLower(target), strings.ToLower(prefix))
	}

	return strings.HasPrefix(target, prefix)
}

// getAllOptions returns all valid options for autocomplete when input is empty
func (m MatchPlayerCharacter) getAllOptions() []string {
	return []string{
		"0",   // all
		"1",   // character index 1
		"2",   // character index 2
		"3",   // character index 3
		"4",   // character index 4
		"5",   // character index 5
		"6",   // character index 6
		"all", // keyword for all
		"fighter",
		"bard",
		"wizard",
		"mage",
	}
}

// CouldOverlapWith returns true conservatively. MatchPlayerCharacter accepts
// integers 0-6, "all", and class names ("fighter", "bard", "wizard", "mage"),
// so it could overlap with many match types.
func (m MatchPlayerCharacter) CouldOverlapWith(_ Match) bool {
	return true
}
