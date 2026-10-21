package sprites

import (
	"fmt"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
)

func TestNewEndgameAssetsRequiresEverySourceArchiveAndTextRecord(t *testing.T) {
	archives := endgameArchiveFixture()
	endData := endgameRecordFixture(endgameArtTextOffsets[:])
	messageData := endgameRecordFixture(endgameMessageOffsets[:])

	assets, err := newEndgameAssets(archives, characterCreationFontFixture(), endData, messageData)

	require.NoError(t, err)
	assert.Len(t, assets.ArtText, len(endgameArtTextOffsets))
	assert.Len(t, assets.Messages, len(endgameMessageOffsets))
	assert.Equal(t, "record-5", assets.ArtText[5])
	assert.Equal(t, "record-10", assets.Messages[10])
	assert.Len(t, assets.Shapes["END1.16"], 3)
	assert.Len(t, assets.Shapes["END2.16"], 3)
	assert.Len(t, assets.Shapes["ENDSC.16"], 1)
	assert.Len(t, assets.Shapes["TEXT.16"], 6)

	broken := endgameArchiveFixture()
	broken["END2.16"].Entries[1].Missing = true
	_, err = newEndgameAssets(broken, characterCreationFontFixture(), endData, messageData)
	require.ErrorContains(t, err, "END2.16 entry 1 is unavailable")
}

func TestParseEndgameRecordsRejectsMissingSourceBoundary(t *testing.T) {
	data := endgameRecordFixture(endgameMessageOffsets[:])
	for index := endgameMessageOffsets[1]; index < endgameMessageOffsets[2]; index++ {
		data[index] = 'X'
	}

	_, err := parseEndgameRecords("ENDMSG.DAT", data, endgameMessageOffsets[:])

	require.ErrorContains(t, err, "record 1 is not terminated")
}

func endgameArchiveFixture() map[string]legacy.Archive {
	counts := map[string]int{"END1.16": 3, "END2.16": 3, "ENDSC.16": 1, "TEXT.16": 6}
	archives := make(map[string]legacy.Archive, len(counts))
	for name, count := range counts {
		entries := make([]legacy.Entry, count)
		for index := range entries {
			entries[index] = legacy.Entry{
				Index: index, Width: 8, Height: 8, StoredWidth: 8, PixelType: legacy.PixelEGA4,
				Image: image.NewPaletted(image.Rect(0, 0, 8, 8), legacy.EGAPalette),
			}
		}
		archives[name] = legacy.Archive{
			Spec: legacy.FileSpec{Name: name, Format: legacy.FormatShape4}, Entries: entries,
		}
	}

	return archives
}

func endgameRecordFixture(offsets []int) []byte {
	data := make([]byte, offsets[len(offsets)-1]+32)
	for index, offset := range offsets {
		copy(data[offset:], fmt.Sprintf("record-%d", index))
	}

	return data
}
