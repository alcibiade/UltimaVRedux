package visual

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type randomShuffler interface {
	Intn(n int) int
}

// bytesPerPixel is the number of bytes per pixel in RGBA format
const bytesPerPixel = 4

// DissolveMode controls whether pixels are revealed or hidden during dissolve
type DissolveMode int

const (
	// DissolveModeReveal starts transparent and progressively shows pixels (0 → 255 alpha)
	DissolveModeReveal DissolveMode = iota
	// DissolveModeHide starts opaque and progressively hides pixels (255 → 0 alpha)
	DissolveModeHide
)

// PixelDissolveEffect manages progressive random pixel reveal/conceal using GPU-accelerated masking.
//
// This implementation uses Ebiten's BlendDestinationIn mode with a dynamically updated alpha mask,
// providing significant performance improvements over pixel-by-pixel Set() operations:
//   - Persistent mask buffer (zero per-frame allocations)
//   - CPU-side pixel updates (no GPU state changes during update)
//   - Single GPU-accelerated DrawImage call per frame
//   - Optional mask scaling for memory efficiency (e.g., 1/4 scale for large dialogs)
//
// Usage:
//
//	effect := NewPixelDissolveEffect(800, 600, 200, 150, DissolveModeReveal, rng)
//	for progress := 0.0; progress <= 1.0; progress += 0.01 {
//	    effect.UpdateProgress(progress)
//	    mask := effect.GetMask()
//	    // Render with BlendDestinationIn...
//	}
type PixelDissolveEffect struct {
	// Target dimensions (final display size)
	targetWidth, targetHeight int

	// Mask dimensions (may be smaller than target for memory efficiency)
	maskWidth, maskHeight int

	// Pixel data buffer (RGBA format, 4 bytes per pixel)
	// Only alpha channel is used for masking, RGB are set to white (255,255,255)
	pixelData []byte

	// Ebiten mask image (GPU-side)
	mask *ebiten.Image

	// Scaled mask for when target != mask dimensions
	scaledMask *ebiten.Image

	// Shuffled pixel coordinates (pre-generated for determinism)
	pixelOrder []image.Point

	// Progress tracking
	currentPixelIdx int // Current position in pixelOrder array
	totalPixels     int // Total pixels in mask

	// Mode and state
	mode      DissolveMode
	maskDirty bool
}

// NewPixelDissolveEffect creates a new dissolve effect with optional mask scaling.
//
// Parameters:
//   - targetWidth, targetHeight: Final display dimensions
//   - maskWidth, maskHeight: Mask buffer dimensions (may be smaller for efficiency)
//   - mode: Reveal (transparent→opaque) or Hide (opaque→transparent)
//   - rng: Presentation-owned deterministic random source; never pass the gameplay source
//
// For 1:1 rendering (sprites), use same values for target and mask dimensions.
// For scaled rendering (MessageDialog), use CalculateMaskSizeForDialog() for mask dimensions.
//
// Example (sprite):
//
//	effect := NewPixelDissolveEffect(32, 32, 32, 32, DissolveModeReveal, gs.rng)
//
// Example (scaled dialog):
//
//	maskW, maskH := CalculateMaskSizeForDialog(800, 600)
//	effect := NewPixelDissolveEffect(800, 600, maskW, maskH, DissolveModeReveal, rng)
func NewPixelDissolveEffect(
	targetWidth, targetHeight int,
	maskWidth, maskHeight int,
	mode DissolveMode,
	rng randomShuffler,
) *PixelDissolveEffect {
	totalPixels := maskWidth * maskHeight

	// Create pixel data buffer (RGBA format: bytesPerPixel bytes per pixel)
	pixelData := make([]byte, totalPixels*bytesPerPixel)

	// Determine initial alpha based on mode
	initialAlpha := uint8(0) // Start transparent for reveal
	if mode == DissolveModeHide {
		initialAlpha = 255 // Start opaque for hide
	}

	// Fill pixel data with initial values
	// RGB = white (255,255,255), A = initial alpha
	// Using white RGB ensures BlendDestinationIn preserves destination colors
	for i := range totalPixels {
		offset := i * bytesPerPixel
		pixelData[offset+0] = 255          // R
		pixelData[offset+1] = 255          // G
		pixelData[offset+2] = 255          // B
		pixelData[offset+3] = initialAlpha // A
	}

	// Create Ebiten mask image
	mask := ebiten.NewImage(maskWidth, maskHeight)
	mask.WritePixels(pixelData)

	// Create scaled mask if needed
	var scaledMask *ebiten.Image
	if maskWidth != targetWidth || maskHeight != targetHeight {
		scaledMask = ebiten.NewImage(targetWidth, targetHeight)
	}

	// Create array of all pixel coordinates
	pixelOrder := make([]image.Point, 0, totalPixels)
	for y := range maskHeight {
		for x := range maskWidth {
			pixelOrder = append(pixelOrder, image.Point{X: x, Y: y})
		}
	}

	// Shuffle pixel order using Fisher-Yates algorithm (deterministic with provided RNG)
	for i := len(pixelOrder) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		pixelOrder[i], pixelOrder[j] = pixelOrder[j], pixelOrder[i]
	}

	return &PixelDissolveEffect{
		targetWidth:     targetWidth,
		targetHeight:    targetHeight,
		maskWidth:       maskWidth,
		maskHeight:      maskHeight,
		pixelData:       pixelData,
		mask:            mask,
		scaledMask:      scaledMask,
		pixelOrder:      pixelOrder,
		currentPixelIdx: 0,
		totalPixels:     totalPixels,
		mode:            mode,
		maskDirty:       false, // Already written initial state
	}
}

// UpdateProgress updates the dissolve effect based on progress (0.0 to 1.0).
//
// Progress interpretation:
//   - 0.0: Initial state (transparent for Reveal, opaque for Hide)
//   - 1.0: Final state (opaque for Reveal, transparent for Hide)
//   - Values outside 0.0-1.0 are clamped
//
// Returns true if effect is complete (all pixels processed).
func (pde *PixelDissolveEffect) UpdateProgress(progress float64) bool {
	// Clamp progress to valid range
	if progress < 0.0 {
		progress = 0.0
	}

	if progress > 1.0 {
		progress = 1.0
	}

	// Calculate target pixel index based on progress
	targetIdx := int(float64(pde.totalPixels) * progress)

	// Determine target alpha value based on mode
	targetAlpha := uint8(255) // Opaque for reveal
	if pde.mode == DissolveModeHide {
		targetAlpha = 0 // Transparent for hide
	}

	// Update pixels from current index to target
	// This allows progress to move forward or backward
	if targetIdx > pde.currentPixelIdx {
		// Progress forward
		for pde.currentPixelIdx < targetIdx && pde.currentPixelIdx < pde.totalPixels {
			pt := pde.pixelOrder[pde.currentPixelIdx]
			offset := (pt.Y*pde.maskWidth + pt.X) * bytesPerPixel
			pde.pixelData[offset+3] = targetAlpha // Update alpha channel only
			pde.currentPixelIdx++
		}

		pde.maskDirty = true
	} else if targetIdx < pde.currentPixelIdx {
		// Progress backward (reverse dissolve)
		reverseAlpha := uint8(0) // Transparent for reveal mode reversal
		if pde.mode == DissolveModeHide {
			reverseAlpha = 255 // Opaque for hide mode reversal
		}

		for pde.currentPixelIdx > targetIdx && pde.currentPixelIdx > 0 {
			pde.currentPixelIdx--
			pt := pde.pixelOrder[pde.currentPixelIdx]
			offset := (pt.Y*pde.maskWidth + pt.X) * bytesPerPixel
			pde.pixelData[offset+3] = reverseAlpha // Update alpha channel only
		}

		pde.maskDirty = true
	}

	return pde.currentPixelIdx >= pde.totalPixels
}

// GetMask returns the current alpha mask as an Ebiten image, scaled to target dimensions if needed.
//
// The mask is cached and only regenerated when dirty (after UpdateProgress calls).
// If mask dimensions match target dimensions, returns mask directly.
// If mask is smaller, scales up using nearest-neighbor filtering for pixelated effect.
func (pde *PixelDissolveEffect) GetMask() *ebiten.Image {
	if pde.maskDirty {
		// Upload updated pixel data to GPU
		pde.mask.WritePixels(pde.pixelData)
		pde.maskDirty = false
	}

	// Check if scaling is needed
	if pde.scaledMask == nil {
		// No scaling needed - return mask directly
		return pde.mask
	}

	// Scale mask to target dimensions
	pde.scaledMask.Clear()

	scaleOp := &ebiten.DrawImageOptions{}
	scaleX := float64(pde.targetWidth) / float64(pde.maskWidth)
	scaleY := float64(pde.targetHeight) / float64(pde.maskHeight)
	scaleOp.GeoM.Scale(scaleX, scaleY)
	scaleOp.Filter = ebiten.FilterNearest // Pixelated scaling for retro aesthetic

	pde.scaledMask.DrawImage(pde.mask, scaleOp)

	return pde.scaledMask
}

// Reset resets the effect to initial state, allowing reuse with the same dimensions.
//
// This is more efficient than creating a new effect if dimensions haven't changed.
// The pixel shuffle order is preserved (same pattern will be used again).
func (pde *PixelDissolveEffect) Reset() {
	pde.currentPixelIdx = 0

	// Reset pixel data to initial alpha
	initialAlpha := uint8(0)
	if pde.mode == DissolveModeHide {
		initialAlpha = 255
	}

	for i := range pde.totalPixels {
		offset := i * bytesPerPixel
		pde.pixelData[offset+3] = initialAlpha // Reset alpha channel only (RGB stays white)
	}

	// Upload reset state to GPU
	pde.mask.WritePixels(pde.pixelData)
	pde.maskDirty = false
}

// VisitLeadingPixels optionally exposes the recently revealed portion of the
// deterministic mask, in target coordinates. Consumers clip highlights to their
// sprite before composing semantic overlays. Ordinary dissolve users are unchanged.
func (pde *PixelDissolveEffect) VisitLeadingPixels(fraction float64, visit func(image.Rectangle)) {
	if pde.mode != DissolveModeReveal || pde.currentPixelIdx == pde.totalPixels {
		return
	}
	count := int(float64(pde.totalPixels) * min(max(fraction, 0), 1))
	for i := max(pde.currentPixelIdx-count, 0); i < pde.currentPixelIdx; i++ {
		point := pde.pixelOrder[i]
		visit(image.Rect(point.X*pde.targetWidth/pde.maskWidth, point.Y*pde.targetHeight/pde.maskHeight,
			(point.X+1)*pde.targetWidth/pde.maskWidth, (point.Y+1)*pde.targetHeight/pde.maskHeight))
	}
}

// Dispose releases optional scaled and native mask images on every effect exit.
func (pde *PixelDissolveEffect) Dispose() {
	if pde.mask != nil {
		pde.mask.Deallocate()
		pde.mask = nil
	}
	if pde.scaledMask != nil {
		pde.scaledMask.Deallocate()
		pde.scaledMask = nil
	}
}
