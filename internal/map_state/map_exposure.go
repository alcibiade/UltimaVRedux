package map_state

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ExposureKind describes structural exposure, independently of visibility and light.
type ExposureKind uint8

// Exposure kinds keep structural boundaries distinct from either side of a wall.
const (
	ExposureUnknown ExposureKind = iota
	ExposureOutdoor
	ExposureIndoor
	ExposureBoundary
)

// MapExposure is an immutable, non-wrapping classification of one source floor.
// Its zero value reports ExposureUnknown everywhere.
type MapExposure struct {
	width, height int
	cells         []ExposureKind
}

// ClassifyMapExposure floods open border cells and explicit outdoor seeds through
// four-neighbor connections. Unreached open cells are indoor; boundaries retain
// their own kind. Boundaries are row-major (y*width+x), and inputs are not retained.
// Invalid dimensions, cell counts, or seeds are programming errors and panic.
func ClassifyMapExposure(width, height int, boundaries []bool, outdoorSeeds []image.Point) MapExposure {
	if width <= 0 || height <= 0 {
		panic("map exposure dimensions must be positive")
	}
	if width > math.MaxInt/height || len(boundaries) != width*height {
		panic("map exposure dimensions overflow or do not match boundary count")
	}

	exposure := MapExposure{width: width, height: height, cells: make([]ExposureKind, len(boundaries))}
	pending := make([]image.Point, 0, len(boundaries))
	markOutdoor := func(x, y int) {
		if exposure.At(x, y) == ExposureIndoor {
			exposure.cells[y*width+x] = ExposureOutdoor
			pending = append(pending, image.Point{X: x, Y: y})
		}
	}
	for cell, boundary := range boundaries {
		if boundary {
			exposure.cells[cell] = ExposureBoundary

			continue
		}
		exposure.cells[cell] = ExposureIndoor
	}
	for x := range width {
		markOutdoor(x, 0)
		markOutdoor(x, height-1)
	}
	for y := range height {
		markOutdoor(0, y)
		markOutdoor(width-1, y)
	}
	for _, seed := range outdoorSeeds {
		kind := exposure.At(seed.X, seed.Y)
		if kind == ExposureUnknown || kind == ExposureBoundary {
			panic("map exposure outdoor seed must be in bounds and not a boundary")
		}
		markOutdoor(seed.X, seed.Y)
	}
	for len(pending) > 0 {
		point := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		markOutdoor(point.X-1, point.Y)
		markOutdoor(point.X+1, point.Y)
		markOutdoor(point.X, point.Y-1)
		markOutdoor(point.X, point.Y+1)
	}

	return exposure
}

// At returns ExposureUnknown outside the source floor, without wrapping.
func (m MapExposure) At(x, y int) ExposureKind {
	if x < 0 || y < 0 || x >= m.width || y >= m.height {
		return ExposureUnknown
	}

	return m.cells[y*m.width+x]
}

// isExposureBoundary uses source sprite identities, not movement or light rules.
// Archways, portcullises, fences, vegetation, water, and ordinary furniture remain open.
func isExposureBoundary(sprite indexes.SpriteIndex) bool {
	//nolint:exhaustive // Only source structural boundaries are included; all other sprites remain open.
	switch sprite {
	case indexes.StoneCrossWindow, indexes.StoneGlassWindow,
		indexes.LargeRockWall, indexes.StoneBrickWallSecret, indexes.StoneBrickWall,
		indexes.Bookcase,
		indexes.MagicLockDoor, indexes.MagicLockDoorWithView,
		indexes.RegularDoor, indexes.LockedDoor, indexes.RegularDoorView, indexes.LockedDoorView,
		indexes.Fireplace, indexes.SignWarning:
		// Source Fireplace (188) partitions embedded walls at SerpentsHold floor 1
		// (8,23) and LordBritishsCastle floor 0 (13,8); it is not freestanding furniture.
		// Britain also embeds Bookcase (90) at floor 0 (22,0) and SignWarning (248)
		// at floor 1 (11,0) in its outer walls. Neither is an opening to the border.
		return true
	default:
		return false
	}
}
