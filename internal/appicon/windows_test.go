package appicon

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const committedResourceDir = "../../cmd/ultimav"

// parsedIcon is one RT_ICON image located through the RT_GROUP_ICON directory.
type parsedIcon struct {
	image  *image.NRGBA
	mask   []byte
	format string
}

// parseWindowsResource validates the COFF wrapper and resource tree the Go linker
// and Windows shell consume, returning the group's icons keyed by edge length.
func parseWindowsResource(t *testing.T, object []byte, machine Machine) map[int]parsedIcon {
	t.Helper()
	file, err := pe.NewFile(bytes.NewReader(object))
	require.NoError(t, err)
	require.Equal(t, uint16(machine), file.Machine)
	require.Len(t, file.Sections, 1)
	section := file.Sections[0]
	require.Equal(t, ".rsrc", section.Name)
	require.Equal(t, uint32(sectionCharacteristics), section.Characteristics)
	require.Len(t, file.COFFSymbols, 1)
	require.Equal(t, uint8(coffSymbolClassStatic), file.COFFSymbols[0].StorageClass)
	data, err := section.Data()
	require.NoError(t, err)
	relocationType, err := machine.relocationType()
	require.NoError(t, err)

	directory := func(offset uint32) map[uint32]uint32 {
		require.Zero(t, binary.LittleEndian.Uint16(data[offset+12:]), "named entries")
		count := binary.LittleEndian.Uint16(data[offset+14:])
		entries := make(map[uint32]uint32, count)
		for index := range uint32(count) {
			entry := offset + directoryHeaderSize + index*directoryEntrySize
			entries[binary.LittleEndian.Uint32(data[entry:])] = binary.LittleEndian.Uint32(data[entry+4:])
		}

		return entries
	}
	relocated := make(map[uint32]bool, len(section.Relocs))
	for _, relocation := range section.Relocs {
		assert.Zero(t, relocation.SymbolTableIndex)
		assert.Equal(t, relocationType, relocation.Type)
		relocated[relocation.VirtualAddress] = true
	}
	body := func(kind, resourceID uint32) []byte {
		names, ok := directory(0)[kind]
		require.True(t, ok, "resource type %d", kind)
		require.NotZero(t, names&subdirectoryFlag)
		languages, ok := directory(names &^ subdirectoryFlag)[resourceID]
		require.True(t, ok, "resource %d/%d", kind, resourceID)
		require.NotZero(t, languages&subdirectoryFlag)
		dataEntry, ok := directory(languages &^ subdirectoryFlag)[resourceLanguage]
		require.True(t, ok, "language for %d/%d", kind, resourceID)
		require.True(t, relocated[dataEntry], "data entry %d/%d has no RVA relocation", kind, resourceID)
		start := binary.LittleEndian.Uint32(data[dataEntry:])
		size := binary.LittleEndian.Uint32(data[dataEntry+4:])
		require.Zero(t, start%resourceAlignment)

		return data[start : start+size]
	}

	require.Len(t, directory(0), 2)
	require.Len(t, section.Relocs, len(WindowsIconSizes())+1)
	group := body(resourceTypeGroupIcon, groupIconID)
	require.Equal(t, uint16(groupResourceIcon), binary.LittleEndian.Uint16(group[2:]))
	count := int(binary.LittleEndian.Uint16(group[4:]))
	require.Len(t, group, groupHeaderSize+count*groupEntrySize)
	icons := make(map[int]parsedIcon, count)
	for index := range count {
		entry := group[groupHeaderSize+index*groupEntrySize:]
		size := int(entry[0])
		if size == 0 {
			size = pngIconSize
		}
		require.Equal(t, entry[0], entry[1], "square icon")
		assert.Equal(t, uint16(iconBitCount), binary.LittleEndian.Uint16(entry[6:]))
		encoded := body(resourceTypeIcon, uint32(binary.LittleEndian.Uint16(entry[12:])))
		require.Equal(t, binary.LittleEndian.Uint32(entry[8:]), uint32(len(encoded)), "group byte count")
		icons[size] = decodeIconImage(t, encoded, size)
	}

	return icons
}

func decodeIconImage(t *testing.T, encoded []byte, size int) parsedIcon {
	t.Helper()
	if size == pngIconSize {
		decoded, err := png.Decode(bytes.NewReader(encoded))
		require.NoError(t, err)
		icon := image.NewNRGBA(decoded.Bounds())
		for y := range size {
			for x := range size {
				icon.Set(x, y, decoded.At(x, y))
			}
		}

		return parsedIcon{image: icon, mask: nil, format: "png"}
	}
	require.Equal(t, uint32(bitmapInfoHeaderSize), binary.LittleEndian.Uint32(encoded[0:]))
	require.Equal(t, uint32(size), binary.LittleEndian.Uint32(encoded[4:]))
	require.Equal(t, uint32(2*size), binary.LittleEndian.Uint32(encoded[8:]))
	require.Equal(t, uint16(iconBitCount), binary.LittleEndian.Uint16(encoded[14:]))
	require.Zero(t, binary.LittleEndian.Uint32(encoded[16:]), "uncompressed")
	pixels := encoded[bitmapInfoHeaderSize : bitmapInfoHeaderSize+size*size*bitmapBytesPerPixel]
	icon := image.NewNRGBA(image.Rect(0, 0, size, size))
	for row := range size {
		for x := range size {
			pixel := pixels[(row*size+x)*bitmapBytesPerPixel:]
			icon.SetNRGBA(x, size-1-row, color.NRGBA{R: pixel[2], G: pixel[1], B: pixel[0], A: pixel[3]})
		}
	}

	return parsedIcon{image: icon, mask: encoded[bitmapInfoHeaderSize+len(pixels):], format: "bmp"}
}

func assertIconsMatchTile(t *testing.T, icons map[int]parsedIcon, tile *image.NRGBA) {
	t.Helper()
	require.Len(t, icons, len(WindowsIconSizes()))
	for _, size := range WindowsIconSizes() {
		want, err := Render(tile, size, size)
		require.NoError(t, err)
		icon, ok := icons[size]
		require.True(t, ok, "missing %dpx icon", size)
		assert.Equal(t, want.Pix, icon.image.Pix, "%dpx icon pixels", size)
		if size < pngIconSize {
			assert.Equal(t, "bmp", icon.format, "%dpx icon format", size)
		} else {
			assert.Equal(t, "png", icon.format)
		}
	}
}

func TestWindowsResourceEmbedsEveryIconSizeForEachTarget(t *testing.T) {
	atlas := bundledAtlas(t)
	tile, err := Tile(atlas)
	require.NoError(t, err)

	for _, target := range WindowsTargets() {
		t.Run(target.Architecture, func(t *testing.T) {
			object, err := WindowsResource(atlas, target.Machine)
			require.NoError(t, err)

			icons := parseWindowsResource(t, object, target.Machine)
			assertIconsMatchTile(t, icons, tile)
			// The opaque Avatar tile needs no transparency mask bits.
			for size, icon := range icons {
				for _, bits := range icon.mask {
					require.Zero(t, bits, "%dpx mask", size)
				}
			}
		})
	}
}

func TestWindowsResourceMasksTransparentPixels(t *testing.T) {
	transparent := map[image.Point]bool{{X: 0, Y: 0}: true, {X: 9, Y: 0}: true, {X: 15, Y: 15}: true}
	atlas := indexedAtlas(t, func(x, y int) color.NRGBA {
		if transparent[image.Pt(x, y)] {
			return color.NRGBA{}
		}

		return color.NRGBA{R: 200, G: 40, B: 10, A: 255}
	})
	tile, err := Tile(atlas)
	require.NoError(t, err)

	object, err := WindowsResource(atlas, MachineAMD64)
	require.NoError(t, err)
	icons := parseWindowsResource(t, object, MachineAMD64)
	assertIconsMatchTile(t, icons, tile)

	icon := icons[TileSize]
	stride := 4 // 16 mask bits padded to a 32-bit word
	require.Len(t, icon.mask, stride*TileSize)
	for y := range TileSize {
		row := TileSize - 1 - y // bitmaps are bottom-up
		for x := range TileSize {
			masked := icon.mask[row*stride+x/8]&(0x80>>(x%8)) != 0
			assert.Equal(t, transparent[image.Pt(x, y)], masked, "mask bit %d,%d", x, y)
		}
	}
}

func TestWindowsResourceRejectsUnsupportedMachines(t *testing.T) {
	_, err := WindowsResource(bundledAtlas(t), Machine(0x014c))
	require.ErrorIs(t, err, errWindowsResource)
}

func TestCommittedWindowsResourcesMatchTheBundledAtlas(t *testing.T) {
	tile, err := Tile(bundledAtlas(t))
	require.NoError(t, err)

	for _, target := range WindowsTargets() {
		t.Run(target.Architecture, func(t *testing.T) {
			object, err := os.ReadFile(filepath.Join(committedResourceDir, target.FileName()))
			require.NoError(t, err, "regenerate with make generate")

			icons := parseWindowsResource(t, object, target.Machine)
			assertIconsMatchTile(t, icons, tile)
		})
	}
}
