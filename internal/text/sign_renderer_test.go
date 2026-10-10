package text

import (
	"bytes"
	"image"
	"testing"
)

// TestSignRenderer_ExpandAndSplit_MultiLine verifies that the sign renderer
// correctly splits sign text into multiple lines using both runic (0x0A)
// and normal/IBM (0x8A) newline encodings.
func TestSignRenderer_ExpandAndSplit_MultiLine(t *testing.T) {
	sr := &SignRenderer{}

	// NOTE: Avoid using ASCII chars ')', '*', '+', ',', '-', '.', '/', '0', '1'
	// (decimal 41-49, hex 0x29-0x31) in test data - these are border macros
	// that expand to 16-character strings.

	tests := []struct {
		name          string
		rawText       []byte
		expectedLines int
		description   string
	}{
		{
			name:          "SingleLine_NoNewlines",
			rawText:       []byte("HELLO"),
			expectedLines: 1,
			description:   "Single line text without newlines",
		},
		{
			name:          "TwoLines_RunicNewline_0x0A",
			rawText:       []byte("AAAA\nBBBB"), // Avoid '1','2' which are border macros
			expectedLines: 2,
			description:   "Two lines separated by runic newline (0x0A)",
		},
		{
			name:          "TwoLines_IBMNewline_0x8A",
			rawText:       []byte{'A', 'A', 'A', 'A', 0x8A, 'B', 'B', 'B', 'B'},
			expectedLines: 2,
			description:   "Two lines separated by IBM/normal newline (0x8A)",
		},
		{
			name:          "ThreeLines_MixedNewlines",
			rawText:       []byte{'A', 'A', '\n', 'B', 'B', 0x8A, 'C', 'C'},
			expectedLines: 3,
			description:   "Three lines with mixed 0x0A and 0x8A newlines",
		},
		{
			name:          "MultipleIBMNewlines",
			rawText:       []byte{'A', 0x8A, 'B', 0x8A, 'C', 0x8A, 'D'},
			expectedLines: 4,
			description:   "Four lines separated by 0x8A newlines",
		},
		{
			name:          "RealSignData_WithBorders",
			rawText:       []byte{0x67, ' ', 'T', 'E', 'S', 'T', ' ', 0x67, 0x8A, 0x67, ' ', 'S', 'I', 'G', 'N', ' ', 0x67},
			expectedLines: 2,
			description:   "Simulated sign data with border chars and 0x8A newline",
		},
		{
			name:          "EmptyLinesBetween",
			rawText:       []byte{'A', 0x8A, 0x8A, 'B'},
			expectedLines: 3,
			description:   "Text with empty line between (consecutive 0x8A)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := sr.expandAndSplit(tt.rawText)

			if len(lines) != tt.expectedLines {
				t.Errorf("%s: expected %d lines, got %d lines",
					tt.description, tt.expectedLines, len(lines))
				t.Logf("Raw text bytes: %v", tt.rawText)
				for i, line := range lines {
					t.Logf("  Line %d: %d chars", i, len(line.chars))
				}
			}
		})
	}
}

// TestSignRenderer_RenderSign_MultiLine verifies that RenderSign produces
// an image with correct height for multi-line signs.
func TestSignRenderer_RenderSign_MultiLine(t *testing.T) {
	sr := NewSignRenderer(1.0)
	charHeight := int(ScaledCharHeight(1.0))

	// NOTE: Avoid using ASCII chars ')', '*', '+', ',', '-', '.', '/', '0', '1'
	// (decimal 41-49, hex 0x29-0x31) in test data - these are border macros.
	tests := []struct {
		name           string
		rawText        []byte
		expectedLines  int
		expectedHeight int
	}{
		{
			name:           "SingleLine",
			rawText:        []byte("TEST"),
			expectedLines:  1,
			expectedHeight: charHeight * 1,
		},
		{
			name:           "TwoLines_0x8A",
			rawText:        []byte{'A', 'A', 'A', 'A', 0x8A, 'B', 'B', 'B', 'B'},
			expectedLines:  2,
			expectedHeight: charHeight * 2,
		},
		{
			name:           "ThreeLines_0x8A",
			rawText:        []byte{'A', 0x8A, 'B', 0x8A, 'C'},
			expectedLines:  3,
			expectedHeight: charHeight * 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sr.RenderSign(tt.rawText)

			if result.Height != tt.expectedHeight {
				t.Errorf("Expected height %d (for %d lines), got %d",
					tt.expectedHeight, tt.expectedLines, result.Height)
			}

			if result.Image == nil {
				t.Error("Expected non-nil image")
			}
		})
	}
}

// TestSignRenderer_0x8A_IsNewline is a focused test ensuring 0x8A
// is treated as a newline character (the IBM-encoded newline).
func TestSignRenderer_0x8A_IsNewline(t *testing.T) {
	sr := &SignRenderer{}

	// Text: "TOP" + 0x8A + "BOTTOM"
	rawText := []byte{'T', 'O', 'P', 0x8A, 'B', 'O', 'T', 'T', 'O', 'M'}

	lines := sr.expandAndSplit(rawText)

	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	// Verify first line has 3 chars (T, O, P)
	if len(lines[0].chars) != 3 {
		t.Errorf("First line: expected 3 chars, got %d", len(lines[0].chars))
	}

	// Verify second line has 6 chars (B, O, T, T, O, M)
	if len(lines[1].chars) != 6 {
		t.Errorf("Second line: expected 6 chars, got %d", len(lines[1].chars))
	}

	// Log for debugging
	t.Logf("Line 0: %d chars", len(lines[0].chars))
	t.Logf("Line 1: %d chars", len(lines[1].chars))
}

// TestSignRenderer_RealUltimaSignData tests with bytes similar to actual SIGNS.DAT content.
// Based on hex dump: 8a67 (newline + 'g' border) appears frequently
func TestSignRenderer_RealUltimaSignData(t *testing.T) {
	sr := &SignRenderer{}

	// Simulated sign data from SIGNS.DAT hex dump:
	// "g SERPENTS :l9" + 0x8A + "g PASS 8lllll;"
	// This simulates a two-line sign with borders
	rawText := []byte{
		0x67, ' ', 'S', 'E', 'R', 'P', 'E', 'N', 'T', 'S', ' ',
		0x8A, // IBM newline
		0x67, ' ', 'P', 'A', 'S', 'S', ' ',
	}

	lines := sr.expandAndSplit(rawText)

	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines from Ultima sign data, got %d", len(lines))
	}

	t.Logf("Successfully parsed %d lines from simulated Ultima sign data", len(lines))
	for i, line := range lines {
		t.Logf("  Line %d: %d characters", i, len(line.chars))
	}
}

// TestSignRenderer_AutoWrap16Chars tests signs that rely on 16-character auto-wrapping
// instead of explicit newlines. This is the second sign format used in Ultima V.
// The original game's message window was exactly 16 characters wide, so signs
// without newline bytes relied on the window boundary for line breaks.
func TestSignRenderer_AutoWrap16Chars(t *testing.T) {
	sr := &SignRenderer{}

	tests := []struct {
		name                 string
		rawText              []byte
		expectedLines        int
		expectedCharsPerLine []int
		description          string
	}{
		{
			name:                 "Exactly16Chars_OneLine",
			rawText:              []byte("ABCDEFGHIJKLMNOP"), // 16 chars exactly
			expectedLines:        1,
			expectedCharsPerLine: []int{16},
			description:          "Exactly 16 chars should be one line",
		},
		{
			name:                 "17Chars_TwoLines",
			rawText:              []byte("ABCDEFGHIJKLMNOPQ"), // 17 chars
			expectedLines:        2,
			expectedCharsPerLine: []int{16, 1},
			description:          "17 chars should wrap to 2 lines (16 + 1)",
		},
		{
			name:                 "32Chars_TwoLines",
			rawText:              []byte("ABCDEFGHIJKLMNOPABCDEFGHIJKLMNOP"), // 32 chars
			expectedLines:        2,
			expectedCharsPerLine: []int{16, 16},
			description:          "32 chars should be exactly 2 lines of 16",
		},
		{
			name:                 "64Chars_FourLines",
			rawText:              make([]byte, 64), // 64 chars - simulates sign at (76, 113)
			expectedLines:        4,
			expectedCharsPerLine: []int{16, 16, 16, 16},
			description:          "64 chars with no newlines should produce 4 lines of 16",
		},
		{
			name:                 "48Chars_ThreeLines",
			rawText:              []byte("AAAAAAAAAAAAAAAA" + "BBBBBBBBBBBBBBBB" + "CCCCCCCCCCCCCCCC"), // 48 chars
			expectedLines:        3,
			expectedCharsPerLine: []int{16, 16, 16},
			description:          "48 chars should be exactly 3 lines of 16",
		},
		{
			name:                 "20Chars_TwoLines",
			rawText:              []byte("AAAABBBBCCCCDDDDEEEE"), // 20 chars
			expectedLines:        2,
			expectedCharsPerLine: []int{16, 4},
			description:          "20 chars should wrap to 2 lines (16 + 4)",
		},
	}

	// Initialize the 64-char test case with 'X' characters (avoid border macro range)
	for i := range tests[3].rawText {
		tests[3].rawText[i] = 'X'
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := sr.expandAndSplit(tt.rawText)

			if len(lines) != tt.expectedLines {
				t.Errorf("%s: expected %d lines, got %d lines",
					tt.description, tt.expectedLines, len(lines))
				for i, line := range lines {
					t.Logf("  Line %d: %d chars", i, len(line.chars))
				}
				return
			}

			// Verify character counts per line
			for i, expectedChars := range tt.expectedCharsPerLine {
				if i >= len(lines) {
					t.Errorf("Line %d missing, expected %d chars", i, expectedChars)
					continue
				}
				if len(lines[i].chars) != expectedChars {
					t.Errorf("Line %d: expected %d chars, got %d",
						i, expectedChars, len(lines[i].chars))
				}
			}
		})
	}
}

// TestSignRenderer_BorderMacroExpansion tests that border macros (0x29-0x31)
// expand to 16-character strings and count toward auto-wrap.
func TestSignRenderer_BorderMacroExpansion(t *testing.T) {
	sr := &SignRenderer{}

	// Single border macro (0x2c = top border plain) should expand to 16 chars
	rawText := []byte{0x2c}
	lines := sr.expandAndSplit(rawText)

	if len(lines) != 1 {
		t.Fatalf("Expected 1 line from single border macro, got %d", len(lines))
	}

	if len(lines[0].chars) != 16 {
		t.Errorf("Border macro 0x2c should expand to 16 chars, got %d", len(lines[0].chars))
	}

	// Two consecutive border macros should wrap to 2 lines
	rawText = []byte{0x2c, 0x2d}
	lines = sr.expandAndSplit(rawText)

	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines from two border macros, got %d", len(lines))
	}

	// Both lines should have 16 chars
	for i, line := range lines {
		if len(line.chars) != 16 {
			t.Errorf("Line %d: expected 16 chars from border macro, got %d", i, len(line.chars))
		}
	}
}

func TestSignRendererClassifiesWarmColourRoles(t *testing.T) {
	lines := (&SignRenderer{}).expandAndSplit([]byte{'a', 'A', 0xC1, 0x26})
	if len(lines) != 1 || len(lines[0].chars) != 4 {
		t.Fatalf("warm-role fixture produced unexpected lines: %+v", lines)
	}
	characters := lines[0].chars
	if !characters[0].isRunic || !characters[0].isStructural {
		t.Error("literal frame glyph must be structural runic art")
	}
	if !characters[1].isRunic || characters[1].isStructural {
		t.Error("ordinary runic lettering must not be structural")
	}
	if characters[2].isRunic || characters[2].isStructural {
		t.Error("IBM lettering must remain non-structural IBM text")
	}
	if !characters[3].isRunic || !characters[3].isStructural {
		t.Error("original horizontal-line byte must be structural runic art")
	}
}

func TestBitmapFontsUseDistinctRunicAndIBMGlyphShapes(t *testing.T) {
	runicFont, _, err := image.Decode(bytes.NewReader(runicFontPNG))
	if err != nil {
		t.Fatalf("decode Runic font: %v", err)
	}
	ibmFont, _, err := image.Decode(bytes.NewReader(ibmFontPNG))
	if err != nil {
		t.Fatalf("decode IBM font: %v", err)
	}

	different := false
	for glyphIndex := 'A'; glyphIndex <= 'Z' && !different; glyphIndex++ {
		glyphX := (int(glyphIndex) % BitmapCharsPerRow) * BitmapCharWidth
		glyphY := (int(glyphIndex) / BitmapCharsPerRow) * BitmapCharHeight
		for y := 0; y < BitmapCharHeight && !different; y++ {
			for x := 0; x < BitmapCharWidth; x++ {
				if runicFont.At(glyphX+x, glyphY+y) != ibmFont.At(glyphX+x, glyphY+y) {
					different = true
					break
				}
			}
		}
	}
	if !different {
		t.Error("Runic and IBM uppercase alphabets use the same pixel shapes")
	}
}

func TestSignRendererClassifiesLiteralBottomFrameAsStructural(t *testing.T) {
	lines := (&SignRenderer{}).expandAndSplit([]byte(":lllllnlllll;"))
	if len(lines) != 1 {
		t.Fatalf("literal bottom-frame fixture produced %d lines, expected 1", len(lines))
	}
	for index, character := range lines[0].chars {
		if !character.isRunic || !character.isStructural {
			t.Errorf("bottom-frame glyph %q at index %d must be structural runic art", character.rawByte, index)
		}
	}
}

func TestSignRenderer_ExactWidthNewlineDoesNotAddBlankLine(t *testing.T) {
	sr := &SignRenderer{}

	lines := sr.expandAndSplit([]byte("ABCDEFGHIJKLMNOP\nSECOND"))

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if len(lines[0].chars) != 16 || len(lines[1].chars) != 6 {
		t.Errorf("line lengths = [%d, %d], want [16, 6]", len(lines[0].chars), len(lines[1].chars))
	}
}

func TestSignRenderer_ConsecutiveNewlinesPreserveBlankLine(t *testing.T) {
	sr := &SignRenderer{}

	lines := sr.expandAndSplit([]byte("FIRST\n\nSECOND"))

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if len(lines[1].chars) != 0 {
		t.Errorf("middle line contains %d characters, want a blank line", len(lines[1].chars))
	}
}

// TestSignRenderer_MixedNewlineAndAutoWrap tests signs that have some explicit
// newlines but also rely on auto-wrap for other lines.
func TestSignRenderer_MixedNewlineAndAutoWrap(t *testing.T) {
	sr := &SignRenderer{}

	// Create test: 10 chars + newline + 20 chars (should wrap at 16)
	// Expected: Line 1: 10 chars, Line 2: 16 chars, Line 3: 4 chars
	rawText := make([]byte, 0, 31)
	// First 10 chars
	for i := 0; i < 10; i++ {
		rawText = append(rawText, 'A')
	}
	// Explicit newline
	rawText = append(rawText, 0x8A)
	// Next 20 chars (will wrap at 16)
	for i := 0; i < 20; i++ {
		rawText = append(rawText, 'B')
	}

	lines := sr.expandAndSplit(rawText)

	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
		for i, line := range lines {
			t.Logf("  Line %d: %d chars", i, len(line.chars))
		}
	}

	expectedChars := []int{10, 16, 4}
	for i, expected := range expectedChars {
		if len(lines[i].chars) != expected {
			t.Errorf("Line %d: expected %d chars, got %d", i, expected, len(lines[i].chars))
		}
	}
}

// TestSignRenderer_PauseSkipped verifies that Redux omits the original
// 0x0D key wait while preserving its no-cursor-movement layout behavior.
func TestSignRenderer_PauseSkipped(t *testing.T) {
	sr := &SignRenderer{}

	// Input: "LINE" + 0x0D + "TWO"
	// Expected: 1 line with 7 chars (0x0D is skipped, text is joined)
	rawText := []byte{'L', 'I', 'N', 'E', 0x0D, 'T', 'W', 'O'}

	lines := sr.expandAndSplit(rawText)

	if len(lines) != 1 {
		t.Fatalf("Expected 1 line (0x0D skipped), got %d", len(lines))
	}

	// Verify line has 7 chars (L, I, N, E, T, W, O - the 0x0D is skipped)
	if len(lines[0].chars) != 7 {
		t.Errorf("Expected 7 chars, got %d", len(lines[0].chars))
	}
}

// TestSignRenderer_MultiplePausesPreserveOneContinuousLayout verifies that
// multiple 0x0D waits do not move the cursor.
func TestSignRenderer_MultiplePausesPreserveOneContinuousLayout(t *testing.T) {
	sr := NewSignRenderer(1.0)

	// Input: "A" + 0x0D + "BB" + 0x0D + "CCC"
	// Expected: 1 line with 6 chars (all 0x0D bytes skipped)
	rawText := []byte{'A', 0x0D, 'B', 'B', 0x0D, 'C', 'C', 'C'}

	result := sr.RenderSign(rawText)

	// Verify height corresponds to 1 line (all text joined, 0x0D skipped)
	charHeight := int(ScaledCharHeight(1.0))
	expectedHeight := charHeight * 1
	if result.Height != expectedHeight {
		t.Errorf("Expected height %d (1 line), got %d", expectedHeight, result.Height)
	}
}
