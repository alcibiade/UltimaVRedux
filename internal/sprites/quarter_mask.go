package sprites

import (
	"math"
	"slices"
)

// DitherCoverageFull is full coverage in the Q12 units accepted by DitherKeeps.
const DitherCoverageFull = 4096

// Neighborhood bits run clockwise from north, followed by the center cell.
const (
	EdgeDitherNorth uint16 = 1 << iota
	EdgeDitherNorthEast
	EdgeDitherEast
	EdgeDitherSouthEast
	EdgeDitherSouth
	EdgeDitherSouthWest
	EdgeDitherWest
	EdgeDitherNorthWest
	EdgeDitherCenter
)

const (
	quarterCenter = 1 << iota
	quarterVertical
	quarterHorizontal
	quarterDiagonal
)

const (
	quarterStateCount   = 16
	quarterMaxTileSize  = 256
	quarterMinTileSize  = 4
	quarterAxisParts    = 2
	quarterPixelCenter  = 0.5
	ditherPeriod        = 4
	ditherThresholdUnit = 128
)

// EdgeDitherOptions measures geometry in logical tile pixels. TileSize must be
// even and in [4,256]. Widths are nonnegative, not both zero, and strictly less
// than half a tile. CornerRadius is nonnegative and its sum with the larger
// width must not exceed half a tile. These bounds preserve quarter locality
// and the four central pixels, and bound table allocation.
type EdgeDitherOptions struct {
	TileSize      int
	OutwardPixels int
	InwardPixels  int
	CornerRadius  int
}

// QuarterMaskSet is immutable after construction and safe for concurrent reads.
// Its zero value is not usable; construct it with NewQuarterMaskSet.
type QuarterMaskSet struct {
	tileSize int
	pieces   [][]uint16
	quarters [quarterStateCount]quarterMaskReference
}

type quarterMaskReference struct {
	piece     int
	transpose bool
}

// NewQuarterMaskSet generates reusable Q12 quarter pieces, not whole-tile masks.
// Invalid options are programming errors and panic. The 16 four-cell states
// share identical pieces and corner-bisector reflections (10 pieces at 16/4/1/2).
func NewQuarterMaskSet(options EdgeDitherOptions) *QuarterMaskSet {
	options.validate()

	set := &QuarterMaskSet{
		tileSize: options.TileSize,
		pieces:   make([][]uint16, 0, quarterStateCount),
		quarters: [quarterStateCount]quarterMaskReference{},
	}
	for state := range quarterStateCount {
		if state&quarterHorizontal != 0 && state&quarterVertical == 0 {
			set.quarters[state] = set.quarters[state^(quarterHorizontal|quarterVertical)]
			set.quarters[state].transpose = true

			continue
		}

		piece := generateQuarterMask(options, state)
		index := len(set.pieces)
		for previous, candidate := range set.pieces {
			if slices.Equal(piece, candidate) {
				index = previous

				break
			}
		}
		if index == len(set.pieces) {
			set.pieces = append(set.pieces, piece)
		}
		set.quarters[state] = quarterMaskReference{piece: index, transpose: false}
	}

	return set
}

func (options EdgeDitherOptions) validate() {
	if options.TileSize < quarterMinTileSize || options.TileSize > quarterMaxTileSize ||
		options.TileSize%quarterAxisParts != 0 {
		panic("edge dither: TileSize must be even and in [4,256]")
	}
	width := max(options.OutwardPixels, options.InwardPixels)
	if options.OutwardPixels < 0 || options.InwardPixels < 0 || width == 0 || width >= options.TileSize/quarterAxisParts {
		panic("edge dither: widths must be nonnegative, not both zero, and less than half a tile")
	}
	if options.CornerRadius < 0 || options.CornerRadius > options.TileSize/quarterAxisParts-width {
		panic("edge dither: radius plus the larger width must not exceed half a tile")
	}
}

func generateQuarterMask(options EdgeDitherOptions, state int) []uint16 {
	half := options.TileSize / quarterAxisParts
	piece := make([]uint16, half*half)
	for y := range half {
		for x := range half {
			distance := quarterSignedDistance(state, float64(x)+quarterPixelCenter, float64(y)+quarterPixelCenter,
				float64(options.CornerRadius))
			coverage := (float64(options.OutwardPixels) - distance) / float64(options.OutwardPixels+options.InwardPixels)
			piece[y*half+x] = uint16(math.Round(DitherCoverageFull * max(0, min(1, coverage))))
		}
	}

	return piece
}

// quarterSignedDistance uses coordinates directed inward from one tile corner.
// Only exposed union edges and convex arcs matter within the validated band.
// In particular, a shared edge is never a candidate for interior distance.
func quarterSignedDistance(state int, pixelX, pixelY, radius float64) float64 {
	cardinals := state & (quarterVertical | quarterHorizontal)
	diagonal := state&quarterDiagonal != 0
	if state&quarterCenter != 0 {
		switch cardinals {
		case quarterVertical | quarterHorizontal:
			if diagonal {
				return math.Inf(-1)
			}

			return -math.Hypot(pixelX, pixelY)
		case quarterVertical:
			return -pixelX
		case quarterHorizontal:
			return -pixelY
		default:
			return math.Hypot(max(radius-pixelX, 0), max(radius-pixelY, 0)) +
				min(max(radius-pixelX, radius-pixelY), 0) - radius
		}
	}
	if diagonal {
		switch cardinals {
		case quarterVertical | quarterHorizontal:
			return min(pixelX, pixelY)
		case quarterVertical:
			return pixelY
		case quarterHorizontal:
			return pixelX
		default:
			return math.Hypot(pixelX+radius, pixelY+radius) - radius
		}
	}
	distance := math.Inf(1)
	if state&quarterVertical != 0 {
		distance = math.Hypot(max(radius-pixelX, 0), pixelY+radius) - radius
	}
	if state&quarterHorizontal != 0 {
		distance = min(distance, math.Hypot(pixelX+radius, max(radius-pixelY, 0))-radius)
	}

	return distance
}

// Coverage returns Q12 coverage in [0,4096] at logical pixel center (x+.5,y+.5).
// It panics for coordinates outside the tile or bits outside the nine-cell mask.
// Quarter selection, reflection, and lookup use only integer operations.
func (set *QuarterMaskSet) Coverage(neighborhood uint16, tileX, tileY int) uint16 {
	if tileX < 0 || tileY < 0 || tileX >= set.tileSize || tileY >= set.tileSize || neighborhood >= EdgeDitherCenter<<1 {
		panic("edge dither: invalid tile coordinate or neighborhood")
	}
	half := set.tileSize / quarterAxisParts
	vertical, horizontal, diagonal := EdgeDitherNorth, EdgeDitherWest, EdgeDitherNorthWest
	if tileX >= half {
		tileX = set.tileSize - 1 - tileX
		horizontal, diagonal = EdgeDitherEast, EdgeDitherNorthEast
	}
	if tileY >= half {
		tileY = set.tileSize - 1 - tileY
		vertical, diagonal = EdgeDitherSouth, EdgeDitherSouthWest
		if horizontal == EdgeDitherEast {
			diagonal = EdgeDitherSouthEast
		}
	}
	state := 0
	if neighborhood&EdgeDitherCenter != 0 {
		state |= quarterCenter
	}
	if neighborhood&vertical != 0 {
		state |= quarterVertical
	}
	if neighborhood&horizontal != 0 {
		state |= quarterHorizontal
	}
	if neighborhood&diagonal != 0 {
		state |= quarterDiagonal
	}
	ref := set.quarters[state]
	if ref.transpose {
		tileX, tileY = tileY, tileX
	}

	return set.pieces[ref.piece][tileY*half+tileX]
}

// DitherKeeps applies the fixed world-space Bayer4 threshold, including negative
// world coordinates. Transform coverage, not this pattern. Values above Q12 one
// are programming errors and panic; equality with a threshold is discarded.
func DitherKeeps(coverage uint16, worldX, worldY int) bool {
	if coverage > DitherCoverageFull {
		panic("edge dither: coverage exceeds Q12 one")
	}
	bayer := [ditherPeriod][ditherPeriod]uint16{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}
	rank := bayer[worldY&(ditherPeriod-1)][worldX&(ditherPeriod-1)]

	return coverage > (2*rank+1)*ditherThresholdUnit
}
