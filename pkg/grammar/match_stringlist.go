package grammar

import "strings"

// MatchStringList matches typed input against a fixed list of accepted strings.
type MatchStringList struct {
	Strings              []string
	Description          string
	CaseSensitive        bool
	SingleCharacterInput bool
}

// ShouldAutofillWithFirstCharacter reports whether typing one character should autofill this slot.
func (m MatchStringList) ShouldAutofillWithFirstCharacter() bool {
	return m.SingleCharacterInput
}

// GetDescription returns the text that describes this slot in command listings.
func (m MatchStringList) GetDescription() string {
	return m.Description
}

// GetPartialMatches returns the accepted strings that start with inputString.
// Matching ignores case unless CaseSensitive is set.
func (m MatchStringList) GetPartialMatches(inputString string) ([]string, error) {
	matches := make([]string, 0)
	if !m.CaseSensitive {
		inputString = strings.ToUpper(inputString)
	}
	for _, str := range m.Strings {
		if strings.HasPrefix(strings.ToUpper(str), inputString) {
			matches = append(matches, str)
		}
	}

	return matches, nil
}

// PartiallyMatches reports whether any accepted string starts with inputString.
func (m MatchStringList) PartiallyMatches(inputString string) (bool, error) {
	var matchErr error
	matches, matchErr := m.GetPartialMatches(inputString)
	_ = matchErr // Error intentionally ignored - partial match failure not critical

	return len(matches) > 0, nil
}

// GetSuffixHint returns the rest of the only accepted string that starts with currentStr.
// It returns empty when zero or several strings match.
func (m MatchStringList) GetSuffixHint(currentStr string) string {
	strs := m.getListOfMatches(currentStr)
	if len(*strs) != 1 {
		return ""
	}
	str := (*strs)[0]
	if m.CaseSensitive {
		return strings.TrimPrefix(str, currentStr)
	}

	return strings.TrimPrefix(strings.ToUpper(str), strings.ToUpper(currentStr))
}

// GetString returns no fixed word, because a string list accepts several.
func (m MatchStringList) GetString() string {
	return ""
}

func (m MatchStringList) getListOfMatches(currentStr string) *[]string {
	strs := make([]string, 0, len(m.Strings))

	if !m.CaseSensitive {
		currentStr = strings.ToUpper(currentStr)
	}
	for _, str := range m.Strings {
		if strings.HasPrefix(strings.ToUpper(str), currentStr) {
			strs = append(strs, str)
		}
	}

	return &strs
}

// CouldOverlapWith returns true if another match slot could accept the same
// input as this MatchStringList. Checks whether any string in this list
// prefix-matches any accepted value of the other match.
func (m MatchStringList) CouldOverlapWith(other Match) bool {
	if o, ok := other.(MatchStringList); ok {
		return anyStringListOverlap(m.Strings, o.Strings)
	}

	return true
}
