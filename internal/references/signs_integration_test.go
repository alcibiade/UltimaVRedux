package references

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestSignReferences_BritainAliasAndCastleBasement(t *testing.T) {
	signsPath := filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT")

	refs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("load SIGNS.DAT: %v", err)
	}

	leftBritainSign := refs.GetSignAt(Britain, int(MainFloor), 14, 27)
	rightBritainSign := refs.GetSignAt(Britain, int(MainFloor), 16, 27)
	if leftBritainSign == nil || rightBritainSign == nil {
		t.Fatal("Britain entrance sign pair was not loaded")
	}
	if !bytes.Equal(rightBritainSign.RawText, leftBritainSign.RawText) {
		t.Error("Britain right entrance sign did not resolve to the left sign's text")
	}

	prisonSign := refs.GetSignAt(LordBritishsCastle, int(Basement), 6, 13)
	armourySign := refs.GetSignAt(LordBritishsCastle, int(Basement), 24, 13)
	if prisonSign == nil || !bytes.Contains(prisonSign.RawText, []byte("ROYAL")) {
		t.Error("Lord British's Castle basement prison sign was not loaded")
	}
	if armourySign == nil || !bytes.Contains(armourySign.RawText, []byte("ARMOURY")) {
		t.Error("Lord British's Castle basement armoury sign was not loaded")
	}
}

func TestSignReferences_PreserveAuthoredSignAndEpitaphFonts(t *testing.T) {
	signsPath := filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT")

	refs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("load SIGNS.DAT: %v", err)
	}

	tests := []struct {
		name        string
		location    Location
		x           int
		y           int
		wantRunic   bool
		wantIBMText bool
	}{
		{name: "Yew forgiveness notice", location: Yew, x: 14, y: 5, wantRunic: true},
		{name: "Moonglow honesty law", location: Moonglow, x: 14, y: 20, wantIBMText: true},
		{name: "Yew epitaph", location: Yew, x: 12, y: 4, wantRunic: true},
		{name: "New Magincia epitaph", location: NewMagincia, x: 20, y: 1, wantIBMText: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sign := refs.GetSignAt(test.location, int(MainFloor), test.x, test.y)
			if sign == nil {
				t.Fatalf("sign at %s (%d,%d) was not loaded", test.location, test.x, test.y)
			}

			runicLetters := 0
			ibmLetters := 0
			for _, value := range sign.RawText {
				glyph := value & SignCharIndexMask
				if glyph < 'A' || glyph > 'Z' {
					continue
				}
				if IsRunicChar(value) {
					runicLetters++
				} else {
					ibmLetters++
				}
			}

			if test.wantRunic && runicLetters == 0 {
				t.Error("expected authored Runic lettering")
			}
			if test.wantIBMText && ibmLetters == 0 {
				t.Error("expected authored IBM lettering")
			}
		})
	}
}

// TestSignReferences_MultiLineSign verifies that signs with 0x8A newlines
// are loaded correctly and contain multiple logical lines.
func TestSignReferences_MultiLineSign(t *testing.T) {
	// Load from actual SIGNS.DAT
	signsPath := filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT")

	refs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("Failed to load SIGNS.DAT: %v", err)
	}

	// Find a sign that should have newlines
	var multiLineSign *SignRecord
	var signLocation Location

	for loc, signs := range refs.SignsByLocation {
		for _, sign := range signs {
			// Look for 0x8A in the raw text (IBM newline)
			for _, b := range sign.RawText {
				if b == 0x8A {
					multiLineSign = sign
					signLocation = loc
					break
				}
			}
			if multiLineSign != nil {
				break
			}
		}
		if multiLineSign != nil {
			break
		}
	}

	if multiLineSign == nil {
		t.Fatal("Could not find any sign with 0x8A newline character in SIGNS.DAT")
	}

	// Count newlines
	newlineCount := 0
	for _, b := range multiLineSign.RawText {
		if b == 0x8A {
			newlineCount++
		}
	}

	expectedLines := newlineCount + 1

	t.Logf("Found multi-line sign at location %d, position (%d, %d)", signLocation, multiLineSign.X, multiLineSign.Y)
	t.Logf("Raw text length: %d bytes", len(multiLineSign.RawText))
	t.Logf("Contains %d newlines (0x8A), expected %d lines", newlineCount, expectedLines)
	t.Logf("Raw bytes (first 50): %v", multiLineSign.RawText[:min(50, len(multiLineSign.RawText))])

	if newlineCount == 0 {
		t.Error("Sign should have at least one 0x8A newline")
	}
}

// TestSignReferences_SpecificOverworldSign tests a specific sign on the overworld.
func TestSignReferences_SpecificOverworldSign(t *testing.T) {
	signsPath := filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT")

	refs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("Failed to load SIGNS.DAT: %v", err)
	}

	// Log all locations that have signs
	t.Log("Locations with signs:")
	for loc, signs := range refs.SignsByLocation {
		t.Logf("  Location %d (%s): %d signs", loc, loc.String(), len(signs))
		for i, sign := range signs {
			newlineCount := 0
			for _, b := range sign.RawText {
				if b == 0x8A {
					newlineCount++
				}
			}
			t.Logf("    Sign %d: floor=%d, pos=(%d,%d), %d bytes, %d newlines",
				i, sign.Floor, sign.X, sign.Y, len(sign.RawText), newlineCount)
		}
	}
}

// TestSignReferences_DumpSignContent dumps the actual content of signs for debugging.
func TestSignReferences_DumpSignContent(t *testing.T) {
	signsPath := filepath.Join(originaldata.Directory(t, "SIGNS.DAT"), "SIGNS.DAT")

	refs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("Failed to load SIGNS.DAT: %v", err)
	}

	// Find first few multi-line signs and dump their content
	count := 0
	maxDump := 3

	for loc, signs := range refs.SignsByLocation {
		for _, sign := range signs {
			hasNewline := false
			for _, b := range sign.RawText {
				if b == 0x8A {
					hasNewline = true
					break
				}
			}

			if hasNewline && count < maxDump {
				t.Logf("\n=== Sign at Location %d (%s), Floor %d, Pos (%d, %d) ===",
					loc, loc.String(), sign.Floor, sign.X, sign.Y)
				t.Logf("Raw bytes (%d total):", len(sign.RawText))

				// Print in rows of 16 bytes
				for i := 0; i < len(sign.RawText); i += 16 {
					end := min(i+16, len(sign.RawText))
					t.Logf("  %04x: % x", i, sign.RawText[i:end])
				}

				// Also show as text with newlines marked
				t.Log("Decoded (| = newline):")
				decoded := ""
				for _, b := range sign.RawText {
					if b == 0x8A {
						decoded += "|"
					} else if b >= 0x20 && b < 0x7F {
						decoded += string(rune(b))
					} else if b >= 0xA0 && b < 0xFF {
						// High-bit character - show the ASCII equivalent
						decoded += string(rune(b & 0x7F))
					} else {
						decoded += "."
					}
				}
				t.Logf("  %s", decoded)

				count++
			}
		}
	}
}

// TestSignReferences_BorderMacroLengths verifies all border macros are exactly 16 characters.
// This is critical because the original Ultima V message window was 16 characters wide,
// and the sign renderer auto-wraps at 16 characters.
func TestSignReferences_BorderMacroLengths(t *testing.T) {
	expectedMacros := map[byte]string{
		0x29: "g              g", // side borders
		0x2a: "jlllllllnllllllk", // top border with center notch
		0x2b: "8lllllllmllllll9", // bottom border with center notch
		0x2c: "jllllllllllllllk", // top border plain
		0x2d: "8llllllllllllll9", // bottom border plain
		0x2e: "hllllk    jlllli", // cross arm (left + gap + right)
		0x2f: "jlllli    hllllk", // cross arm mirrored
		0x30: "     g    g     ", // cross vertical bars
		0x31: "     hlllli     ", // cross bottom center
	}

	for byteVal := byte(0x29); byteVal <= 0x31; byteVal++ {
		if !IsBorderMacro(byteVal) {
			t.Errorf("IsBorderMacro(0x%02x) should return true", byteVal)
			continue
		}

		macro := GetBorderMacro(byteVal)
		expectedMacro := expectedMacros[byteVal]

		if len(macro) != 16 {
			t.Errorf("Border macro 0x%02x has length %d, expected 16. Content: %q",
				byteVal, len(macro), macro)
		}

		if macro != expectedMacro {
			t.Errorf("Border macro 0x%02x content mismatch:\n  got:      %q\n  expected: %q",
				byteVal, macro, expectedMacro)
		}
	}

	// Also verify non-macro bytes return empty
	if IsBorderMacro(0x28) {
		t.Error("IsBorderMacro(0x28) should return false")
	}
	if IsBorderMacro(0x32) {
		t.Error("IsBorderMacro(0x32) should return false")
	}
}
