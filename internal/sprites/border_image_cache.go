package sprites

import (
	"container/list"
	"math/bits"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	borderCacheMaxBytes   = 32 * 1024 * 1024
	borderCacheMaxEntries = 128
)

// borderImageCache bounds cache-held standalone texture payload and metadata.
// Include the pinned engine's minimum and power-of-two storage rounding: logical
// rectangle bytes substantially underestimate differently sized frame textures.
// Driver overhead and images still used by callers are outside this budget.
type borderImageCache struct {
	mu         sync.Mutex
	entries    map[string]*list.Element
	recent     list.List
	bytes      int64
	maxBytes   int64
	maxEntries int
}

type borderImageEntry struct {
	key   string
	image *ebiten.Image
	bytes int64
}

func newBorderImageCache(maxBytes int64, maxEntries int) *borderImageCache {
	if maxBytes <= 0 || maxEntries <= 0 {
		panic("border cache requires positive budgets")
	}

	return &borderImageCache{
		mu: sync.Mutex{}, entries: make(map[string]*list.Element), recent: list.List{},
		bytes: 0, maxBytes: maxBytes, maxEntries: maxEntries,
	}
}

// ClearBorderCache drops reusable frame references when an owning application or
// independent fixture retires. Live borrowers remain valid; no shared image is
// deallocated by this operation.
func ClearBorderCache() {
	globalBorderCache.mu.Lock()
	defer globalBorderCache.mu.Unlock()
	clear(globalBorderCache.entries)
	globalBorderCache.recent.Init()
	globalBorderCache.bytes = 0
}

func (c *borderImageCache) get(key string) *ebiten.Image {
	c.mu.Lock()
	defer c.mu.Unlock()
	element := c.entries[key]
	if element == nil {
		return nil
	}
	c.recent.MoveToFront(element)

	return borderCacheValue(element).image
}

func (c *borderImageCache) put(key string, img *ebiten.Image) {
	bounds := img.Bounds()
	const bytesPerPixel = 4
	bytes := int64(borderTextureExtent(bounds.Dx())) * int64(borderTextureExtent(bounds.Dy())) * bytesPerPixel
	c.mu.Lock()
	defer c.mu.Unlock()
	if previous := c.entries[key]; previous != nil {
		c.remove(previous)
	}
	if bytes > c.maxBytes {
		return
	}
	for len(c.entries) >= c.maxEntries || bytes > c.maxBytes-c.bytes {
		c.remove(c.recent.Back())
	}
	c.entries[key] = c.recent.PushFront(borderImageEntry{key: key, image: img, bytes: bytes})
	c.bytes += bytes
}

func borderTextureExtent(size int) int {
	const minimumTextureExtent = 16
	if size <= minimumTextureExtent {
		return minimumTextureExtent
	}

	return 1 << bits.Len(uint(size-1))
}

// remove only drops the cache's strong reference. A Border or scene can still
// be drawing this shared image; disposing it here would blank live UI artwork.
func (c *borderImageCache) remove(element *list.Element) {
	entry := borderCacheValue(element)
	delete(c.entries, entry.key)
	c.recent.Remove(element)
	c.bytes -= entry.bytes
}

func borderCacheValue(element *list.Element) borderImageEntry {
	entry, ok := element.Value.(borderImageEntry)
	if !ok {
		panic("border cache contains a non-image entry")
	}

	return entry
}
