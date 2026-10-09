package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/profiling"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// Presentation tuning in native artwork pixels, independent of light/LOS radii.
const (
	terrainDitherEnabled       = true
	terrainDitherOutwardPixels = 6
	terrainDitherInwardPixels  = 1
	terrainDitherCornerRadius  = 2
)

type terrainDitherSample struct {
	visible  bool
	trusted  bool
	hardEdge bool
}

type terrainDitherCell struct {
	visible bool
	bonus   bool
}

// The scene adapter owns world sampling; reusable sprites code only sees bits
// and pixels. No part of this plan is written back to gameplay visibility.
type terrainDitherFrame struct {
	enabled       bool
	wrapped       bool
	origin        references.Position
	width, height int
	margin        int
	masks         *sprites.QuarterMaskSet
	samples       []terrainDitherSample
	cells         []terrainDitherCell
	image         sprites.DitherMaskImage
	maskValid     bool
	maskBuilds    uint64
}

func (g *GameScene) shouldDitherTerrain() bool {
	if !terrainDitherEnabled || g.disposed || !g.usesSurfaceMapFrame() || g.xRayVisionActive || g.mapFadeLevel >= 1 {
		return false
	}
	if !g.gameState.CanAcceptGameplayCommands() {
		return false
	}
	for i := range g.activeTileDissolves {
		state := &g.activeTileDissolves[i]
		if state.RenderLayer == RenderLayerOverFade && state.Effect.IsActive() {
			return false
		}
	}

	return true
}

func (g *GameScene) prepareTerrainDither(layer *map_state.LayeredMap) {
	frame := &g.terrainDither
	oldOrigin, oldWidth, oldHeight, oldWrapped := frame.origin, frame.width, frame.height, frame.wrapped
	// Decide before terrain admission and retain the decision through final draw.
	frame.enabled = g.shouldDitherTerrain()
	if !frame.enabled {
		return
	}
	if frame.masks == nil {
		frame.masks = sprites.NewQuarterMaskSet(sprites.EdgeDitherOptions{
			TileSize: sprites.TileSize, OutwardPixels: terrainDitherOutwardPixels,
			InwardPixels: terrainDitherInwardPixels, CornerRadius: terrainDitherCornerRadius,
		})
	}
	smallMap := g.gameState.MapState.PlayerLocation.Location.GetMapType() == references.SmallMapType
	// Soften indoor light edges on every small-map floor, but retain
	// structural boundaries and their touching quarters.
	margin := g.surfaceOverscanTiles()
	frame.margin = int(margin)
	frame.width, frame.height = xTilesVisibleOnGameScreen+2*frame.margin, yTilesVisibleOnGameScreen+2*frame.margin
	frame.origin = references.Position{
		X: g.gameState.MapState.PlayerLocation.Position.X - xCenter - margin,
		Y: g.gameState.MapState.PlayerLocation.Position.Y - yCenter - margin,
	}
	frame.wrapped = layer.IsWrappedMap()
	dirty := !frame.maskValid || frame.origin != oldOrigin || frame.width != oldWidth ||
		frame.height != oldHeight || frame.wrapped != oldWrapped
	sampleWidth, sampleHeight := frame.width+2, frame.height+2
	if cap(frame.samples) < sampleWidth*sampleHeight {
		frame.samples = make([]terrainDitherSample, sampleWidth*sampleHeight)
		dirty = true
	}
	frame.samples = frame.samples[:sampleWidth*sampleHeight]
	if cap(frame.cells) < frame.width*frame.height {
		frame.cells = make([]terrainDitherCell, frame.width*frame.height)
	}
	frame.cells = frame.cells[:frame.width*frame.height]
	for y := range sampleHeight {
		for x := range sampleWidth {
			unwrapped := references.Position{X: frame.origin.X + references.Coordinate(x-1), Y: frame.origin.Y + references.Coordinate(y-1)}
			pos := unwrapped
			if frame.wrapped {
				pos = *pos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}
			_, hasDistance := layer.GetPrimaryDistanceMap()[pos]
			exposure := layer.ExposureAt(pos)
			sample := terrainDitherSample{
				visible:  g.isPositionVisible(layer, &pos),
				hardEdge: smallMap && exposure == map_state.ExposureBoundary,
				trusted: hasDistance && !g.gameState.IsOutOfBounds(pos) &&
					unwrapped.X >= layer.TopLeft.X && unwrapped.X < layer.BottomRight.X &&
					unwrapped.Y >= layer.TopLeft.Y && unwrapped.Y < layer.BottomRight.Y,
			}
			index := y*sampleWidth + x
			dirty = dirty || frame.samples[index] != sample
			frame.samples[index] = sample
		}
	}
	if dirty {
		frame.buildMask()
	}
}

func (frame *terrainDitherFrame) buildMask() {
	const bytesPerPixel, opaque = 4, 255
	pixelWidth := frame.width * sprites.TileSize
	pixels := frame.image.Resize(pixelWidth, frame.height*sprites.TileSize)
	stride := frame.width + 2
	neighbors := [...]struct {
		dx, dy int
		bit    uint16
	}{
		{0, -1, sprites.EdgeDitherNorth}, {1, -1, sprites.EdgeDitherNorthEast},
		{1, 0, sprites.EdgeDitherEast}, {1, 1, sprites.EdgeDitherSouthEast},
		{0, 1, sprites.EdgeDitherSouth}, {-1, 1, sprites.EdgeDitherSouthWest},
		{-1, 0, sprites.EdgeDitherWest}, {-1, -1, sprites.EdgeDitherNorthWest},
		{0, 0, sprites.EdgeDitherCenter},
	}
	for y := range frame.height {
		for x := range frame.width {
			center := (y+1)*stride + x + 1
			trusted := true
			var bits, hardEdgeBits uint16
			for _, neighbor := range neighbors {
				sample := frame.samples[center+neighbor.dy*stride+neighbor.dx]
				trusted = trusted && sample.trusted
				if sample.visible {
					bits |= neighbor.bit
				}
				if sample.hardEdge {
					hardEdgeBits |= neighbor.bit
				}
			}
			// Preserve protected indoor/structural tiles. Neighboring soft quarters
			// cannot introduce a bonus fringe through the same room boundary.
			quarterBits := [...]uint16{
				sprites.EdgeDitherCenter | sprites.EdgeDitherNorth | sprites.EdgeDitherWest | sprites.EdgeDitherNorthWest,
				sprites.EdgeDitherCenter | sprites.EdgeDitherNorth | sprites.EdgeDitherEast | sprites.EdgeDitherNorthEast,
				sprites.EdgeDitherCenter | sprites.EdgeDitherSouth | sprites.EdgeDitherWest | sprites.EdgeDitherSouthWest,
				sprites.EdgeDitherCenter | sprites.EdgeDitherSouth | sprites.EdgeDitherEast | sprites.EdgeDitherSouthEast,
			}
			var hardQuarters [4]bool
			for quarter, touching := range quarterBits {
				hardQuarters[quarter] = hardEdgeBits&touching != 0
			}
			cell := terrainDitherCell{visible: frame.samples[center].visible, bonus: false}
			for pixelY := range sprites.TileSize {
				for pixelX := range sprites.TileSize {
					keep := cell.visible
					quarter := (pixelY/(sprites.TileSize/2))*2 + pixelX/(sprites.TileSize/2)
					if trusted && !hardQuarters[quarter] {
						coverage := frame.masks.Coverage(bits, pixelX, pixelY)
						keep = sprites.DitherKeeps(coverage,
							(int(frame.origin.X)+x)*sprites.TileSize+pixelX,
							(int(frame.origin.Y)+y)*sprites.TileSize+pixelY)
					}
					if !keep {
						pixels[((y*sprites.TileSize+pixelY)*pixelWidth+x*sprites.TileSize+pixelX)*bytesPerPixel+3] = opaque
					} else if !cell.visible {
						cell.bonus = true
					}
				}
			}
			frame.cells[y*frame.width+x] = cell
		}
	}
	frame.image.Upload()
	frame.maskValid = true
	frame.maskBuilds++
	profiling.Count(profiling.DitherBuilds)
}

func (frame *terrainDitherFrame) cellAt(pos references.Position) (terrainDitherCell, bool) {
	x, y := int(pos.X-frame.origin.X), int(pos.Y-frame.origin.Y)
	if frame.wrapped {
		x = (x%int(references.XLargeMapTiles) + int(references.XLargeMapTiles)) % int(references.XLargeMapTiles)
		y = (y%int(references.YLargeMapTiles) + int(references.YLargeMapTiles)) % int(references.YLargeMapTiles)
	}
	if x < 0 || y < 0 || x >= frame.width || y >= frame.height {
		return terrainDitherCell{}, false
	}

	return frame.cells[y*frame.width+x], true
}

// The final mask cannot distinguish artwork layers. Keep unguarded world effects
// out of hidden cells before composition; bonus permission is terrain-only.
func (g *GameScene) ditherAllowsWorldEffect(pos references.Position) bool {
	if !g.terrainDither.enabled {
		return true
	}
	cell, found := g.terrainDither.cellAt(pos)

	return found && cell.visible
}

func (g *GameScene) ditherRelativePosition(pos references.Position) references.Position {
	player := g.gameState.MapState.PlayerLocation.Position
	relative := references.Position{X: pos.X - player.X, Y: pos.Y - player.Y}
	if g.terrainDither.wrapped {
		width, height := references.XLargeMapTiles, references.YLargeMapTiles
		relative.X = (relative.X%width+width+width/2)%width - width/2
		relative.Y = (relative.Y%height+height+height/2)%height - height/2
	}

	return relative
}

func (g *GameScene) drawTerrainDither(target *ebiten.Image) {
	frame := &g.terrainDither
	if !frame.enabled {
		return
	}
	transform := ebiten.GeoM{}
	transform.Translate(-float64(frame.margin*sprites.TileSize), -float64(frame.margin*sprites.TileSize))
	transform.Concat(g.worldDrawOptions().GeoM)
	frame.image.Draw(target, transform)

	// Camera/shake translate the source, but the aperture is fixed. Explicitly
	// reject uncovered strips instead of relying on the decorative border.
	// DrawImage computes vertices in float32 and snaps half-pixel edges upward.
	// Use the same precision and tie direction so no uncovered row/column leaks.
	left, top := float32(transform.Element(0, 2)), float32(transform.Element(1, 2))
	right := float32(frame.width*sprites.TileSize)*float32(transform.Element(0, 0)) + left
	bottom := float32(frame.height*sprites.TileSize)*float32(transform.Element(1, 1)) + top
	bounds := target.Bounds()
	covered := image.Rect(int(math.Floor(float64(left)+0.5)), int(math.Floor(float64(top)+0.5)),
		int(math.Floor(float64(right)+0.5)), int(math.Floor(float64(bottom)+0.5))).Intersect(bounds)
	strips := [...]image.Rectangle{
		image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, covered.Min.Y),
		image.Rect(bounds.Min.X, covered.Max.Y, bounds.Max.X, bounds.Max.Y),
		image.Rect(bounds.Min.X, covered.Min.Y, covered.Min.X, covered.Max.Y),
		image.Rect(covered.Max.X, covered.Min.Y, bounds.Max.X, covered.Max.Y),
	}
	if covered.Empty() {
		target.Fill(color.Black)

		return
	}
	for _, strip := range strips {
		if !strip.Empty() {
			vector.FillRect(target, float32(strip.Min.X), float32(strip.Min.Y),
				float32(strip.Dx()), float32(strip.Dy()), color.Black, false)
		}
	}
}
