package map_state

import (
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	minimumDaylightBrightness = 0.3
	maximumDaylightReach      = 8
)

// DaylightOptions controls cosmetic full-day targets, not visibility or emission.
// Brightness values must be finite in [0.3, 1], with IndoorBrightness <= OutdoorBrightness.
// OpeningFalloff contains at most eight finite, nonincreasing weights in [0, 1].
// Weight d-1 blends toward outdoor brightness in the Euclidean band (d-1,d].
// The pool faces inward and includes structural tiles along the opening plane.
// An empty slice disables all opening effects.
type DaylightOptions struct {
	OutdoorBrightness float32
	IndoorBrightness  float32
	OpeningFalloff    []float32
}

// DefaultDaylightOptions returns independent options: outdoor 1, indoor 0.75,
// and inward opening weights 1, 0.6, 0.25 at distances one, two, and three.
// The indoor baseline remains a visual-tuning choice.
func DefaultDaylightOptions() DaylightOptions {
	//nolint:mnd // Central cosmetic daylight policy, intentionally defined in one place.
	return DaylightOptions{OutdoorBrightness: 1, IndoorBrightness: 0.75, OpeningFalloff: []float32{1, 0.6, 0.25}}
}

// DaylightField holds full-day cosmetic targets. The caller owns world-context
// eligibility and time-of-day blending. Construct with NewDaylightField.
// Preparing a previously sized floor reuses storage without retaining its map or callback.
type DaylightField struct {
	options       DaylightOptions
	values        []float32 // Row-major; zero means no target, below every valid brightness.
	width, height int
	outdoor       bool
}

// NewDaylightField validates and copies options, including the falloff slice.
// Invalid configuration is a programming error and panics.
func NewDaylightField(options DaylightOptions) *DaylightField {
	for _, brightness := range [...]float32{options.OutdoorBrightness, options.IndoorBrightness} {
		if !(brightness >= minimumDaylightBrightness && brightness <= 1) {
			panic("daylight brightness must be finite and in [0.3, 1]")
		}
	}
	if options.IndoorBrightness > options.OutdoorBrightness {
		panic("daylight indoor brightness must not exceed outdoor brightness")
	}
	if len(options.OpeningFalloff) > maximumDaylightReach {
		panic("daylight opening falloff must contain at most eight weights")
	}
	previous := float32(1)
	for _, weight := range options.OpeningFalloff {
		if !(weight >= 0 && weight <= previous) {
			panic("daylight opening weights must be finite, nonincreasing, and in [0, 1]")
		}
		previous = weight
	}
	options.OpeningFalloff = slices.Clone(options.OpeningFalloff)

	return &DaylightField{options: options, values: nil, width: 0, height: 0, outdoor: false}
}

// Clear deactivates all targets while retaining the values buffer for reuse.
func (d *DaylightField) Clear() {
	d.values = d.values[:0]
	d.width, d.height = 0, 0
	d.outdoor = false
}

// PrepareOutdoor selects a uniform outdoor target without coordinate bounds.
func (d *DaylightField) PrepareOutdoor() {
	d.Clear()
	d.outdoor = true
}

// PrepareSmallMap recomputes baselines and live opening activation. Nil maps and
// unknown exposure produce no targets. isDoorOpen is required, must only observe
// its position argument, and should be the current MapState.IsPositionOpenDoor.
// Runtime floor overrides never determine opening identity or orientation.
func (d *DaylightField) PrepareSmallMap(layer *LayeredMap, isDoorOpen func(*references.Position) bool) {
	if isDoorOpen == nil {
		panic("daylight requires an isDoorOpen callback")
	}
	d.Clear()
	if layer == nil {
		return
	}
	exposure := layer.exposure
	d.width, d.height = exposure.width, exposure.height
	count := len(exposure.cells)
	if cap(d.values) < count {
		d.values = make([]float32, count)
	} else {
		d.values = d.values[:count]
	}
	for cell, kind := range exposure.cells {
		switch kind {
		case ExposureOutdoor:
			d.values[cell] = d.options.OutdoorBrightness
		case ExposureIndoor, ExposureBoundary:
			d.values[cell] = d.options.IndoorBrightness
		case ExposureUnknown:
			d.values[cell] = 0
		}
	}
	if len(d.options.OpeningFalloff) == 0 {
		return
	}
	for index := range layer.daylightOpenings {
		opening := &layer.daylightOpenings[index]
		if opening.requiresOpen && !isDoorOpen(&opening.position) {
			continue
		}
		d.applyOpening(opening, exposure)
	}
}

// TargetAt returns the full-day target, or false for inactive/unknown/out-of-bounds cells.
func (d *DaylightField) TargetAt(pos references.Position) (float32, bool) {
	if d.outdoor {
		return d.options.OutdoorBrightness, true
	}
	x, y := int(pos.X), int(pos.Y)
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return 0, false
	}
	value := d.values[y*d.width+x]

	return value, value != 0
}

// daylightOpening is immutable source-floor metadata, never a live light source.
type daylightOpening struct {
	position     references.Position
	inward       references.Position
	requiresOpen bool
}

func (d *DaylightField) applyOpening(opening *daylightOpening, exposure MapExposure) {
	ox, oy := int(opening.position.X), int(opening.position.Y)
	d.values[oy*d.width+ox] = d.options.OutdoorBrightness
	reach := len(d.options.OpeningFalloff)
	for row := max(0, oy-reach); row <= min(d.height-1, oy+reach); row++ {
		for column := max(0, ox-reach); column <= min(d.width-1, ox+reach); column++ {
			dx, dy := column-ox, row-oy
			forward := dx*int(opening.inward.X) + dy*int(opening.inward.Y)
			distanceSquared := dx*dx + dy*dy
			kind := exposure.At(column, row)
			if distanceSquared == 0 || distanceSquared > reach*reach || forward < 0 ||
				(kind != ExposureIndoor && kind != ExposureBoundary) || (forward == 0 && kind != ExposureBoundary) {
				continue
			}
			// Integer squared thresholds give ceil-distance bands without admitting
			// pixels outside the radius. Structural receivers do not relay light.
			distance := 1
			for distance*distance < distanceSquared {
				distance++
			}
			target := d.options.IndoorBrightness +
				(d.options.OutdoorBrightness-d.options.IndoorBrightness)*d.options.OpeningFalloff[distance-1]
			cell := row*d.width + column
			d.values[cell] = max(d.values[cell], target)
		}
	}
}

func collectDaylightOpenings(
	source *references.SmallLocationReference, floor references.FloorNumber, exposure MapExposure,
) []daylightOpening {
	var openings []daylightOpening
	for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			pos := references.Position{X: x, Y: y}
			if opening, ok := daylightOpeningAt(exposure, pos, source.GetTileNumber(pos, floor)); ok {
				openings = append(openings, opening)
			}
		}
	}

	return openings
}

func daylightOpeningAt(
	exposure MapExposure, pos references.Position, sprite indexes.SpriteIndex,
) (daylightOpening, bool) {
	isWindow := sprite == indexes.StoneCrossWindow || sprite == indexes.StoneGlassWindow || sprite.IsWindowedDoor()
	opening := daylightOpening{
		position: pos, inward: references.Position{X: 0, Y: 0}, requiresOpen: sprite.IsDoor() && !isWindow,
	}
	if !opening.requiresOpen && !isWindow {
		return opening, false
	}
	if exposure.At(int(pos.X), int(pos.Y)) != ExposureBoundary {
		return opening, false
	}
	matches := 0
	for _, direction := range [...]references.Position{{X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: -1}, {X: 0, Y: 1}} {
		x, y := int(pos.X), int(pos.Y)
		dx, dy := int(direction.X), int(direction.Y)
		if exposure.At(x+dx, y+dy) == ExposureIndoor && exposure.At(x-dx, y-dy) == ExposureOutdoor {
			opening.inward = direction
			matches++
		}
	}

	return opening, matches == 1
}
