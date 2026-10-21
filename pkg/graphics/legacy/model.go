// Package legacy decodes graphics files shipped with the DOS version of Ultima V.
package legacy

import (
	"fmt"
	"image"
	"image/color"
)

// Format identifies a shipped graphics container layout.
type Format string

const (
	// FormatShape4 is an offset-indexed archive of packed EGA shapes.
	FormatShape4 Format = "shape-4bpp"
	// FormatPaired is an archive of alternating color images and masks.
	FormatPaired Format = "paired-image-mask"
	// FormatBitmap1 is an offset-indexed monochrome bitmap archive.
	FormatBitmap1 Format = "bitmap-1bpp"
	// FormatFontPCS is the proportional 8-pixel-high font archive.
	FormatFontPCS Format = "proportional-font"
	// FormatFontCH is a raw fixed-width 8x8 font.
	FormatFontCH Format = "fixed-font"
	// FormatPath is the Lord British signature path stream.
	FormatPath Format = "vector-path"
)

// PixelType identifies how an entry's pixels are encoded.
type PixelType string

const (
	// PixelEGA4 identifies packed four-bit EGA color indexes.
	PixelEGA4 PixelType = "ega-4bpp"
	// PixelMask identifies a one-bit image mask.
	PixelMask PixelType = "mask-1bpp"
	// PixelFont identifies a one-bit font glyph.
	PixelFont PixelType = "font-1bpp"
	// PixelPath identifies vector path commands.
	PixelPath PixelType = "path"
)

// FileSpec describes one known original graphics file.
type FileSpec struct {
	Name       string
	Format     Format
	Compressed bool
	EntryNames []string
}

// Archive is a decoded original graphics file.
type Archive struct {
	Spec         FileSpec
	OriginalSize int
	ExpandedSize int
	Entries      []Entry
}

// Entry is one decoded shape, mask, glyph, or path stroke.
type Entry struct {
	Index       int
	Name        string
	Offset      int
	Width       int
	Height      int
	StoredWidth int
	PixelType   PixelType
	Missing     bool
	Image       *image.Paletted
	Path        []PathCommand
}

// PathCommand is one decoded BRITISH.PTH movement command.
type PathCommand struct {
	DX       int
	DY       int
	PlotNext bool
}

// EGAPalette is the canonical 16-color EGA palette used for review exports.
// The original files store palette indexes, not RGB values.
var EGAPalette = color.Palette{ //nolint:gochecknoglobals // Immutable format palette.
	color.RGBA{R: 0, G: 0, B: 0, A: 255},
	color.RGBA{R: 0, G: 0, B: 170, A: 255},
	color.RGBA{R: 0, G: 170, B: 0, A: 255},
	color.RGBA{R: 0, G: 170, B: 170, A: 255},
	color.RGBA{R: 170, G: 0, B: 0, A: 255},
	color.RGBA{R: 170, G: 0, B: 170, A: 255},
	color.RGBA{R: 170, G: 85, B: 0, A: 255},
	color.RGBA{R: 170, G: 170, B: 170, A: 255},
	color.RGBA{R: 85, G: 85, B: 85, A: 255},
	color.RGBA{R: 85, G: 85, B: 255, A: 255},
	color.RGBA{R: 85, G: 255, B: 85, A: 255},
	color.RGBA{R: 85, G: 255, B: 255, A: 255},
	color.RGBA{R: 255, G: 85, B: 85, A: 255},
	color.RGBA{R: 255, G: 85, B: 255, A: 255},
	color.RGBA{R: 255, G: 255, B: 85, A: 255},
	color.RGBA{R: 255, G: 255, B: 255, A: 255},
}

var monochromePalette = color.Palette{ //nolint:gochecknoglobals // Immutable format palette.
	color.RGBA{R: 0, G: 0, B: 0, A: 255},
	color.RGBA{R: 255, G: 255, B: 255, A: 255},
}

func entryName(spec FileSpec, index int) string {
	if index < len(spec.EntryNames) && spec.EntryNames[index] != "" {
		return spec.EntryNames[index]
	}

	return fmt.Sprintf("entry-%03d", index)
}
