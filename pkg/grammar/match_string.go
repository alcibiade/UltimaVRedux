package grammar

import "strings"

type MatchString struct {
	Str           string
	Description   string
	CaseSensitive bool
}

func (m MatchString) ShouldAutofillWithFirstCharacter() bool {
	return false
}

func (m MatchString) GetDescription() string {
	return m.Description
}

func (m MatchString) GetString() string {
	return m.Str
}

func (m MatchString) GetPartialMatches(s string) ([]string, error) {
	if s == "" {
		return []string{}, nil
	}
	if strings.HasPrefix(strings.ToUpper(m.Str), strings.ToUpper(s)) {
		return []string{strings.ToUpper(s)}, nil
	}

	return []string{}, nil
}

func (m MatchString) PartiallyMatches(str string) (bool, error) {
	matches, err := m.GetPartialMatches(str)

	return len(matches) > 0, err
}

func (m MatchString) GetSuffixHint(currentStr string) string {
	checkMatch, err := m.PartiallyMatches(currentStr)
	if err != nil || !checkMatch {
		// don't give a hint because it doesn't match in the first place
		return ""
	}

	// return the second half after the matched prefix
	if m.CaseSensitive {
		return strings.TrimPrefix(m.Str, currentStr)
	}

	return strings.TrimPrefix(strings.ToUpper(m.Str), strings.ToUpper(currentStr))
}

// CouldOverlapWith returns true if another match slot could accept the same
// input as this MatchString. Overlap means a user could type a prefix that
// PartiallyMatches both this slot and the other slot.
func (m MatchString) CouldOverlapWith(other Match) bool {
	switch o := other.(type) {
	case MatchString:
		return prefixOverlaps(m.Str, o.Str)
	case MatchStringList:
		return anyStringSharesFirstChar(m.Str, o.Strings)
	case MatchInt:
		return anyStringOverlapsInt([]string{m.Str}, o.IntMin, o.IntMax)
	case MatchPlayerCharacter:
		// Conservative: MatchPlayerCharacter accepts ints, "all", class names
		return true
	default:
		// Unknown type — conservative true
		return true
	}
}
