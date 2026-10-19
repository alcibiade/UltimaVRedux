package sprites

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"slices"
	"testing"
)

func quarterTestOptions() EdgeDitherOptions {
	return EdgeDitherOptions{TileSize: 16, OutwardPixels: 4, InwardPixels: 1, CornerRadius: 2}
}

// This oracle constructs the complete finite union boundary, independently of
// the production four-cell formulas and quarter assembly. Coordinates follow
// the public clockwise neighborhood bit layout, with center last.
func quarterTestCoordinates() [9][2]int {
	return [9][2]int{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, 0}}
}

type quarterOracleCell struct {
	x, y    float64
	rounded [4]bool
}

type quarterOracle struct {
	options EdgeDitherOptions
	cells   []quarterOracleCell
	edges   [][4]float64
	arcs    [][4]float64 // Circle center and quadrant signs.
}

func newQuarterOracle(options EdgeDitherOptions, neighborhood uint16) quarterOracle {
	occupied := make(map[[2]int]bool)
	for bit, cell := range quarterTestCoordinates() {
		occupied[cell] = neighborhood&(1<<bit) != 0
	}
	oracle := quarterOracle{options: options}
	size, radius := float64(options.TileSize), float64(options.CornerRadius)
	for _, cell := range quarterTestCoordinates() {
		if !occupied[cell] {
			continue
		}
		gx, gy := cell[0], cell[1]
		x, y := float64(gx)*size, float64(gy)*size
		north, east := occupied[[2]int{gx, gy - 1}], occupied[[2]int{gx + 1, gy}]
		south, west := occupied[[2]int{gx, gy + 1}], occupied[[2]int{gx - 1, gy}]
		rounded := [4]bool{!north && !west, !north && !east, !south && !east, !south && !west}
		oracle.cells = append(oracle.cells, quarterOracleCell{x: x, y: y, rounded: rounded})
		centers := [4][4]float64{
			{x + radius, y + radius, -1, -1}, {x + size - radius, y + radius, 1, -1},
			{x + size - radius, y + size - radius, 1, 1}, {x + radius, y + size - radius, -1, 1},
		}
		var trim [4]float64
		for corner, round := range rounded {
			if round && radius > 0 {
				trim[corner] = radius
				oracle.arcs = append(oracle.arcs, centers[corner])
			}
		}
		if !north {
			oracle.edges = append(oracle.edges, [4]float64{x + trim[0], y, x + size - trim[1], y})
		}
		if !east {
			oracle.edges = append(oracle.edges, [4]float64{x + size, y + trim[1], x + size, y + size - trim[2]})
		}
		if !south {
			oracle.edges = append(oracle.edges, [4]float64{x + trim[3], y + size, x + size - trim[2], y + size})
		}
		if !west {
			oracle.edges = append(oracle.edges, [4]float64{x, y + trim[0], x, y + size - trim[3]})
		}
	}
	return oracle
}

func (oracle quarterOracle) coverage(x, y float64) (float64, float64) {
	size, radius := float64(oracle.options.TileSize), float64(oracle.options.CornerRadius)
	distance, originalDistance := math.Inf(1), math.Inf(1)
	for _, edge := range oracle.edges {
		distance = min(distance, math.Hypot(x-max(edge[0], min(edge[2], x)), y-max(edge[1], min(edge[3], y))))
	}
	for _, arc := range oracle.arcs {
		dx, dy := x-arc[0], y-arc[1]
		candidate := min(math.Hypot(dx-arc[2]*radius, dy), math.Hypot(dx, dy-arc[3]*radius))
		if dx*arc[2] >= 0 && dy*arc[3] >= 0 {
			candidate = math.Abs(math.Hypot(dx, dy) - radius)
		}
		distance = min(distance, candidate)
	}
	inside := false
	for _, cell := range oracle.cells {
		originalDistance = min(originalDistance, math.Hypot(max(cell.x-x, 0, x-cell.x-size), max(cell.y-y, 0, y-cell.y-size)))
		if x < cell.x || x > cell.x+size || y < cell.y || y > cell.y+size {
			continue
		}
		inCell := true
		corners := [4][2]float64{{x - cell.x, y - cell.y}, {cell.x + size - x, y - cell.y},
			{cell.x + size - x, cell.y + size - y}, {x - cell.x, cell.y + size - y}}
		for corner, point := range corners {
			if cell.rounded[corner] && point[0] < radius && point[1] < radius &&
				math.Hypot(point[0]-radius, point[1]-radius) > radius {
				inCell = false
			}
		}
		inside = inside || inCell
	}
	if inside {
		distance = -distance
	}
	coverage := (float64(oracle.options.OutwardPixels) - distance) / float64(oracle.options.OutwardPixels+oracle.options.InwardPixels)
	return max(0, min(1, coverage)), originalDistance
}

func TestQuarterMaskExhaustive(t *testing.T) {
	options := quarterTestOptions()
	set := NewQuarterMaskSet(options)
	bayer := [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}
	unique := [2]map[string]bool{{}, {}}
	for neighborhood := uint16(0); neighborhood < 512; neighborhood++ {
		oracle := newQuarterOracle(options, neighborhood)
		var encoded [512]byte
		for y := range 16 {
			for x := range 16 {
				floating, originalDistance := oracle.coverage(float64(x)+0.5, float64(y)+0.5)
				want := uint16(math.Round(floating * 4096))
				got := set.Coverage(neighborhood, x, y)
				if got != want {
					t.Fatalf("state %09b pixel (%d,%d): got %d, full-union oracle %d", neighborhood, x, y, got, want)
				}
				if got > 4096 || (got > 0 && originalDistance >= 4) {
					t.Fatalf("coverage/support bound: state %09b pixel (%d,%d), q=%d distance=%g", neighborhood, x, y, got, originalDistance)
				}
				if x >= 7 && x <= 8 && y >= 7 && y <= 8 {
					if got != (neighborhood>>8)*4096 {
						t.Fatalf("tile center changed: state %09b pixel (%d,%d)", neighborhood, x, y)
					}
				}
				binary.LittleEndian.PutUint16(encoded[(y*16+x)*2:], got)
				for phaseY := range 4 {
					for phaseX := range 4 {
						threshold := (float64(bayer[(y+phaseY)%4][(x+phaseX)%4]) + 0.5) / 16
						if DitherKeeps(got, x+phaseX, y+phaseY) != (floating > threshold) {
							t.Fatalf("float/Q12 decision mismatch: state %09b pixel (%d,%d) phase (%d,%d)", neighborhood, x, y, phaseX, phaseY)
						}
					}
				}
				if (neighborhood == 0 && got != 0) || (neighborhood == 511 && got != 4096) {
					t.Fatal("constant full/empty mask changed")
				}
			}
		}
		unique[neighborhood>>8][string(encoded[:])] = true
	}
	if len(unique[0]) != 256 || len(unique[1]) != 47 {
		t.Fatalf("distinct full-tile arrays: hidden=%d visible=%d; want 256/47", len(unique[0]), len(unique[1]))
	}
	t.Log("131072 full-union comparisons and 2097152 float/Q12 Bayer comparisons passed")
}

func TestQuarterMaskTemplatesAndLocality(t *testing.T) {
	set := NewQuarterMaskSet(quarterTestOptions())
	if len(set.pieces) != 10 {
		t.Fatalf("stored quarter templates: got %d, want 10", len(set.pieces))
	}
	bits := [4][3]uint16{
		{EdgeDitherNorth, EdgeDitherWest, EdgeDitherNorthWest},
		{EdgeDitherNorth, EdgeDitherEast, EdgeDitherNorthEast},
		{EdgeDitherSouth, EdgeDitherWest, EdgeDitherSouthWest},
		{EdgeDitherSouth, EdgeDitherEast, EdgeDitherSouthEast},
	}
	groups := make(map[int][64]uint16)
	unique := make(map[[64]uint16]bool)
	for neighborhood := uint16(0); neighborhood < 512; neighborhood++ {
		for quadrant, touching := range bits {
			local := int(neighborhood >> 8)
			for index, bit := range touching {
				if neighborhood&bit != 0 {
					local |= 1 << (index + 1)
				}
			}
			var quarter [64]uint16
			for y := range 8 {
				for x := range 8 {
					px, py := x, y
					if quadrant&1 != 0 {
						px = 15 - x
					}
					if quadrant&2 != 0 {
						py = 15 - y
					}
					quarter[y*8+x] = set.Coverage(neighborhood, px, py)
				}
			}
			if previous, exists := groups[local]; exists && previous != quarter {
				t.Fatalf("quarter locality/orientation mismatch: state %09b quadrant %d local %04b", neighborhood, quadrant, local)
			}
			groups[local] = quarter
			unique[quarter] = true
		}
	}
	if len(groups) != 16 || len(unique) != 13 {
		t.Fatalf("quarter groups=%d distinct=%d; want 16/13", len(groups), len(unique))
	}
}

func quarterTestTransform(x, y, transform int) (int, int) {
	if transform >= 4 {
		x = -x
	}
	for range transform % 4 {
		x, y = -y, x
	}
	return x, y
}

func TestQuarterMaskD4(t *testing.T) {
	coordinates := quarterTestCoordinates()
	for _, options := range []EdgeDitherOptions{quarterTestOptions(), {16, 0, 2, 2}} {
		t.Run(fmt.Sprintf("%d/%d/%d/%d", options.TileSize, options.OutwardPixels, options.InwardPixels, options.CornerRadius), func(t *testing.T) {
			set := NewQuarterMaskSet(options)
			for neighborhood := uint16(0); neighborhood < 512; neighborhood++ {
				for transform := 1; transform < 8; transform++ {
					var transformed uint16
					for bit, point := range coordinates {
						if neighborhood&(1<<bit) == 0 {
							continue
						}
						x, y := quarterTestTransform(point[0], point[1], transform)
						for target, candidate := range coordinates {
							if candidate == [2]int{x, y} {
								transformed |= 1 << target
							}
						}
					}
					for y := range 16 {
						for x := range 16 {
							tx, ty := quarterTestTransform(2*x-15, 2*y-15, transform)
							if set.Coverage(neighborhood, x, y) != set.Coverage(transformed, (tx+15)/2, (ty+15)/2) {
								t.Fatalf("D4 mismatch: state %09b transform %d pixel (%d,%d)", neighborhood, transform, x, y)
							}
						}
					}
				}
			}
			t.Log("917504 exact integer D4 comparisons passed")
		})
	}
}

func TestQuarterMaskTuning(t *testing.T) {
	for _, options := range []EdgeDitherOptions{
		{4, 1, 0, 1}, {6, 0, 1, 2}, {8, 2, 1, 2},
		{16, 0, 1, 2}, {16, 0, 2, 2}, {16, 0, 3, 2}, {16, 0, 4, 2},
		{16, 0, 3, 4}, {16, 3, 0, 4}, {16, 1, 1, 7},
		{16, 4, 2, 0}, {16, 6, 1, 2}, {24, 5, 2, 3}, {32, 6, 3, 2},
	} {
		t.Run(fmt.Sprintf("%d/%d/%d/%d", options.TileSize, options.OutwardPixels, options.InwardPixels, options.CornerRadius), func(t *testing.T) {
			set := NewQuarterMaskSet(options)
			repeated := NewQuarterMaskSet(options)
			for index, piece := range set.pieces {
				if !slices.Equal(piece, repeated.pieces[index]) {
					t.Fatal("construction is not deterministic")
				}
			}
			for neighborhood := uint16(0); neighborhood < 512; neighborhood++ {
				oracle := newQuarterOracle(options, neighborhood)
				for y := range options.TileSize {
					for x := range options.TileSize {
						floating, originalDistance := oracle.coverage(float64(x)+0.5, float64(y)+0.5)
						got, want := set.Coverage(neighborhood, x, y), uint16(math.Round(floating*4096))
						if got != want {
							t.Fatalf("state %09b pixel (%d,%d): got %d, oracle %d", neighborhood, x, y, got, want)
						}
						if got > 4096 || (got > 0 && originalDistance > 0 && originalDistance >= float64(options.OutwardPixels)) {
							t.Fatal("coverage/support bound exceeded")
						}
						half := options.TileSize / 2
						if x >= half-1 && x <= half && y >= half-1 && y <= half && got != (neighborhood>>8)*4096 {
							t.Fatal("central pixels changed")
						}
					}
				}
			}
		})
	}
}

func TestQuarterMaskIndoor(t *testing.T) {
	set := NewQuarterMaskSet(EdgeDitherOptions{16, 0, 2, 2})
	t.Run("HiddenCenter", func(t *testing.T) {
		for neighborhood := uint16(0); neighborhood < 256; neighborhood++ {
			for y := range 16 {
				for x := range 16 {
					got := set.Coverage(neighborhood, x, y)
					if got != 0 {
						t.Fatalf("hidden center: state %09b pixel (%d,%d): Q12=%d, want 0", neighborhood, x, y, got)
					}
					for phaseY := range 4 {
						for phaseX := range 4 {
							if DitherKeeps(got, x+phaseX, y+phaseY) {
								t.Fatalf("hidden center kept: state %09b pixel (%d,%d) phase (%d,%d)", neighborhood, x, y, phaseX, phaseY)
							}
						}
					}
				}
			}
		}
	})
	t.Run("VisibleIsolatedCorners", func(t *testing.T) {
		for _, point := range [][2]int{{0, 0}, {15, 0}, {0, 15}, {15, 15}} {
			if got := set.Coverage(EdgeDitherCenter, point[0], point[1]); got != 0 {
				t.Fatalf("isolated corner (%d,%d): Q12=%d, want 0", point[0], point[1], got)
			}
		}
	})
	t.Run("StraightNorthEdge", func(t *testing.T) {
		neighborhood := EdgeDitherCenter | EdgeDitherEast | EdgeDitherWest |
			EdgeDitherSouth | EdgeDitherSouthEast | EdgeDitherSouthWest
		// Sample at depths 0.5, 1.5, 2.5, and 3.5, away from either corner.
		for y, want := range [4]uint16{1024, 3072, 4096, 4096} {
			for x := 4; x < 12; x++ {
				if got := set.Coverage(neighborhood, x, y); got != want {
					t.Fatalf("straight edge pixel (%d,%d): Q12=%d, want %d", x, y, got, want)
				}
			}
		}
	})
}

func quarterTestPanics(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected programming-error panic")
		}
	}()
	action()
}

func TestQuarterMaskValidation(t *testing.T) {
	for _, options := range []EdgeDitherOptions{
		{}, {-16, 4, 1, 2}, {2, 1, 0, 0}, {15, 4, 1, 2}, {258, 4, 1, 2},
		{16, -1, 1, 2}, {16, 4, -1, 2}, {16, 4, 1, -1}, {16, 0, 0, 2},
		{16, 8, 1, 0}, {16, 1, 8, 0}, {16, 4, 1, 5},
		{math.MaxInt, 1, 1, 0}, {16, math.MaxInt, 1, 0}, {16, 1, 1, math.MaxInt},
	} {
		t.Run(fmt.Sprint(options), func(t *testing.T) {
			quarterTestPanics(t, func() { NewQuarterMaskSet(options) })
		})
	}
	set := NewQuarterMaskSet(quarterTestOptions())
	for _, point := range [][2]int{{-1, 0}, {0, -1}, {16, 0}, {0, 16}, {math.MaxInt, 0}} {
		quarterTestPanics(t, func() { set.Coverage(0, point[0], point[1]) })
	}
	quarterTestPanics(t, func() { set.Coverage(512, 0, 0) })
	quarterTestPanics(t, func() { new(QuarterMaskSet).Coverage(0, 0, 0) })
	large := NewQuarterMaskSet(EdgeDitherOptions{256, 4, 1, 2})
	if large.Coverage(511, 127, 127) != 4096 {
		t.Fatal("maximum supported tile size failed")
	}
}

func TestQuarterMaskCoverageAllocations(t *testing.T) {
	set := NewQuarterMaskSet(quarterTestOptions())
	var assembled [256]bool
	allocations := testing.AllocsPerRun(100, func() {
		for y := range 16 {
			for x := range 16 {
				assembled[y*16+x] = DitherKeeps(set.Coverage(301, x, y), x-16, y+16)
			}
		}
	})
	if allocations != 0 {
		t.Fatalf("assembly allocated %g times, want zero", allocations)
	}
	runtime.KeepAlive(&assembled)
}

func BenchmarkQuarterMaskCoverage(b *testing.B) {
	set := NewQuarterMaskSet(quarterTestOptions())
	var sum uint64
	b.ReportAllocs()
	b.ResetTimer()
	for index := range b.N {
		sum += uint64(set.Coverage(uint16(index&511), index&15, (index>>4)&15))
	}
	runtime.KeepAlive(sum)
}

func BenchmarkQuarterMaskAssembly(b *testing.B) {
	set := NewQuarterMaskSet(quarterTestOptions())
	var assembled [256]bool
	b.ReportAllocs()
	b.ResetTimer()
	for index := range b.N {
		for y := range 16 {
			for x := range 16 {
				assembled[y*16+x] = DitherKeeps(set.Coverage(uint16(index&511), x, y), x-16, y+16)
			}
		}
	}
	runtime.KeepAlive(&assembled)
}
