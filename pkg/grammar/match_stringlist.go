package grammar

import "strings"

type MatchStringList struct {
	Strings              []string
	Description          string
	CaseSensitive        bool
	SingleCharacterInput bool
}

func (m MatchStringList) ShouldAutofillWithFirstCharacter() bool {
	return m.SingleCharacterInput
}

func (m MatchStringList) GetDescription() string {
	return m.Description
}

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

func (m MatchStringList) PartiallyMatches(inputString string) (bool, error) {
	var matchErr error
	matches, matchErr := m.GetPartialMatches(inputString)
	_ = matchErr // Error intentionally ignored - partial match failure not critical

	return len(matches) > 0, nil
}

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
	switch o := other.(type) {
	case MatchString:
		return anyStringSharesFirstChar(o.Str, m.Strings)
	case MatchStringList:
		return anyStringListOverlap(m.Strings, o.Strings)
	case MatchInt:
		return anyStringOverlapsInt(m.Strings, o.IntMin, o.IntMax)
	case MatchPlayerCharacter:
		// Conservative: MatchPlayerCharacter accepts ints, "all", class names
		return true
	default:
		return true
	}
}
