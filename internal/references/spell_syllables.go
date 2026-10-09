package references

// Ultima V uses 24 magic syllables (no J or O).
// Each syllable is uniquely identified by its first letter.

// GetSyllableMap returns a map of first-letter abbreviation to full syllable word.
// Used for spell input parsing and display throughout the magic system.
func GetSyllableMap() map[rune]string {
	return map[rune]string{
		'A': "An", 'B': "Bet", 'C': "Corp", 'D': "Des",
		'E': "Ex", 'F': "Flam", 'G': "Grav", 'H': "Hur",
		'I': "In", 'K': "Kal", 'L': "Lor", 'M': "Mani",
		'N': "Nox", 'P': "Por", 'Q': "Quas", 'R': "Rel",
		'S': "Sanct", 'T': "Tym", 'U': "Uus", 'V': "Vas",
		'W': "Wis", 'X': "Xen", 'Y': "Ylem", 'Z': "Zu",
	}
}

// IsValidSyllable returns true if the rune represents a valid spell syllable.
func IsValidSyllable(r rune) bool {
	_, exists := GetSyllableMap()[r]

	return exists
}

// GetSyllableWord returns the full word for a syllable abbreviation.
// Returns the word and true if found, empty string and false otherwise.
func GetSyllableWord(r rune) (string, bool) {
	word, exists := GetSyllableMap()[r]

	return word, exists
}
