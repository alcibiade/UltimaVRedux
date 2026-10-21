package sprites

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestLoadEndgameAssetsWithOriginalData(t *testing.T) {
	directory := originaldata.Directory(t,
		"END1.16", "END2.16", "ENDSC.16", "TEXT.16", "PROPORT.PCS", "END.DAT", "ENDMSG.DAT",
	)

	for name, count := range map[string]int{"END1.16": 3, "END2.16": 3, "ENDSC.16": 1, "TEXT.16": 6} {
		archive, err := loadIntroArchive(directory, name)
		require.NoError(t, err)
		require.Equal(t, name, archive.Spec.Name)
		require.Equal(t, legacy.FormatShape4, archive.Spec.Format)
		require.Len(t, archive.Entries, count)
		for _, entry := range archive.Entries {
			require.False(t, entry.Missing)
			require.NotNil(t, entry.Image)
			require.Equal(t, legacy.PixelEGA4, entry.PixelType)
			require.Equal(t, entry.Width, entry.Image.Bounds().Dx())
			require.Equal(t, entry.Height, entry.Image.Bounds().Dy())
		}
	}
	font, err := loadIntroArchive(directory, "PROPORT.PCS")
	require.NoError(t, err)
	require.Equal(t, legacy.FormatFontPCS, font.Spec.Format)
	require.Len(t, font.Entries, 91)
	for index, entry := range font.Entries {
		require.NoError(t, validateCharacterCreationGlyph(entry, index, characterCreationGlyphWidths[index]))
	}
	endData, err := os.ReadFile(filepath.Join(directory, "END.DAT"))
	require.NoError(t, err)
	artText, err := parseEndgameRecords("END.DAT", endData, endgameArtTextOffsets[:])
	require.NoError(t, err)
	messageData, err := os.ReadFile(filepath.Join(directory, "ENDMSG.DAT"))
	require.NoError(t, err)
	messages, err := parseEndgameRecords("ENDMSG.DAT", messageData, endgameMessageOffsets[:])
	require.NoError(t, err)
	require.Len(t, artText, len(endgameArtTextOffsets))
	require.Len(t, messages, len(endgameMessageOffsets))
	for _, record := range append(artText, messages...) {
		require.NotEmpty(t, record)
	}
}
