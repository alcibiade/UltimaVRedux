package appicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const atlasPath = "../sprites/assets/transparent_u5_tiles.png"

func bundledAtlas(t *testing.T) []byte {
	t.Helper()
	atlas, err := os.ReadFile(atlasPath)
	require.NoError(t, err)

	return atlas
}

// indexedAtlas fills every tile with a colour encoding its index, optionally
// replacing the icon tile's pixels.
func indexedAtlas(t *testing.T, iconPixel func(x, y int) color.NRGBA) []byte {
	t.Helper()
	atlas := image.NewNRGBA(image.Rect(0, 0, atlasTilesPerRow*TileSize, 16*TileSize))
	for y := range atlas.Bounds().Dy() {
		for x := range atlas.Bounds().Dx() {
			index := y/TileSize*atlasTilesPerRow + x/TileSize
			pixel := color.NRGBA{R: byte(index), G: byte(index >> 8), B: 7, A: 255}
			if index == int(TileIndex) && iconPixel != nil {
				pixel = iconPixel(x%TileSize, y%TileSize)
			}
			atlas.SetNRGBA(x, y, pixel)
		}
	}
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, atlas))

	return encoded.Bytes()
}

// patternTile gives every pixel a distinct colour so scaling errors are visible.
func patternTile() *image.NRGBA {
	tile := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	for y := range TileSize {
		for x := range TileSize {
			tile.SetNRGBA(x, y, color.NRGBA{R: byte(x * 16), G: byte(y * 16), B: 99, A: 255})
		}
	}

	return tile
}

func TestTileCropsTheAvatarTileFromTheAtlas(t *testing.T) {
	tile, err := Tile(indexedAtlas(t, nil))
	require.NoError(t, err)

	require.Equal(t, image.Rect(0, 0, TileSize, TileSize), tile.Bounds())
	index := int(TileIndex)
	want := color.NRGBA{R: byte(index), G: byte(index >> 8), B: 7, A: 255}
	for y := range TileSize {
		for x := range TileSize {
			require.Equal(t, want, tile.NRGBAAt(x, y), "pixel %d,%d", x, y)
		}
	}
}

func TestTileUsesTheOpaqueBundledAvatar(t *testing.T) {
	tile, err := Tile(bundledAtlas(t))
	require.NoError(t, err)

	// The icon relies on the tile's own black background on every platform.
	for y := range TileSize {
		for x := range TileSize {
			require.Equal(t, uint8(255), tile.NRGBAAt(x, y).A, "pixel %d,%d", x, y)
		}
	}
	assert.Equal(t, color.NRGBA{A: 255}, tile.NRGBAAt(0, 0), "background")
	assert.NotEqual(t, color.NRGBA{A: 255}, tile.NRGBAAt(7, 6), "body")
}

func TestTileRejectsUnusableAtlases(t *testing.T) {
	_, err := Tile([]byte("not a png"))
	require.ErrorContains(t, err, "decoding sprite atlas")

	var small bytes.Buffer
	require.NoError(t, png.Encode(&small, image.NewNRGBA(image.Rect(0, 0, 64, 64))))
	_, err = Tile(small.Bytes())
	require.ErrorIs(t, err, errIconGeometry)
}

func TestRenderReplicatesPixelsOntoACentredCanvas(t *testing.T) {
	tile := patternTile()
	icon, err := Render(tile, 2*TileSize, 3*TileSize)
	require.NoError(t, err)

	require.Equal(t, image.Rect(0, 0, 3*TileSize, 3*TileSize), icon.Bounds())
	margin := TileSize / 2
	for y := range icon.Bounds().Dy() {
		for x := range icon.Bounds().Dx() {
			inside := x >= margin && x < margin+2*TileSize && y >= margin && y < margin+2*TileSize
			if !inside {
				require.Equal(t, color.NRGBA{}, icon.NRGBAAt(x, y), "margin pixel %d,%d", x, y)

				continue
			}
			want := tile.NRGBAAt((x-margin)/2, (y-margin)/2)
			require.Equal(t, want, icon.NRGBAAt(x, y), "content pixel %d,%d", x, y)
		}
	}
}

func TestRenderRejectsGeometryThatBlursOrMisalignsPixels(t *testing.T) {
	for _, test := range []struct {
		name    string
		content int
		canvas  int
	}{
		{name: "empty", content: 0, canvas: 16},
		{name: "fractional scale", content: 24, canvas: 24},
		{name: "canvas too small", content: 32, canvas: 16},
		{name: "uneven margin", content: 32, canvas: 33},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Render(patternTile(), test.content, test.canvas)
			require.ErrorIs(t, err, errIconGeometry)
		})
	}

	_, err := Render(image.NewNRGBA(image.Rect(0, 0, 8, 8)), 16, 16)
	require.ErrorIs(t, err, errIconGeometry)
}

func TestWindowIconsCoverEverySharpSize(t *testing.T) {
	atlas := bundledAtlas(t)
	tile, err := Tile(atlas)
	require.NoError(t, err)

	icons, err := WindowIcons(atlas)
	require.NoError(t, err)

	require.Len(t, icons, len(WindowIconSizes()))
	for index, size := range WindowIconSizes() {
		want, err := Render(tile, size, size)
		require.NoError(t, err)
		assert.Equal(t, want, icons[index], "size %d", size)
	}
}

func TestDockIconKeepsTheMacOSGridMargin(t *testing.T) {
	atlas := bundledAtlas(t)
	tile, err := Tile(atlas)
	require.NoError(t, err)

	encoded, err := DockIconPNG(atlas)
	require.NoError(t, err)
	decoded, err := png.Decode(bytes.NewReader(encoded))
	require.NoError(t, err)

	want, err := Render(tile, dockContentSize, dockCanvasSize)
	require.NoError(t, err)
	got := image.NewNRGBA(decoded.Bounds())
	for y := range decoded.Bounds().Dy() {
		for x := range decoded.Bounds().Dx() {
			got.Set(x, y, decoded.At(x, y))
		}
	}
	require.Equal(t, want.Bounds(), got.Bounds())
	assert.Equal(t, want.Pix, got.Pix)
	assert.Equal(t, color.NRGBA{}, got.NRGBAAt(0, 0), "transparent corner")
	assert.Equal(t, tile.NRGBAAt(0, 0), got.NRGBAAt(dockCanvasSize/2-dockContentSize/2, dockCanvasSize/2-dockContentSize/2))
}
