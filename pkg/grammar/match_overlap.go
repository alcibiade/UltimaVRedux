package grammar

import (
	"strconv"
	"strings"
)

// prefixOverlaps returns true if either string is a case-insensitive prefix of the other.
// This mirrors the PartiallyMatches behavior: "s" matches "south" because
// HasPrefix("SOUTH", "S") is true.
func prefixOverlaps(a, b string) bool {
	ua := strings.ToUpper(a)
	ub := strings.ToUpper(b)

	return strings.HasPrefix(ua, ub) || strings.HasPrefix(ub, ua)
}

// anyStringOverlapsInt returns true if any string in the list could parse as
// an integer within [min, max]. This is conservative: we check all list items.
func anyStringOverlapsInt(strs []string, intMin, intMax int) bool {
	for _, s := range strs {
		n, err := strconv.Atoi(s)
		if err == nil && n >= intMin && n <= intMax {
			return true
		}
	}

	return false
}

// anyStringSharesFirstChar returns true if the single string shares a
// case-insensitive first character with any string in the list. This mirrors
// PartiallyMatches: typing the first character of `single` would also match
// any list item starting with that same character.
func anyStringSharesFirstChar(single string, list []string) bool {
	if len(single) == 0 {
		return false
	}

	firstChar := strings.ToUpper(single)[0]

	for _, item := range list {
		if len(item) > 0 && strings.ToUpper(item)[0] == firstChar {
			return true
		}
	}

	return false
}

// anyStringListOverlap returns true if any string from listA and any string
// from listB could both be matched by the same user-typed prefix. Two strings
// overlap if they share a common prefix of length >= 1 (case-insensitive),
// since the user could type that shared prefix and PartiallyMatches would
// accept it for both items.
//
// For example, "south" and "skeleton" share the prefix "s", so typing "s"
// matches both. "east" and "ettin" share "e".
func anyStringListOverlap(listA, listB []string) bool {
	// Build a set of first characters from listA
	firstCharsA := make(map[byte]bool, len(listA))
	for _, a := range listA {
		if len(a) > 0 {
			firstCharsA[strings.ToUpper(a)[0]] = true
		}
	}

	// Check if any item in listB shares a first character
	for _, b := range listB {
		if len(b) > 0 && firstCharsA[strings.ToUpper(b)[0]] {
			return true
		}
	}

	return false
}
