package text

import (
	"bytes"
	"sync"
	"testing"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func TestUltimaFontsShareSourceButNotMutableFaces(t *testing.T) {
	first, second := NewUltimaFont(20), NewUltimaFont(20)
	require.NotSame(t, first, second)
	require.NotSame(t, first.TextFace, second.TextFace)
	require.Same(t, first.TextFace.Source, second.TextFace.Source)
	first.TextFace.Size = 31.5
	first.TextFace.Language = language.French
	first.TextFace.Direction = etext.DirectionRightToLeft
	first.TextFace.SetFeature(etext.MustParseTag("kern"), 0)
	require.Equal(t, float64(20), second.TextFace.Size)
	require.Equal(t, language.English, second.TextFace.Language)
	require.Equal(t, etext.DirectionLeftToRight, second.TextFace.Direction)
	otherSize := NewUltimaFont(14.25)
	require.Same(t, second.TextFace.Source, otherSize.TextFace.Source)
}

func TestSharedSourcePreservesMetricsUnderConcurrentUse(t *testing.T) {
	independent, err := etext.NewGoTextFaceSource(bytes.NewReader(rawUltimaTTF))
	require.NoError(t, err)
	for _, size := range []float64{12, 14.25, 20, 23, 31.5} {
		shared := NewUltimaFont(size)
		fresh := &etext.GoTextFace{Source: independent, Size: size, Language: language.English}
		for _, value := range []string{"Strength: 20", "Tab Next character", "Mixed punctuation -- 999/1000HP", ""} {
			width, height := shared.MeasureString(value)
			wantWidth, wantHeight := etext.Measure(value, fresh, 0)
			require.Equal(t, wantWidth, width)
			require.Equal(t, wantHeight, height)
		}
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 25 {
				font := NewUltimaFont(23)
				width, _ := font.MeasureString("Compassion")
				if width <= 0 || font.TextFace.Source != ultimaFontSource() {
					t.Error("independent faces must safely share parsed source and measurements")
				}
			}
		})
	}
	workers.Wait()
}

func BenchmarkUltimaFontConstruction(b *testing.B) {
	b.Run("shared", func(b *testing.B) {
		_ = ultimaFontSource()
		b.ReportAllocs()
		for range b.N {
			font := NewUltimaFont(23)
			if font.TextFace.Source == nil {
				b.Fatal("missing source")
			}
		}
	})
	b.Run("independent_parse", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := etext.NewGoTextFaceSource(bytes.NewReader(rawUltimaTTF))
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
