// Package sprites provides the LRU cache for dynamic visual effects.
package sprites

import (
	"container/list"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// DefaultDynamicCacheSize is the default maximum number of cached effect images.
const DefaultDynamicCacheSize = 500

// EffectCache provides an LRU cache for dynamically generated effect images.
// This prevents regenerating identical effect combinations each frame.
type EffectCache struct {
	maxSize int
	cache   map[uint64]*cachedEntry
	lruList *list.List
	mu      sync.RWMutex
}

type cachedEntry struct {
	key     uint64
	image   *ebiten.Image
	element *list.Element
}

// NewEffectCache creates a new LRU cache with the specified maximum size.
func NewEffectCache(maxSize int) *EffectCache {
	if maxSize <= 0 {
		maxSize = DefaultDynamicCacheSize
	}

	return &EffectCache{
		maxSize: maxSize,
		cache:   make(map[uint64]*cachedEntry),
		lruList: list.New(),
	}
}

// Get retrieves a cached effect image by key.
// Returns nil if not found. Moves the entry to the front of the LRU list.
func (c *EffectCache) Get(key uint64) *ebiten.Image {
	c.mu.RLock()
	entry, exists := c.cache[key]
	c.mu.RUnlock()

	if !exists {
		return nil
	}

	// Move to front (most recently used)
	c.mu.Lock()
	c.lruList.MoveToFront(entry.element)
	c.mu.Unlock()

	return entry.image
}

// Put stores an effect image in the cache.
// If the cache is full, evicts the least recently used entry.
func (c *EffectCache) Put(key uint64, img *ebiten.Image) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if entry, exists := c.cache[key]; exists {
		// Update existing entry
		entry.image = img
		c.lruList.MoveToFront(entry.element)

		return
	}

	// Evict if at capacity
	for len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	// Add new entry
	entry := &cachedEntry{
		key:   key,
		image: img,
	}
	entry.element = c.lruList.PushFront(entry)
	c.cache[key] = entry
}

// evictOldest removes the least recently used entry.
// Must be called with lock held.
func (c *EffectCache) evictOldest() {
	oldest := c.lruList.Back()
	if oldest == nil {
		return
	}

	entry, ok := oldest.Value.(*cachedEntry)
	if !ok {
		panic("effect cache LRU contains a non-effect entry")
	}
	c.lruList.Remove(oldest)
	delete(c.cache, entry.key)
}

// Size returns the current number of cached entries.
func (c *EffectCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// Clear removes all entries from the cache.
func (c *EffectCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[uint64]*cachedEntry)
	c.lruList.Init()
}
