// Package exporter writes decoded Ultima V graphics to local review files.
package exporter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
)

const (
	manifestVersion = 1
	contactMaxWidth = 1024
	contactPadding  = 8
)

// Options configures one export run.
type Options struct {
	DataDir   string
	OutputDir string
}

// Manifest describes one deterministic extraction run.
type Manifest struct {
	Version  int               `json:"version"`
	Palette  []PaletteColor    `json:"palette"`
	Archives []ArchiveManifest `json:"archives"`
}

// PaletteColor records one review-palette entry.
type PaletteColor struct {
	Index int   `json:"index"`
	R     uint8 `json:"r"`
	G     uint8 `json:"g"`
	B     uint8 `json:"b"`
}

// ArchiveManifest records source and output metadata for one archive.
type ArchiveManifest struct {
	Name         string          `json:"name"`
	Format       legacy.Format   `json:"format"`
	Compressed   bool            `json:"compressed"`
	SHA256       string          `json:"sha256"`
	OriginalSize int             `json:"originalSize"`
	ExpandedSize int             `json:"expandedSize"`
	ContactSheet string          `json:"contactSheet"`
	Entries      []EntryManifest `json:"entries"`
}

// EntryManifest records one decoded image or missing archive slot.
type EntryManifest struct {
	Index       int              `json:"index"`
	Name        string           `json:"name"`
	Offset      int              `json:"offset"`
	Width       int              `json:"width"`
	Height      int              `json:"height"`
	StoredWidth int              `json:"storedWidth"`
	PixelType   legacy.PixelType `json:"pixelType"`
	Missing     bool             `json:"missing"`
	PathCommand int              `json:"pathCommands,omitempty"`
	PNG         string           `json:"png,omitempty"`
}

// Export decodes the complete supported catalog and writes PNG review files.
func Export(options Options) (Manifest, error) {
	if options.DataDir == "" {
		return Manifest{}, errors.New("data directory is required")
	}
	if options.OutputDir == "" {
		return Manifest{}, errors.New("output directory is required")
	}
	if err := os.MkdirAll(options.OutputDir, 0750); err != nil {
		return Manifest{}, fmt.Errorf("creating output directory: %w", err)
	}

	manifest := Manifest{Version: manifestVersion, Palette: manifestPalette(), Archives: make([]ArchiveManifest, 0, len(legacy.Catalog()))}
	for _, spec := range legacy.Catalog() {
		archiveManifest, err := exportArchive(options, spec)
		if err != nil {
			return Manifest{}, err
		}
		manifest.Archives = append(manifest.Archives, archiveManifest)
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "manifest.json"), manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func exportArchive(options Options, spec legacy.FileSpec) (ArchiveManifest, error) {
	sourcePath := filepath.Join(options.DataDir, spec.Name)
	data, err := os.ReadFile(sourcePath) //nolint:gosec // The user explicitly selects the source directory.
	if err != nil {
		return ArchiveManifest{}, fmt.Errorf("reading %s: %w", sourcePath, err)
	}
	archive, err := legacy.Decode(spec, data)
	if err != nil {
		return ArchiveManifest{}, err
	}
	directoryName := strings.ToLower(strings.ReplaceAll(spec.Name, ".", "_"))
	archiveDir := filepath.Join(options.OutputDir, directoryName)
	if err := os.MkdirAll(archiveDir, 0750); err != nil {
		return ArchiveManifest{}, fmt.Errorf("creating archive directory %s: %w", archiveDir, err)
	}

	hash := sha256.Sum256(data)
	result := ArchiveManifest{
		Name: spec.Name, Format: spec.Format, Compressed: spec.Compressed,
		SHA256: hex.EncodeToString(hash[:]), OriginalSize: archive.OriginalSize, ExpandedSize: archive.ExpandedSize,
		ContactSheet: "", Entries: make([]EntryManifest, 0, len(archive.Entries)),
	}
	for _, entry := range archive.Entries {
		entryManifest := EntryManifest{
			Index: entry.Index, Name: entry.Name, Offset: entry.Offset, Width: entry.Width, Height: entry.Height,
			StoredWidth: entry.StoredWidth, PixelType: entry.PixelType, Missing: entry.Missing, PathCommand: len(entry.Path),
			PNG: "",
		}
		if !entry.Missing && entry.Image != nil {
			filename := fmt.Sprintf("%03d-%s.png", entry.Index, sanitizeName(entry.Name))
			path := filepath.Join(archiveDir, filename)
			if err := writePNG(path, entry.Image); err != nil {
				return ArchiveManifest{}, err
			}
			entryManifest.PNG = filepath.ToSlash(filepath.Join(directoryName, filename))
		}
		result.Entries = append(result.Entries, entryManifest)
	}

	contact := makeContactSheet(archive.Entries)
	contactFilename := directoryName + "-contact.png"
	if err := writePNG(filepath.Join(options.OutputDir, contactFilename), contact); err != nil {
		return ArchiveManifest{}, err
	}
	result.ContactSheet = contactFilename

	return result, nil
}

func makeContactSheet(entries []legacy.Entry) image.Image {
	type placement struct {
		img image.Image
		x   int
		y   int
	}
	placements := make([]placement, 0, len(entries))
	x, y, rowHeight, usedWidth := contactPadding, contactPadding, 0, 1
	for _, entry := range entries {
		if entry.Missing || entry.Image == nil {
			continue
		}
		bounds := entry.Image.Bounds()
		if x > contactPadding && x+bounds.Dx()+contactPadding > contactMaxWidth {
			x, y, rowHeight = contactPadding, y+rowHeight+contactPadding, 0
		}
		placements = append(placements, placement{img: entry.Image, x: x, y: y})
		x += bounds.Dx() + contactPadding
		rowHeight = max(rowHeight, bounds.Dy())
		usedWidth = max(usedWidth, x)
	}
	height := max(1, y+rowHeight+contactPadding)
	sheet := image.NewRGBA(image.Rect(0, 0, min(contactMaxWidth, usedWidth), height))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.RGBA{R: 48, G: 48, B: 48, A: 255}}, image.Point{}, draw.Src)
	for _, item := range placements {
		destination := image.Rect(item.x, item.y, item.x+item.img.Bounds().Dx(), item.y+item.img.Bounds().Dy())
		draw.Draw(sheet, destination, item.img, item.img.Bounds().Min, draw.Src)
	}

	return sheet
}

func manifestPalette() []PaletteColor {
	result := make([]PaletteColor, 0, len(legacy.EGAPalette))
	for index, value := range legacy.EGAPalette {
		r, g, b, _ := value.RGBA()
		result = append(result, PaletteColor{
			Index: index,
			R:     uint8(r >> 8), //nolint:gosec // color.Color returns replicated 8-bit channels.
			G:     uint8(g >> 8), //nolint:gosec // color.Color returns replicated 8-bit channels.
			B:     uint8(b >> 8), //nolint:gosec // color.Color returns replicated 8-bit channels.
		})
	}

	return result
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	var builder strings.Builder
	for _, value := range name {
		if value >= 'a' && value <= 'z' || value >= '0' && value <= '9' || value == '-' {
			builder.WriteRune(value)
		} else {
			builder.WriteByte('-')
		}
	}

	return strings.Trim(builder.String(), "-")
}

func writePNG(path string, img image.Image) error {
	return writeFile(path, func(writer io.Writer) error {
		if err := png.Encode(writer, img); err != nil {
			return fmt.Errorf("encoding PNG: %w", err)
		}

		return nil
	})
}

func writeJSON(path string, value Manifest) error {
	return writeFile(path, func(writer io.Writer) error {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(value); err != nil {
			return fmt.Errorf("encoding manifest: %w", err)
		}

		return nil
	})
}

func writeFile(path string, encode func(io.Writer) error) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600) //nolint:gosec // Caller owns output path.
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	writeErr := encode(file)
	closeErr := file.Close()
	if writeErr != nil {
		writeErr = fmt.Errorf("writing %s: %w", path, writeErr)
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("closing %s: %w", path, closeErr)
	}

	return errors.Join(writeErr, closeErr)
}
