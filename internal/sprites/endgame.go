package sprites

import (
	"errors"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
)

var errEndgameAssetsInvalid = errors.New("endgame assets are invalid")

var (
	//nolint:gochecknoglobals // Immutable finale text offsets; ref:DNG-0015.
	endgameArtTextOffsets = [...]int{0, 424, 956, 1530, 2280, 2932}
	//nolint:gochecknoglobals // Immutable finale message offsets; ref:DNG-0015.
	endgameMessageOffsets = [...]int{0, 33, 73, 171, 211, 296, 359, 457, 529, 587, 725}
)

const (
	endgameFirstArtArchive  = "END1.16"
	endgameSecondArtArchive = "END2.16"
	endgameCertificate      = "ENDSC.16"
	endgameTitleArchive     = "TEXT.16"
	endgameArtShapeCount    = 3
	endgameTitleShapeCount  = 6
)

// EndgameAssets contains the original finale art, proportional font, and text records; ref:DNG-0015.
type EndgameAssets struct {
	Shapes    map[string][]*ebiten.Image
	Narrative *NarrativeIntroAssets
	ArtText   []string
	Messages  []string
}

// LoadEndgameAssets decodes the installed original ending files without embedding them.
func LoadEndgameAssets(dataDir string) (*EndgameAssets, error) {
	archiveNames := []string{
		endgameFirstArtArchive, endgameSecondArtArchive, endgameCertificate, endgameTitleArchive,
	}
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
	endData, err := os.ReadFile(files.Resolve(dataDir, "END.DAT"))
	if err != nil {
		return nil, fmt.Errorf("reading endgame narrative: %w", err)
	}
	//nolint:gosec // Configured original-data directory.
	messageData, err := os.ReadFile(files.Resolve(dataDir, "ENDMSG.DAT"))
	if err != nil {
		return nil, fmt.Errorf("reading endgame dialogue: %w", err)
	}

	return newEndgameAssets(archives, font, endData, messageData)
}

func newEndgameAssets(
	archives map[string]legacy.Archive,
	font legacy.Archive,
	endData, messageData []byte,
) (*EndgameAssets, error) {
	expectedCounts := map[string]int{
		endgameFirstArtArchive: endgameArtShapeCount, endgameSecondArtArchive: endgameArtShapeCount,
		endgameCertificate: 1, endgameTitleArchive: endgameTitleShapeCount,
	}
	shapes := make(map[string][]*ebiten.Image, len(expectedCounts))
	for name, expected := range expectedCounts {
		archive, ok := archives[name]
		images, err := endgameArchiveImages(name, archive, ok, expected)
		if err != nil {
			return nil, err
		}
		shapes[name] = images
	}
	glyphs, glyphWidths, err := narrativeIntroFont(font)
	if err != nil {
		return nil, err
	}
	artText, err := parseEndgameRecords("END.DAT", endData, endgameArtTextOffsets[:])
	if err != nil {
		return nil, err
	}
	messages, err := parseEndgameRecords("ENDMSG.DAT", messageData, endgameMessageOffsets[:])
	if err != nil {
		return nil, err
	}
	narrative := &NarrativeIntroAssets{
		Shapes: shapes, Glyphs: glyphs, GlyphWidths: glyphWidths, StoryBlocks: artText,
	}

	return &EndgameAssets{Shapes: shapes, Narrative: narrative, ArtText: artText, Messages: messages}, nil
}

func endgameArchiveImages(
	name string,
	archive legacy.Archive,
	found bool,
	expected int,
) ([]*ebiten.Image, error) {
	if !found || archive.Spec.Name != name || archive.Spec.Format != legacy.FormatShape4 ||
		len(archive.Entries) != expected {
		return nil, fmt.Errorf("%w: %s archive does not match the expected finale layout", errEndgameAssetsInvalid, name)
	}
	images := make([]*ebiten.Image, len(archive.Entries))
	for index, entry := range archive.Entries {
		if entry.Missing || entry.Image == nil || entry.PixelType != legacy.PixelEGA4 {
			return nil, fmt.Errorf("%w: %s entry %d is unavailable", errEndgameAssetsInvalid, name, index)
		}
		images[index] = ebiten.NewImageFromImage(entry.Image)
	}

	return images, nil
}

func parseEndgameRecords(name string, data []byte, offsets []int) ([]string, error) {
	result := make([]string, len(offsets))
	for index, offset := range offsets {
		if offset < 0 || offset >= len(data) {
			return nil, fmt.Errorf(
				"%w: %s record %d offset %d exceeds %d bytes",
				errEndgameAssetsInvalid, name, index, offset, len(data),
			)
		}
		limit := len(data)
		if index+1 < len(offsets) {
			limit = offsets[index+1]
		}
		end := offset
		for end < limit && data[end] != 0 {
			end++
		}
		if end == limit {
			return nil, fmt.Errorf("%w: %s record %d is not terminated before %d", errEndgameAssetsInvalid, name, index, limit)
		}
		result[index] = string(data[offset:end])
	}

	return result, nil
}
