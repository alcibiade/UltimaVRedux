package map_state

import (
	"fmt"
	"image"
	"math"
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyMapExposure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		rows  []string
		seeds []image.Point
	}{
		{name: "enclosed room", rows: []string{
			".......",
			".#####.",
			".#III#.",
			".#III#.",
			".#####.",
			".......",
		}},
		{name: "no diagonal corner leak", rows: []string{".#.", "#I#", ".#."}},
		{name: "wall-only map border does not expose room", rows: []string{
			"#####",
			"#III#",
			"#III#",
			"#####",
		}},
		{name: "seed changes only its enclosed component", rows: []string{
			"#########",
			"#...#III#",
			"#...#III#",
			"#########",
		}, seeds: []image.Point{image.Pt(2, 1)}},
		{name: "all boundaries", rows: []string{"###", "###"}},
		{name: "single open cell", rows: []string{"."}},
		{name: "single boundary", rows: []string{"#"}},
		{name: "single column", rows: []string{".", "#", ".", "."}},
		{name: "single row", rows: []string{".#..#."}},
		{name: "duplicate border seeds", rows: []string{"..", ".."},
			seeds: []image.Point{image.Pt(0, 0), image.Pt(0, 0), image.Pt(1, 1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			width, height := len(test.rows[0]), len(test.rows)
			boundaries := exposureTestBoundaries(test.rows)
			originalBoundaries, originalSeeds := slices.Clone(boundaries), slices.Clone(test.seeds)
			exposure := ClassifyMapExposure(width, height, boundaries, test.seeds)
			assert.Equal(t, originalBoundaries, boundaries)
			assert.Equal(t, originalSeeds, test.seeds)
			assert.Equal(t, exposure, ClassifyMapExposure(width, height, boundaries, test.seeds))
			for y, row := range test.rows {
				for x, cell := range row {
					want := ExposureOutdoor
					switch cell {
					case '#':
						want = ExposureBoundary
					case 'I':
						want = ExposureIndoor
					}
					assert.Equal(t, want, exposure.At(x, y), "at (%d,%d)", x, y)
				}
			}
			// The result must not alias either caller-owned input.
			before := slices.Clone(exposure.cells)
			for cell := range boundaries {
				boundaries[cell] = !boundaries[cell]
			}
			for seed := range test.seeds {
				test.seeds[seed] = image.Pt(-1, -1)
			}
			assert.Equal(t, before, exposure.cells)
		})
	}
}

func TestMapExposureFourNeighborBorderConnections(t *testing.T) {
	t.Parallel()
	for _, border := range []image.Point{image.Pt(0, 2), image.Pt(4, 2), image.Pt(2, 0), image.Pt(2, 4)} {
		t.Run(border.String(), func(t *testing.T) {
			boundaries := exposureTestBoundaries([]string{"#####", "#III#", "#III#", "#III#", "#####"})
			assert.Equal(t, ExposureIndoor, ClassifyMapExposure(5, 5, boundaries, nil).At(2, 2))
			boundaries[border.Y*5+border.X] = false
			exposure := ClassifyMapExposure(5, 5, boundaries, nil)
			for y := 1; y <= 3; y++ {
				for x := 1; x <= 3; x++ {
					assert.Equal(t, ExposureOutdoor, exposure.At(x, y))
				}
			}
		})
	}
}

func TestMapExposureBoundsAndZeroValue(t *testing.T) {
	t.Parallel()
	exposure := ClassifyMapExposure(2, 3, make([]bool, 6), nil)
	for _, point := range []image.Point{
		image.Pt(-1, 0), image.Pt(0, -1), image.Pt(2, 0), image.Pt(0, 3),
		image.Pt(math.MaxInt, math.MaxInt), image.Pt(math.MinInt, math.MinInt),
	} {
		assert.Equal(t, ExposureUnknown, exposure.At(point.X, point.Y), "at %v", point)
	}
	assert.Equal(t, ExposureOutdoor, exposure.At(1, 2))
	var zero MapExposure
	assert.Equal(t, ExposureUnknown, zero.At(0, 0))
	assert.Equal(t, ExposureUnknown, zero.At(-1, -1))
	var layeredMap LayeredMap
	assert.Equal(t, ExposureUnknown, layeredMap.ExposureAt(references.Position{}))
	layeredMap.exposure = exposure
	assert.Equal(t, ExposureOutdoor, layeredMap.ExposureAt(references.Position{X: 1, Y: 2}))
	assert.Equal(t, ExposureUnknown, layeredMap.ExposureAt(references.Position{X: -1, Y: 0}))
	assert.Equal(t, ExposureUnknown, layeredMap.ExposureAt(references.Position{X: 2, Y: 0}))
}

func TestClassifyMapExposureInvalidInputs(t *testing.T) {
	t.Parallel()
	for _, dimensions := range []image.Point{
		image.Pt(0, 1), image.Pt(1, 0), image.Pt(-1, 1), image.Pt(1, -1),
		image.Pt(-1, -1), image.Pt(math.MaxInt, 2), image.Pt(2, math.MaxInt),
		image.Pt(math.MinInt, 1), image.Pt(1, math.MinInt), image.Pt(math.MaxInt, math.MaxInt),
		image.Pt(math.MaxInt, 1),
	} {
		assert.Panics(t, func() { ClassifyMapExposure(dimensions.X, dimensions.Y, nil, nil) }, "%v", dimensions)
	}
	for _, length := range []int{0, 5, 7} {
		assert.Panics(t, func() { ClassifyMapExposure(2, 3, make([]bool, length), nil) })
	}
	boundaries := exposureTestBoundaries([]string{"...", ".#.", "..."})
	for _, seed := range []image.Point{
		image.Pt(-1, 0), image.Pt(0, -1), image.Pt(3, 0), image.Pt(0, 3), image.Pt(1, 1),
		image.Pt(math.MaxInt, 0), image.Pt(math.MinInt, 0),
	} {
		assert.Panics(t, func() { ClassifyMapExposure(3, 3, boundaries, []image.Point{image.Pt(0, 0), seed}) }, "%v", seed)
	}
}

func TestMapExposureStructuralTilePolicy(t *testing.T) {
	t.Parallel()
	barriers := []indexes.SpriteIndex{
		indexes.StoneCrossWindow, indexes.StoneGlassWindow, indexes.LargeRockWall,
		indexes.StoneBrickWallSecret, indexes.StoneBrickWall, indexes.Bookcase, indexes.MagicLockDoor,
		indexes.MagicLockDoorWithView, indexes.RegularDoor, indexes.LockedDoor,
		indexes.RegularDoorView, indexes.LockedDoorView, indexes.Fireplace, indexes.SignWarning,
	}
	assert.Equal(t, []indexes.SpriteIndex{74, 75, 77, 78, 79, 90, 151, 152, 184, 185, 186, 187, 188, 248}, barriers)
	// Exercise every source byte, including archway 135, portcullis 153, fences
	// 202/203, forests, trees, water, and furniture, against the exact boundary set.
	for sprite := indexes.SpriteIndex(0); sprite <= 255; sprite++ {
		t.Run(fmt.Sprint(sprite), func(t *testing.T) {
			boundary := slices.Contains(barriers, sprite)
			assert.Equal(t, boundary, isExposureBoundary(sprite))
			boundaries := exposureTestBoundaries([]string{".....", ".###.", ".#I#.", ".###.", "....."})
			boundaries[1*5+2] = isExposureBoundary(sprite)
			exposure := ClassifyMapExposure(5, 5, boundaries, nil)
			if boundary {
				assert.Equal(t, ExposureBoundary, exposure.At(2, 1))
				assert.Equal(t, ExposureIndoor, exposure.At(2, 2))
			} else {
				assert.Equal(t, ExposureOutdoor, exposure.At(2, 1))
				assert.Equal(t, ExposureOutdoor, exposure.At(2, 2))
			}
		})
	}
}

func TestMapExposureOutdoorSeedMetadata(t *testing.T) {
	t.Parallel()
	wants := map[struct {
		location references.Location
		floor    references.FloorNumber
	}][]image.Point{
		{references.Yew, 0}:                {image.Pt(15, 15)},
		{references.Minoc, 0}:              {image.Pt(15, 15)},
		{references.LordBritishsCastle, 1}: {image.Pt(4, 12), image.Pt(26, 12), image.Pt(15, 4)},
	}
	for key, want := range wants {
		require.Equal(t, want, smallMapOutdoorSeeds(key.location, key.floor))
		seeds := smallMapOutdoorSeeds(key.location, key.floor)
		seeds[0] = image.Pt(-1, -1)
		assert.Equal(t, want, smallMapOutdoorSeeds(key.location, key.floor), "metadata must not share writable storage")
	}
	for location := references.Location(1); location <= references.TotalSmallMapLocations; location++ {
		for floor := references.FloorNumber(-1); floor <= 3; floor++ {
			if _, ok := wants[struct {
				location references.Location
				floor    references.FloorNumber
			}{location, floor}]; ok {
				continue
			}
			assert.Empty(t, smallMapOutdoorSeeds(location, floor), "%s floor %d", location, floor)
		}
	}
}

func exposureTestBoundaries(rows []string) []bool {
	boundaries := make([]bool, len(rows)*len(rows[0]))
	for y, row := range rows {
		for x, cell := range row {
			boundaries[y*len(rows[0])+x] = cell == '#'
		}
	}
	return boundaries
}
