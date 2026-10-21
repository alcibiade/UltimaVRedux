package sprites

import (
	"image"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
)

func TestParseCharacterCreationScriptMapsEveryVerifiedBlock(t *testing.T) {
	data := characterCreationQuestionFixture()

	script, err := parseCharacterCreationScript(data)

	require.NoError(t, err)
	assert.Equal(t, blockText(0), script.Introduction)
	assert.Equal(t, blockText(1), script.Conclusion)
	blockIndex := 2
	for lower := 0; lower < characterCreationVirtues-1; lower++ {
		for higher := lower + 1; higher < characterCreationVirtues; higher++ {
			assert.Equal(t, blockText(blockIndex), script.Dilemma(lower, higher))
			blockIndex++
		}
	}
	assert.Equal(t, len(characterCreationTextOffsets), blockIndex)
}

func TestParseCharacterCreationScriptRejectsMalformedBoundaries(t *testing.T) {
	t.Run("truncated", func(t *testing.T) {
		_, err := parseCharacterCreationScript(make([]byte, characterCreationTextSize-1))
		require.ErrorContains(t, err, "truncated before byte 7745")
	})

	t.Run("early terminator", func(t *testing.T) {
		data := characterCreationQuestionFixture()
		data[1] = 0
		_, err := parseCharacterCreationScript(data)
		require.ErrorContains(t, err, "block 0 terminator is not at expected offset 801")
	})

	t.Run("missing terminator", func(t *testing.T) {
		data := characterCreationQuestionFixture()
		data[characterCreationTextSize-1] = 'X'
		_, err := parseCharacterCreationScript(data)
		require.ErrorContains(t, err, "block 29 terminator is not at expected offset 7744")
	})
}

func TestNewCharacterCreationAssetsRequiresCompleteShapesAndFont(t *testing.T) {
	shapes := characterCreationShapeFixture()
	font := characterCreationFontFixture()
	data := characterCreationQuestionFixture()

	assets, err := newCharacterCreationAssets(shapes, font, data)

	require.NoError(t, err)
	require.NotNil(t, assets.Oracle)
	require.NotNil(t, assets.Incense)
	require.NotNil(t, assets.Shrine)
	for index, symbol := range assets.VirtueSymbols {
		require.NotNilf(t, symbol, "virtue symbol %d", index)
	}
	require.Len(t, assets.Glyphs, 91)
	require.Len(t, assets.GlyphWidths, 91)

	shapes.Entries[4].Missing = true
	_, err = newCharacterCreationAssets(shapes, font, data)
	require.ErrorContains(t, err, "CREATE.16 entry 4 is unavailable")

	shapes = characterCreationShapeFixture()
	shapes.Entries[4].Width++
	_, err = newCharacterCreationAssets(shapes, font, data)
	require.ErrorContains(t, err, "CREATE.16 entry 4 has dimensions 35x69, want 34x69")

	shapes = characterCreationShapeFixture()
	font.Entries = font.Entries[:90]
	_, err = newCharacterCreationAssets(shapes, font, data)
	require.ErrorContains(t, err, "PROPORT.PCS has an unexpected format or entry count")

	font = characterCreationFontFixture()
	font.Entries[1].Width++
	_, err = newCharacterCreationAssets(shapes, font, data)
	require.ErrorContains(t, err, "PROPORT.PCS entry 1 has invalid glyph geometry")
}

func TestCharacterCreationScriptRejectsInvalidDilemma(t *testing.T) {
	script, err := parseCharacterCreationScript(characterCreationQuestionFixture())
	require.NoError(t, err)
	require.Panics(t, func() {
		script.Dilemma(1, 0)
	})
}

func characterCreationQuestionFixture() []byte {
	data := make([]byte, characterCreationTextSize)
	for index, offset := range characterCreationTextOffsets {
		limit := characterCreationTextSize
		if index+1 < len(characterCreationTextOffsets) {
			limit = characterCreationTextOffsets[index+1]
		}
		copy(data[offset:limit-1], blockText(index))
	}

	return data
}

func blockText(index int) string {
	limit := characterCreationTextSize
	if index+1 < len(characterCreationTextOffsets) {
		limit = characterCreationTextOffsets[index+1]
	}
	length := limit - characterCreationTextOffsets[index] - 1

	return strings.Repeat(string(rune('A'+index%26)), length)
}

func characterCreationShapeFixture() legacy.Archive {
	entries := make([]legacy.Entry, characterCreationShapeCount)
	for index := range entries {
		size := characterCreationShapeSizes[index]
		entries[index] = legacy.Entry{
			Index: index, Width: size.X, Height: size.Y, StoredWidth: (size.X + 7) &^ 7,
			PixelType: legacy.PixelEGA4,
			Image:     image.NewPaletted(image.Rectangle{Max: size}, legacy.EGAPalette),
		}
	}

	return legacy.Archive{
		Spec:    legacy.FileSpec{Name: "CREATE.16", Format: legacy.FormatShape4},
		Entries: entries,
	}
}

func characterCreationFontFixture() legacy.Archive {
	entries := make([]legacy.Entry, 91)
	for index := range entries {
		width := characterCreationGlyphWidths[index]
		var glyph *image.Paletted
		if width > 0 {
			glyph = image.NewPaletted(image.Rect(0, 0, 8, 8), legacy.EGAPalette[:2])
		}
		entries[index] = legacy.Entry{
			Index: index, Width: width, Height: 8, StoredWidth: 8, PixelType: legacy.PixelFont,
			Image: glyph,
		}
	}

	return legacy.Archive{
		Spec:    legacy.FileSpec{Name: "PROPORT.PCS", Format: legacy.FormatFontPCS},
		Entries: entries,
	}
}
