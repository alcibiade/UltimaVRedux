package sprites

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestLoadCharacterCreationAssetsWithOriginalData(t *testing.T) {
	directory := originaldata.Directory(t, "CREATE.16", "PROPORT.PCS", "QUESTION.DAT")

	shapes, err := loadIntroArchive(directory, "CREATE.16")
	require.NoError(t, err)
	require.Equal(t, legacy.FormatShape4, shapes.Spec.Format)
	require.Len(t, shapes.Entries, characterCreationShapeCount)
	for index, entry := range shapes.Entries {
		require.NoError(t, validateCharacterCreationShape(entry, index, characterCreationShapeSizes[index]))
	}
	font, err := loadIntroArchive(directory, "PROPORT.PCS")
	require.NoError(t, err)
	require.Equal(t, legacy.FormatFontPCS, font.Spec.Format)
	require.Len(t, font.Entries, 91)
	for index, entry := range font.Entries {
		require.NoError(t, validateCharacterCreationGlyph(entry, index, characterCreationGlyphWidths[index]))
	}
	data, err := os.ReadFile(filepath.Join(directory, "QUESTION.DAT"))
	require.NoError(t, err)
	script, err := parseCharacterCreationScript(data)
	require.NoError(t, err)
	require.NotEmpty(t, script.Introduction)
	require.NotEmpty(t, script.Conclusion)
	for lower := 0; lower < characterCreationVirtues-1; lower++ {
		for higher := lower + 1; higher < characterCreationVirtues; higher++ {
			require.NotEmpty(t, script.Dilemma(lower, higher))
		}
	}
}
