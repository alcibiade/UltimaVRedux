package appicon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"slices"
)

// Machine is a Windows COFF machine type.
type Machine uint16

// Windows targets that receive the executable icon resource.
const (
	MachineAMD64 Machine = 0x8664
	MachineARM64 Machine = 0xaa64
)

// Windows resource and COFF layout constants. Field offsets follow the PE/COFF
// specification; zero-valued fields (timestamps, code pages, reserved words) are
// left at zero so regenerated objects stay reproducible.
const (
	resourceTypeIcon      = 3
	resourceTypeGroupIcon = 14
	resourceLanguage      = 0x0409 // en-US
	groupIconID           = 1

	relocationAMD64Addr32NB = 0x0003
	relocationARM64Addr32NB = 0x0002

	coffFileHeaderSize     = 20
	coffSectionHeaderSize  = 40
	coffRelocationSize     = 10
	coffSymbolSize         = 18
	coffStringTableSize    = 4
	coffSymbolClassStatic  = 3
	sectionCharacteristics = 0x40000040 // initialized data, readable

	directoryHeaderSize = 16
	directoryEntrySize  = 8
	dataEntrySize       = 16
	subdirectoryFlag    = 0x80000000
	resourceAlignment   = 8

	groupHeaderSize      = 6
	groupEntrySize       = 14
	groupResourceIcon    = 1
	iconPlanes           = 1
	iconBitCount         = 32
	bitmapInfoHeaderSize = 40
	bitmapBytesPerPixel  = 4
	maskWordBits         = 32
	maskWordBytes        = 4
	bitsPerByte          = 8
	maskHighBit          = 0x80
	// Windows stores 256-pixel icons as PNG and encodes their width as zero.
	pngIconSize = 256
	// Keep every offset representable in the 32-bit PE fields.
	maxObjectSize = math.MaxUint32
)

var errWindowsResource = errors.New("windows icon resource")

// WindowsTarget names one committed executable-resource object.
type WindowsTarget struct {
	Architecture string
	Machine      Machine
}

// FileName is the object name the Go toolchain links only for this target.
func (t WindowsTarget) FileName() string {
	return "rsrc_windows_" + t.Architecture + ".syso"
}

// WindowsTargets lists the architectures whose executables embed the icon.
func WindowsTargets() []WindowsTarget {
	return []WindowsTarget{
		{Architecture: "amd64", Machine: MachineAMD64},
		{Architecture: "arm64", Machine: MachineARM64},
	}
}

// WindowsIconSizes lists the icon images embedded in the executable. Sizes below
// 256 are stored as 32-bit bitmaps for the widest shell compatibility.
func WindowsIconSizes() []int {
	return []int{TileSize, 2 * TileSize, 3 * TileSize, 4 * TileSize, pngIconSize}
}

type resource struct {
	kind uint16
	id   uint16
	data []byte
}

type resourceSection struct {
	data []byte
	// relocations holds the section offsets of fields that need the section RVA.
	relocations []uint32
}

// WindowsResource returns a COFF object holding the RT_ICON images and the
// RT_GROUP_ICON directory that Explorer, the taskbar and shortcuts display.
func WindowsResource(atlasPNG []byte, machine Machine) ([]byte, error) {
	relocationType, err := machine.relocationType()
	if err != nil {
		return nil, err
	}
	tile, err := Tile(atlasPNG)
	if err != nil {
		return nil, err
	}
	resources, err := iconResources(tile)
	if err != nil {
		return nil, err
	}
	section, err := buildResourceSection(resources)
	if err != nil {
		return nil, err
	}

	return coffObject(machine, relocationType, section)
}

func (m Machine) relocationType() (uint16, error) {
	switch m {
	case MachineAMD64:
		return relocationAMD64Addr32NB, nil
	case MachineARM64:
		return relocationARM64Addr32NB, nil
	}

	return 0, fmt.Errorf("%w: unsupported machine %#04x", errWindowsResource, uint16(m))
}

func iconResources(tile *image.NRGBA) ([]resource, error) {
	sizes := WindowsIconSizes()
	group := make([]byte, groupHeaderSize, groupHeaderSize+groupEntrySize*len(sizes))
	binary.LittleEndian.PutUint16(group[2:], groupResourceIcon)
	binary.LittleEndian.PutUint16(group[4:], field16(len(sizes)))
	resources := make([]resource, 0, len(sizes)+1)
	for index, size := range sizes {
		icon, err := Render(tile, size, size)
		if err != nil {
			return nil, err
		}
		data, err := encodeIconImage(icon)
		if err != nil {
			return nil, err
		}
		resourceID := field16(index + 1)
		entry := make([]byte, groupEntrySize)
		entry[0] = iconDimension(size)
		entry[1] = iconDimension(size)
		binary.LittleEndian.PutUint16(entry[4:], iconPlanes)
		binary.LittleEndian.PutUint16(entry[6:], iconBitCount)
		binary.LittleEndian.PutUint32(entry[8:], field32(len(data)))
		binary.LittleEndian.PutUint16(entry[12:], resourceID)
		group = append(group, entry...)
		resources = append(resources, resource{kind: resourceTypeIcon, id: resourceID, data: data})
	}

	return append(resources, resource{kind: resourceTypeGroupIcon, id: groupIconID, data: group}), nil
}

// encodeIconImage stores 256-pixel icons as PNG and smaller icons as a bottom-up
// BGRA bitmap followed by its 1-bit transparency mask.
func encodeIconImage(icon *image.NRGBA) ([]byte, error) {
	size := icon.Bounds().Dx()
	if size >= pngIconSize {
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, icon); err != nil {
			return nil, fmt.Errorf("encoding %dpx Windows icon: %w", size, err)
		}

		return encoded.Bytes(), nil
	}
	maskStride := (size + maskWordBits - 1) / maskWordBits * maskWordBytes
	data := make([]byte, bitmapInfoHeaderSize, bitmapInfoHeaderSize+size*size*bitmapBytesPerPixel+maskStride*size)
	binary.LittleEndian.PutUint32(data[0:], bitmapInfoHeaderSize)
	binary.LittleEndian.PutUint32(data[4:], field32(size))
	// The height covers the colour bitmap and the mask that follows it.
	binary.LittleEndian.PutUint32(data[8:], field32(size+size))
	binary.LittleEndian.PutUint16(data[12:], iconPlanes)
	binary.LittleEndian.PutUint16(data[14:], iconBitCount)
	mask := make([]byte, maskStride*size)
	for row := range size {
		y := size - 1 - row
		for x := range size {
			pixel := icon.NRGBAAt(x, y)
			data = append(data, pixel.B, pixel.G, pixel.R, pixel.A)
			if pixel.A == 0 {
				mask[row*maskStride+x/bitsPerByte] |= maskHighBit >> (x % bitsPerByte)
			}
		}
	}

	return append(data, mask...), nil
}

// sectionPlan records where each directory, data entry and resource body lives
// within the resource section.
type sectionPlan struct {
	kinds               []uint16
	kindDirectories     map[uint16]int
	languageDirectories []int
	dataEntries         []int
	bodies              []int
	size                int
}

// planResourceSection lays out the type, name and language directories, then the
// data entries, then the 8-byte-aligned resource bodies.
func planResourceSection(ordered []resource) sectionPlan {
	var kinds []uint16
	for _, entry := range ordered {
		if len(kinds) == 0 || kinds[len(kinds)-1] != entry.kind {
			kinds = append(kinds, entry.kind)
		}
	}
	plan := sectionPlan{
		kinds:               kinds,
		kindDirectories:     make(map[uint16]int, len(kinds)),
		languageDirectories: make([]int, len(ordered)),
		dataEntries:         make([]int, len(ordered)),
		bodies:              make([]int, len(ordered)),
		size:                0,
	}
	offset := directorySize(len(kinds))
	for _, kind := range kinds {
		plan.kindDirectories[kind] = offset
		offset += directorySize(countKind(ordered, kind))
	}
	for index := range ordered {
		plan.languageDirectories[index] = offset
		offset += directorySize(1)
	}
	for index := range ordered {
		plan.dataEntries[index] = offset
		offset += dataEntrySize
	}
	for index, entry := range ordered {
		offset = alignUp(offset, resourceAlignment)
		plan.bodies[index] = offset
		offset += len(entry.data)
	}
	plan.size = alignUp(offset, resourceAlignment)

	return plan
}

func buildResourceSection(resources []resource) (resourceSection, error) {
	ordered := slices.Clone(resources)
	slices.SortFunc(ordered, func(left, right resource) int {
		if left.kind != right.kind {
			return int(left.kind) - int(right.kind)
		}

		return int(left.id) - int(right.id)
	})
	plan := planResourceSection(ordered)
	if plan.size > maxObjectSize {
		return resourceSection{}, fmt.Errorf("%w: %d-byte section is too large", errWindowsResource, plan.size)
	}
	section := make([]byte, plan.size)
	writeDirectoryHeader(section, 0, len(plan.kinds))
	for index, kind := range plan.kinds {
		writeDirectoryEntry(section, 0, index, uint32(kind), subdirectoryFlag|field32(plan.kindDirectories[kind]))
		writeDirectoryHeader(section, plan.kindDirectories[kind], countKind(ordered, kind))
	}
	relocations := make([]uint32, 0, len(ordered))
	for index, entry := range ordered {
		languages := plan.languageDirectories[index]
		dataEntry := plan.dataEntries[index]
		position := index - slices.IndexFunc(ordered, func(other resource) bool { return other.kind == entry.kind })
		writeDirectoryEntry(section, plan.kindDirectories[entry.kind], position, uint32(entry.id),
			subdirectoryFlag|field32(languages))
		writeDirectoryHeader(section, languages, 1)
		writeDirectoryEntry(section, languages, 0, resourceLanguage, field32(dataEntry))
		binary.LittleEndian.PutUint32(section[dataEntry:], field32(plan.bodies[index]))
		binary.LittleEndian.PutUint32(section[dataEntry+4:], field32(len(entry.data)))
		copy(section[plan.bodies[index]:], entry.data)
		relocations = append(relocations, field32(dataEntry))
	}

	return resourceSection{data: section, relocations: relocations}, nil
}

// coffObject wraps the resource section in a single-section COFF object whose
// relocations make each data-entry offset image-relative when linked.
func coffObject(machine Machine, relocationType uint16, section resourceSection) ([]byte, error) {
	rawOffset := coffFileHeaderSize + coffSectionHeaderSize
	relocationOffset := rawOffset + len(section.data)
	symbolOffset := relocationOffset + coffRelocationSize*len(section.relocations)
	objectSize := symbolOffset + coffSymbolSize + coffStringTableSize
	if objectSize > maxObjectSize || len(section.relocations) > math.MaxUint16 {
		return nil, fmt.Errorf("%w: %d-byte object with %d relocations is too large",
			errWindowsResource, objectSize, len(section.relocations))
	}
	object := make([]byte, objectSize)

	binary.LittleEndian.PutUint16(object[0:], uint16(machine))
	binary.LittleEndian.PutUint16(object[2:], 1) // one section
	binary.LittleEndian.PutUint32(object[8:], field32(symbolOffset))
	binary.LittleEndian.PutUint32(object[12:], 1) // one symbol

	header := object[coffFileHeaderSize:rawOffset]
	copy(header, ".rsrc")
	binary.LittleEndian.PutUint32(header[16:], field32(len(section.data)))
	binary.LittleEndian.PutUint32(header[20:], field32(rawOffset))
	binary.LittleEndian.PutUint32(header[24:], field32(relocationOffset))
	binary.LittleEndian.PutUint16(header[32:], field16(len(section.relocations)))
	binary.LittleEndian.PutUint32(header[36:], sectionCharacteristics)

	copy(object[rawOffset:], section.data)
	entry := relocationOffset
	for _, address := range section.relocations {
		binary.LittleEndian.PutUint32(object[entry:], address)
		// Symbol index zero is the section symbol written below.
		binary.LittleEndian.PutUint16(object[entry+8:], relocationType)
		entry += coffRelocationSize
	}

	symbol := object[symbolOffset:]
	copy(symbol, ".rsrc")
	binary.LittleEndian.PutUint16(symbol[12:], 1) // section number
	symbol[16] = coffSymbolClassStatic
	binary.LittleEndian.PutUint32(object[symbolOffset+coffSymbolSize:], coffStringTableSize)

	return object, nil
}

func writeDirectoryHeader(section []byte, offset, entries int) {
	binary.LittleEndian.PutUint16(section[offset+14:], field16(entries))
}

func writeDirectoryEntry(section []byte, directory, index int, name, target uint32) {
	entry := directory + directoryHeaderSize + index*directoryEntrySize
	binary.LittleEndian.PutUint32(section[entry:], name)
	binary.LittleEndian.PutUint32(section[entry+4:], target)
}

func directorySize(entries int) int {
	return directoryHeaderSize + entries*directoryEntrySize
}

func countKind(resources []resource, kind uint16) int {
	count := 0
	for _, entry := range resources {
		if entry.kind == kind {
			count++
		}
	}

	return count
}

func alignUp(value, alignment int) int {
	return (value + alignment - 1) / alignment * alignment
}

// iconDimension encodes an icon edge for a group entry, where 256 is stored as 0.
func iconDimension(size int) byte {
	if size == pngIconSize {
		return 0
	}
	if size <= 0 || size > math.MaxUint8 {
		panic(fmt.Sprintf("windows icon resource: unsupported icon size %d", size))
	}

	return byte(size)
}

// field32 and field16 convert layout quantities that the object-size checks have
// already bounded; an out-of-range value is a programming error.
func field32(value int) uint32 {
	if value < 0 || value > math.MaxUint32 {
		panic(fmt.Sprintf("windows icon resource: %d does not fit a 32-bit field", value))
	}

	return uint32(value)
}

func field16(value int) uint16 {
	if value < 0 || value > math.MaxUint16 {
		panic(fmt.Sprintf("windows icon resource: %d does not fit a 16-bit field", value))
	}

	return uint16(value)
}
