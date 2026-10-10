package text

import (
	"container/list"
	"image"
	"image/color"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/profiling"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

const (
	// DefaultMaxCacheEntries is the default maximum number of cached text images
	DefaultMaxCacheEntries = 512
	// DefaultMaxCacheBytes bounds cache-held pixel payload and owned text keys.
	DefaultMaxCacheBytes = 16 * 1024 * 1024

	// paddingPixels is the additional padding added to text images for safety
	paddingPixels = 2
)

// TextRenderParams represents parameters for text rendering that can be cached
type TextRenderParams struct {
	Content string
	Face    text.Face
	Options *text.LayoutOptions
	Color   color.Color
}

// CacheKey is a comparable rendering identity; its fields remain cache-owned.
type CacheKey struct {
	content   string
	face      text.Face // Strong identity prevents recycled pointer addresses from aliasing old entries.
	source    *text.GoTextFaceSource
	size      float64
	direction text.Direction
	language  language.Tag
	script    language.Script
	layout    text.LayoutOptions
	rgba      [4]uint32
	colorSet  bool
}

// CacheKey uses numeric colours and value layout options without formatting.
// Faces must keep opaque feature/variation settings immutable while cached;
// replace the face or ClearTextCache after changing those engine-private settings.
func (t TextRenderParams) CacheKey() CacheKey {
	var key CacheKey
	key.content, key.face, key.layout = t.Content, t.Face, *t.Options
	if face, ok := t.Face.(*text.GoTextFace); ok {
		key.source, key.size, key.direction = face.Source, face.Size, face.Direction
		//nolint:staticcheck // Retain the legacy Script hint in cache identity to avoid stale layouts.
		key.language, key.script = face.Language, face.Script
	}
	if t.Color != nil {
		key.colorSet = true
		key.rgba[0], key.rgba[1], key.rgba[2], key.rgba[3] = t.Color.RGBA()
	}

	return key
}

// CacheEntry represents a single cached text image with LRU tracking
type CacheEntry struct {
	key           CacheKey
	image         *ebiten.Image
	width, height float64 // Unpadded measurements, safe for styled-segment advances.
	bytes         int
	listElement   *list.Element // Position in LRU list
}

// TextCache manages cached rendered text images with LRU eviction
// This cache is designed to be reusable for other graphics (tiles, sprites) in the future
type TextCache struct {
	cache                      map[CacheKey]*CacheEntry
	lruList                    *list.List // Front = most recently used, Back = least recently used
	maxEntries                 int
	maxBytes, bytes, peakBytes int
}

var globalCache = NewTextCache(DefaultMaxCacheEntries)

// NewTextCache creates a new cache with specified maximum entries
func NewTextCache(maxEntries int) *TextCache {
	return NewTextCacheWithBudget(maxEntries, DefaultMaxCacheBytes)
}

// NewTextCacheWithBudget bounds entries and accounted bytes, not total GPU memory.
func NewTextCacheWithBudget(maxEntries, maxBytes int) *TextCache {
	if maxEntries <= 0 || maxBytes <= 0 {
		panic("text cache requires positive entry and byte budgets")
	}
	return &TextCache{
		cache:      make(map[CacheKey]*CacheEntry),
		lruList:    list.New(),
		maxEntries: maxEntries,
		maxBytes:   maxBytes, bytes: 0, peakBytes: 0,
	}
}

// GetCachedText returns a cached text image or renders and caches a new one
func GetCachedText(content string, face text.Face, options *text.LayoutOptions, textColor color.Color) *ebiten.Image {
	params := TextRenderParams{
		Content: content,
		Face:    face,
		Options: options,
		Color:   textColor,
	}

	return globalCache.GetOrRender(params)
}

// ClearTextCache clears all cached text images (e.g., on resolution change)
func ClearTextCache() {
	globalCache.Clear()
}

// GetOrRender returns cached image or renders new one, updating LRU order
func (tc *TextCache) GetOrRender(params TextRenderParams) *ebiten.Image {
	return tc.getOrRenderEntry(params).image
}

func (tc *TextCache) getOrRenderEntry(params TextRenderParams) *CacheEntry {
	key := params.CacheKey()

	// Check cache hit
	if entry, ok := tc.cache[key]; ok {
		profiling.Count(profiling.TextHits)
		// Move to front of LRU list (most recently used)
		tc.lruList.MoveToFront(entry.listElement)

		return entry
	}

	// Cache miss - render new image
	profiling.Count(profiling.TextMisses)
	width, height := text.Measure(params.Content, params.Face, params.Options.LineSpacing)
	img := tc.renderText(params, width, height)
	const bytesPerPixel = 4
	bytes := int64(img.Bounds().Dx())*int64(img.Bounds().Dy())*bytesPerPixel + int64(len(params.Content))
	entry := &CacheEntry{
		key: key, image: img, width: width, height: height,
		bytes: 0, listElement: nil,
	}
	if bytes > int64(tc.maxBytes) {
		profiling.Count(profiling.TextCacheBypasses)

		return entry // Oversized results must not evict the entire useful working set.
	}
	entry.bytes = int(bytes)

	// Evict LRU if cache is full
	for len(tc.cache) >= tc.maxEntries || entry.bytes > tc.maxBytes-tc.bytes {
		tc.evictLRU()
	}

	// Add new entry to cache and LRU list
	key.content = strings.Clone(key.content) // Do not retain a large substring's backing storage.
	entry.key = key
	element := tc.lruList.PushFront(entry)
	entry.listElement = element
	tc.cache[key] = entry
	tc.bytes += entry.bytes
	tc.peakBytes = max(tc.peakBytes, tc.bytes)
	tc.publishAccounting()

	return entry
}

// Clear removes all entries from the cache
func (tc *TextCache) Clear() {
	clear(tc.cache)
	tc.lruList.Init()
	tc.bytes = 0
	tc.publishAccounting()
}

// evictLRU removes the least recently used entry from the cache
func (tc *TextCache) evictLRU() {
	element := tc.lruList.Back()
	if element != nil {
		profiling.Count(profiling.TextEvictions)
		tc.lruList.Remove(element)
		entry, ok := element.Value.(*CacheEntry)
		if !ok {
			panic("text cache LRU contains a non-text entry")
		}
		delete(tc.cache, entry.key)
		tc.bytes -= entry.bytes
	}
}

// renderText performs the actual text rendering to an image
func (tc *TextCache) renderText(params TextRenderParams, textWidth, textHeight float64) *ebiten.Image {
	// Create image of appropriate size (add small padding for safety)
	width := int(textWidth) + paddingPixels
	height := int(textHeight) + paddingPixels

	// Cache accounting bounds these composed images themselves. Keeping them off
	// shared atlases prevents a tiny persistent text entry from retaining a much
	// larger atlas grown by transient full-screen targets during resolution changes.
	img := ebiten.NewImageWithOptions(image.Rect(0, 0, width, height), &ebiten.NewImageOptions{Unmanaged: true})

	// Render text to image with color applied
	// The alignment will be handled by the caller when drawing the cached image
	// We must NOT apply alignment here as text.Draw already handles it from LayoutOptions
	dop := &text.DrawOptions{
		LayoutOptions: *params.Options,
	}
	if params.Color != nil {
		dop.ColorScale.ScaleWithColor(params.Color)
	} else {
		// Use white as default color when none specified
		dop.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
	}
	text.Draw(img, params.Content, params.Face, dop)

	return img
}

// Size returns the current number of cached entries (useful for testing/debugging)
func (tc *TextCache) Size() int {
	return len(tc.cache)
}

func (tc *TextCache) publishAccounting() {
	if tc.bytes < 0 || tc.peakBytes < 0 {
		panic("text cache accounting cannot be negative")
	}
	if tc == globalCache {
		profiling.SetGauge(profiling.TextCacheEntries, uint64(len(tc.cache)))
		profiling.SetGauge(profiling.TextCacheBytes, uint64(tc.bytes))
		profiling.SetGauge(profiling.TextCachePeakBytes, uint64(tc.peakBytes))
	}
}
