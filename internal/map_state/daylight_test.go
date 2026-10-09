package map_state

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaylightOptions(t *testing.T) {
	t.Parallel()
	options := DefaultDaylightOptions()
	assert.GreaterOrEqual(t, options.IndoorBrightness, float32(minimumDaylightBrightness))
	assert.LessOrEqual(t, options.IndoorBrightness, options.OutdoorBrightness)
	assert.Equal(t, float32(1), options.OutdoorBrightness)
	assert.Equal(t, []float32{1, 0.6, 0.25}, options.OpeningFalloff)
	field := NewDaylightField(options)
	options.OutdoorBrightness = 0.9
	options.OpeningFalloff[0] = 0
	assert.Equal(t, DefaultDaylightOptions(), field.options, "constructor owns a detached copy")
	assert.Equal(t, float32(1), DefaultDaylightOptions().OpeningFalloff[0], "defaults never share writable storage")

	for _, value := range []float32{0, 0.299, 1.001, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			outdoor := DefaultDaylightOptions()
			outdoor.OutdoorBrightness = value
			assert.Panics(t, func() { NewDaylightField(outdoor) })
			indoor := DefaultDaylightOptions()
			indoor.IndoorBrightness = value
			assert.Panics(t, func() { NewDaylightField(indoor) })
		})
	}
	for _, weights := range [][]float32{
		{-0.1}, {1.1}, {0.2, 0.3}, {1, float32(math.NaN())}, {float32(math.Inf(1))},
		{float32(math.Inf(-1))}, make([]float32, 9),
	} {
		invalid := DefaultDaylightOptions()
		invalid.OpeningFalloff = weights
		assert.Panics(t, func() { NewDaylightField(invalid) }, "%v", weights)
	}
	assert.Panics(t, func() {
		NewDaylightField(DaylightOptions{OutdoorBrightness: 0.8, IndoorBrightness: 0.9})
	})
	for _, weights := range [][]float32{nil, {}, {0}, {1, 1, 0}, make([]float32, 8)} {
		for _, brightness := range []float32{0.3, 1} {
			assert.NotPanics(t, func() {
				NewDaylightField(DaylightOptions{OutdoorBrightness: brightness, IndoorBrightness: brightness, OpeningFalloff: weights})
			})
		}
	}
}

func TestDaylightWindowFootprint(t *testing.T) {
	t.Parallel()
	defaults := DefaultDaylightOptions()
	indoor, outdoor := defaults.IndoorBrightness, defaults.OutdoorBrightness
	near, far := indoor+(outdoor-indoor)*0.6, indoor+(outdoor-indoor)*0.25
	for _, sprite := range []indexes.SpriteIndex{indexes.StoneCrossWindow, indexes.StoneGlassWindow} {
		for _, direction := range []struct {
			name   string
			pos    references.Position
			inward references.Position
		}{
			{"east", references.Position{X: 1, Y: 5}, references.Position{X: 1, Y: 0}},
			{"west", references.Position{X: 9, Y: 5}, references.Position{X: -1, Y: 0}},
			{"south", references.Position{X: 5, Y: 1}, references.Position{X: 0, Y: 1}},
			{"north", references.Position{X: 5, Y: 9}, references.Position{X: 0, Y: -1}},
		} {
			t.Run(fmt.Sprintf("%d/%s", sprite, direction.name), func(t *testing.T) {
				layer := daylightTestRoom()
				opening, ok := daylightOpeningAt(layer.exposure, direction.pos, sprite)
				require.True(t, ok)
				assert.Equal(t, direction.inward, opening.inward)
				assert.False(t, opening.requiresOpen)
				layer.daylightOpenings = []daylightOpening{opening}
				field := NewDaylightField(DefaultDaylightOptions())
				field.PrepareSmallMap(layer, func(*references.Position) bool {
					t.Fatal("windows must not consult live door state")
					return false
				})
				footprint := map[references.Position]float32{direction.pos: outdoor}
				for _, cell := range []struct {
					forward, side references.Coordinate
					target        float32
				}{
					{0, -1, outdoor}, {0, 1, outdoor}, {0, -2, near}, {0, 2, near}, {0, -3, far}, {0, 3, far},
					{1, 0, outdoor}, {2, 0, near}, {3, 0, far},
					{1, -1, near}, {1, 1, near}, {1, -2, far}, {1, 2, far},
					{2, -1, far}, {2, 1, far}, {2, -2, far}, {2, 2, far},
					{3, -1, indoor}, {3, 1, indoor},
				} {
					pos := references.Position{
						X: direction.pos.X + cell.forward*direction.inward.X - cell.side*direction.inward.Y,
						Y: direction.pos.Y + cell.forward*direction.inward.Y + cell.side*direction.inward.X,
					}
					footprint[pos] = cell.target
				}
				for y := references.Coordinate(0); y < 11; y++ {
					for x := references.Coordinate(0); x < 11; x++ {
						pos := references.Position{X: x, Y: y}
						want := indoor
						if layer.ExposureAt(pos) == ExposureOutdoor {
							want = outdoor
						}
						if target, exists := footprint[pos]; exists {
							want = target
						}
						assertDaylightTarget(t, field, pos, want, true)
					}
				}
			})
		}
	}
}

func TestDaylightReceiverPolicy(t *testing.T) {
	t.Parallel()
	defaults := DefaultDaylightOptions()
	for _, kind := range []ExposureKind{ExposureIndoor, ExposureBoundary, ExposureOutdoor, ExposureUnknown} {
		for _, receiver := range []struct {
			name string
			pos  references.Position
		}{
			{"outward", references.Position{X: 0, Y: 6}},
			{"wall plane", references.Position{X: 1, Y: 6}},
			{"inward", references.Position{X: 2, Y: 6}},
			{"diagonal outer band", references.Position{X: 3, Y: 7}},
			{"outside radius", references.Position{X: 4, Y: 6}},
		} {
			t.Run(fmt.Sprintf("%d/%s", kind, receiver.name), func(t *testing.T) {
				layer := daylightTestRoom()
				opening, ok := daylightOpeningAt(layer.exposure, references.Position{X: 1, Y: 5}, indexes.StoneCrossWindow)
				require.True(t, ok)
				layer.daylightOpenings = []daylightOpening{opening}
				// These receivers are off the axis used to identify the source opening.
				layer.exposure.cells[int(receiver.pos.Y)*layer.exposure.width+int(receiver.pos.X)] = kind
				exposure := slices.Clone(layer.exposure.cells)
				field := NewDaylightField(defaults)
				field.PrepareSmallMap(layer, daylightTestClosedDoor)
				want := daylightOriginalTarget(layer.exposure, layer.daylightOpenings, receiver.pos, false, defaults)
				assertDaylightTarget(t, field, receiver.pos, want, kind != ExposureUnknown)
				assertDaylightTarget(t, field, opening.position, defaults.OutdoorBrightness, true)
				assert.Equal(t, exposure, layer.exposure.cells)
				assert.Equal(t, []daylightOpening{opening}, layer.daylightOpenings)
			})
		}
	}
}

func TestDaylightSourceOpeningPolicy(t *testing.T) {
	t.Parallel()
	allowed := []indexes.SpriteIndex{
		indexes.StoneCrossWindow, indexes.StoneGlassWindow,
		indexes.RegularDoor, indexes.LockedDoor, indexes.RegularDoorView, indexes.LockedDoorView,
		indexes.MagicLockDoor, indexes.MagicLockDoorWithView,
	}
	layer := daylightTestRoom()
	pos := references.Position{X: 1, Y: 5}
	for sprite := indexes.SpriteIndex(0); sprite <= 255; sprite++ {
		opening, ok := daylightOpeningAt(layer.exposure, pos, sprite)
		assert.Equal(t, slices.Contains(allowed, sprite), ok, "source sprite %d", sprite)
		if ok {
			opaqueDoor := sprite == indexes.RegularDoor || sprite == indexes.LockedDoor || sprite == indexes.MagicLockDoor
			assert.Equal(t, opaqueDoor, opening.requiresOpen)
			assert.Equal(t, references.Position{X: 1, Y: 0}, opening.inward)
		}
	}
	for _, test := range []struct {
		name                     string
		west, east, north, south ExposureKind
	}{
		{"internal", ExposureIndoor, ExposureIndoor, ExposureBoundary, ExposureBoundary},
		{"all outdoor", ExposureOutdoor, ExposureOutdoor, ExposureOutdoor, ExposureOutdoor},
		{"outdoor boundary", ExposureOutdoor, ExposureBoundary, ExposureBoundary, ExposureBoundary},
		{"indoor boundary", ExposureIndoor, ExposureBoundary, ExposureBoundary, ExposureBoundary},
		{"outdoor unknown", ExposureOutdoor, ExposureUnknown, ExposureBoundary, ExposureBoundary},
		{"indoor unknown", ExposureIndoor, ExposureUnknown, ExposureBoundary, ExposureBoundary},
		{"two qualifying axes", ExposureOutdoor, ExposureIndoor, ExposureOutdoor, ExposureIndoor},
	} {
		t.Run(test.name, func(t *testing.T) {
			exposure := MapExposure{width: 3, height: 3, cells: []ExposureKind{
				ExposureBoundary, test.north, ExposureBoundary,
				test.west, ExposureBoundary, test.east,
				ExposureBoundary, test.south, ExposureBoundary,
			}}
			for _, sprite := range allowed {
				_, ok := daylightOpeningAt(exposure, references.Position{X: 1, Y: 1}, sprite)
				assert.False(t, ok, "source sprite %d", sprite)
			}
		})
	}
	for _, pos := range []references.Position{
		{X: -1, Y: 5}, {X: 11, Y: 5}, {X: 5, Y: -1}, {X: 5, Y: 11}, {X: 0, Y: 0}, {X: 5, Y: 5},
		{X: math.MinInt16, Y: math.MaxInt16},
	} {
		_, ok := daylightOpeningAt(layer.exposure, pos, indexes.StoneCrossWindow)
		assert.False(t, ok, "invalid opening position %v", pos)
	}
	_, ok := daylightOpeningAt(MapExposure{}, references.Position{}, indexes.RegularDoor)
	assert.False(t, ok)
	// A border window cannot infer an outdoor neighbor beyond the classified floor.
	border := ClassifyMapExposure(3, 3, exposureTestBoundaries([]string{"###", "#I#", "###"}), nil)
	_, ok = daylightOpeningAt(border, references.Position{X: 0, Y: 1}, indexes.StoneCrossWindow)
	assert.False(t, ok)
}

func TestDaylightDoorsUseLiveStateNotRuntimeTiles(t *testing.T) {
	t.Parallel()
	defaults := DefaultDaylightOptions()
	for _, sprite := range []indexes.SpriteIndex{
		indexes.RegularDoor, indexes.LockedDoor, indexes.RegularDoorView,
		indexes.LockedDoorView, indexes.MagicLockDoor, indexes.MagicLockDoorWithView,
	} {
		t.Run(fmt.Sprint(sprite), func(t *testing.T) {
			layer := daylightTestRoom()
			pos, inside := references.Position{X: 1, Y: 5}, references.Position{X: 2, Y: 5}
			opening, ok := daylightOpeningAt(layer.exposure, pos, sprite)
			require.True(t, ok)
			require.Equal(t, !sprite.IsWindowedDoor(), opening.requiresOpen)
			layer.daylightOpenings = []daylightOpening{opening}
			layer.SetTileByLayer(MapLayer, &pos, sprite)
			state := &MapState{}
			field := NewDaylightField(DefaultDaylightOptions())
			for _, open := range []bool{false, true, false, true} {
				state.openDoorPos = nil
				want := defaults.IndoorBrightness
				if open {
					state.openDoorPos = &pos
				}
				if open || sprite.IsWindowedDoor() {
					want = defaults.OutdoorBrightness
				}
				field.PrepareSmallMap(layer, state.IsPositionOpenDoor)
				assertDaylightTarget(t, field, pos, want, true)
				assertDaylightTarget(t, field, inside, want, true)
				assert.Equal(t, open, state.IsPositionOpenDoor(&pos), "window daylight must not open or unlock the door")
				// Replacing both live layers cannot change cached source orientation or activation.
				layer.SetTileByLayer(MapLayer, &pos, indexes.StoneGlassWindow)
				layer.SetTileByLayer(MapOverrideLayer, &pos, indexes.BrickFloor)
			}
			assert.Equal(t, []daylightOpening{opening}, layer.daylightOpenings)
			other := references.Position{X: 9, Y: 5}
			state.openDoorPos = &other
			field.PrepareSmallMap(layer, state.IsPositionOpenDoor)
			want := defaults.IndoorBrightness
			if sprite.IsWindowedDoor() {
				want = defaults.OutdoorBrightness
			}
			assertDaylightTarget(t, field, inside, want, true)
		})
	}
}

func TestDaylightOverlapUsesMaximumWithoutRelaysOrOcclusion(t *testing.T) {
	t.Parallel()
	defaults := DefaultDaylightOptions()
	indoor, outdoor := defaults.IndoorBrightness, defaults.OutdoorBrightness
	near, far := indoor+(outdoor-indoor)*0.6, indoor+(outdoor-indoor)*0.25
	layer := daylightTestRoom()
	for _, pos := range []references.Position{{X: 3, Y: 1}, {X: 5, Y: 1}} {
		opening, ok := daylightOpeningAt(layer.exposure, pos, indexes.StoneCrossWindow)
		require.True(t, ok)
		layer.daylightOpenings = append(layer.daylightOpenings, opening)
	}
	field := NewDaylightField(DefaultDaylightOptions())
	field.PrepareSmallMap(layer, daylightTestClosedDoor)
	assertDaylightTarget(t, field, references.Position{X: 4, Y: 2}, near, true)
	assertDaylightTarget(t, field, references.Position{X: 5, Y: 2}, outdoor, true)
	assertDaylightTarget(t, field, references.Position{X: 4, Y: 3}, far, true)
	assertDaylightTarget(t, field, references.Position{X: 4, Y: 4}, indoor, true)
	first := slices.Clone(field.values)
	slices.Reverse(layer.daylightOpenings)
	field.PrepareSmallMap(layer, daylightTestClosedDoor)
	assert.Equal(t, first, field.values, "opening order cannot matter")
	for _, value := range field.values {
		assert.LessOrEqual(t, value, float32(1))
	}
	// Structural receivers can receive light without occluding it or becoming sources.
	layer.exposure.cells[2*11+3] = ExposureBoundary
	layer.exposure.cells[2*11+4] = ExposureUnknown
	layer.exposure.cells[3*11+5] = ExposureOutdoor
	wall := references.Position{X: 3, Y: 2}
	layer.SetTileByLayer(MapUnitLayer, &wall, indexes.StoneBrickWall)
	door := references.Position{X: 3, Y: 3}
	layer.exposure.cells[3*11+3] = ExposureBoundary
	layer.SetTileByLayer(MapLayer, &door, indexes.RegularDoor)
	_, ok := daylightOpeningAt(layer.exposure, door, indexes.RegularDoor)
	require.False(t, ok, "an internal door is not a source")
	openings, exposure := slices.Clone(layer.daylightOpenings), slices.Clone(layer.exposure.cells)
	field.PrepareSmallMap(layer, daylightTestClosedDoor)
	assertDaylightTarget(t, field, wall, outdoor, true)
	assertDaylightTarget(t, field, references.Position{X: 4, Y: 2}, 0, false)
	assertDaylightTarget(t, field, door, near, true)
	assertDaylightTarget(t, field, references.Position{X: 3, Y: 4}, far, true)
	assertDaylightTarget(t, field, references.Position{X: 5, Y: 3}, outdoor, true)
	assertDaylightTarget(t, field, references.Position{X: 5, Y: 4}, far, true)
	assertDaylightTarget(t, field, references.Position{X: 3, Y: 5}, indoor, true)
	assertDaylightTarget(t, field, references.Position{X: 5, Y: 5}, indoor, true)
	assert.Equal(t, openings, layer.daylightOpenings)
	assert.Equal(t, exposure, layer.exposure.cells)
}

func TestDaylightWindowDoorOverlapOrdering(t *testing.T) {
	t.Parallel()
	defaults := DefaultDaylightOptions()
	layer := daylightTestRoom()
	door := references.Position{X: 4, Y: 1}
	for _, source := range []struct {
		pos    references.Position
		sprite indexes.SpriteIndex
	}{
		{references.Position{X: 3, Y: 1}, indexes.StoneCrossWindow},
		{door, indexes.RegularDoor},
		{references.Position{X: 5, Y: 1}, indexes.StoneGlassWindow},
	} {
		opening, ok := daylightOpeningAt(layer.exposure, source.pos, source.sprite)
		require.True(t, ok)
		layer.daylightOpenings = append(layer.daylightOpenings, opening)
	}
	openings, exposure := slices.Clone(layer.daylightOpenings), slices.Clone(layer.exposure.cells)
	state := &MapState{}
	field := NewDaylightField(defaults)
	for _, open := range []bool{false, true, false} {
		state.openDoorPos = nil
		if open {
			state.openDoorPos = &door
		}
		field.PrepareSmallMap(layer, state.IsPositionOpenDoor)
		first := slices.Clone(field.values)
		// Receiving light from neighboring windows must not activate a closed door.
		for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
			for index, source := range order {
				layer.daylightOpenings[index] = openings[source]
			}
			ordered := slices.Clone(layer.daylightOpenings)
			field.PrepareSmallMap(layer, state.IsPositionOpenDoor)
			assert.Equal(t, first, field.values, "order %v, door open %t", order, open)
			for y := references.Coordinate(0); y < 11; y++ {
				for x := references.Coordinate(0); x < 11; x++ {
					pos := references.Position{X: x, Y: y}
					want := daylightOriginalTarget(layer.exposure, openings, pos, open, defaults)
					assertDaylightTarget(t, field, pos, want, true)
				}
			}
			assert.Equal(t, ordered, layer.daylightOpenings)
			assert.Equal(t, exposure, layer.exposure.cells)
		}
		assertDaylightTarget(t, field, door, defaults.OutdoorBrightness, true)
		want := defaults.IndoorBrightness
		if open {
			want += (defaults.OutdoorBrightness - defaults.IndoorBrightness) * defaults.OpeningFalloff[2]
		}
		assertDaylightTarget(t, field, references.Position{X: 4, Y: 4}, want, true)
	}
}

func TestDaylightCustomAndDisabledFalloff(t *testing.T) {
	t.Parallel()
	layer := daylightTestRoom()
	pos := references.Position{X: 1, Y: 5}
	opening, ok := daylightOpeningAt(layer.exposure, pos, indexes.RegularDoor)
	require.True(t, ok)
	layer.daylightOpenings = []daylightOpening{opening}
	options := DaylightOptions{OutdoorBrightness: 0.9, IndoorBrightness: 0.3, OpeningFalloff: []float32{0.5, 0.25, 0}}
	field := NewDaylightField(options)
	field.PrepareSmallMap(layer, func(*references.Position) bool { return true })
	assertDaylightTarget(t, field, pos, 0.9, true)
	assertDaylightTarget(t, field, references.Position{X: 2, Y: 5}, 0.6, true)
	assertDaylightTarget(t, field, references.Position{X: 3, Y: 5}, 0.45, true)
	assertDaylightTarget(t, field, references.Position{X: 4, Y: 5}, 0.3, true)
	for reach := 1; reach <= 8; reach++ {
		t.Run(fmt.Sprintf("reach %d", reach), func(t *testing.T) {
			custom := options
			custom.OpeningFalloff = make([]float32, reach)
			for band := range custom.OpeningFalloff {
				custom.OpeningFalloff[band] = 1 / float32(band+2)
			}
			field := NewDaylightField(custom)
			field.PrepareSmallMap(layer, func(*references.Position) bool { return true })
			for y := references.Coordinate(0); y < 11; y++ {
				for x := references.Coordinate(0); x < 11; x++ {
					pos := references.Position{X: x, Y: y}
					want := daylightOriginalTarget(layer.exposure, layer.daylightOpenings, pos, true, custom)
					assertDaylightTarget(t, field, pos, want, true)
				}
			}
		})
	}
	for _, weights := range [][]float32{nil, {}} {
		options.OpeningFalloff = weights
		field = NewDaylightField(options)
		field.PrepareSmallMap(layer, func(*references.Position) bool {
			t.Fatal("disabled openings should not consult doors")
			return true
		})
		assertDaylightTarget(t, field, pos, 0.3, true)
		assertDaylightTarget(t, field, references.Position{X: 2, Y: 5}, 0.3, true)
	}
}

func TestDaylightClearBoundsAndBufferReuse(t *testing.T) {
	defaults := DefaultDaylightOptions()
	field := NewDaylightField(defaults)
	pos := references.Position{X: 0, Y: 0}
	assertDaylightTarget(t, field, pos, 0, false)
	layer := daylightTestRoom()
	opening, ok := daylightOpeningAt(layer.exposure, references.Position{X: 1, Y: 5}, indexes.RegularDoor)
	require.True(t, ok)
	layer.daylightOpenings = []daylightOpening{opening}
	state := &MapState{openDoorPos: &opening.position}
	callback := state.IsPositionOpenDoor
	field.PrepareSmallMap(layer, callback)
	buffer := &field.values[0]
	capacity := cap(field.values)
	assert.Zero(t, testing.AllocsPerRun(100, func() { field.PrepareSmallMap(layer, callback) }))
	assert.Same(t, buffer, &field.values[0])
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		field.Clear()
		field.PrepareOutdoor()
		field.PrepareSmallMap(layer, callback)
	}))
	field.Clear()
	assert.Empty(t, field.values)
	assert.Equal(t, capacity, cap(field.values))
	assert.Zero(t, field.width)
	assert.Zero(t, field.height)
	assert.False(t, field.outdoor)
	assertDaylightTarget(t, field, pos, 0, false)
	field.PrepareOutdoor()
	for _, pos := range []references.Position{{X: 0, Y: 0}, {X: -1, Y: -1}, {X: math.MinInt16, Y: math.MaxInt16}} {
		assertDaylightTarget(t, field, pos, defaults.OutdoorBrightness, true)
	}
	for _, absent := range []*LayeredMap{nil, {}} {
		field.PrepareSmallMap(absent, callback)
		assertDaylightTarget(t, field, pos, 0, false)
		assert.False(t, field.outdoor)
		field.PrepareOutdoor()
	}
	for _, candidate := range []*LayeredMap{nil, {}, layer} {
		assert.Panics(t, func() { field.PrepareSmallMap(candidate, nil) })
	}
	// A differently shaped floor rewrites every cell, including unknown values.
	smaller := &LayeredMap{exposure: MapExposure{width: 2, height: 3, cells: []ExposureKind{
		ExposureOutdoor, ExposureIndoor, ExposureBoundary, ExposureUnknown, ExposureIndoor, ExposureOutdoor,
	}}}
	field.PrepareSmallMap(smaller, callback)
	assert.Same(t, buffer, &field.values[0])
	assert.Equal(t, []float32{defaults.OutdoorBrightness, defaults.IndoorBrightness, defaults.IndoorBrightness, 0,
		defaults.IndoorBrightness, defaults.OutdoorBrightness}, field.values)
	assertDaylightTarget(t, field, references.Position{X: 1, Y: 1}, 0, false)
	assertDaylightTarget(t, field, references.Position{X: 1, Y: 2}, defaults.OutdoorBrightness, true)
	for _, outside := range []references.Position{{X: -1, Y: 0}, {X: 0, Y: -1}, {X: 2, Y: 0}, {X: 0, Y: 3}} {
		assertDaylightTarget(t, field, outside, 0, false)
	}
	field.PrepareSmallMap(layer, callback)
	assert.Same(t, buffer, &field.values[0])
	assertDaylightTarget(t, field, references.Position{X: 2, Y: 5}, defaults.OutdoorBrightness, true)
	assert.Equal(t, []daylightOpening{opening}, layer.daylightOpenings)
}

func daylightTestRoom() *LayeredMap {
	rows := []string{
		"...........", ".#########.", ".#IIIIIII#.", ".#IIIIIII#.", ".#IIIIIII#.",
		".#IIIIIII#.", ".#IIIIIII#.", ".#IIIIIII#.", ".#IIIIIII#.", ".#########.", "...........",
	}
	layer := newLayeredMap(11, 11, nil, 11, 11, false)
	layer.exposure = ClassifyMapExposure(11, 11, exposureTestBoundaries(rows), nil)
	return layer
}

func daylightTestClosedDoor(*references.Position) bool { return false }

func assertDaylightTarget(t *testing.T, field *DaylightField, pos references.Position, want float32, valid bool) {
	t.Helper()
	target, ok := field.TargetAt(pos)
	assert.Equal(t, valid, ok, "target validity at %v", pos)
	assert.InDelta(t, want, target, 0.000001, "target at %v", pos)
}
