package sprites

import (
	_ "embed"
	"fmt"
	"image"

	graphics "github.com/bradhannah/berad-sdk/modules/berad-graphics"
	graphicsbackend "github.com/bradhannah/berad-sdk/modules/berad-graphics/backend/ebitengine"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// SpriteSheet owns original atlas identities and game-selected artwork over SDK resources.
type SpriteSheet struct {
	// Populate custom atlases before the first lookup; cached images are read-only.
	SpriteImage *ebiten.Image

	rawSpriteCache      [totalSprites]*ebiten.Image
	transparentRaw      *ebiten.Image
	transparentBordered *ebiten.Image

	spriteImageCache [1024]*ebiten.Image

	// Hybrid border cache - pre-generated 18x18 bordered sprites for static tiles
	staticBorderedCache [1024]*ebiten.Image

	// Tile effect processor for flames, lava, water, and mixed land/water effects
	tileEffectProcessor *TileEffectProcessor

	backgroundCache *EffectCache

	// TileEffectsEnabled controls whether tiles with registered effects use visual animation.
	// When true, flame pixels flicker (original U5 style) and lava waves.
	TileEffectsEnabled bool
}

// Original atlas dimensions and tile-grid policy remain game-owned.
const (
	origTilesWidth  = 512
	origTilesHeight = 256
	TileSize        = 16
	spritesPerRow   = 32
	spriteRows      = 16
	totalSprites    = spriteRows * spritesPerRow
)

//go:embed assets/transparent_u5_tiles.png
var transparentTiles []byte

// TileAtlasPNG returns a detached copy of the embedded sprite atlas for local developer tooling.
func TileAtlasPNG() []byte {
	return append([]byte(nil), transparentTiles...)
}

// NewSpriteSheet loads the bundled game atlas and configures its presentation adapters.
func NewSpriteSheet() *SpriteSheet {
	spriteSheet := &SpriteSheet{
		SpriteImage:         nil,
		transparentRaw:      nil,
		transparentBordered: nil,
		rawSpriteCache:      [totalSprites]*ebiten.Image{},
		spriteImageCache:    [1024]*ebiten.Image{},
		staticBorderedCache: [1024]*ebiten.Image{},
		tileEffectProcessor: nil,
		backgroundCache:     NewEffectCache(DefaultDynamicCacheSize),
		TileEffectsEnabled:  true,
	}

	spriteSheet.SpriteImage = NewPngSprite(transparentTiles) // Use transparent tiles for proper alpha blending

	// Initialize tile effect processor for flames, lava, water, and mixed land/water effects
	spriteSheet.tileEffectProcessor = NewTileEffectProcessor(spriteSheet)

	// Pre-generate bordered sprites for static tiles
	spriteSheet.preGenerateStaticBorders()

	return spriteSheet
}

// preGenerateStaticBorders pre-generates 18x18 bordered sprites for static (non-animated) tiles
func (s *SpriteSheet) preGenerateStaticBorders() {
	for sprite := indexes.SpriteIndex(0); sprite < totalSprites; sprite++ {
		if s.isStaticSprite(sprite) {
			s.staticBorderedCache[sprite] = s.copySpriteRegion(sprite, graphics.Insets{Left: 1, Top: 1, Right: 1, Bottom: 1})
		}
	}
}

func (s *SpriteSheet) copySpriteRegion(sprite indexes.SpriteIndex, padding graphics.Insets) *ebiten.Image {
	result, err := graphicsbackend.CopyRegion(s.SpriteImage, s.getSpriteImageRectangle(sprite), padding)
	if err != nil {
		panic(fmt.Sprintf("copy original atlas region: %v", err))
	}

	return result
}

func (s *SpriteSheet) getSpriteImageRectangle(spriteIndex indexes.SpriteIndex) image.Rectangle {
	nSprite := int(spriteIndex)
	topLeft := image.Point{X: (nSprite % spritesPerRow) * TileSize, Y: (nSprite / spritesPerRow) * TileSize}
	bottomRight := image.Point{
		X: topLeft.X + TileSize,
		Y: topLeft.Y + TileSize,
	}

	return image.Rectangle{
		Min: topLeft,
		Max: bottomRight,
	}
}

// GetSprite returns borrowed, read-only bordered artwork, including the shared
// transparent fallback. Callers must copy before modifying and must not dispose it.
func (s *SpriteSheet) GetSprite(nSprite indexes.SpriteIndex) *ebiten.Image {
	if nSprite < 0 || int(nSprite) >= totalSprites {
		if s.transparentBordered == nil {
			s.transparentBordered = ebiten.NewImage(ExpandedTileSize, ExpandedTileSize)
		}

		return s.transparentBordered
	}

	// For static sprites, use pre-generated bordered cache
	if s.isStaticSprite(nSprite) {
		if s.staticBorderedCache[nSprite] != nil {
			return s.staticBorderedCache[nSprite]
		}
		// Fallback if pre-generation failed
		sprite18 := s.copySpriteRegion(nSprite, graphics.Insets{Left: 1, Top: 1, Right: 1, Bottom: 1})
		s.staticBorderedCache[nSprite] = sprite18

		return sprite18
	}

	// For dynamic sprites, use original lazy-loading approach
	if s.spriteImageCache[nSprite] == nil {
		sprite18 := s.copySpriteRegion(nSprite, graphics.Insets{Left: 1, Top: 1, Right: 1, Bottom: 1})
		s.spriteImageCache[nSprite] = sprite18

		return sprite18
	}

	return s.spriteImageCache[nSprite]
}

// GetRawSprite returns a borrowed, read-only, zero-origin 16x16 tile.
// Callers must copy it before modifying pixels and must not dispose cached images.
// Invalid indices share a sheet-owned transparent fallback.
func (s *SpriteSheet) GetRawSprite(sprite indexes.SpriteIndex) *ebiten.Image {
	if int(sprite) < 0 || int(sprite) >= totalSprites {
		if s.transparentRaw == nil {
			s.transparentRaw = ebiten.NewImage(TileSize, TileSize)
		}

		return s.transparentRaw
	}

	return s.getOriginal16x16Sprite(sprite)
}

// GetRawAnimatedSprite returns a borrowed, read-only deterministic 16x16 frame.
func (s *SpriteSheet) GetRawAnimatedSprite(
	sprite indexes.SpriteIndex,
	positionHash int32,
	elapsedMs int64,
) *ebiten.Image {
	if s.TileEffectsEnabled && s.tileEffectProcessor.HasEffect(sprite) {
		if frame := s.tileEffectProcessor.GetEffectSprite(sprite, positionHash, elapsedMs); frame != nil {
			return frame
		}
	}
	animated := GetSpriteIndexWithAnimationBySpriteIndexTick(sprite, positionHash, elapsedMs)

	return s.GetRawSprite(animated)
}

// getOriginal16x16Sprite returns the original 16x16 sprite without border expansion
// Used internally by water tile generator for pixel-perfect animations
func (s *SpriteSheet) getOriginal16x16Sprite(nSprite indexes.SpriteIndex) *ebiten.Image {
	if nSprite < 0 || int(nSprite) >= totalSprites {
		panic("raw sprite index is outside the atlas")
	}
	if s.rawSpriteCache[nSprite] == nil {
		// Keep zero-origin bounds: portrait transforms and finale clipping rely on
		// copied-tile coordinates rather than the atlas subimage's original bounds.
		var padding graphics.Insets
		s.rawSpriteCache[nSprite] = s.copySpriteRegion(nSprite, padding)
	}

	return s.rawSpriteCache[nSprite]
}

// GetAnimatedSprite returns a sprite image with animation applied based on position and time
func (s *SpriteSheet) GetAnimatedSprite(
	nSprite indexes.SpriteIndex, posHash int32, animationTimeMs int64,
) *ebiten.Image {
	// Tiles with registered visual effects (flames, lava, water, mixed land/water) use the tile effect processor
	// All frames are pre-generated at initialization, so we can return directly without caching overhead
	if s.TileEffectsEnabled && s.tileEffectProcessor.HasEffect(nSprite) {
		// Get pre-generated 18x18 bordered frame directly (no cache, no allocation)
		borderedFrame := s.tileEffectProcessor.GetBorderedEffectSprite(nSprite, posHash, animationTimeMs)
		if borderedFrame != nil {
			return borderedFrame
		}
		// Fallback to standard sprite if effect generation failed
		return s.GetSprite(nSprite)
	}

	// Position/time choose the frame, not a separate copy of its artwork. GetSprite
	// already owns the resolved frame cache, so no position-specific LRU is needed.
	animatedSpriteIndex := GetSpriteIndexWithAnimationBySpriteIndexTick(nSprite, posHash, animationTimeMs)

	return s.GetSprite(animatedSpriteIndex)
}

// isStaticSprite determines if a sprite should be pre-generated (non-animated)
func (s *SpriteSheet) isStaticSprite(spriteIndex indexes.SpriteIndex) bool {
	// Tiles with registered visual effects are dynamic when effects are enabled
	// This covers: flames, lava, pure water tiles, mixed land/water tiles
	if s.TileEffectsEnabled && s.tileEffectProcessor.HasEffect(spriteIndex) {
		return false
	}

	// All other sprites are static and can be pre-generated
	return true
}

// GetTileEffectProcessor returns the tile effect processor for applying dynamic effects.
// This adapts game outline/contour presets to the SDK's bounded native processor.
func (s *SpriteSheet) GetTileEffectProcessor() *TileEffectProcessor {
	return s.tileEffectProcessor
}
