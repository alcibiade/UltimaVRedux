package sprites

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestBorderCacheBudgetsPreserveBorrowedArtwork(t *testing.T) {
	const imageBytes = 16 * 16 * 4
	const pixelBytes = 4 * 4 * 4
	for _, budget := range []struct {
		name    string
		bytes   int64
		entries int
	}{
		{name: "pixel budget", bytes: 2 * imageBytes, entries: 8},
		{name: "entry budget", bytes: 8 * imageBytes, entries: 2},
	} {
		t.Run(budget.name, func(t *testing.T) {
			cache := newBorderImageCache(budget.bytes, budget.entries)
			images := make([]*ebiten.Image, 3)
			for i := range images {
				images[i] = ebiten.NewImage(4, 4)
				t.Cleanup(images[i].Deallocate)
				images[i].Fill(color.RGBA{R: byte(i + 1), A: 255})
			}
			cache.put("first", images[0])
			cache.put("borrowed", images[1])
			require.Same(t, images[0], cache.get("first"))
			cache.put("third", images[2])
			require.Nil(t, cache.get("borrowed"), "least-recently-used lookup must be retired")
			require.Same(t, images[0], cache.get("first"))
			require.Same(t, images[2], cache.get("third"))
			require.LessOrEqual(t, cache.bytes, budget.bytes)
			require.LessOrEqual(t, len(cache.entries), budget.entries)
			pixels := make([]byte, pixelBytes)
			images[1].ReadPixels(pixels)
			for i := 0; i < len(pixels); i += 4 {
				require.Equal(t, []byte{2, 0, 0, 255}, pixels[i:i+4], "eviction must not dispose an image borrowed by a live scene")
			}
		})
	}
}

func TestBorderCacheOversizeAndReplacementStayBounded(t *testing.T) {
	cache := newBorderImageCache(4096, 2)
	first, replacement, large := ebiten.NewImage(32, 32), ebiten.NewImage(2, 2), ebiten.NewImage(65, 65)
	t.Cleanup(first.Deallocate)
	t.Cleanup(replacement.Deallocate)
	t.Cleanup(large.Deallocate)
	cache.put("frame", first)
	cache.put("frame", replacement)
	require.Equal(t, int64(1024), cache.bytes)
	require.Len(t, cache.entries, 1)
	require.Same(t, replacement, cache.get("frame"))
	cache.put("oversize", large)
	require.Nil(t, cache.get("oversize"))
	require.Same(t, replacement, cache.get("frame"), "oversize bypass must not flush useful cached artwork")
	require.Equal(t, int64(1024), cache.bytes)
}

func TestBorderCacheAccountsPaddedTextureStorage(t *testing.T) {
	cache := newBorderImageCache(4096, 8)
	frame := ebiten.NewImage(17, 17)
	t.Cleanup(frame.Deallocate)
	cache.put("odd", frame)
	require.Equal(t, int64(32*32*4), cache.bytes, "17-square frame occupies a 32-square texture, not 1156 bytes")
	other := ebiten.NewImage(3, 3)
	t.Cleanup(other.Deallocate)
	cache.put("small", other)
	require.Nil(t, cache.get("odd"), "logical-byte accounting would incorrectly retain both")
	require.Same(t, other, cache.get("small"))
	require.Equal(t, int64(16*16*4), cache.bytes)
}
