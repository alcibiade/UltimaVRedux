package text

import (
	_ "embed"
	"fmt"
	"sync"

	beradtext "github.com/bradhannah/berad-sdk/modules/berad-text"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/bradhannah/UltimaVRedux/internal/profiling"
)

//go:embed assets/ultima-v-warriors-of-destiny/ultima-v-warriors-of-destiny.ttf
var rawUltimaTTF []byte

//nolint:gochecknoglobals // Game-owned embedded data; SDK parsing is shared read-only.
var ultimaSDKSource = sync.OnceValue(func() *beradtext.Source {
	profiling.Count(profiling.FontSourceParses)
	source, err := beradtext.ParseFont(rawUltimaTTF)
	if err != nil {
		panic(fmt.Sprintf("parsing mandatory Ultima font: %v", err))
	}

	return source
})

//nolint:gochecknoglobals // Existing engine callers share the SDK's read-only source.
var ultimaFontSource = sync.OnceValue(func() *etext.GoTextFaceSource {
	font, err := ultimaSDKSource().NewFont(beradtext.FontOptions{
		Size: 1, Language: language.English, Features: nil, Variations: nil,
	})
	if err != nil {
		panic(fmt.Sprintf("creating mandatory font face: %v", err))
	}

	return font.CopyEngineFace().Source
})

// UltimaFont adapts game typography and the legacy engine view to Berad Text.
type UltimaFont struct {
	TextFace    *etext.GoTextFace
	sdkFont     *beradtext.Font
	scaledFont  *beradtext.Font
	sdkLanguage language.Tag
}

// NewUltimaFont creates independently sized typography sharing the game source.
func NewUltimaFont(size float64) *UltimaFont {
	profiling.Count(profiling.FontCreates)
	font := new(UltimaFont)
	font.TextFace = new(etext.GoTextFace)
	font.TextFace.Source = ultimaFontSource()
	font.TextFace.Size = size
	font.TextFace.Language = language.English

	return font
}

// MeasureString uses the same immutable SDK configuration as rendering.
func (f *UltimaFont) MeasureString(value string) (float64, float64) {
	width, height, err := f.sdkForScale(1).Measure(value, 0)
	if err != nil {
		panic(fmt.Sprintf("measuring game text: %v", err))
	}

	return width, height
}

// Layout supplies immutable SDK geometry to host-owned editors and selectors.
func (f *UltimaFont) Layout(value string) *beradtext.Layout {
	return newTextLayout(value, f.sdkForScale(1))
}

// Metrics returns the same typography used for rendering, not a second TTF face.
func (f *UltimaFont) Metrics() beradtext.Metrics {
	return f.sdkForScale(1).Metrics()
}

// sdkForScale converts legacy size/language settings to immutable SDK typography.
// TextFace is a host compatibility view; its source/features remain read-only.
func (f *UltimaFont) sdkForScale(scale float64) *beradtext.Font {
	if f.sdkFont == nil || f.sdkFont.Size() != f.TextFace.Size || f.sdkLanguage != f.TextFace.Language {
		font, err := ultimaSDKSource().NewFont(beradtext.FontOptions{
			Size: f.TextFace.Size, Language: f.TextFace.Language, Features: nil, Variations: nil,
		})
		if err != nil {
			panic(fmt.Sprintf("configuring game font: %v", err))
		}
		f.sdkFont, f.sdkLanguage, f.scaledFont = font, f.TextFace.Language, nil
	}
	if scale == 1 {
		return f.sdkFont
	}
	size := f.TextFace.Size * scale
	if f.scaledFont == nil || f.scaledFont.Size() != size {
		font, err := f.sdkFont.WithSize(size)
		if err != nil {
			panic(fmt.Sprintf("scaling game font: %v", err))
		}
		f.scaledFont = font
	}

	return f.scaledFont
}

// CharsForWidth preserves the game's conservative legacy column convention.
// New proportional-layout consumers use the SDK's measured wrapping instead.
func (f *UltimaFont) CharsForWidth(pixelWidth int) int {
	const minimumLegacyColumns = 10
	width, _ := f.MeasureString("M")
	if width <= 0 {
		return minimumLegacyColumns
	}

	return max(minimumLegacyColumns, int(float64(pixelWidth)/width))
}
