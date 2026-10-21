// Package grammar matches typed command words against per-slot vocabularies and
// resolves the command that the typed words fit.
package grammar

// Match is one word slot of a text command: the vocabulary a typed word is checked against.
type Match interface {
	PartiallyMatches(inputString string) (bool, error)
	GetPartialMatches(inputString string) ([]string, error)
	GetSuffixHint(currentStr string) string
	GetString() string
	GetDescription() string
	ShouldAutofillWithFirstCharacter() bool
	// CouldOverlapWith returns true if this match slot could accept at least one
	// input string that the other match slot also accepts. Used for ambiguity
	// detection between commands with the same slot count.
	CouldOverlapWith(other Match) bool
}
