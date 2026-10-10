package grammar

type Match interface {
	PartiallyMatches(string) (bool, error)
	GetPartialMatches(string) ([]string, error)
	GetSuffixHint(currentStr string) string
	GetString() string
	GetDescription() string
	ShouldAutofillWithFirstCharacter() bool
	// CouldOverlapWith returns true if this match slot could accept at least one
	// input string that the other match slot also accepts. Used for ambiguity
	// detection between commands with the same slot count.
	CouldOverlapWith(other Match) bool
}
