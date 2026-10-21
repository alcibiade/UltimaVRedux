package grammar

import (
	"strings"
)

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
