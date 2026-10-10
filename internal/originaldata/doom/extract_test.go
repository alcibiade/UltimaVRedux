package doom

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRejectsTruncatedStructuralFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Files)
		asset  string
	}{
		{name: "dungeon dat", mutate: func(files *Files) { files.DungeonDAT = make([]byte, dungeonMinimumSize-1) }, asset: "DUNGEON.DAT"},
		{name: "dungeon cbt", mutate: func(files *Files) { files.DungeonCBT = make([]byte, dungeonCBTMinimumSize-1) }, asset: "DUNGEON.CBT"},
		{name: "misc maps", mutate: func(files *Files) { files.MiscMapsDAT = make([]byte, miscMapsMinimumSize-1) }, asset: "MISCMAPS.DAT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := syntheticFiles()
			test.mutate(&files)

			_, err := Extract(files)

			require.Error(t, err)
			assert.ErrorContains(t, err, test.asset)
			assert.ErrorContains(t, err, "need at least")
		})
	}
}

func TestExtractRejectsMissingRequiredAsset(t *testing.T) {
	t.Parallel()

	files := syntheticFiles()
	files.End1 = nil

	_, err := Extract(files)

	require.Error(t, err)
	assert.ErrorIs(t, err, errRequiredAssetMissing)
	assert.ErrorContains(t, err, "END1.16")
}

func TestExtractRejectsMalformedActiveEntityCoordinates(t *testing.T) {
	t.Parallel()

	files := syntheticFiles()
	record := syntheticRoomRecord(files, 2)
	record[6*combatRowSize+11] = combatMapSize

	_, err := Extract(files)

	require.Error(t, err)
	assert.ErrorContains(t, err, "active entity slot 0")
	assert.ErrorContains(t, err, "out-of-range position")
}

func TestExtractRejectsMalformedPartyEntryCoordinates(t *testing.T) {
	t.Parallel()

	files := syntheticFiles()
	record := syntheticRoomRecord(files, 2)
	record[combatRowSize+11] = combatMapSize

	_, err := Extract(files)

	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedPartyCoordinates)
	assert.ErrorContains(t, err, "row 1 party slot 0")
}

func TestExtractRejectsNoFinalMirrorCandidate(t *testing.T) {
	t.Parallel()

	files := syntheticFiles()
	syntheticRoomRecord(files, 2)[5*combatRowSize+11] = 0

	_, err := Extract(files)

	require.Error(t, err)
	assert.ErrorContains(t, err, "found 0 referenced IN_MIRROR")
}

func TestExtractRejectsMultipleFinalMirrorCandidates(t *testing.T) {
	t.Parallel()

	files := syntheticFiles()
	record := syntheticRoomRecord(files, 2)
	record[5*combatRowSize+12] = inMirrorFamily + 1
	record[6*combatRowSize+12] = 6
	record[7*combatRowSize+12] = 1
	record[1*combatRowSize+6] = mirrorTerrain

	_, err := Extract(files)

	require.Error(t, err)
	assert.ErrorContains(t, err, "found 2 referenced IN_MIRROR")
}

func TestExtractPreservesRepeatedRoomThresholdsAndZeroTriggers(t *testing.T) {
	t.Parallel()

	report, err := Extract(syntheticFiles())

	require.NoError(t, err)
	require.Len(t, report.Doom.Thresholds, 2)
	assert.Equal(t, byte(0xf2), report.Doom.Thresholds[0].RawValue)
	assert.Equal(t, byte(0xa2), report.Doom.Thresholds[1].RawValue)
	require.Len(t, report.Doom.Rooms, 1, "repeated room IDs share one semantic room report")
	room := report.Doom.Rooms[0]
	assert.Equal(t, 2, room.RoomID)
	require.Len(t, room.Triggers, triggerCount)
	for _, trigger := range room.Triggers {
		assert.Equal(t, Coordinate{}, trigger.Trigger)
		assert.False(t, trigger.Disabled, "zero is a preserved source coordinate, not a disabled sentinel")
		require.Len(t, trigger.Destinations, 2)
		assert.True(t, trigger.Destinations[0].InBounds)
		assert.True(t, trigger.Destinations[1].InBounds)
	}
}

func TestExtractRejectsMalformedRescueMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func([]byte)
		want   error
	}{
		{
			name: "missing mirror",
			mutate: func(data []byte) {
				data[rescueMapOffset+9*16+5] = 0
			},
			want: errRescueMirrorMismatch,
		},
		{
			name: "nonzero row padding",
			mutate: func(data []byte) {
				data[rescueMapOffset+11] = 1
			},
			want: errRescuePadding,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := syntheticFiles()
			test.mutate(files.MiscMapsDAT)

			_, err := Extract(files)

			require.Error(t, err)
			assert.ErrorIs(t, err, test.want)
		})
	}
}

func TestExtractRejectsTruncatedOrUnterminatedEndingText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Files)
		want   error
	}{
		{
			name: "truncated END.DAT",
			mutate: func(files *Files) {
				files.EndDAT = files.EndDAT[:endRecordOffsets()[len(endRecordOffsets())-1]]
			},
			want: errTextRecordOffset,
		},
		{
			name: "truncated ENDMSG.DAT",
			mutate: func(files *Files) {
				files.EndMsgDAT = files.EndMsgDAT[:endMessageRecordOffsets()[len(endMessageRecordOffsets())-1]]
			},
			want: errTextRecordOffset,
		},
		{
			name: "unterminated END.DAT record",
			mutate: func(files *Files) {
				for index := 0; index < endRecordOffsets()[1]; index++ {
					files.EndDAT[index] = 'x'
				}
			},
			want: errTextRecordTermination,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := syntheticFiles()
			test.mutate(&files)

			_, err := Extract(files)

			require.Error(t, err)
			assert.ErrorIs(t, err, test.want)
		})
	}
}

func TestExtractRejectsUnderspecifiedOrUnusableGraphicsArchives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Files)
		want   error
	}{
		{
			name: "END1 entry missing from archive",
			mutate: func(files *Files) {
				files.End1 = compressedGraphics(shapeArchive(2, -1))
			},
			want: errGraphicsEntryCount,
		},
		{
			name: "TEXT required entry absent",
			mutate: func(files *Files) {
				files.Text = compressedGraphics(shapeArchive(6, 4))
			},
			want: errGraphicsEntryUnusable,
		},
		{
			name: "PROPORT has too few glyphs",
			mutate: func(files *Files) {
				files.Proport = compressedGraphics(fontArchive(proportEntryCount-1, -1))
			},
			want: errGraphicsEntryCount,
		},
		{
			name: "PROPORT glyph is missing",
			mutate: func(files *Files) {
				files.Proport = compressedGraphics(fontArchive(proportEntryCount, 10))
			},
			want: errGraphicsEntryUnusable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := syntheticFiles()
			test.mutate(&files)

			_, err := Extract(files)

			require.Error(t, err)
			assert.ErrorIs(t, err, test.want)
		})
	}
}

func TestCanonicalJSONIsDeterministic(t *testing.T) {
	t.Parallel()

	report, err := Extract(syntheticFiles())
	require.NoError(t, err)

	first, err := CanonicalJSON(report)
	require.NoError(t, err)
	second, err := CanonicalJSON(report)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, byte('\n'), first[len(first)-1])
	assert.NotContains(t, string(first), "tiles")

	compactFirst, err := CanonicalEvidenceJSON(report)
	require.NoError(t, err)
	compactSecond, err := CanonicalEvidenceJSON(report)
	require.NoError(t, err)
	assert.Equal(t, compactFirst, compactSecond)
	assert.NotContains(t, string(compactFirst), "fixedEntities")
	assert.NotContains(t, string(compactFirst), "triggerSlots")
	assert.NotContains(t, string(compactFirst), "recordOffsets")
	assert.NotContains(t, string(compactFirst), "entry-090")
	assert.Contains(t, string(compactFirst), "roomFingerprints")
	assert.Contains(t, string(compactFirst), `"entryCount": 91`)
}

func TestIntegrationOriginalDataMatchesGolden(t *testing.T) {
	dataDir := originaldata.Directory(t)

	files := Files{}
	assets := []struct {
		name string
		dest *[]byte
	}{
		{"DUNGEON.DAT", &files.DungeonDAT}, {"DUNGEON.CBT", &files.DungeonCBT},
		{"MISCMAPS.DAT", &files.MiscMapsDAT}, {"END1.16", &files.End1},
		{"END2.16", &files.End2}, {"ENDSC.16", &files.EndScreen}, {"TEXT.16", &files.Text},
		{"PROPORT.PCS", &files.Proport}, {"END.DAT", &files.EndDAT}, {"ENDMSG.DAT", &files.EndMsgDAT},
	}
	for _, asset := range assets {
		data, err := os.ReadFile(filepath.Join(dataDir, asset.name))
		require.NoError(t, err, asset.name)
		*asset.dest = data
	}

	report, err := Extract(files)
	require.NoError(t, err)
	actual, err := CanonicalEvidenceJSON(report)
	require.NoError(t, err)
	expected, err := os.ReadFile(filepath.Join("testdata", "doom_finale_evidence.json"))
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(actual))
}

func syntheticFiles() Files {
	dungeonDAT := make([]byte, dungeonMinimumSize)
	dungeonDAT[doomBlockOffset] = 0xf2
	dungeonDAT[doomBlockOffset+1] = 0xa2
	dungeonCBT := make([]byte, dungeonCBTMinimumSize)
	recordOffset := (doomRoomBaseIndex + 2) * dungeonRoomSize
	record := dungeonCBT[recordOffset : recordOffset+dungeonRoomSize]
	record[5*combatRowSize+11] = inMirrorFamily
	record[6*combatRowSize+11] = 5
	record[7*combatRowSize+11] = 1
	record[1*combatRowSize+5] = mirrorTerrain

	miscMaps := make([]byte, miscMapsMinimumSize)
	miscMaps[rescueMapOffset+9*16+5] = mirrorTerrain

	return Files{
		DungeonDAT: dungeonDAT, DungeonCBT: dungeonCBT, MiscMapsDAT: miscMaps,
		End1:      compressedGraphics(shapeArchive(3, -1)),
		End2:      compressedGraphics(shapeArchive(3, -1)),
		EndScreen: compressedGraphics(shapeArchive(1, -1)),
		Text:      compressedGraphics(shapeArchive(6, -1)),
		Proport:   compressedGraphics(fontArchive(proportEntryCount, -1)),
		EndDAT:    textRecords(endRecordOffsets()),
		EndMsgDAT: textRecords(endMessageRecordOffsets()),
	}
}

func syntheticRoomRecord(files Files, roomID int) []byte {
	offset := (doomRoomBaseIndex + roomID) * dungeonRoomSize

	return files.DungeonCBT[offset : offset+dungeonRoomSize]
}

func shapeArchive(count, missingIndex int) []byte {
	tableEnd := 2 + count*4
	data := make([]byte, tableEnd)
	offset := tableEnd
	binary.LittleEndian.PutUint16(data[0:2], uint16(count)) //nolint:gosec // Synthetic count is small.
	for index := range count {
		if index == missingIndex {
			continue
		}
		binary.LittleEndian.PutUint16(data[2+index*4:4+index*4], uint16(offset)) //nolint:gosec // Tiny fixture.
		entry := make([]byte, 8)
		binary.LittleEndian.PutUint16(entry[0:2], 8)
		binary.LittleEndian.PutUint16(entry[2:4], 1)
		data = append(data, entry...)
		offset += len(entry)
	}

	return data
}

func fontArchive(count, missingIndex int) []byte {
	tableEnd := 2 + count*2
	data := make([]byte, tableEnd)
	offset := tableEnd
	binary.LittleEndian.PutUint16(data[0:2], uint16(count)) //nolint:gosec // Synthetic count is small.
	for index := range count {
		if index == missingIndex {
			continue
		}
		binary.LittleEndian.PutUint16(data[2+index*2:4+index*2], uint16(offset)) //nolint:gosec // Tiny fixture.
		entry := make([]byte, 12)
		width := uint16(5)
		if index == 0 {
			width = 0
		}
		binary.LittleEndian.PutUint16(entry[0:2], width)
		binary.LittleEndian.PutUint16(entry[2:4], proportGlyphHeight)
		data = append(data, entry...)
		offset += len(entry)
	}

	return data
}

func textRecords(offsets []int) []byte {
	return make([]byte, offsets[len(offsets)-1]+2)
}

func compressedGraphics(expanded []byte) []byte {
	codes := make([]int, 0, len(expanded)*2+2)
	for _, value := range expanded {
		codes = append(codes, 256, int(value))
	}
	codes = append(codes, 257)
	packed := packNineBitCodes(codes)
	result := make([]byte, 4, 4+len(packed))
	binary.LittleEndian.PutUint32(result, uint32(len(expanded))) //nolint:gosec // Tiny synthetic fixture.

	return append(result, packed...)
}

func packNineBitCodes(codes []int) []byte {
	var buffer uint64
	bits := 0
	result := make([]byte, 0, (len(codes)*9+7)/8)
	for _, code := range codes {
		buffer |= uint64(code) << bits
		bits += 9
		for bits >= 8 {
			result = append(result, byte(buffer))
			buffer >>= 8
			bits -= 8
		}
	}
	if bits > 0 {
		result = append(result, byte(buffer))
	}

	return result
}
