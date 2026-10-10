package atlasterrain

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T) (Terrain, []byte) {
	t.Helper()
	var terrain Terrain
	for x := range WorldSize {
		for y := range WorldSize {
			terrain[x][y] = 1
		}
	}
	for _, guard := range siteGuards {
		terrain[guard.x][guard.y] = uint16(guard.source)
	}
	for x := range 50 {
		terrain[x][0] = 20
	}
	// Distinct retained geometry: a clearing, road, coast and asymmetric corners.
	terrain[50][70] = 5
	terrain[51][70] = 32
	terrain[52][70] = 3
	terrain[0][255] = 12
	terrain[255][0] = 7
	sheet := image.NewNRGBA(image.Rect(0, 0, 512, 256))
	for id := range 256 {
		sheet.SetNRGBA(id%32*16, id/32*16, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	}
	var buffer bytes.Buffer
	require.NoError(t, png.Encode(&buffer, sheet))
	return terrain, buffer.Bytes()
}

func TestAtlasPolicyGuardsAndPreservesNaturalGeometry(t *testing.T) {
	terrain, _ := fixture(t)
	normalized, count, err := Normalize(terrain)
	require.NoError(t, err)
	require.Equal(t, 69, count)
	require.Equal(t, terrain[50][70], normalized[50][70])
	require.Equal(t, terrain[51][70], normalized[51][70])
	require.Equal(t, uint16(12), normalized[36][229])
	require.Equal(t, uint16(8), normalized[45][62])
	for _, guard := range siteGuards {
		modified := terrain
		modified[guard.x][guard.y] = 5
		_, _, err = Normalize(modified)
		require.Error(t, err)
	}
	terrain[80][70] = 255
	_, _, err = Normalize(terrain)
	require.Error(t, err)
}

func TestAtlasCacheHitIdentityCorruptionAndUnwritable(t *testing.T) {
	terrain, art := fixture(t)
	root := t.TempDir()
	cold, err := LoadOrGenerate(root, terrain, art)
	require.NoError(t, err)
	require.NoError(t, cold.Diagnostic)
	require.False(t, cold.CacheHit)
	require.Equal(t, image.Rect(0, 0, ImageSize, ImageSize), cold.Image.Bounds())
	for _, sample := range []struct {
		x, y int
		want color.RGBA
	}{
		{50 * 16, 70 * 16, rgb(201, 202, 177)},
		{50*16 + 1, 70 * 16, rgb(111, 143, 73)},
		{1, 255 * 16, rgb(105, 109, 96)},
		{255*16 + 1, 0, rgb(188, 155, 86)},
	} {
		require.Equal(t, sample.want, cold.Image.At(sample.x, sample.y))
	}
	warm, err := LoadOrGenerate(root, terrain, art)
	require.NoError(t, err)
	require.True(t, warm.CacheHit)
	require.Zero(t, warm.ComposeTime)
	require.Zero(t, warm.EncodeTime)
	require.Equal(t, cold.Identity, warm.Identity)
	require.Equal(t, cold.Path, warm.Path)
	require.Equal(t, cold.Image.Bounds(), warm.Image.Bounds())
	require.NotSame(t, cold.Image, warm.Image, "warm decode transfers independent CPU ownership")
	for _, point := range []image.Point{{0, 0}, {800, 1120}, {801, 1120}, {1, 4080}, {4081, 0}, {4095, 4095}} {
		require.Equal(t, cold.Image.At(point.X, point.Y), warm.Image.At(point.X, point.Y), "pixel %v", point)
	}
	changed := terrain
	changed[70][71] = 5
	require.NotEqual(t, Identity(terrain, art), Identity(changed, art))
	require.NotEqual(t, Identity(terrain, art), Identity(terrain, append(append([]byte(nil), art...), 0)))
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0600))
	inMemory, err := LoadOrGenerate(blocker, terrain, art)
	require.NoError(t, err)
	require.Error(t, inMemory.Diagnostic)
	require.NotNil(t, inMemory.Image)
	require.Equal(t, cold.Image.Bounds(), inMemory.Image.Bounds())
	require.Equal(t, cold.Image.At(800, 1120), inMemory.Image.At(800, 1120))
	files, err := filepath.Glob(filepath.Join(root, ".atlas-*"))
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestAtlasCacheRejectsWrongDimensionsBeforeDecode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "small.png")
	require.NoError(t, writeCache(path, "identity", image.NewRGBA(image.Rect(0, 0, 16, 16))))
	_, ok := readCache(path, "identity")
	require.False(t, ok)
	_, ok = readCache(path, "other")
	require.False(t, ok)
	// A tiny file exercises corruption rejection without recomposing 4096² pixels.
	require.NoError(t, os.WriteFile(path, []byte("truncated"), 0600))
	_, ok = readCache(path, "identity")
	require.False(t, ok, "surviving identity metadata cannot authenticate changed PNG bytes")
}
