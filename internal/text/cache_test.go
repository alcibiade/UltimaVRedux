package text

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Keep real rendering for cache insertion/ownership checks, with small glyphs.
func createMockFont() text.Face {
	return NewUltimaFont(2).TextFace
}

func createMockLayoutOptions() *text.LayoutOptions {
	return &text.LayoutOptions{LineSpacing: 1, PrimaryAlign: text.AlignStart}
}

// Identity-only hits borrow a sentinel and never invoke its rendering methods.
func preseedTextCache(cache *TextCache, params TextRenderParams) *ebiten.Image {
	entry := &CacheEntry{key: params.CacheKey(), image: &ebiten.Image{}}
	entry.listElement = cache.lruList.PushFront(entry)
	cache.cache[entry.key] = entry
	return entry.image
}

func TestTextRenderParams_CacheKey(t *testing.T) {
	// Key identity does not require a font source or rendered glyphs.
	font := &text.GoTextFace{Size: 12}
	params := TextRenderParams{Content: "Hello World", Face: font, Options: createMockLayoutOptions(), Color: color.White}
	identical := params
	identical.Options = createMockLayoutOptions()
	assert.Equal(t, params.CacheKey(), identical.CacheKey())
	different := params
	different.Content = "Different Text"
	assert.NotEqual(t, params.CacheKey(), different.CacheKey())
	different = params
	different.Color = color.RGBA{255, 255, 0, 255}
	assert.NotEqual(t, params.CacheKey(), different.CacheKey())
}

func TestNewTextCache(t *testing.T) {
	cache := NewTextCache(50)
	assert.NotNil(t, cache)
	assert.Zero(t, cache.Size())
	assert.Equal(t, 50, cache.maxEntries)
}

func TestTextCache_CacheHit(t *testing.T) {
	cache := NewTextCache(10)
	t.Cleanup(cache.Clear)
	params := TextRenderParams{Face: createMockFont(), Options: createMockLayoutOptions(), Color: color.White}
	for _, content := range []string{"A", "B", ""} {
		params.Content = content
		before := cache.Size()
		first := cache.GetOrRender(params)
		require.NotNil(t, first)
		width, height := text.Measure(content, params.Face, params.Options.LineSpacing)
		entry := cache.getOrRenderEntry(params)
		require.Equal(t, width, entry.width)
		require.Equal(t, height, entry.height)
		require.Equal(t, before+1, cache.Size(), "distinct content, including empty text, must miss")
		require.Same(t, first, cache.GetOrRender(params))
		require.Equal(t, before+1, cache.Size())
		require.Equal(t, params.CacheKey(), cache.lruList.Front().Value.(*CacheEntry).key)
	}
}

func TestTextCache_LRUEviction(t *testing.T) {
	cache := NewTextCache(3)
	t.Cleanup(cache.Clear)
	font, options := &text.GoTextFace{Size: 12}, createMockLayoutOptions()
	paramsA := TextRenderParams{Content: "A", Face: font, Options: options}
	paramsB := TextRenderParams{Content: "B", Face: font, Options: options}
	paramsC := TextRenderParams{Content: "C", Face: font, Options: options}
	imgA := preseedTextCache(cache, paramsA)
	preseedTextCache(cache, paramsB)
	imgC := preseedTextCache(cache, paramsC)
	require.Equal(t, 3, cache.Size())
	cache.GetOrRender(paramsA)
	require.Equal(t, paramsA.CacheKey(), cache.lruList.Front().Value.(*CacheEntry).key)
	require.Equal(t, paramsB.CacheKey(), cache.lruList.Back().Value.(*CacheEntry).key)
	// Exercise hit promotion and the cache's eviction primitive without glyphs.
	// Native insertion enforcing entry/byte limits is covered by budget tests.
	cache.evictLRU()
	require.Equal(t, 2, cache.Size())
	require.NotContains(t, cache.cache, paramsB.CacheKey())
	paramsD := TextRenderParams{Content: "D", Face: font, Options: options}
	imgD := preseedTextCache(cache, paramsD)
	require.Equal(t, 3, cache.Size())
	assert.Same(t, imgA, cache.GetOrRender(paramsA))
	assert.Same(t, imgC, cache.GetOrRender(paramsC))
	assert.Same(t, imgD, cache.GetOrRender(paramsD))
	cache.evictLRU()
	require.NotContains(t, cache.cache, paramsA.CacheKey(), "the latest hit ordering must select A next")
	require.Equal(t, 2, cache.lruList.Len())
}

func TestTextCache_Clear(t *testing.T) {
	cache := NewTextCache(10)
	t.Cleanup(cache.Clear)
	font, options := createMockFont(), createMockLayoutOptions()
	for index := range 5 {
		cache.GetOrRender(TextRenderParams{Content: fmt.Sprint(index), Face: font, Options: options})
	}
	require.Equal(t, 5, cache.Size())
	cache.Clear()
	assert.Zero(t, cache.Size())
	params := TextRenderParams{Content: "A", Face: font, Options: options}
	first := cache.GetOrRender(params)
	cache.Clear()
	assert.NotSame(t, first, cache.GetOrRender(params))
}

func TestGetCachedText_GlobalCache(t *testing.T) {
	ClearTextCache()
	t.Cleanup(ClearTextCache)
	font, options := createMockFont(), createMockLayoutOptions()
	first := GetCachedText("A", font, options, color.White)
	assert.Same(t, first, GetCachedText("A", font, options, color.White))
	ClearTextCache()
	assert.NotSame(t, first, GetCachedText("A", font, options, color.White))
}

func TestTextCache_SingleEntry(t *testing.T) {
	cache := NewTextCache(1)
	t.Cleanup(cache.Clear)
	font, options := createMockFont(), createMockLayoutOptions()
	paramsA := TextRenderParams{Content: "A", Face: font, Options: options}
	paramsB := TextRenderParams{Content: "B", Face: font, Options: options}
	first := cache.GetOrRender(paramsA)
	assert.Equal(t, 1, cache.Size())
	second := cache.GetOrRender(paramsB)
	assert.Equal(t, 1, cache.Size())
	assert.NotSame(t, first, cache.GetOrRender(paramsA))
	assert.Equal(t, 1, cache.Size())
	assert.NotSame(t, second, cache.GetOrRender(paramsB))
	assert.Equal(t, 1, cache.Size())
}
