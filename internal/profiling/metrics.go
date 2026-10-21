// Package profiling supplies opt-in presentation diagnostics. Its wall-clock
// samples never drive gameplay, input acceptance, or animation decisions.
package profiling

import (
	"sort"
	"sync/atomic"
	"time"
)

// Counter identifies measured resource work, independently of sampled CPU time.
type Counter int

// Resource-work counters distinguish rebuilds from ordinary cached composition.
const (
	FontCreates Counter = iota
	ProvisionFontRebuilds
	TextHits
	TextMisses
	TextEvictions
	ReadyRebuilds
	ReadySelections
	FontSourceParses
	TextLayoutBuilds
	TextCacheEntries
	TextCacheBytes
	TextCachePeakBytes
	TextCacheBypasses
	DitherBuilds
	OuterDitherBuilds
	counterCount
)

const retainedSamples = 512

type durationSamples struct {
	count  atomic.Uint64
	values [retainedSamples]atomic.Int64
}

// Metrics holds bounded samples and counters safe to read from the profile server.
type Metrics struct {
	counters [counterCount]atomic.Uint64
	updates  durationSamples
	draws    durationSamples
	ready    durationSamples
}

//nolint:gochecknoglobals // Startup-owned diagnostic sink shared by presentation and HTTP readers.
var active atomic.Pointer[Metrics]

// Enable starts a fresh diagnostic session; production enables it at startup.
func Enable() *Metrics {
	metrics := new(Metrics)
	active.Store(metrics)

	return metrics
}

// Count records work only while diagnostics are enabled.
func Count(counter Counter) {
	if metrics := active.Load(); metrics != nil {
		metrics.counters[counter].Add(1)
	}
}

// SetGauge publishes resource accounting only while diagnostics are enabled.
func SetGauge(counter Counter, value uint64) {
	if metrics := active.Load(); metrics != nil {
		metrics.counters[counter].Store(value)
	}
}

// Begin returns zero when disabled, avoiding wall-time reads in ordinary play.
func Begin() time.Time {
	if active.Load() == nil {
		return time.Time{}
	}

	return time.Now()
}

func (samples *durationSamples) record(start time.Time) {
	if start.IsZero() {
		return
	}
	// Timing producers run on the single application update/draw thread.
	index := samples.count.Load()
	samples.values[index%retainedSamples].Store(time.Since(start).Nanoseconds())
	samples.count.Store(index + 1)
}

// EndUpdate records the complete application Update, including debug processing.
func EndUpdate(start time.Time) {
	if metrics := active.Load(); metrics != nil {
		metrics.updates.record(start)
	}
}

// EndDraw records a content-rendered Draw, excluding the frame-limiter early return.
func EndDraw(start time.Time) {
	if metrics := active.Load(); metrics != nil {
		metrics.draws.record(start)
	}
}

// EndReadySelection measures latest accepted arrow selection to its next Draw.
// Multiple accepted moves before a Draw coalesce; this is not OS keypress latency.
func EndReadySelection(start time.Time) {
	if metrics := active.Load(); metrics != nil {
		metrics.ready.record(start)
	}
}

// DurationSummary reports cumulative sample count and recent wall-time quantiles.
// Concurrent reads are an approximate rolling window, not an atomic frame trace.
type DurationSummary struct {
	Count   uint64  `json:"count"`
	Samples int     `json:"recentSamples"`
	P50Ms   float64 `json:"p50Ms"`
	P95Ms   float64 `json:"p95Ms"`
	P99Ms   float64 `json:"p99Ms"`
	MaxMs   float64 `json:"recentMaxMs"`
}

func (samples *durationSamples) snapshot() DurationSummary {
	count := samples.count.Load()
	values := make([]int64, 0, min(count, retainedSamples))
	for index := range min(count, retainedSamples) {
		values = append(values, samples.values[index].Load())
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	var result DurationSummary
	result.Count, result.Samples = count, len(values)
	if len(values) != 0 {
		const median, high, tail = 50, 95, 99
		quantile := func(percent int) float64 {
			return float64(values[(len(values)-1)*percent/100]) / float64(time.Millisecond)
		}
		result.P50Ms, result.P95Ms, result.P99Ms = quantile(median), quantile(high), quantile(tail)
		result.MaxMs = float64(values[len(values)-1]) / float64(time.Millisecond)
	}

	return result
}

// Snapshot contains only counters/timing storage, never live game state.
type Snapshot struct {
	Counters               map[string]uint64 `json:"counters"`
	Update                 DurationSummary   `json:"update"`
	ContentDraw            DurationSummary   `json:"contentDraw"`
	ReadySelectionNextDraw DurationSummary   `json:"readySelectionNextDraw"`
}

// Snapshot allocates/serializes only on request, not inside the frame loop.
func (metrics *Metrics) Snapshot() Snapshot {
	names := [...]string{
		"font_creates", "provision_font_rebuilds", "text_hits", "text_misses",
		"text_evictions", "ready_rebuilds", "ready_selections",
		"font_source_parses",
		"text_layout_builds",
		"text_cache_entries", "text_cache_bytes", "text_cache_peak_bytes", "text_cache_bypasses",
		"dither_builds", "outer_dither_builds",
	}
	result := Snapshot{
		Counters: make(map[string]uint64, counterCount), Update: metrics.updates.snapshot(),
		ContentDraw: metrics.draws.snapshot(), ReadySelectionNextDraw: metrics.ready.snapshot(),
	}
	for index, name := range names {
		result.Counters[name] = metrics.counters[index].Load()
	}

	return result
}
