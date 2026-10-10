package text

import (
	"bytes"
	_ "embed"
	"fmt"
	"sync"

	"github.com/bradhannah/UltimaVRedux/internal/profiling"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

//go:embed assets/ultima-v-warriors-of-destiny/ultima-v-warriors-of-destiny.ttf
var rawUltimaTTF []byte

//nolint:gochecknoglobals // Immutable embedded source shares parsing/glyph caches across independent faces.
var ultimaFontSource = sync.OnceValue(func() *text.GoTextFaceSource {
	profiling.Count(profiling.FontSourceParses)
	source, err := text.NewGoTextFaceSource(bytes.NewReader(rawUltimaTTF))
	if err != nil {
		panic(fmt.Sprintf("creating face source from embedded Ultima font: %v", err))
	}

	return source
})

type UltimaFont struct {
	rawUltimaTTF     *[]byte
	ultimaFaceSource *text.GoTextFaceSource
	TextFace         *text.GoTextFace
}

func NewUltimaFont(size float64) *UltimaFont {
	profiling.Count(profiling.FontCreates)
	ultimaFont := UltimaFont{}
	ultimaFont.rawUltimaTTF = &rawUltimaTTF

	ultimaFont.ultimaFaceSource = ultimaFontSource()

	ultimaFont.TextFace = &text.GoTextFace{
		Source:    ultimaFont.ultimaFaceSource,
		Direction: text.DirectionLeftToRight,
		Size:      size,
		Language:  language.English,
	}

	return &ultimaFont
}

// MeasureString returns the width and height of the given text string in pixels.
// This is a convenience wrapper around text.Measure for use in dialog sizing.
func (f *UltimaFont) MeasureString(str string) (width, height float64) {
	return text.Measure(str, f.TextFace, 0)
}

// CharsForWidth calculates how many characters fit in a given pixel width.
// Uses 'M' as the reference character (em-width) for conservative estimation.
// Returns a minimum of 10 to prevent pathological edge cases.
func (f *UltimaFont) CharsForWidth(pixelWidth int) int {
	avgCharWidth, _ := f.MeasureString("M")
	if avgCharWidth <= 0 {
		return 10 // Fallback for invalid font measurement
	}
	maxChars := int(float64(pixelWidth) / avgCharWidth)
	if maxChars < 10 {
		maxChars = 10
	}

	return maxChars
}
