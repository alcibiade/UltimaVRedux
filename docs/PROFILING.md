# Profiling and Performance Optimization

Use profiling data to identify a measured bottleneck, make the smallest relevant change, and repeat the same measurement to verify the result. This document is the authoritative project guide for both profiling and proven optimization patterns.

## Profile Output

Keep generated profiles under the project-local `PROFILING/` directory. The directory is created on demand and profile files are not source artifacts.

```bash
mkdir -p PROFILING
```

Use workload-specific names so that the scenario and before/after state remain clear:

```text
PROFILING/movement_before_cpu.prof
PROFILING/movement_after_cpu.prof
PROFILING/combat_before_heap.prof
```

Do not mix profile files into the repository root or `bin/`.

## Build and Capture

Build the normal optimized binary with debug information retained by Go:

```bash
make build
```

### File-Based Profiles

The game supports `-cpuprofile` and `-memprofile`. Both profiles are finalized when the game exits, so close the game normally before analysis.

Startup/profile failures return through the application boundary so deferred CPU
profile cleanup still runs. Profile-status diagnostics use the centralized
`system` logging category rather than unconditional terminal prints; enable the
applicable logging controls to see those messages. Profile capture itself does
not depend on logging being enabled. The opt-in loopback HTTP server limits header
and idle waits, while leaving long CPU captures free of a response-write deadline.

```bash
./bin/ultimav \
  -cpuprofile=PROFILING/movement_cpu.prof \
  -memprofile=PROFILING/movement_heap.prof
```

The memory flag writes a heap profile after a forced garbage collection. Select an in-use or cumulative-allocation sample index during analysis; use the live endpoints when a profile is needed before exit.

### Live Profiles

Start the loopback-only pprof server:

```bash
./bin/ultimav -webprofile
```

Capture profiles from another terminal while reproducing the workload:

```bash
curl -o PROFILING/combat_cpu.prof \
  "http://localhost:6060/debug/pprof/profile?seconds=30"
curl -o PROFILING/combat_heap.prof \
  "http://localhost:6060/debug/pprof/heap"
curl -o PROFILING/combat_allocs.prof \
  "http://localhost:6060/debug/pprof/allocs"
curl -o PROFILING/combat_goroutine.prof \
  "http://localhost:6060/debug/pprof/goroutine"
```

Use CPU capture intervals long enough to contain repeated examples of the behavior under investigation. Thirty seconds is a useful starting point, not a required duration.

### Opt-In Frame And Cache Metrics

`-webprofile` also enables `/debug/performance` on the same loopback listener.
It serializes diagnostic atomics from `internal/profiling`, never live gameplay
state. Capture JSON snapshots before and after an interval and subtract counts:

```bash
curl -o PROFILING/ready_before_metrics.json http://localhost:6060/debug/performance
curl -o PROFILING/ready_after_metrics.json http://localhost:6060/debug/performance
```

Counters include font creation, provision-font initialization, text cache
hits/misses/evictions, Ready parent rebuilds, and accepted Ready arrow-selection
changes. Timing summaries report cumulative Update/content-Draw counts and
p50/p95/p99/max over at most 512 recent wall-time samples. Content Draw is counted
after the frame limiter admits it; delta counts divided by interval duration
distinguish actual content rendering from engine Draw calls. Draw timing covers
application composition, not later render-thread/GPU completion.

Ready timing measures the latest accepted arrow-selection change to its next
application Draw. Moves accepted before that Draw coalesce; this does not measure
OS keypress delivery or debounce/release rejection. Counter deltas identify that
coalescing. Quantiles are approximate concurrent rolling snapshots, not a complete
frame trace. Fetching/encoding a snapshot allocates only on request.

Outside `-webprofile`, timing does not read wall time or retain samples, and the
resource-counter calls take only a disabled atomic-pointer check. Diagnostic
wall time never influences gameplay or input policy.

Additional counters distinguish lightweight font-face construction from actual
source parsing, retained line-layout builds, and visibility/outer-dither mask
builds. Text-cache entries/current bytes/peak bytes are **gauges**, not cumulative
event counts; oversized-entry bypasses are cumulative. Accounted cache bytes
cover logical pixel payload and cloned text strings, not driver/atlas memory.

### Shared Font And World-Preparation Benchmarks

The general font/world-preparation changes are verified with
ownership, concurrency, source-order/distance/tint parity, buffer reuse, and
pixel-equivalence tests. Local three-sample, 200ms-per-sample benchmarks on the
Apple M3 Pro (Go 1.27.1) provide narrow-function evidence:

| Operation | Earlier/exhaustive path | Improved/shared path |
|---|---|---|
| Font constructor/parser | Fresh source parse: 29-30 us, about 71 KB, 535 allocations | Warm constructor: 44 ns, 168 bytes, 2 allocations |
| Static-light collection | 3.3-3.9 ms, about 240 KB | About 18 us, 2.2 KB |
| Complete synthetic lighting recalculation | 3.5-3.7 ms, about 297 KB | About 0.32 ms, 59 KB |
| Text raster-cache key/hit | Previous formatting alone: about 466-471 ns, 224 bytes, 12 allocations | Complete warm structured lookup: about 67 ns, zero allocations |

These numbers describe the pre-SDK implementation, not a performance claim for
Berad Text. `BenchmarkUltimaFontConstruction` remains in `internal/text`; the old
formatted-key comparison was retired with the old cache. SDK tests now cover real
scaled-text raster reuse and fresh/cached pixels. Use current rendering workloads
when profiling the new adapter. `BenchmarkStaticLightCollection` and
`BenchmarkLightingRecalculation` remain in `internal/map_state`. The light
fixture is deliberately dense and covers a wrapped corner; the complete
recalculation comparison uses the exhaustive compatibility branch with the same
current visibility bookkeeping in both cases. These are microbenchmarks, not
live FPS/whole-process forecasts. Font sharing mainly benefits construction,
opening/reopening, and resize; steady-state font churn was already fixed.

Static emitter collection records current raw visible positions, sorts them in
the exhaustive scan's X-major order, filters its original half-open domain, and
keeps dynamic-source ordering/duplicates. It has no cross-frame result cache.
A primed legacy cache preserves its full-source-list compatibility path.
Terrain dither still samples each eligible frame and reuses cells/pixels only
when the derived sample/geometry inputs match; outer dither reuses its geometric
mask. Ambient scratch storage is refilled each Update, not cached by position.

Keep the raster entry limit unchanged until a representative working-set/budget
comparison justifies increasing it. Source sharing and cheap keys do not alone
eliminate capacity pressure during Ready navigation. Match live world/Ready and
opening/resize workloads before reporting end-to-end gains, and record retained
heap/resource cost separately from temporary allocation churn.

## Analyze Profiles

Open the browser UI:

```bash
go tool pprof -http=:8080 PROFILING/combat_cpu.prof
```

Or use the interactive shell:

```bash
go tool pprof PROFILING/combat_cpu.prof
```

Useful interactive commands are:

```text
top
top -cum
list FunctionName
web
```

`web` requires Graphviz. The browser UI provides flame graphs, call graphs, source views, and profile tables without changing the profile file.

### CPU Interpretation

- `flat` is time spent directly in a function.
- `cum` is time spent in the function and its callees.
- High cumulative time with low flat time often identifies orchestration, not the operation to optimize.
- Confirm a hotspot in source view before changing its caller.

### Memory Interpretation

Select the sample type that matches the question:

```bash
go tool pprof -sample_index=inuse_space PROFILING/combat_heap.prof
go tool pprof -sample_index=inuse_objects PROFILING/combat_heap.prof
go tool pprof -sample_index=alloc_space PROFILING/combat_allocs.prof
go tool pprof -sample_index=alloc_objects PROFILING/combat_allocs.prof
```

- `inuse_space` and `inuse_objects` show retained live heap.
- `alloc_space` and `alloc_objects` show cumulative allocation churn.
- A large allocation attributed to a caller may occur in a callee; use `list` and the call graph to locate the allocation site.

### Compare Before and After

Capture the same scenario, duration, build mode, data, and controls before and after a change. Compare the result directly:

```bash
go tool pprof \
  -base PROFILING/movement_before_cpu.prof \
  PROFILING/movement_after_cpu.prof
```

Retain the change only when profiles, benchmarks, or frame measurements demonstrate an improvement without a gameplay regression.

## Representative Scenarios

Profile one controlled workload at a time:

- Idle gameplay isolates baseline update and draw costs.
- Continuous movement exercises map queries, visibility, animation, and rendering.
- Crowded small maps exercise NPC scheduling and pathfinding.
- Combat exercises turn selection, AI, spells, animation, and visual effects.
- Dialog dissolve and other transitions exercise temporary render targets and masking.
- Time-of-day changes exercise lighting and visibility recalculation.

Record the location, party state, action sequence, capture duration, and relevant flags with the profile name or investigation notes. Use deterministic game-state seeds when randomness affects the scenario.

## Rendering Reuse And Capture Procedure

### Current Rendering-Reuse Contracts

- The shared text cache, owned by the SDK's `berad-text` renderer, admits up to
  **512 entries** within a **16 MiB** accounted-byte cap, with an oversized
  bypass. Measure misses, evictions and retained bytes when evaluating it; entry
  capacity alone does not prove a frame-time or GPU-memory improvement.
- `GameScene.Draw` scopes `combatRenderFrame` after FPS admission and discards
  it on every exit. Battlefield and HUD share one lazy attack preview. Sprite
  and health passes share one lazy manager-order traversal for disclosed,
  presented party insects. Groups of two through four retain their stable
  representative and segment identities; other group sizes do not aggregate.
  Calls outside Draw remain fresh. Nothing survives into Update or the next
  rendered frame, and action admission still revalidates authoritative state.
- Non-effect sprite animation resolves the existing position/time phase before
  returning the sheet's cached frame artwork directly. Registered tile effects
  retain their generated-frame path. There is no additional position-keyed LRU.
  Raw/bordered invalid indices reuse distinct sheet-owned transparent images;
  all returned artwork is borrowed and read-only. Copy before editing and do
  not dispose a borrowed image. Absent terrain skips tinting and composition.
- CharacterSummary retains at most one generated status outline per party
  slot, keyed by source image identity and numeric colour. Draw scaling and
  selection do not rebuild source-sized artwork. Source/colour changes, return
  to healthy status, inactive slots, resolution replacement, and scene disposal
  release owned outlines. Atlas artwork remains borrowed. Readback failure
  returns the source without retaining a failed outline, allowing a later retry.

Focused rendering tests cover phase/effect toggles, transparent fallback pixels,
portrait baseline pixels and ownership, insect disclosure/identity, fresh next-
frame previews, and unchanged RNG call counts. Benchmarks for warmed resolved
sprite lookup and portrait reuse report allocations independently of live FPS;
both should stay at zero allocations per call.
Do not continuously generate GPU images in a benchmark running inside one test-
game Update: that prevents normal frame-end driver-resource retirement. Use
bounded equivalence tests for that path and live captures for sustained churn.
Crowded-combat contribution and end-to-end improvement still require playtesting
and matching profiles.

### Controlled Capture Matrix

| Workload | Procedure and comparison |
|---|---|
| World idle | Same loaded save/location, warm assets, remain stationary without menus. Establish baseline before Ready. |
| Ready idle | Open once, allow warm-up, leave selection unchanged. Keep the same character/item/details page. |
| Ready navigation | Scroll repeatedly; record whether fresh taps or held repeat feel delayed. Capture character/hand changes and equip/confirmation separately from arrow-only navigation. |
| Combat player boundary | Small and large active crowds with the same loadout/terrain. Stay at player input to exclude enemy action costs. |
| Combat active | Record ordinary monster turns separately from ranged impacts, signature effects, and area attacks. Annotate the action/phase at each perceived hitch. |
| Combat history | Fresh/churned encounters with comparable active actors; record retained monster/queue counts when instrumentation provides them. |
| Isolation | After baseline, compare existing `-no-sound`, controlled logging, resolution, and drawing limit. Hold update rate and scenario constant. |
| Long session | Comparable start/end heap snapshots and cache/resource counts; exclude scene/load warm-up from steady-frame conclusions. |

Keep build revision/dirty state, Go/build mode, machine, resolution, configured
draw limit, audio/logging/Workbench settings, scenario/save identity, party
status, loadout, crowd/phase, duration, and perceived events beside each capture
under `PROFILING/`. Do not commit local saves or capture logs. Profile one CPU
interval at a time and do not run builds/tests/validation during captures.

Live CPU profiles isolate interaction windows without startup dilution. An
`allocs?seconds=30` endpoint capture gives a matching allocation delta; cumulative
allocation totals from a heap snapshot also include earlier workloads. Run
heap collection/forced GC and short traces separately when their observer cost
could alter CPU captures. Do not run `-cpuprofile` and a live CPU capture together.

If initial sampling cannot explain a hitch, add opt-in bounded timing/counter
storage for Update, content-rendered Draw, cache misses/evictions, image
generation/readbacks/uploads, and input observation/admission/ownership. Avoid
per-frame logging. Report p50/p95/p99/max frame durations separately from average
CPU usage. Native/GPU tools can supplement Go profiles for driver waits; a Go
Draw duration is not completed presentation time.

## Benchmarks

Use a benchmark when a narrow function can be measured independently. Keep setup outside the timed loop and report allocations:

```go
func BenchmarkOperation(b *testing.B) {
	state := setupBenchmarkState(b)
	b.ResetTimer()
	for range b.N {
		state.Operation()
	}
}
```

Run multiple samples:

```bash
go test -run '^$' -bench '^BenchmarkOperation$' -benchmem -count=5 ./internal/package
```

Capture a benchmark profile from one package when deeper analysis is needed:

```bash
go test -run '^$' -bench '^BenchmarkOperation$' \
  -cpuprofile=PROFILING/operation_cpu.prof \
  -memprofile=PROFILING/operation_heap.prof \
  ./internal/package
```

Do not replace an end-to-end game profile with a microbenchmark. A benchmark proves a local change; the representative workload proves that the local cost matters.

## Proven Project Patterns

These patterns are already useful in rendering and effects code. Apply them only after profiling identifies the corresponding cost.

### Reuse Persistent Render Buffers

Creating an `ebiten.Image` in a per-frame draw path can produce substantial allocation and texture-management overhead. Allocate render targets during initialization or when dimensions change, call `Clear` before reuse, and keep ownership with the component that uses them.

The message dialog implementation demonstrates this with the `MessageDialog.maskedBuffer` field in `internal/ui/widgets`. Its rendering path reuses the image while applying dissolve masks.

Lifecycle rules:

- Allocate once for the current dimensions.
- Recreate only when dimensions or format requirements change.
- Clear and reuse during each frame.
- Release references when the owning component is discarded.
- Prefer component ownership over a speculative global buffer pool.

### Cache Static Content with Dirty State

When content is static but an effect changes each frame, render the content only when its inputs change. `MessageDialog.contentDirty` separates text/content rendering from per-frame dissolve composition.

Set dirty state in every mutator that changes rendered content. Clear it only after a successful refresh. Avoid caching until profiles show repeated work, because invalidation paths add correctness risk.

### Use GPU Composition for Image Effects

For large image masks, prefer Ebitengine draw composition over reading and rewriting pixels on the CPU. The dissolve implementations use `ebiten.BlendDestinationIn` to combine a cached content image with an alpha mask.

Avoid per-frame `ReadPixels`, large byte slices, and `WritePixels` unless the effect specifically requires CPU access and profiling justifies it.

### Reuse Scratch Storage in Hot Loops

If allocation profiles identify temporary slices, maps, or arrays inside a hot loop, pre-size or reuse them where ownership is clear. Moving a value declaration outside a loop does not automatically improve performance: Go often keeps small values on the stack. Confirm heap behavior with `-benchmem`, escape analysis, or an allocation profile before changing code.

### Eliminate Redundant Work

Use cached paths, dirty flags, and precomputed data only when the inputs and invalidation rules are explicit. Optimize repeated work before introducing generic pools or broad caches.

## Anti-Patterns

- Do not optimize a suspected hot path without a profile or benchmark.
- Do not infer a bottleneck from cumulative CPU time alone.
- Do not treat retained heap and cumulative allocations as interchangeable.
- Do not optimize startup or other cold paths while frame-time hotspots remain.
- Do not add a generic cache or buffer pool for a single measured use case.
- Do not accept an optimization based only on reduced code complexity or intuition; re-profile it.
- Do not use wall-clock timing or uncontrolled randomness in core gameplay benchmarks.

## Investigation Checklist

1. Define one reproducible workload and the metric that matters.
2. Build with `make build` and capture a baseline under `PROFILING/`.
3. Inspect flat and cumulative costs or the appropriate memory sample type.
4. Locate the exact source operation and classify the cost.
5. Apply the smallest relevant pattern.
6. Run correctness tests for the affected package.
7. Repeat the same capture and compare against the baseline.
8. Record the result; revert speculative complexity when no material improvement appears.

For GC diagnostics, `GODEBUG=gctrace=1 ./bin/ultimav` can supplement a heap or allocation profile. It is diagnostic output, not a substitute for a saved profile.

## Further Reading

- [Go blog: Profiling Go Programs](https://go.dev/blog/pprof)
- [`runtime/pprof` package](https://pkg.go.dev/runtime/pprof)
- [`net/http/pprof` package](https://pkg.go.dev/net/http/pprof)
- [pprof documentation](https://github.com/google/pprof/blob/main/doc/README.md)
