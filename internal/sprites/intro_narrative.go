package sprites

import (
	"errors"
	"fmt"
	"image"
	stdcolor "image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
)

const introductionStoryBlockCount = 20

//nolint:gochecknoglobals // Immutable introduction text offsets; ref:INTRO-0003.
var introductionStoryOffsets = [...]int{
	0, 273, 971, 1424, 2051, 2756, 3437, 3619, 4022, 4467,
	4906, 5310, 5678, 6443, 7066, 7803, 8562, 9291, 10112, 10903,
}

//nolint:gochecknoglobals,mnd // Immutable introduction picture counts; ref:INTRO-0003.
var introductionShapeCounts = map[string]int{
	"TEXT.16": 6, "STORY1.16": 3, "STORY2.16": 4, "STORY3.16": 2,
	"STORY4.16": 2, "STORY5.16": 2, "STORY6.16": 8,
}

var errIntroStoryInvalid = errors.New("introduction story data is invalid")

// NarrativeIntroAssets contains the original graphics, proportional font, and
// story blocks used by the 21-frame Ultima V Introduction.
type NarrativeIntroAssets struct {
	Shapes      map[string][]*ebiten.Image
	Glyphs      []*ebiten.Image
	GlyphWidths []int
	StoryBlocks []string
}

// LoadNarrativeIntroAssets loads the optional narrative Introduction assets.
// It is intentionally separate from title loading so missing story data does
// not prevent the main menu from starting.
func LoadNarrativeIntroAssets(dataDir string) (*NarrativeIntroAssets, error) {
	archiveNames := []string{"TEXT.16", "STORY1.16", "STORY2.16", "STORY3.16", "STORY4.16", "STORY5.16", "STORY6.16"}
	archives := make(map[string]legacy.Archive, len(archiveNames))
	for _, name := range archiveNames {
		archive, err := loadIntroArchive(dataDir, name)
		if err != nil {
			return nil, err
		}
		archives[name] = archive
	}
	font, err := loadIntroArchive(dataDir, "PROPORT.PCS")
	if err != nil {
		return nil, err
	}
	storyPath := files.Resolve(dataDir, "STORY.DAT")
	storyData, err := os.ReadFile(storyPath) //nolint:gosec // Configured original-data directory is user-selected.
	if err != nil {
		return nil, fmt.Errorf("reading introduction text %s: %w", storyPath, err)
	}

	return newNarrativeIntroAssets(archives, font, storyData)
}

func newNarrativeIntroAssets(
	archives map[string]legacy.Archive,
	font legacy.Archive,
	storyData []byte,
) (*NarrativeIntroAssets, error) {
	shapes, err := narrativeIntroShapes(archives)
	if err != nil {
		return nil, err
	}
	glyphs, glyphWidths, err := narrativeIntroFont(font)
	if err != nil {
		return nil, err
	}
	storyBlocks, err := parseIntroductionStoryBlocks(storyData)
	if err != nil {
		return nil, err
	}

	return &NarrativeIntroAssets{
		Shapes: shapes, Glyphs: glyphs, GlyphWidths: glyphWidths, StoryBlocks: storyBlocks,
	}, nil
}

func narrativeIntroShapes(archives map[string]legacy.Archive) (map[string][]*ebiten.Image, error) {
	shapes := make(map[string][]*ebiten.Image, len(archives))
	for name, archive := range archives {
		if archive.Spec.Name != name || archive.Spec.Format != legacy.FormatShape4 {
			return nil, fmt.Errorf("%w: %s is not a matching shape archive", errIntroArchiveInvalid, name)
		}
		expectedCount, known := introductionShapeCounts[name]
		if !known || len(archive.Entries) != expectedCount {
			return nil, fmt.Errorf(
				"%w: %s has %d entries, want %d", errIntroArchiveInvalid, name, len(archive.Entries), expectedCount,
			)
		}
		images := make([]*ebiten.Image, len(archive.Entries))
		for index, entry := range archive.Entries {
			if entry.Missing || entry.Image == nil || entry.PixelType != legacy.PixelEGA4 {
				return nil, fmt.Errorf("%w: %s entry %d is unavailable", errIntroArchiveInvalid, name, index)
			}
			images[index] = ebiten.NewImageFromImage(entry.Image)
		}
		shapes[name] = images
	}

	return shapes, nil
}

func narrativeIntroFont(font legacy.Archive) ([]*ebiten.Image, []int, error) {
	if font.Spec.Name != "PROPORT.PCS" || font.Spec.Format != legacy.FormatFontPCS || len(font.Entries) != 91 {
		return nil, nil, fmt.Errorf("%w: PROPORT.PCS has an unexpected format or entry count", errIntroArchiveInvalid)
	}
	glyphs := make([]*ebiten.Image, len(font.Entries))
	glyphWidths := make([]int, len(font.Entries))
	for index, entry := range font.Entries {
		if entry.Missing || entry.PixelType != legacy.PixelFont {
			return nil, nil, fmt.Errorf("%w: PROPORT.PCS entry %d is unavailable", errIntroArchiveInvalid, index)
		}
		glyphWidths[index] = entry.Width
		if entry.Width > 0 {
			glyphs[index] = ebiten.NewImageFromImage(fontGlyphImage(entry.Image))
		}
	}

	return glyphs, glyphWidths, nil
}

func parseIntroductionStoryBlocks(data []byte) ([]string, error) {
	blocks := make([]string, introductionStoryBlockCount)
	for index, offset := range introductionStoryOffsets {
		if offset >= len(data) {
			return nil, fmt.Errorf(
				"%w: block %d offset %d exceeds %d-byte file", errIntroStoryInvalid, index, offset, len(data),
			)
		}
		var limit int
		if index+1 < len(introductionStoryOffsets) {
			limit = introductionStoryOffsets[index+1]
		} else {
			const finalTerminator = 11677
			limit = finalTerminator + 1
			if limit > len(data) {
				return nil, fmt.Errorf("%w: block %d is truncated before %d", errIntroStoryInvalid, index, limit)
			}
		}
		if limit > len(data) {
			return nil, fmt.Errorf("%w: block %d is truncated before %d", errIntroStoryInvalid, index, limit)
		}
		end := offset
		for end < limit && data[end] != 0 {
			end++
		}
		if end != limit-1 {
			return nil, fmt.Errorf(
				"%w: block %d terminator is not at expected offset %d", errIntroStoryInvalid, index, limit-1,
			)
		}
		blocks[index] = string(data[offset:end])
	}

	return blocks, nil
}

func fontGlyphImage(source *image.Paletted) *image.NRGBA {
	const opaqueChannel = 255

	result := image.NewNRGBA(source.Bounds())
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			if source.ColorIndexAt(x, y) != 0 {
				result.SetNRGBA(x, y, stdcolor.NRGBA{
					R: opaqueChannel, G: opaqueChannel, B: opaqueChannel, A: opaqueChannel,
				})
			}
		}
	}

	return result
}
