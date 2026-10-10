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

func TestGetSyllableWord(t *testing.T) {
	// Valid syllable
	word, ok := GetSyllableWord('V')
	if !ok || word != "Vas" {
		t.Errorf("Expected 'Vas', got %q (ok=%v)", word, ok)
	}

	// Another valid syllable
	word, ok = GetSyllableWord('I')
	if !ok || word != "In" {
		t.Errorf("Expected 'In', got %q (ok=%v)", word, ok)
	}

	// Invalid syllable (J not used in Ultima V)
	_, ok = GetSyllableWord('J')
	if ok {
		t.Error("Expected 'J' to not exist")
	}

	// Invalid syllable (O not used in Ultima V)
	_, ok = GetSyllableWord('O')
	if ok {
		t.Error("Expected 'O' to not exist")
	}
}
