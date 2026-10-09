package sprites

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
)

const (
	characterCreationShapeCount = 11
	characterCreationTextSize   = 7745
	characterCreationVirtues    = 8
	characterCreationGlyphSize  = 8
	// CharacterCreationDilemmaCount is the number of unordered pairs among eight virtues.
	CharacterCreationDilemmaCount = 28
)

//nolint:gochecknoglobals // Immutable creation-art record offsets; ref:CREATE-0004.
var characterCreationTextOffsets = [...]int{
	0, 802,
	1730, 1959, 2188, 2400, 2631, 2861, 3089,
	3290, 3471, 3703, 3926, 4153, 4379,
	4607, 4827, 4980, 5206, 5357,
	5585, 5807, 6036, 6235,
	6447, 6680, 6912,
	7139, 7355,
	7559,
}

//nolint:gochecknoglobals,mnd // Verified CREATE.16 geometry is immutable original data.
var characterCreationShapeSizes = [...]image.Point{
	image.Pt(168, 96), image.Pt(120, 148), image.Pt(51, 67), image.Pt(43, 67),
	image.Pt(34, 69), image.Pt(55, 58), image.Pt(48, 61), image.Pt(42, 64),
	image.Pt(50, 65), image.Pt(42, 65), image.Pt(152, 100),
}

//nolint:gochecknoglobals // Verified PROPORT.PCS glyph widths are immutable original data.
var characterCreationGlyphWidths = [...]int{
	0, 2, 6, 7, 7, 7, 7, 3, 4, 4, 5, 6, 3, 3, 2, 7, 6, 5, 6, 6, 6, 6, 6,
	6, 6, 6, 2, 3, 5, 5, 5, 6, 7, 7, 6, 6, 6, 6, 6, 6, 6, 4, 7, 7, 6, 7,
	6, 6, 7, 6, 7, 6, 6, 6, 6, 7, 6, 6, 6, 3, 7, 3, 7, 8, 3, 5, 5, 4, 5,
	5, 5, 5, 5, 2, 3, 5, 2, 7, 6, 5, 5, 5, 4, 4, 4, 5, 5, 7, 5, 5, 4,
}

var errCharacterCreationAssetsInvalid = errors.New("character creation assets are invalid")

// CharacterCreationScript contains the two framing blocks and every pairwise dilemma.
type CharacterCreationScript struct {
	Introduction string
	Conclusion   string
	dilemmas     [characterCreationVirtues][characterCreationVirtues]string
}

// NewCharacterCreationScript constructs a complete semantic script from source-order dilemmas.
func NewCharacterCreationScript(
	introduction, conclusion string,
	dilemmaTexts [CharacterCreationDilemmaCount]string,
) (CharacterCreationScript, error) {
	if introduction == "" || conclusion == "" {
		return CharacterCreationScript{}, fmt.Errorf(
			"%w: framing text must not be empty",
			errCharacterCreationAssetsInvalid,
		)
	}
	script := CharacterCreationScript{
		Introduction: introduction,
		Conclusion:   conclusion,
		dilemmas:     [characterCreationVirtues][characterCreationVirtues]string{},
	}
	textIndex := 0
	for lower := range characterCreationVirtues - 1 {
		for higher := lower + 1; higher < characterCreationVirtues; higher++ {
			if dilemmaTexts[textIndex] == "" {
				return CharacterCreationScript{}, fmt.Errorf(
					"%w: dilemma text %d must not be empty",
					errCharacterCreationAssetsInvalid,
					textIndex,
				)
			}
			script.dilemmas[lower][higher] = dilemmaTexts[textIndex]
			textIndex++
		}
	}
	if textIndex != len(dilemmaTexts) {
		panic("NewCharacterCreationScript: pairwise question count mismatch")
	}

	return script, nil
}

// Dilemma returns the source text for a validated lower-index A/higher-index B pair.
func (script CharacterCreationScript) Dilemma(lowerVirtue, higherVirtue int) string {
	if lowerVirtue < 0 || higherVirtue >= characterCreationVirtues || lowerVirtue >= higherVirtue {
		panic("CharacterCreationScript.Dilemma: invalid ordered virtue pair")
	}
	text := script.dilemmas[lowerVirtue][higherVirtue]
	if text == "" {
		panic("CharacterCreationScript.Dilemma: validated question text is unavailable")
	}

	return text
}

// CharacterCreationAssets contains semantic images, proportional glyphs, and question text.
type CharacterCreationAssets struct {
	Oracle        *ebiten.Image
	Incense       *ebiten.Image
	VirtueSymbols [characterCreationVirtues]*ebiten.Image
	Shrine        *ebiten.Image
	Glyphs        []*ebiten.Image
	GlyphWidths   []int
	Script        CharacterCreationScript
}

// LoadCharacterCreationAssets loads optional original assets only when creation starts.
func LoadCharacterCreationAssets(dataDir string) (*CharacterCreationAssets, error) {
	shapes, err := loadIntroArchive(dataDir, "CREATE.16")
	if err != nil {
		return nil, fmt.Errorf("loading character creation shapes: %w", err)
	}
	font, err := loadIntroArchive(dataDir, "PROPORT.PCS")
	if err != nil {
		return nil, fmt.Errorf("loading character creation font: %w", err)
	}
	questionPath := filepath.Join(dataDir, "QUESTION.DAT")
	questionData, err := os.ReadFile(questionPath) //nolint:gosec // Configured original-data directory is user-selected.
	if err != nil {
		return nil, fmt.Errorf("reading character creation text %s: %w", questionPath, err)
	}

	return newCharacterCreationAssets(shapes, font, questionData)
}

func newCharacterCreationAssets(
	shapes legacy.Archive,
	font legacy.Archive,
	questionData []byte,
) (*CharacterCreationAssets, error) {
	images, err := characterCreationShapes(shapes)
	if err != nil {
		return nil, err
	}
	glyphs, glyphWidths, err := characterCreationFont(font)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCharacterCreationAssetsInvalid, err)
	}
	script, err := parseCharacterCreationScript(questionData)
	if err != nil {
		return nil, err
	}

	assets := &CharacterCreationAssets{
		Oracle:        images[0],
		Incense:       images[1],
		VirtueSymbols: [characterCreationVirtues]*ebiten.Image{},
		Shrine:        images[10],
		Glyphs:        glyphs,
		GlyphWidths:   glyphWidths,
		Script:        script,
	}
	copy(assets.VirtueSymbols[:], images[2:10])

	return assets, nil
}

func characterCreationShapes(archive legacy.Archive) ([]*ebiten.Image, error) {
	if archive.Spec.Name != "CREATE.16" || archive.Spec.Format != legacy.FormatShape4 ||
		len(archive.Entries) != characterCreationShapeCount {
		return nil, fmt.Errorf(
			"%w: CREATE.16 has an unexpected format or entry count",
			errCharacterCreationAssetsInvalid,
		)
	}
	images := make([]*ebiten.Image, characterCreationShapeCount)
	for index, entry := range archive.Entries {
		expectedSize := characterCreationShapeSizes[index]
		if err := validateCharacterCreationShape(entry, index, expectedSize); err != nil {
			return nil, err
		}
		images[index] = ebiten.NewImageFromImage(entry.Image)
	}

	return images, nil
}

func validateCharacterCreationShape(entry legacy.Entry, index int, expectedSize image.Point) error {
	if entry.Missing || entry.Image == nil || entry.PixelType != legacy.PixelEGA4 {
		return fmt.Errorf(
			"%w: CREATE.16 entry %d is unavailable",
			errCharacterCreationAssetsInvalid,
			index,
		)
	}
	if entry.Width != expectedSize.X || entry.Height != expectedSize.Y ||
		entry.Image.Bounds().Size() != expectedSize {
		return fmt.Errorf(
			"%w: CREATE.16 entry %d has dimensions %dx%d, want %dx%d",
			errCharacterCreationAssetsInvalid,
			index,
			entry.Width,
			entry.Height,
			expectedSize.X,
			expectedSize.Y,
		)
	}

	return nil
}

func characterCreationFont(font legacy.Archive) ([]*ebiten.Image, []int, error) {
	if font.Spec.Name != "PROPORT.PCS" || font.Spec.Format != legacy.FormatFontPCS ||
		len(font.Entries) != len(characterCreationGlyphWidths) {
		return nil, nil, fmt.Errorf(
			"%w: PROPORT.PCS has an unexpected format or entry count",
			errCharacterCreationAssetsInvalid,
		)
	}
	for index, entry := range font.Entries {
		expectedWidth := characterCreationGlyphWidths[index]
		if err := validateCharacterCreationGlyph(entry, index, expectedWidth); err != nil {
			return nil, nil, err
		}
	}

	return narrativeIntroFont(font)
}

func validateCharacterCreationGlyph(entry legacy.Entry, index, expectedWidth int) error {
	valid := !entry.Missing && entry.PixelType == legacy.PixelFont && entry.Width == expectedWidth &&
		entry.Height == characterCreationGlyphSize && entry.StoredWidth == characterCreationGlyphSize
	if expectedWidth == 0 {
		valid = valid && entry.Image == nil
	} else {
		valid = valid && entry.Image != nil
	}
	if !valid {
		return fmt.Errorf(
			"%w: PROPORT.PCS entry %d has invalid glyph geometry",
			errCharacterCreationAssetsInvalid,
			index,
		)
	}

	return nil
}

func parseCharacterCreationScript(data []byte) (CharacterCreationScript, error) {
	if len(data) < characterCreationTextSize {
		return CharacterCreationScript{}, fmt.Errorf(
			"%w: QUESTION.DAT is truncated before byte %d",
			errCharacterCreationAssetsInvalid,
			characterCreationTextSize,
		)
	}
	blocks := make([]string, len(characterCreationTextOffsets))
	for index, offset := range characterCreationTextOffsets {
		limit := characterCreationTextSize
		if index+1 < len(characterCreationTextOffsets) {
			limit = characterCreationTextOffsets[index+1]
		}
		block := data[offset:limit]
		terminator := bytes.IndexByte(block, 0)
		if terminator != len(block)-1 {
			return CharacterCreationScript{}, fmt.Errorf(
				"%w: QUESTION.DAT block %d terminator is not at expected offset %d",
				errCharacterCreationAssetsInvalid,
				index,
				limit-1,
			)
		}
		blocks[index] = string(block[:terminator])
	}

	var dilemmaTexts [CharacterCreationDilemmaCount]string
	copy(dilemmaTexts[:], blocks[2:])

	return NewCharacterCreationScript(blocks[0], blocks[1], dilemmaTexts)
}
