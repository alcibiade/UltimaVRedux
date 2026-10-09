package legacy

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeShapeArchivePreservesDeclaredWidthAndPadding(t *testing.T) {
	data := make([]byte, 18)
	binary.LittleEndian.PutUint16(data[0:2], 1)
	binary.LittleEndian.PutUint16(data[2:4], 6)
	binary.LittleEndian.PutUint16(data[6:8], 3)
	binary.LittleEndian.PutUint16(data[8:10], 2)
	copy(data[10:], []byte{0x12, 0x30, 0x00, 0x00, 0x45, 0x60, 0x00, 0x00})
	spec := FileSpec{Name: "TEST.16", Format: FormatShape4}

	archive, err := Decode(spec, data)

	require.NoError(t, err)
	require.Len(t, archive.Entries, 1)
	entry := archive.Entries[0]
	assert.Equal(t, 3, entry.Width)
	assert.Equal(t, 8, entry.StoredWidth)
	assert.Equal(t, 3, entry.Image.Bounds().Dx())
	assert.Equal(t, uint8(1), entry.Image.ColorIndexAt(0, 0))
	assert.Equal(t, uint8(3), entry.Image.ColorIndexAt(2, 0))
	assert.Equal(t, uint8(4), entry.Image.ColorIndexAt(0, 1))
}

func TestDecodeDungeonArchivePreservesMissingEntry(t *testing.T) {
	data := make([]byte, 18)
	binary.LittleEndian.PutUint16(data[0:2], 2)
	binary.LittleEndian.PutUint16(data[6:8], 10)
	binary.LittleEndian.PutUint16(data[10:12], 8)
	binary.LittleEndian.PutUint16(data[12:14], 1)
	copy(data[14:], []byte{0x12, 0x34, 0x56, 0x78})
	spec := FileSpec{Name: "DNG1.16", Format: FormatShape4}

	archive, err := Decode(spec, data)

	require.NoError(t, err)
	require.Len(t, archive.Entries, 2)
	assert.True(t, archive.Entries[0].Missing)
	assert.False(t, archive.Entries[1].Missing)
}

func TestDecodePairedArchiveAssignsImageAndMaskTypes(t *testing.T) {
	data := make([]byte, 19)
	binary.LittleEndian.PutUint16(data[0:2], 1)
	binary.LittleEndian.PutUint16(data[2:4], 6)
	binary.LittleEndian.PutUint16(data[4:6], 14)
	binary.LittleEndian.PutUint16(data[6:8], 8)
	binary.LittleEndian.PutUint16(data[8:10], 1)
	copy(data[10:14], []byte{0x12, 0x34, 0x56, 0x78})
	binary.LittleEndian.PutUint16(data[14:16], 8)
	binary.LittleEndian.PutUint16(data[16:18], 1)
	data[18] = 0x81
	spec := FileSpec{Name: "ITEMS.16", Format: FormatPaired}

	archive, err := Decode(spec, data)

	require.NoError(t, err)
	assert.Equal(t, PixelEGA4, archive.Entries[0].PixelType)
	assert.Equal(t, PixelMask, archive.Entries[1].PixelType)
	assert.Equal(t, uint8(1), archive.Entries[1].Image.ColorIndexAt(0, 0))
	assert.Equal(t, uint8(1), archive.Entries[1].Image.ColorIndexAt(7, 0))
}

func TestDecodeRejectsPayloadSizeMismatch(t *testing.T) {
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data[0:2], 1)
	binary.LittleEndian.PutUint16(data[2:4], 6)
	binary.LittleEndian.PutUint16(data[6:8], 8)
	binary.LittleEndian.PutUint16(data[8:10], 1)

	_, err := Decode(FileSpec{Name: "BAD.16", Format: FormatShape4}, data)

	require.ErrorContains(t, err, "payload is 1 bytes, want 4")
}

func TestDecodeProportionalSpacePreservesZeroWidthWithoutImage(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint16(data[0:2], 1)
	binary.LittleEndian.PutUint16(data[2:4], 4)
	binary.LittleEndian.PutUint16(data[6:8], 8)
	spec := FileSpec{Name: "PROPORT.PCS", Format: FormatFontPCS}

	archive, err := Decode(spec, data)

	require.NoError(t, err)
	require.Len(t, archive.Entries, 1)
	assert.Zero(t, archive.Entries[0].Width)
	assert.Nil(t, archive.Entries[0].Image)
	assert.False(t, archive.Entries[0].Missing)
}

func TestDecodePathSplitsAndRasterizesStrokes(t *testing.T) {
	spec := FileSpec{Name: "BRITISH.PTH", Format: FormatPath, EntryNames: []string{"one", "two"}}

	archive, err := Decode(spec, []byte{0x11, 0x11, 0, 0x91, 0})

	require.NoError(t, err)
	require.Len(t, archive.Entries, 2)
	assert.Equal(t, "one", archive.Entries[0].Name)
	assert.NotNil(t, archive.Entries[0].Image)
	assert.Equal(t, uint8(1), archive.Entries[0].Image.ColorIndexAt(3, 3))
	assert.Equal(t, -1, archive.Entries[1].Path[0].DX)
}

func TestCatalogIncludesCompleteNonTileSet(t *testing.T) {
	catalog := Catalog()

	assert.Len(t, catalog, 32)
	assert.Equal(t, "DNG1.16", catalog[0].Name)
	assert.Equal(t, "BRITISH.PTH", catalog[len(catalog)-1].Name)
}

func TestFindSpec(t *testing.T) {
	spec, found := FindSpec("ULTIMA.16")

	require.True(t, found)
	assert.Equal(t, FormatShape4, spec.Format)
	_, found = FindSpec("MISSING.16")
	assert.False(t, found)
}
