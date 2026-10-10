package text

import (
	"fmt"
	"image/color"
	"testing"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
)

func TestTextCacheUsesValueKeysAndDetectsPublicFaceChanges(t *testing.T) {
	face := &etext.GoTextFace{Size: 20}
	params := TextRenderParams{Content: "Strength", Face: face,
		Options: &etext.LayoutOptions{LineSpacing: 24}, Color: color.RGBA64{R: 1, G: 0x23, B: 4, A: 0xffff}}
	cache := NewTextCache(100)
	first := preseedTextCache(cache, params)
	t.Cleanup(cache.Clear)
	params.Options = &etext.LayoutOptions{LineSpacing: 24}
	require.Same(t, first, cache.GetOrRender(params), "equivalent layout values at different addresses must hit")
	require.Zero(t, testing.AllocsPerRun(100, func() { cache.GetOrRender(params) }), "warm keys must not allocate formatting strings")
	before := params.CacheKey()
	params.Color = color.RGBA64{R: 0x12, G: 3, B: 4, A: 0xffff}
	require.NotEqual(t, before, params.CacheKey(), "numeric colours must not collide")
	require.NotContains(t, cache.cache, params.CacheKey())
	before = params.CacheKey()
	face.Size = 31.5
	require.NotEqual(t, before, params.CacheKey())
	require.NotContains(t, cache.cache, params.CacheKey())
}

func TestTextCacheBudgetEvictsAndBypassesOversizedResults(t *testing.T) {
	font := NewUltimaFont(2)
	params := TextRenderParams{Content: "One", Face: font.TextFace,
		Options: &etext.LayoutOptions{LineSpacing: 2}, Color: color.White}
	probe := NewTextCache(10)
	t.Cleanup(probe.Clear)
	probe.GetOrRender(params)
	budget := probe.bytes * 2
	cache := NewTextCacheWithBudget(2, budget)
	t.Cleanup(cache.Clear)
	for _, value := range []string{"One", "Two", "Three", "Four"} {
		params.Content = value
		cache.GetOrRender(params)
		require.LessOrEqual(t, cache.bytes, budget)
		require.LessOrEqual(t, cache.Size(), 2)
		require.Equal(t, cache.Size(), cache.lruList.Len())
	}
	oldest := params
	oldest.Content = "One"
	require.NotContains(t, cache.cache, oldest.CacheKey(), "repeated misses must evict old entries")
	latest := cache.GetOrRender(params)
	before := cache.Size()
	params.Content = "An oversized line must not evict every useful label in the working set."
	cache.GetOrRender(params)
	require.Equal(t, before, cache.Size())
	params.Content = "Four"
	require.Same(t, latest, cache.GetOrRender(params), "a bypass must preserve cached image ownership")
	require.LessOrEqual(t, cache.peakBytes, budget)
	cache.Clear()
	require.Zero(t, cache.bytes)
	require.Zero(t, cache.Size())
	require.Panics(t, func() { NewTextCacheWithBudget(0, 100) })
	require.Panics(t, func() { NewTextCacheWithBudget(1, 0) })
}

func TestOutputLineGeometryTracksOnlyContentAndTypography(t *testing.T) {
	output := NewOutput(NewUltimaFont(20), 24, 3, 100)
	segments := []ColorSegment{{Text: "Strength: ", Color: color.White}, {Text: "20", Color: color.Black}}
	first := output.layoutLine(0, segments)
	advances := &first.advances[0]
	width := first.total
	segments[0].Color = color.RGBA{R: 255, A: 255}
	require.Same(t, advances, &output.layoutLine(0, segments).advances[0])
	require.Equal(t, width, output.lineLayouts[0].total, "recolouring must not invalidate geometry")
	segments[1].Text = "2000"
	require.Greater(t, output.layoutLine(0, segments).total, width)
	output.Font.TextFace.Size = 31.5
	require.Greater(t, output.layoutLine(0, segments).total, width)
	copy := output.WrappedCopy(20)
	require.Nil(t, copy.lineLayouts, "wrapped copies must own independent mutable layout storage")
	output.Reset()
	require.False(t, output.lineLayouts[0].valid)
	require.Empty(t, output.lineLayouts[0].inputs[0].text)
}

func TestUnwrappedStyledFastPathKeepsCallerIsolation(t *testing.T) {
	output := NewOutput(NewUltimaFont(20), 24, 2, 8)
	segments := []ColorSegment{{Text: "éé", Color: color.White}, {Text: " x", Color: color.Black, FontType: FontTypeIBM}}
	output.AddStyledText(segments, etext.AlignEnd)
	require.Equal(t, segments, output.lines[0].Segments)
	require.Equal(t, etext.AlignEnd, output.lines[0].Alignment)
	segments[0].Text = "mutated"
	require.Equal(t, "éé", output.lines[0].Segments[0].Text)
}

func BenchmarkTextCacheKeyLookup(b *testing.B) {
	params := TextRenderParams{Content: "Strength: 20", Face: &etext.GoTextFace{Size: 20},
		Options: &etext.LayoutOptions{LineSpacing: 24}, Color: color.White}
	b.Run("structured_hit", func(b *testing.B) {
		cache := NewTextCache(100)
		preseedTextCache(cache, params)
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			cache.GetOrRender(params)
		}
	})
	b.Run("previous_formatted_key", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			r, g, blue, a := params.Color.RGBA()
			colour := fmt.Sprintf("%x%x%x%x", r, g, blue, a)
			key := fmt.Sprintf("text_%s_%p_%+v_%s", params.Content, params.Face, params.Options, colour)
			if key == "" {
				b.Fatal("missing key")
			}
		}
	})
}
