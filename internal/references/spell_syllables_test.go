package references

import "testing"

func TestGetSyllableMap(t *testing.T) {
	m := GetSyllableMap()

	// Verify expected count (24 syllables - no J or O in Ultima V)
	if len(m) != 24 {
		t.Errorf("Expected 24 syllables, got %d", len(m))
	}

	// Verify specific mappings
	tests := []struct {
		abbrev rune
		word   string
	}{
		{'V', "Vas"}, {'F', "Flam"}, {'A', "An"}, {'N', "Nox"},
		{'I', "In"}, {'L', "Lor"}, {'G', "Grav"}, {'P', "Por"},
		{'S', "Sanct"}, {'T', "Tym"}, {'M', "Mani"}, {'R', "Rel"},
	}

	for _, tt := range tests {
		if got := m[tt.abbrev]; got != tt.word {
			t.Errorf("Syllable %c: expected %q, got %q", tt.abbrev, tt.word, got)
		}
	}
}

func TestIsValidSyllable(t *testing.T) {
	// Valid syllables (all 24 used in Ultima V)
	validSyllables := "ABCDEFGHIKLMNPQRSTUVWXYZ"
	for _, r := range validSyllables {
		if !IsValidSyllable(r) {
			t.Errorf("Expected %c to be valid syllable", r)
		}
	}

	// Invalid syllables (J and O not used in Ultima V)
	invalidSyllables := "JO"
	for _, r := range invalidSyllables {
		if IsValidSyllable(r) {
			t.Errorf("Expected %c to be invalid syllable", r)
		}
	}

	// Lowercase should not match (syllables are uppercase)
	if IsValidSyllable('a') {
		t.Error("Expected lowercase 'a' to be invalid syllable")
	}
}
