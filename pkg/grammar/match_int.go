package grammar

import (
	"fmt"
	"strconv"
)

type MatchInt struct {
	IntMin      int
	IntMax      int
	Description string
}

func (m MatchInt) ShouldAutofillWithFirstCharacter() bool {
	return false
}

func (m MatchInt) GetDescription() string {
	return m.Description
}

func (m MatchInt) GetPartialMatches(s string) ([]string, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return []string{}, fmt.Errorf("parsing integer match: %w", err)
	}
	if n >= m.IntMin && n <= m.IntMax {
		return []string{strconv.Itoa(n)}, nil
	}

	return []string{}, nil
}

func (m MatchInt) GetString() string {
	return fmt.Sprintf("%d to %d", m.IntMin, m.IntMax)
}

func (m MatchInt) PartiallyMatches(str string) (bool, error) {
	n, err := strconv.Atoi(str)
	if err != nil {
		return false, fmt.Errorf("parsing integer match: %w", err)
	}

	return n >= m.IntMin && n <= m.IntMax, nil
}

func (m MatchInt) GetSuffixHint(_ string) string {
	return ""
}

// CouldOverlapWith returns true if another match slot could accept the same
// input as this MatchInt. For MatchInt vs MatchInt, checks range intersection.
// For string types, checks if any string parses as an integer within range.
func (m MatchInt) CouldOverlapWith(other Match) bool {
	switch o := other.(type) {
	case MatchString:
		return anyStringOverlapsInt([]string{o.Str}, m.IntMin, m.IntMax)
	case MatchStringList:
		return anyStringOverlapsInt(o.Strings, m.IntMin, m.IntMax)
	case MatchInt:
		// Ranges overlap if one starts before the other ends
		return m.IntMin <= o.IntMax && o.IntMin <= m.IntMax
	case MatchPlayerCharacter:
		// MatchPlayerCharacter accepts ints 0-6, so check range intersection
		return m.IntMin <= 6 && 0 <= m.IntMax
	default:
		return true
	}
}
