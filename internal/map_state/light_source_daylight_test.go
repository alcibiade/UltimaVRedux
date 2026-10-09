package map_state

import (
	"math"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

// legacyLightTintOracle freezes the pre-daylight arithmetic independently of the
// production composition, clamp, and breathing helpers. Properties remain data.
func legacyLightTintOracle(contributions []LightContribution, elapsedMs int64, ambientDarkness float32) *LightTintColor {
	darkness := math.Max(0, math.Min(1, float64(ambientDarkness)))
	var deltas [3]float64
	var totalWeight float64
	closestDist, closestRadius := 1000.0, 0
	for _, contribution := range contributions {
		props := GetLightSourceProperties(contribution.SourceIndex)
		multiplier := 1.0
		switch props.TintMode {
		case TintNightOnly:
			if darkness == 0 {
				continue
			}
			multiplier = darkness
		case TintHalfDuringDay:
			multiplier = 0.5 + 0.5*darkness
		}

		breath := 1.0
		if props.HasBreathing {
			period := props.BreathingPeriodMs
			if period == 0 {
				period = 2250
			}
			amplitude := props.BreathingAmplitude
			if amplitude == 0 {
				amplitude = 0.065
			}
			hash := int32(contribution.SourcePos.X)*256 + int32(contribution.SourcePos.Y)
			if props.IndependentBreathing {
				variation := 0.85 + 0.30*float64(hash%100)/100.0
				period = int(float64(period) * variation)
			}
			adjusted := elapsedMs + int64(hash)*17
			phase := float64(adjusted%int64(period)) / float64(period) * 2 * math.Pi
			breath = 1.0 + math.Sin(phase)*amplitude
		}
		for channel, tint := range [3]float32{props.TintColor.R, props.TintColor.G, props.TintColor.B} {
			delta := (float64(tint) - 1.0) * multiplier
			if props.HasBreathing {
				delta *= breath
			}
			delta *= 0.25
			deltas[channel] += delta * contribution.Weight
		}
		totalWeight += contribution.Weight
		if contribution.Distance < closestDist {
			closestDist, closestRadius = contribution.Distance, contribution.Radius
		}
	}
	if totalWeight == 0 {
		return nil
	}
	brightness := 1.0
	if darkness > 0 && closestRadius > 0 {
		ratio := math.Max(0, math.Min(1, closestDist/(float64(closestRadius)+0.5)))
		night := 1.05 + (0.55-1.05)*ratio
		brightness = 1 + (night-1)*darkness
	}
	var result [3]float32
	for channel, delta := range deltas {
		result[channel] = float32(math.Max(0.3, math.Min(1.5, brightness+delta/totalWeight)))
	}
	return &LightTintColor{R: result[0], G: result[1], B: result[2]}
}

func TestLightSourceLegacyFormula(t *testing.T) {
	pos := references.Position{X: 12, Y: 10}
	contribution := func(index indexes.SpriteIndex, sourcePos references.Position, distance float64, radius int) LightContribution {
		return LightContribution{
			SourceIndex: index, SourcePos: sourcePos, Distance: distance, Radius: radius,
			Weight: 1 / math.Pow(distance+1, 1.5),
		}
	}
	tests := []struct {
		name          string
		contributions []LightContribution
	}{
		{name: "unlit"},
		{name: "empty", contributions: []LightContribution{}},
		{name: "night_only_marker", contributions: []LightContribution{contribution(0, pos, 0, 2)}},
		{name: "unknown_sprite", contributions: []LightContribution{contribution(1, pos, 2.5, 2)}},
		{name: "steady_center", contributions: []LightContribution{contribution(indexes.LampPost, pos, 0, 2)}},
		{name: "steady_edge", contributions: []LightContribution{contribution(indexes.CookStove, pos, 3.5, 3)}},
		{name: "independent_diagonal", contributions: []LightContribution{contribution(indexes.RightSconce, pos, math.Sqrt(8), 3)}},
		{name: "default_breathing", contributions: []LightContribution{contribution(indexes.Brazier, pos, 1, 2)}},
		{name: "wide_position_hash", contributions: []LightContribution{contribution(indexes.LeftSconce, references.Position{X: 255, Y: 254}, 1, 2)}},
		{name: "negative_position_hash", contributions: []LightContribution{contribution(indexes.Fireplace, references.Position{X: -1, Y: -2}, 2, 3)}},
		{name: "always_tint", contributions: []LightContribution{contribution(indexes.BlueFlame, pos, 3.5, 3)}},
		{name: "custom_breathing_period", contributions: []LightContribution{contribution(MoongateMarkerIndex, pos, 0, 4)}},
		{name: "lower_channel_clamp", contributions: []LightContribution{contribution(indexes.RightSconce, references.Position{}, 2.5, 2)}},
		{name: "zero_radius", contributions: []LightContribution{contribution(indexes.CookStove, pos, 0, 0)}},
		{name: "negative_radius", contributions: []LightContribution{contribution(indexes.LampPost, pos, 0, -1)}},
		{name: "distance_clamp", contributions: []LightContribution{contribution(indexes.LampPost, pos, 9, 2)}},
		{name: "zero_weight", contributions: []LightContribution{{SourceIndex: indexes.LampPost, Radius: 2}}},
		{name: "filtered_closest", contributions: []LightContribution{
			contribution(0, pos, 0, 2), contribution(indexes.LampPost, references.Position{X: 10, Y: 10}, 2, 2),
		}},
		{name: "equal_distances_keep_first_radius", contributions: []LightContribution{
			contribution(indexes.LampPost, pos, 2, 2), contribution(indexes.CookStove, pos, 2, 5),
		}},
		{name: "four_overlapping_sources", contributions: []LightContribution{
			contribution(indexes.RightSconce, references.Position{X: 10, Y: 10}, 2, 3),
			contribution(indexes.BlueFlame, references.Position{X: 12, Y: 11}, 1, 3),
			contribution(MoongateMarkerIndex, references.Position{X: 13, Y: 10}, 1, 4),
			contribution(0, pos, 0, 2),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := NewLightSourceTintingState()
			if test.contributions != nil {
				state.litTilesMap[pos] = test.contributions
			}
			for _, elapsed := range []int64{-2251, 0, 1, 375, 562, 750, 1125, 1499, 1500, 2250, 4499, 987654321} {
				for _, darkness := range []float32{-1, 0, math.SmallestNonzeroFloat32, 0.01, 0.25, 0.5, 0.73, math.Nextafter32(1, 0), 1, 2} {
					want := legacyLightTintOracle(test.contributions, elapsed, darkness)
					require.Equal(t, want, state.GetTintScaleFactors(pos, elapsed, darkness), "legacy elapsed=%d darkness=%g", elapsed, darkness)
					require.Equal(t, want, state.GetTintScaleFactorsWithDaylight(pos, elapsed, darkness, 1), "outdoors elapsed=%d darkness=%g", elapsed, darkness)
					if darkness >= 1 {
						for _, target := range []float32{0.3, 0.4, 0.75, math.Nextafter32(1, 0)} {
							require.Equal(t, want, state.GetTintScaleFactorsWithDaylight(pos, elapsed, darkness, target), "night elapsed=%d darkness=%g target=%g", elapsed, darkness, target)
						}
					}
				}
			}
		})
	}
}

func TestDaylightTintComposition(t *testing.T) {
	tests := []struct {
		name          string
		distance      float64
		radius        int
		target        float32
		dayBrightness float64
	}{
		{name: "center_caps_scalar_at_one", radius: 2, target: 0.3, dayBrightness: 1},
		{name: "near_source_lifts_indoor_brightness", distance: 1.25, radius: 2, target: 0.4, dayBrightness: 0.8},
		{name: "far_source_keeps_indoor_floor", distance: 2.5, radius: 2, target: 0.8, dayBrightness: float64(float32(0.8))},
		{name: "minimum_target_still_receives_light", distance: 2.5, radius: 2, target: 0.3, dayBrightness: 0.55},
		{name: "outdoor_baseline", distance: 2.5, radius: 2, target: 1, dayBrightness: 1},
		{name: "zero_radius_keeps_legacy_default", target: 0.3, dayBrightness: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pos := references.Position{X: 8, Y: 8}
			state := NewLightSourceTintingState()
			state.litTilesMap[pos] = []LightContribution{{
				SourceIndex: indexes.CookStove, SourcePos: pos, Distance: test.distance, Radius: test.radius, Weight: 1,
			}}
			for _, darkness := range []float32{0, 0.25, 0.5, 0.9, 1, 2} {
				nightFraction := math.Min(1, float64(darkness))
				nightBrightness := 1.0
				if test.radius > 0 {
					ratio := test.distance / (float64(test.radius) + 0.5)
					nightBrightness = 1.05 + (0.55-1.05)*ratio
				}
				brightness := test.dayBrightness + (nightBrightness-test.dayBrightness)*nightFraction
				if test.target == 1 || nightFraction == 1 {
					brightness = 1 + (nightBrightness-1)*nightFraction
				}
				var channels [3]float32
				for channel, tint := range [3]float32{1.55, 0.75, 0.15} {
					delta := (float64(tint) - 1) * (0.5 + 0.5*nightFraction)
					delta *= 0.25
					channels[channel] = float32(math.Max(0.3, math.Min(1.5, brightness+delta)))
				}
				want := &LightTintColor{R: channels[0], G: channels[1], B: channels[2]}
				got := state.GetTintScaleFactorsWithDaylight(pos, 1125, darkness, test.target)
				require.Equal(t, want, got, "darkness=%g", darkness)
				if darkness == 0 && test.name == "far_source_keeps_indoor_floor" {
					require.Greater(t, got.R, test.target)
					require.Less(t, got.G, test.target, "daylight is a scalar floor, not a per-channel hue clamp")
					require.Greater(t, got.G, got.B)
				}
			}
		})
	}
}

func TestDaylightTintFilteringAndFallback(t *testing.T) {
	pos := references.Position{X: 8, Y: 8}
	nightOnly := LightContribution{SourceIndex: 0, SourcePos: pos, Radius: 2, Weight: 1}
	stove := LightContribution{SourceIndex: indexes.CookStove, SourcePos: references.Position{X: 10, Y: 8}, Distance: 2.5, Radius: 2, Weight: 0.5}
	state := NewLightSourceTintingState()
	for _, target := range []float32{0.3, 0.75, 1} {
		require.Nil(t, state.GetTintScaleFactorsWithDaylight(pos, 0, 0, target))
	}
	for _, contributions := range [][]LightContribution{
		{}, {nightOnly}, {{SourceIndex: indexes.LampPost, Radius: 2, Weight: 0}},
	} {
		state.litTilesMap[pos] = contributions
		for _, target := range []float32{0.3, 0.75, 1} {
			require.Nil(t, state.GetTintScaleFactorsWithDaylight(pos, 0, 0, target))
		}
	}

	state.litTilesMap[pos] = []LightContribution{nightOnly, stove}
	got := state.GetTintScaleFactorsWithDaylight(pos, 0, 0, 0.75)
	want := &LightTintColor{
		R: float32(0.75 + (float64(float32(1.55))-1)*0.5*0.25),
		G: float32(0.75 + (float64(float32(0.75))-1)*0.5*0.25),
		B: float32(0.75 + (float64(float32(0.15))-1)*0.5*0.25),
	}
	require.Equal(t, want, got, "filtered center must affect neither brightness nor color weights")
	state.litTilesMap[pos] = []LightContribution{nightOnly}
	require.Equal(t, &LightTintColor{R: 0.675, G: 0.675, B: 0.675}, state.GetTintScaleFactorsWithDaylight(pos, 0, 0.5, 0.3))
	state.Clear()
	require.Nil(t, state.GetTintScaleFactorsWithDaylight(pos, 0, 1, 0.3))
}

func TestDaylightTintNightOnlyEndpoint(t *testing.T) {
	const target float32 = 0.86
	pos := references.Position{X: 8, Y: 8}
	for _, source := range []struct {
		name     string
		distance float64
		radius   int
	}{
		{name: "center", radius: 2},
		{name: "edge", distance: 2.5, radius: 2},
		{name: "zero_radius"},
	} {
		t.Run(source.name, func(t *testing.T) {
			for _, transition := range []struct {
				name     string
				darkness []float32
			}{
				{name: "dusk", darkness: []float32{0, math.SmallestNonzeroFloat32, 0.001, 1.0 / 30, 0.5, 1}},
				{name: "dawn", darkness: []float32{1, 0.5, 1.0 / 30, 0.001, math.SmallestNonzeroFloat32, 0}},
			} {
				t.Run(transition.name, func(t *testing.T) {
					state := NewLightSourceTintingState()
					for _, darkness := range transition.darkness {
						// Model source appearance/removal at the global daylight boundary.
						state.Clear()
						if darkness > 0 {
							state.litTilesMap[pos] = []LightContribution{{
								SourceIndex: 0, SourcePos: pos, Distance: source.distance, Radius: source.radius, Weight: 1,
							}}
						}
						got := state.GetTintScaleFactorsWithDaylight(pos, 0, darkness, target)
						if darkness == 0 {
							require.Nil(t, got, "unlit ambient fallback remains the caller's responsibility")
							got = &LightTintColor{R: target, G: target, B: target}
						}
						nightBrightness := 1.0
						if source.radius > 0 {
							ratio := source.distance / (float64(source.radius) + 0.5)
							nightBrightness = 1.05 + (0.55-1.05)*ratio
						}
						brightness := float64(target) + (nightBrightness-float64(target))*float64(darkness)
						if darkness == 1 {
							brightness = 1 + (nightBrightness-1)*float64(darkness)
						}
						want := float32(brightness)
						require.Equal(t, &LightTintColor{R: want, G: want, B: want}, got, "darkness=%g", darkness)
						if darkness == math.SmallestNonzeroFloat32 {
							require.Equal(t, target, got.R, "night-only source must not lift the zero-darkness limit")
						}
						legacy := legacyLightTintOracle(state.litTilesMap[pos], 0, darkness)
						require.Equal(t, legacy, state.GetTintScaleFactors(pos, 0, darkness))
						require.Equal(t, legacy, state.GetTintScaleFactorsWithDaylight(pos, 0, darkness, 1))
						if darkness == 1 {
							require.Equal(t, legacy, got)
						}
					}
				})
			}
		})
	}
}

func TestDaylightTintSeparateDayAndNightSources(t *testing.T) {
	pos := references.Position{}
	for _, source := range []struct {
		name   string
		index  indexes.SpriteIndex
		tint   [3]float32
		always bool
	}{
		{name: "half_day", index: indexes.LampPost, tint: [3]float32{1.25, 1.15, 0.90}},
		{name: "always", index: indexes.BlueFlame, tint: [3]float32{0.70, 0.85, 1.45}, always: true},
	} {
		t.Run(source.name, func(t *testing.T) {
			for _, radius := range []int{0, 2} {
				state := NewLightSourceTintingState()
				state.litTilesMap[pos] = []LightContribution{
					{SourceIndex: 0, SourcePos: pos, Radius: 2, Weight: 1},
					{SourceIndex: source.index, SourcePos: references.Position{X: 2}, Distance: 2, Radius: radius, Weight: 0.25},
				}
				// Blue flame's position phase is zero at 296ms, so breathing is exactly 1.
				const elapsed int64 = 296
				for _, target := range []float32{0.4, 0.86} {
					daySourceBrightness := 1.0
					if radius > 0 {
						ratio := 2 / (float64(radius) + 0.5)
						daySourceBrightness = 1.05 + (0.55-1.05)*ratio
					}
					dayBrightness := math.Max(float64(target), math.Min(1, daySourceBrightness))
					for _, darkness := range []float32{0, math.SmallestNonzeroFloat32, 0.001, 0.5, 1} {
						brightness := dayBrightness + (1.05-dayBrightness)*float64(darkness)
						if darkness == 1 {
							brightness = 1 + (1.05-1)*float64(darkness)
						}
						multiplier := 0.5 + 0.5*float64(darkness)
						if source.always {
							multiplier = 1
						}
						// Preserve the preexisting normalization change when night-only weight joins.
						totalWeight := 0.25
						if darkness > 0 {
							totalWeight += 1
						}
						var channels [3]float32
						for channel, tint := range source.tint {
							delta := (float64(tint) - 1) * multiplier
							delta *= 0.25
							average := delta * 0.25 / totalWeight
							channels[channel] = float32(math.Max(0.3, math.Min(1.5, brightness+average)))
						}
						want := &LightTintColor{R: channels[0], G: channels[1], B: channels[2]}
						got := state.GetTintScaleFactorsWithDaylight(pos, elapsed, darkness, target)
						require.Equal(t, want, got, "radius=%d target=%g darkness=%g", radius, target, darkness)
						legacy := legacyLightTintOracle(state.litTilesMap[pos], elapsed, darkness)
						require.Equal(t, legacy, state.GetTintScaleFactors(pos, elapsed, darkness))
						require.Equal(t, legacy, state.GetTintScaleFactorsWithDaylight(pos, elapsed, darkness, 1))
						if darkness == 1 {
							require.Equal(t, legacy, got)
						}
					}
				}
			}
		})
	}
}

func TestDaylightTintOverlappingSourceSelection(t *testing.T) {
	pos := references.Position{X: 10, Y: 10}
	sources := LightSources{
		{Tile: &references.Tile{Index: indexes.RightSconce, LightEmission: 5}, Pos: references.Position{X: 14, Y: 10}},
		{Tile: &references.Tile{Index: indexes.BlueFlame, LightEmission: 5}, Pos: references.Position{X: 13, Y: 10}},
		{Tile: &references.Tile{Index: indexes.Brazier, LightEmission: 5}, Pos: references.Position{X: 12, Y: 10}},
		{Tile: &references.Tile{Index: indexes.CookStove, LightEmission: 5}, Pos: references.Position{X: 11, Y: 10}},
		{Tile: &references.Tile{Index: indexes.LampPost, LightEmission: 5}, Pos: pos},
	}
	state := NewLightSourceTintingState()
	state.BuildLitTilesMap(sources, nil)
	expected := make([]LightContribution, 0, 4)
	for _, sourceIndex := range []int{4, 1, 2, 3} {
		source := sources[sourceIndex]
		distance := float64(source.Pos.X - pos.X)
		expected = append(expected, LightContribution{
			SourceIndex: source.Tile.Index, SourcePos: source.Pos, Distance: distance, Radius: 5,
			Weight: 1 / math.Pow(distance+1, 1.5),
		})
	}
	require.Equal(t, expected, state.litTilesMap[pos], "fifth source replaces weakest without reordering the rest")
	for _, elapsed := range []int64{0, 562, 1125, 2250} {
		for _, darkness := range []float32{0, 0.25, 0.5, 1} {
			want := legacyLightTintOracle(expected, elapsed, darkness)
			require.Equal(t, want, state.GetTintScaleFactors(pos, elapsed, darkness))
			// The closest source is at its center: every daylight target has the same capped endpoint.
			for _, target := range []float32{0.3, 0.75, 1} {
				require.Equal(t, want, state.GetTintScaleFactorsWithDaylight(pos, elapsed, darkness, target))
			}
		}
	}
}

func TestDaylightTintTargetValidation(t *testing.T) {
	state := NewLightSourceTintingState()
	pos := references.Position{}
	for _, target := range []float32{
		-1, 0, math.Nextafter32(0.3, 0), math.Nextafter32(1, 2), 2,
		float32(math.NaN()), float32(math.Inf(-1)), float32(math.Inf(1)),
	} {
		require.PanicsWithValue(t, "daylight target must be finite and in [0.3, 1]", func() {
			state.GetTintScaleFactorsWithDaylight(pos, 0, 0, target)
		}, "target=%g must be rejected even without contributions", target)
	}
	for _, target := range []float32{0.3, math.Nextafter32(0.3, 1), 0.75, math.Nextafter32(1, 0), 1} {
		require.NotPanics(t, func() {
			require.Nil(t, state.GetTintScaleFactorsWithDaylight(pos, 0, 0, target))
		}, "target=%g", target)
	}
}
