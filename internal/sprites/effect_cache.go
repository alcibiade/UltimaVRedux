package sprites

import (
	"fmt"

	graphicsbackend "github.com/bradhannah/berad-sdk/modules/berad-graphics/backend/ebitengine"
	"github.com/hajimehoshi/ebiten/v2"
)

// DefaultDynamicCacheSize retains the game's variant-entry budget.
const DefaultDynamicCacheSize = 500

// EffectCache adapts the game's complete packed variant keys to an SDK image LRU.
// The rendering owner serializes calls; returned artwork is borrowed read-only.
type EffectCache struct{ cache *graphicsbackend.ImageCache }

// NewEffectCache supplies the game's default for unspecified entry limits.
func NewEffectCache(entries int) *EffectCache {
	if entries <= 0 {
		entries = DefaultDynamicCacheSize
	}
	const cacheBytes = 32 << 20
	cache, err := graphicsbackend.NewImageCache(entries, cacheBytes)
	if err != nil {
		panic(fmt.Sprintf("configure game variant cache: %v", err))
	}

	return &EffectCache{cache: cache}
}

// Get borrows an existing image for this game variant.
func (cache *EffectCache) Get(key uint64) *ebiten.Image { return cache.cache.Get(key) }

// Put retains a valid game-created image within the configured resource budget.
func (cache *EffectCache) Put(key uint64, image *ebiten.Image) {
	if err := cache.cache.Put(key, image); err != nil {
		panic(fmt.Sprintf("cache game variant: %v", err))
	}
}
