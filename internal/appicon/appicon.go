// Package appicon derives the application icon from the original Avatar tile in
// the bundled sprite atlas, so the window, Dock and executable icons share one
// artwork source. It has no Ebitengine dependency.
package appicon

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TileIndex identifies the atlas tile used as the application icon.
const TileIndex = indexes.BasicAvatar

// TileSize is the edge length of an original atlas tile in pixels.
const TileSize = 16

const (
	atlasTilesPerRow = 32
	// The Dock artwork follows the macOS icon grid: a transparent margin keeps the
	// opaque tile from looking larger than neighbouring application icons. The
	// content is a whole multiple of the tile so pixels stay square.
	dockCanvasSize  = 1024
	dockContentSize = 768
	// A centred image splits its margin equally between opposite sides.
	marginSides = 2
)

var errIconGeometry = errors.New("application icon geometry")

// WindowIconSizes lists the square window-icon candidates. Each is a whole
// multiple of the tile so enlarged pixels stay sharp.
func WindowIconSizes() []int {
	return []int{TileSize, 2 * TileSize, 3 * TileSize, 4 * TileSize, 8 * TileSize, 16 * TileSize}
}

// Tile crops the Avatar tile from the sprite atlas PNG.
func Tile(atlasPNG []byte) (*image.NRGBA, error) {
	atlas, err := png.Decode(bytes.NewReader(atlasPNG))
	if err != nil {
		return nil, fmt.Errorf("decoding sprite atlas: %w", err)
	}
	index := int(TileIndex)
	origin := atlas.Bounds().Min.Add(image.Pt(index%atlasTilesPerRow*TileSize, index/atlasTilesPerRow*TileSize))
	source := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(TileSize, TileSize))}
	if !source.In(atlas.Bounds()) {
		return nil, fmt.Errorf("%w: tile %d at %v lies outside atlas %v", errIconGeometry, index, source, atlas.Bounds())
	}
	tile := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	draw.Draw(tile, tile.Bounds(), atlas, source.Min, draw.Src)

	return tile, nil
}

// Render enlarges the tile by whole-pixel replication to contentSize and centres
// it on a transparent canvasSize square.
func Render(tile *image.NRGBA, contentSize, canvasSize int) (*image.NRGBA, error) {
	if tile.Bounds().Dx() != TileSize || tile.Bounds().Dy() != TileSize {
		return nil, fmt.Errorf("%w: tile is %v, want %dx%d", errIconGeometry, tile.Bounds().Size(), TileSize, TileSize)
	}
	if contentSize <= 0 || contentSize%TileSize != 0 {
		return nil, fmt.Errorf("%w: content size %d is not a positive multiple of %d", errIconGeometry, contentSize, TileSize)
	}
	if canvasSize < contentSize || (canvasSize-contentSize)%marginSides != 0 {
		return nil, fmt.Errorf("%w: content %d cannot be centred on canvas %d", errIconGeometry, contentSize, canvasSize)
	}
	scale := contentSize / TileSize
	margin := (canvasSize - contentSize) / marginSides
	icon := image.NewNRGBA(image.Rect(0, 0, canvasSize, canvasSize))
	for y := range contentSize {
		for x := range contentSize {
			icon.SetNRGBA(margin+x, margin+y, tile.NRGBAAt(x/scale, y/scale))
		}
	}

	return icon, nil
}

// WindowIcons returns full-bleed icon candidates for ebiten.SetWindowIcon, one
// per WindowIconSizes entry.
func WindowIcons(atlasPNG []byte) ([]image.Image, error) {
	tile, err := Tile(atlasPNG)
	if err != nil {
		return nil, err
	}
	sizes := WindowIconSizes()
	icons := make([]image.Image, 0, len(sizes))
	for _, size := range sizes {
		icon, err := Render(tile, size, size)
		if err != nil {
			return nil, err
		}
		icons = append(icons, icon)
	}

	return icons, nil
}

// DockIconPNG returns the PNG-encoded macOS Dock artwork.
func DockIconPNG(atlasPNG []byte) ([]byte, error) {
	tile, err := Tile(atlasPNG)
	if err != nil {
		return nil, err
	}
	icon, err := Render(tile, dockContentSize, dockCanvasSize)
	if err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, icon); err != nil {
		return nil, fmt.Errorf("encoding Dock icon: %w", err)
	}

	return encoded.Bytes(), nil
}
