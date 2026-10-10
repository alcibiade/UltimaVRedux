package legacy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
)

const shapeHeaderSize = 4

// Decode decodes one original graphics file according to its catalog specification.
func Decode(spec FileSpec, original []byte) (Archive, error) {
	data := original
	if spec.Compressed {
		var err error
		data, err = Decompress(original)
		if err != nil {
			return Archive{}, fmt.Errorf("decompressing %s: %w", spec.Name, err)
		}
	}
	archive := Archive{Spec: spec, OriginalSize: len(original), ExpandedSize: len(data), Entries: nil}
	var err error
	switch spec.Format {
	case FormatShape4:
		archive.Entries, err = decodeOffsetArchive(spec, data, 4, false, PixelEGA4)
	case FormatPaired:
		archive.Entries, err = decodePairedArchive(spec, data)
	case FormatBitmap1:
		archive.Entries, err = decodeOffsetArchive(spec, data, 2, false, PixelMask)
	case FormatFontPCS:
		archive.Entries, err = decodeOffsetArchive(spec, data, 2, true, PixelFont)
	case FormatFontCH:
		archive.Entries, err = decodeFixedFont(spec, data)
	case FormatPath:
		archive.Entries, err = decodePath(spec, data)
	default:
		err = fmt.Errorf("unsupported graphics format %q", spec.Format)
	}
	if err != nil {
		return Archive{}, fmt.Errorf("decoding %s: %w", spec.Name, err)
	}

	return archive, nil
}

func decodeOffsetArchive(spec FileSpec, data []byte, offsetSize int, proportional bool, pixelType PixelType) ([]Entry, error) {
	if len(data) < 2 {
		return nil, errors.New("archive count is truncated")
	}
	count := int(binary.LittleEndian.Uint16(data[:2]))
	tableEnd := 2 + count*offsetSize
	if count == 0 || tableEnd > len(data) {
		return nil, fmt.Errorf("offset table for %d entries exceeds %d-byte archive", count, len(data))
	}
	offsets := make([]int, count)
	for index := range count {
		position := 2 + index*offsetSize
		offsets[index] = int(binary.LittleEndian.Uint16(data[position : position+2]))
		if offsetSize == 4 && binary.LittleEndian.Uint16(data[position+2:position+4]) != 0 {
			return nil, fmt.Errorf("entry %d has unsupported nonzero high offset word", index)
		}
	}

	return decodeEntries(spec, data, offsets, tableEnd, proportional, func(int) PixelType { return pixelType })
}

func decodePairedArchive(spec FileSpec, data []byte) ([]Entry, error) {
	if len(data) < 2 {
		return nil, errors.New("paired archive count is truncated")
	}
	logicalCount := int(binary.LittleEndian.Uint16(data[:2]))
	physicalCount := logicalCount * 2
	tableEnd := 2 + physicalCount*2
	if logicalCount == 0 || tableEnd > len(data) {
		return nil, fmt.Errorf("paired offset table for %d entries exceeds %d-byte archive", physicalCount, len(data))
	}
	offsets := make([]int, physicalCount)
	for index := range physicalCount {
		position := 2 + index*2
		offsets[index] = int(binary.LittleEndian.Uint16(data[position : position+2]))
	}

	return decodeEntries(spec, data, offsets, tableEnd, false, func(index int) PixelType {
		if index%2 == 0 {
			return PixelEGA4
		}

		return PixelMask
	})
}

func decodeEntries(
	spec FileSpec,
	data []byte,
	offsets []int,
	tableEnd int,
	proportional bool,
	pixelType func(int) PixelType,
) ([]Entry, error) {
	entries := make([]Entry, len(offsets))
	previous := tableEnd
	for index, offset := range offsets {
		entry := Entry{Index: index, Name: entryName(spec, index), Offset: offset, PixelType: pixelType(index)}
		if offset == 0 {
			entry.Missing = true
			entries[index] = entry

			continue
		}
		if offset < tableEnd || offset < previous || offset+shapeHeaderSize > len(data) {
			return nil, fmt.Errorf("entry %d has invalid offset %d", index, offset)
		}
		end := nextNonzeroOffset(offsets, index+1, len(data))
		if end <= offset+shapeHeaderSize || end > len(data) {
			return nil, fmt.Errorf("entry %d has invalid end offset %d", index, end)
		}
		entry.Width = int(binary.LittleEndian.Uint16(data[offset : offset+2]))
		entry.Height = int(binary.LittleEndian.Uint16(data[offset+2 : offset+4]))
		if entry.Height == 0 || (!proportional && entry.Width == 0) {
			return nil, fmt.Errorf("entry %d has invalid dimensions %dx%d", index, entry.Width, entry.Height)
		}
		payload := data[offset+shapeHeaderSize : end]
		var err error
		entry.Image, entry.StoredWidth, err = decodePixels(entry.Width, entry.Height, payload, entry.PixelType, proportional)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", index, err)
		}
		entries[index] = entry
		previous = offset
	}

	return entries, nil
}

func nextNonzeroOffset(offsets []int, start, fallback int) int {
	for _, offset := range offsets[start:] {
		if offset != 0 {
			return offset
		}
	}

	return fallback
}

func decodePixels(width, height int, payload []byte, pixelType PixelType, proportional bool) (*image.Paletted, int, error) {
	storedWidth := (width + 7) &^ 7
	if proportional {
		storedWidth = 8
	}
	bitsPerPixel := 1
	palette := monochromePalette
	if pixelType == PixelEGA4 {
		bitsPerPixel = 4
		palette = EGAPalette
	}
	expected := storedWidth * height * bitsPerPixel / 8
	if len(payload) != expected {
		return nil, 0, fmt.Errorf("payload is %d bytes, want %d for %dx%d stored pixels at %dbpp", len(payload), expected, storedWidth, height, bitsPerPixel)
	}
	if width == 0 {
		return nil, storedWidth, nil
	}
	img := image.NewPaletted(image.Rect(0, 0, storedWidth, height), palette)
	if bitsPerPixel == 4 {
		for byteIndex, value := range payload {
			pixelIndex := byteIndex * 2
			img.Pix[pixelIndex] = value >> 4
			img.Pix[pixelIndex+1] = value & 0x0f
		}
	} else {
		for byteIndex, value := range payload {
			row := byteIndex / (storedWidth / 8)
			byteColumn := byteIndex % (storedWidth / 8)
			for bit := range 8 {
				img.SetColorIndex(byteColumn*8+bit, row, (value>>uint(7-bit))&1)
			}
		}
	}
	img.Rect = image.Rect(0, 0, width, height)

	return img, storedWidth, nil
}

func decodeFixedFont(_ FileSpec, data []byte) ([]Entry, error) {
	const (
		glyphCount = 128
		glyphSize  = 8
	)
	if len(data) != glyphCount*glyphSize {
		return nil, fmt.Errorf("fixed font is %d bytes, want %d", len(data), glyphCount*glyphSize)
	}
	entries := make([]Entry, glyphCount)
	for index := range glyphCount {
		img, storedWidth, err := decodePixels(glyphSize, glyphSize, data[index*glyphSize:(index+1)*glyphSize], PixelFont, false)
		if err != nil {
			return nil, fmt.Errorf("glyph %d: %w", index, err)
		}
		entries[index] = Entry{
			Index: index, Name: fmt.Sprintf("glyph-%03d", index), Offset: index * glyphSize,
			Width: glyphSize, Height: glyphSize, StoredWidth: storedWidth, PixelType: PixelFont, Image: img,
		}
	}

	return entries, nil
}

func decodePath(spec FileSpec, data []byte) ([]Entry, error) {
	entries := make([]Entry, 0, len(spec.EntryNames))
	start := 0
	for index, value := range data {
		if value != 0 {
			continue
		}
		commands := make([]PathCommand, 0, index-start)
		for _, command := range data[start:index] {
			dx, dy := int(command>>4)&7, int(command)&7
			plotNext := dx <= 2 && dy <= 2
			if command&0x80 != 0 {
				dx = -dx
			}
			if command&0x08 != 0 {
				dy = -dy
			}
			commands = append(commands, PathCommand{DX: dx, DY: dy, PlotNext: plotNext})
		}
		entryIndex := len(entries)
		img := rasterizePath(commands)
		entries = append(entries, Entry{
			Index: entryIndex, Name: entryName(spec, entryIndex), Offset: start,
			Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), StoredWidth: img.Bounds().Dx(),
			PixelType: PixelPath, Path: commands, Image: img,
		})
		start = index + 1
	}
	if start != len(data) {
		return nil, errors.New("path data has no final terminator")
	}
	if len(entries) != len(spec.EntryNames) {
		return nil, fmt.Errorf("path has %d strokes, want %d", len(entries), len(spec.EntryNames))
	}

	return entries, nil
}

func rasterizePath(commands []PathCommand) *image.Paletted {
	type point struct{ x, y int }
	points := make([]point, 0, len(commands))
	x, y, shouldPlot := 0, 0, true
	minX, maxX, minY, maxY := 0, 0, 0, 0
	for _, command := range commands {
		if shouldPlot {
			points = append(points, point{x: x, y: y})
			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
		}
		shouldPlot = command.PlotNext
		x, y = x+command.DX, y+command.DY
	}
	if shouldPlot {
		points = append(points, point{x: x, y: y})
		minX, maxX = min(minX, x), max(maxX, x)
		minY, maxY = min(minY, y), max(maxY, y)
	}
	const padding = 1
	img := image.NewPaletted(image.Rect(0, 0, maxX-minX+1+padding*2, maxY-minY+1+padding*2), monochromePalette)
	for _, plotted := range points {
		img.SetColorIndex(plotted.x-minX+padding, plotted.y-minY+padding, 1)
	}

	return img
}
