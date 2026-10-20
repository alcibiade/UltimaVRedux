package text

import (
	"strings"
	"testing"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/stretchr/testify/require"
)

func translatedRowsText(rows [][]ColorSegment) string {
	lines := make([]string, len(rows))
	for index, row := range rows {
		for _, segment := range row {
			lines[index] += segment.Text
		}
	}
	return strings.Join(lines, "\n")
}

func TestSignTranslationSharesSourceCellClassification(t *testing.T) {
	ibm := func(value string) []byte {
		result := []byte(value)
		for index := range result {
			result[index] |= 0x80
		}
		return result
	}
	tests := []struct {
		name string
		raw  []byte
		want string
	}{
		{"compounds", []byte("[\\]_@ABC"), "THEENGST ABC"},
		{"IBM lookalikes", ibm("g[\\]_@Jane!?"), "g[\\]_@Jane!?"},
		{"literal frame", []byte("abbbbbbbbbbbbbbc\ng  LIVE@BY@[E  g\ng  EIGHT@LAWS  g\ndeeeeeeeeeeeeeef\n"), "LIVE BY THE\nEIGHT LAWS"},
		{"macro frame", []byte{0x2c, 'A', '@', 'B', '\n', 0x2d}, "A B"},
		{"art separates words", []byte("AgB"), "A B"},
		{"spacing and newlines", []byte{'A', '@', '@', 'B', 0x8a, '\n', 'C'}, "A  B\n\nC"},
		{"pause does not add space", []byte{'A', 0x0d, 'B'}, "AB"},
		{"implicit wrap with explicit newline", []byte("ABCDEFGHIJKLMNOP\nQRST"), "ABCDEFGHIJKLMNOP\nQRST"},
		{"implicit wrap", []byte("ABCDEFGHIJKLMNOPQRST"), "ABCDEFGHIJKLMNOP\nQRST"},
		{"IBM macro lookalikes", ibm("&')*+,-./01"), "&')*+,-./01"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original := append([]byte(nil), test.raw...)
			var parser SignRenderer
			require.Equal(t, test.want, translatedRowsText(decodeSignTranslation(parser.expandAndSplit(test.raw))))
			require.Equal(t, original, test.raw)
		})
	}
}

func TestSignTranslationRetainsUnknownGlyphInItsSourceFont(t *testing.T) {
	var parser SignRenderer
	rows := decodeSignTranslation(parser.expandAndSplit([]byte{'A', '?', 'B', '?' | 0x80}))
	require.Equal(t, "A?B?", translatedRowsText(rows))
	require.Len(t, rows, 1)
	require.Len(t, rows[0], 4)
	require.Equal(t, FontTypeRunic, rows[0][1].FontType)
	require.Equal(t, "?", rows[0][1].Text)
	require.Equal(t, FontTypeNormal, rows[0][2].FontType)
	require.Equal(t, ucolor.EGAYellow, rows[0][2].Color)
	require.Equal(t, ucolor.EGAWhite, rows[0][3].Color)
}

func TestSignTranslationSpanClassificationPreservesSourceBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		cell      signChar
		want      string
		translate bool
	}{
		{"letter", signChar{rawByte: 'A', isRunic: true}, "A", true},
		{"TH", signChar{rawByte: '[', isRunic: true}, "TH", true},
		{"EE", signChar{rawByte: '\\', isRunic: true}, "EE", true},
		{"NG", signChar{rawByte: ']', isRunic: true}, "NG", true},
		{"ST", signChar{rawByte: '_', isRunic: true}, "ST", true},
		{"encoded space", signChar{rawByte: '@', isRunic: true}, " ", true},
		{"padding", signChar{rawByte: ' ', isRunic: true}, " ", true},
		{"unknown rune", signChar{rawByte: '?', isRunic: true}, "?", false},
		{"artwork", signChar{rawByte: 'g', isRunic: true, isStructural: true}, " ", false},
		{"macro padding", signChar{rawByte: ' ', isRunic: true, isStructural: true}, " ", false},
		{"IBM letter", signChar{rawByte: 'A' | 0x80}, "A", false},
		{"IBM compound lookalike", signChar{rawByte: '[' | 0x80}, "[", false},
		{"IBM padding", signChar{rawByte: ' ' | 0x80}, " ", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, translate := translatedRunicCell(test.cell)
			require.Equal(t, test.want, value)
			require.Equal(t, test.translate, translate)
		})
	}
}
