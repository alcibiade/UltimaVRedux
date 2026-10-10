package references

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestParseSignsInRangeNormalizesBasementFloor(t *testing.T) {
	data := []byte{byte(LordBritishsCastle), 0xff, 6, 13, 'S', 'I', 'G', 'N', 0}

	signs := parseSignsInRange(data, 0, len(data))

	if len(signs) != 1 {
		t.Fatalf("parseSignsInRange() returned %d signs, want 1", len(signs))
	}
	if signs[0].Floor != int(Basement) {
		t.Errorf("sign floor = %d, want basement floor %d", signs[0].Floor, Basement)
	}
}

func TestParseSignsInRangeResolvesLeadingNewlineAliases(t *testing.T) {
	data := []byte{
		byte(Britain), 0, 16, 27, '\n', 0,
		byte(Britain), 0, 15, 27, '\n', 0,
		byte(Britain), 0, 14, 27, 'L', 'A', 'W', 0,
	}

	signs := parseSignsInRange(data, 0, len(data))

	if len(signs) != 3 {
		t.Fatalf("parseSignsInRange() returned %d signs, want 3", len(signs))
	}
	for index, sign := range signs {
		if !bytes.Equal(sign.RawText, []byte("LAW")) {
			t.Errorf("sign %d text = %q, want %q", index, sign.RawText, "LAW")
		}
	}

	signs[2].RawText[0] = 'X'
	if string(signs[0].RawText) != "LAW" || string(signs[1].RawText) != "LAW" {
		t.Error("resolved aliases must not share mutable text storage with their target")
	}
}

func TestParseSignsInRangePreservesLeadingNewlineWithContent(t *testing.T) {
	data := []byte{
		byte(Britain), 0, 16, 27, '\n', 'T', 'E', 'X', 'T', 0,
		byte(Britain), 0, 14, 27, 'O', 'T', 'H', 'E', 'R', 0,
	}

	signs := parseSignsInRange(data, 0, len(data))

	if !bytes.Equal(signs[0].RawText, []byte("\nTEXT")) {
		t.Errorf("sign text = %q, want a normal leading newline", signs[0].RawText)
	}
}

func TestNewSignReferences_Integration(t *testing.T) {
	// This test requires actual game data files
	signsPath := filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT")

	refs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("Failed to load SIGNS.DAT: %v", err)
	}

	// Verify we loaded some signs
	totalSigns := 0
	for _, signs := range refs.SignsByLocation {
		totalSigns += len(signs)
	}

	if totalSigns == 0 {
		t.Error("Expected to load at least one sign, got none")
	}

	t.Logf("Loaded %d total signs across %d locations", totalSigns, len(refs.SignsByLocation))

	// Log signs by location for debugging
	for loc, signs := range refs.SignsByLocation {
		t.Logf("Location %d (%s): %d signs", loc, loc.String(), len(signs))
		for _, sign := range signs {
			t.Logf("  Sign at (%d, %d) floor %d: %d bytes of text", sign.X, sign.Y, sign.Floor, len(sign.RawText))
		}
	}
}

func TestSignReferences_GetSignAt(t *testing.T) {
	signsPath := filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT")

	refs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("Failed to load SIGNS.DAT: %v", err)
	}

	// Test that we can retrieve a sign that exists
	// First find any sign to test with
	var testSign *SignRecord
	var testLoc Location
	for loc, signs := range refs.SignsByLocation {
		if len(signs) > 0 {
			testSign = signs[0]
			testLoc = loc
			break
		}
	}

	if testSign == nil {
		t.Fatal("No signs found in selected SIGNS.DAT")
	}

	// Test GetSignAt returns the correct sign
	found := refs.GetSignAt(testLoc, testSign.Floor, testSign.X, testSign.Y)
	if found == nil {
		t.Errorf("GetSignAt failed to find sign at location %d (%d, %d)", testLoc, testSign.X, testSign.Y)
	}

	// Test HasSignAt
	if !refs.HasSignAt(testLoc, testSign.Floor, testSign.X, testSign.Y) {
		t.Error("HasSignAt returned false for existing sign")
	}

	// Test non-existent sign
	if refs.HasSignAt(testLoc, 99, 99, 99) {
		t.Error("HasSignAt returned true for non-existent sign")
	}
}

func TestBorderMacros(t *testing.T) {
	// Test valid border macros
	testCases := []struct {
		input    byte
		expected string
	}{
		{0x29, "g              g"},
		{0x2a, "jlllllllnllllllk"},
		{0x2b, "8lllllllmllllll9"},
		{0x2c, "jllllllllllllllk"},
		{0x2d, "8llllllllllllll9"},
		{0x2e, "hllllk    jlllli"}, // cross arm (left + gap + right)
		{0x2f, "jlllli    hllllk"}, // cross arm mirrored
		{0x30, "     g    g     "}, // cross vertical bars
		{0x31, "     hlllli     "}, // cross bottom center
	}

	for _, tc := range testCases {
		result := GetBorderMacro(tc.input)
		if result != tc.expected {
			t.Errorf("GetBorderMacro(0x%02x) = %q, expected %q", tc.input, result, tc.expected)
		}
		if !IsBorderMacro(tc.input) {
			t.Errorf("IsBorderMacro(0x%02x) = false, expected true", tc.input)
		}
	}

	// Test invalid border macro
	if GetBorderMacro(0x28) != "" {
		t.Error("GetBorderMacro(0x28) should return empty string")
	}
	if IsBorderMacro(0x28) {
		t.Error("IsBorderMacro(0x28) should return false")
	}
}

func TestSignBorderGlyphClassification(t *testing.T) {
	for _, glyph := range []byte{'8', '9', ':', ';', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n'} {
		if !IsSignBorderGlyph(glyph) {
			t.Errorf("IsSignBorderGlyph(%q) = false, expected true", glyph)
		}
	}
	for _, glyph := range []byte{'7', '<', 'A', 'N', 'o', 0xC1} {
		if IsSignBorderGlyph(glyph) {
			t.Errorf("IsSignBorderGlyph(%#x) = true, expected false", glyph)
		}
	}
}

func TestCharacterClassification(t *testing.T) {
	// Test runic characters (high bit clear)
	for b := byte(0); b <= 0x7F; b++ {
		if !IsRunicChar(b) {
			t.Errorf("IsRunicChar(0x%02x) = false, expected true", b)
		}
		if IsNormalChar(b) {
			t.Errorf("IsNormalChar(0x%02x) = true, expected false", b)
		}
		if GetCharIndex(b) != int(b) {
			t.Errorf("GetCharIndex(0x%02x) = %d, expected %d", b, GetCharIndex(b), int(b))
		}
	}

	// Test normal characters (high bit set)
	for b := byte(0x80); b <= 0xFF; b++ {
		if IsRunicChar(b) {
			t.Errorf("IsRunicChar(0x%02x) = true, expected false", b)
		}
		if !IsNormalChar(b) {
			t.Errorf("IsNormalChar(0x%02x) = false, expected true", b)
		}
		expected := int(b & 0x7F)
		if GetCharIndex(b) != expected {
			t.Errorf("GetCharIndex(0x%02x) = %d, expected %d", b, GetCharIndex(b), expected)
		}
		// Overflow check - stop at 255
		if b == 0xFF {
			break
		}
	}
}

func TestSignPause(t *testing.T) {
	if !IsSignPause(0x0D) {
		t.Error("IsSignPause(0x0D) = false, expected true")
	}
	if IsSignPause(0x0A) {
		t.Error("IsSignPause(0x0A) = true, expected false")
	}
}
