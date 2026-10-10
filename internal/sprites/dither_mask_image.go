package sprites

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// DitherMaskImage reuses a CPU pixel buffer and GPU image for a rejection mask.
// Its zero value is ready for Resize. Do not copy it after first use.
type DitherMaskImage struct {
	image         *ebiten.Image
	pixels        []byte
	width, height int
}

// Resize returns a cleared, premultiplied RGBA buffer of the requested size.
// The caller fills its contents: transparent pixels keep the world visible and
// opaque black pixels reject it. The buffer aliases this resource until a size
// change or Dispose; even same-size calls clear it. Upload commits the contents
// to the image. Same-size calls reuse both allocations. Dimensions must be positive.
func (m *DitherMaskImage) Resize(width, height int) []byte {
	const bytesPerPixel = 4
	if width <= 0 || height <= 0 {
		panic("sprites: dither mask dimensions must be positive")
	}

	if m.width != width || m.height != height {
		m.Dispose()
		// Keep nearest-sampling ties independent of texture-atlas placement.
		m.image = ebiten.NewImageWithOptions(image.Rect(0, 0, width, height), &ebiten.NewImageOptions{Unmanaged: true})
		m.pixels = make([]byte, width*height*bytesPerPixel)
		m.width, m.height = width, height
	}
	clear(m.pixels)

	return m.pixels
}

// Upload writes the current buffer to the image. Resize must be called first.
func (m *DitherMaskImage) Upload() {
	m.image.WritePixels(m.pixels)
}

// Draw composites the uploaded mask with nearest filtering, neutral color scale,
// and SourceOver blending. The caller supplies placement and any target SubImage
// clipping; no world coordinates or viewport are inferred. Resize must precede Draw.
func (m *DitherMaskImage) Draw(target *ebiten.Image, transform ebiten.GeoM) {
	target.DrawImage(m.image, &ebiten.DrawImageOptions{
		GeoM:   transform,
		Filter: ebiten.FilterNearest,
	})
}

// Dispose releases the image and buffer, restoring the zero value. It is idempotent.
func (m *DitherMaskImage) Dispose() {
	if m.image != nil {
		m.image.Deallocate()
	}
	m.image = nil
	m.pixels = nil
	m.width, m.height = 0, 0
}
