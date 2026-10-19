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

## Ready And Crowded Combat: Pre-Profiling Cost Map

Source review: 2026-10-05, before the rendering-reuse changes below. The observed symptoms are
lag in Ready, stuttering with many combat monsters, lesser general lag, and
occasional apparent input locks during rapid interactions. The operations below
are verified from source; their runtime contribution is **unmeasured**. This is
a guide to interpreting captures, not an optimization backlog or proof of a
particular bottleneck. Recheck the named entry points when profiling another
revision.

The cost map records the baseline paths. The current rendering-reuse contracts
below describe later changes; baseline captures remain historical evidence.

### Timing And Evidence Boundaries

- `GameScene.Activate` in `cmd/ultimav/gamescene_initialization.go` explicitly
  configures 60 updates per second. This setting predates the recent regression.
- `defaultDisplayConfiguration` in `internal/config/config.go` defaults drawing
  to 30 FPS; saved settings can override it. `GameScene.Draw` in
  `cmd/ultimav/gamescene_draw_core.go` also has a millisecond-based early-return
  frame limiter. Distinguish engine Draw calls from frames containing new content.
- Rendering cached images each frame is expected. Distinguish artwork generation,
  layout/projection rebuilding, cache lookup, CPU pixel access, and graphics
  submission. A cache hit can still leave substantial layout work.
- CPU samples establish where CPU time is spent, not input-to-display latency,
  p99 frame time, exact call counts, or completed GPU work. Allocation profiles
  show Go heap activity, not the complete graphics-resource footprint.
- Separate intentional presentation waits, input ownership/release gates,
  synchronous action I/O, and expensive frames. None is interchangeable with FPS.

### Ready

| ID | Verified operation and entry points | Capture question |
|---|---|---|
| R1 | `ReadyEquipmentDialog.Draw` in `internal/ui/widgets/ready_equipment_dialog.go` reconstructs header, slots, visible item rows, full detail content, and footer every Draw. `populateItemRow` in `ready_equipment_rows.go` measures prefix/name/quantity; `appendStrengthBudget` in `ready_strength_budget.go` measures and formats comparison rows. `Output.drawStyledLine` in `internal/text/output_rendering.go` measures segments again, including left-aligned text. | Does warmed, unchanged Ready still spend significant CPU/allocations in layout, formatting, and measurements? Compare idle with navigation and detail paging. |
| R2 | `internal/text/cache.go` has a global 100-entry rendered-text LRU, keyed by content, face identity, layout, and colour. Ready and the underlying HUD/transcript share it. Texture hits do not cache finished Output layout. | Does the unchanged screen settle into hits, or keep reaching `TextCache.renderText` and eviction? Measure the combined screen rather than an isolated string. |
| R3 | `GameScene.Draw` composes world, effects, and HUD before the dialog stack. Ready also draws its parent panes beneath its confirmation card. | What fraction of the full frame belongs to Ready versus the underlying scene? Preserve still-visible content when evaluating occlusion. |
| R4 | `ReadyEquipmentDialog.Update`, `ready_equipment_review.go`, and the Ready workflow in `cmd/ultimav/gamescene_dialogs.go` distinguish navigation, confirmation/release gates, pending submission, and later authoritative result polling. Several command keys use ordinary same-key debounce. | Does delay occur on fresh arrows, repeated character/command keys, equip confirmation, transaction return, or dismissal? CPU sampling alone cannot answer. |

Existing reuse and negative findings:

- Available equipment and value-type previews are retained. Ordinary Up/Down
  does **not** run `PreviewReadyEquipment`. Opening, actor/hand changes, rejected
  operations, and transaction results can rebuild candidates.
- Candidate construction scans 48 equipment types, with another bounded
  return-capacity scan in applicable previews. Inventory quantity does not
  multiply the candidate count.
- Fonts and pane geometry are retained within a fixed-size dialog. Reopening
  creates new font faces; it is not per-frame font parsing.
- Ready's portrait is a cached raw sprite. It does not use the uncached right-rail
  status-outline path described below.
- Ready checks Escape before list navigation. Do not attribute another widget's
  held-arrow cancellation behavior to Ready.
- Commit must retain fresh authoritative equipment validation, even if
  presentation layouts or previews are later cached.

### Crowded Combat

Use separate counts: **A** active monsters, **M** retained monster records,
**Q** retained queue entries, and **S** attack sources (at most three). The
battlefield is 11x11; the settled rendering viewport is 19x13. Historical
records can outlive active occupants, so live crowd size is not the only scale.

| ID | Verified operation and entry points | Capture question |
|---|---|---|
| C1 | Battlefield eligibility and HUD both call `combatAttackPreview` in `cmd/ultimav/gamescene_combat_attack_preview.go`. `PreviewCombatAttack` in `internal/game_state/combat_attack_preview.go` reviews candidate/source pairs; each review performs actor-disclosure and occupancy scans. Monster-related traversal is approximately O(S*A*M), quadratic when A and M grow together. | Compare small/large crowds at the same eligible player-input boundary, melee/ranged loadouts, and blocked terrain. This preview is suppressed during scheduled/blocked enemy work. |
| C2 | Sprite and health passes call `combatPartyInsectStack` in `cmd/ultimav/gamescene_combat_invocation_effect.go`. `GetMonsterIndicesAtPosition` in `internal/game_state/combat_monster_manager.go` scans retained monsters and constructs a slice before stack filtering, including ordinary non-insect crowds. | Look for repeated O(A*M) occupancy scans and temporary slices in both passes. Preserve co-location/disclosure behavior if sharing frame-local results. |
| C3 | At baseline, non-effect `SpriteSheet.GetAnimatedSprite` in `internal/sprites/spritesheet.go` used position-dependent keys in a 200-entry cache. The viewport visits 247 terrain positions before actor lookups. Absent combat-margin sprites reached `GetSprite`, allocating a transparent image on each miss. | Are absent-sprite generation and eviction sustained after warm-up? Cache churn depends on terrain/effect paths and traversal order; valid base-sprite misses generally reuse artwork. |
| C4 | `CombatTurnQueue.FillForecast` in `internal/game_state/combat_turn_queue.go` copies Q simulation states each Draw for four forecast entries. Party-spell reviews and signature-layer preparation have multiple presentation consumers; cone targeting rebuilds a footprint. | Separate unchanged turns, targeting, and signature bursts. Forecast work is O(Q) with bounded constants, not another crowd-quadratic loop. |
| C5 | Actor rendering scans active impact/signature/division effects; impact label lanes also scan previous impacts. Buffers may be retained while composition repeats. | Compare ordinary movement with multi-target bursts. Actor-by-effect work can grow with the affected crowd without generating new images on every call. |
| C6 | Source-slot lookup scans retained Q entries; death/flee/replacement can retain actor/queue records. Authoritative collision, target selection, turn advancement, and admission also use these lookups. | Compare fresh and churned encounters with equal living crowds. Do not compact/reindex actor identities or change slot precedence casually. |

Simulation and false-positive boundaries:

- Enemy AI runs on admitted turns, not every frame. Normal presented scheduling
  does not ordinarily drain a backlog of hostile turns in a single Update.
  Immediate/headless scheduling is different.
- Enemy movement is bounded directional probing, not repeated A* pathfinding.
  Player Tab assistance can instead run up to eight adjacent-goal searches per
  visible enemy (`combat_auto_action.go`, `internal/astar/map.go`).
- Congested movement, failed admission near capacity, cone/earthquake impacts,
  and death/division are action-specific workloads. Scalar damage formulas are
  not a full-battle per-frame simulation.
- Random rejection loops are not all hard-bounded, but there is no evidence yet
  that rare tails explain the observed stutter. Retry caps or changed sampling
  would change gameplay RNG consumption.
- Combat bypasses exploration visibility/tint at rendering, but still enters
  some preparation in `gamescene_tiles.go`. Its base light scan is 22x22, not
  the overworld's 512x512. Surface dither/overscan is not a combat workload.
- Warm ordinary sprites and many outlines reuse artwork. Monster count adds
  queries/composition and does not necessarily add unique rasterized images.
- Preserve RNG order/count, source-slot order, exact clone identities, multiple
  occupants, visibility rules, actor-local effects, and independent action
  revalidation. Initial presentation reuse should favor frame-local results.

### Shared Frame Costs And Retention

| ID | Verified operation and entry points | Boundary or isolation |
|---|---|---|
| S1 | `Game.Update` / `refreshDebugDiscoverySnapshot` in `cmd/ultimav/scene.go` rebuild party/merchant metadata and publish catalog copies every Update without requiring Workbench startup. `Catalog.Definitions` and `CloneDiscoverySnapshot` both defensively copy definitions. | Deep merchant reviews are blocked during Ready/combat, but rows, JSON, fingerprints, and copies remain. Idle exploration can additionally construct full merchant quotes/projections. |
| S2 | `updateAmbientSounds` in `gamescene_update.go` gathers terrain every ordinary Update, even while stationary. Current routing permits radius 15, or 961 candidates before bounds rejection. Music context lookup reconstructs authoring-derived mappings in `internal/audio/music_manager.go` and `routing.go`. | Session-level `-no-sound` avoids manager construction; muting volume or runtime disabling is not equivalent. Combat bounds reject most ambient candidates, so this is not inherently crowd-proportional. |
| S3 | `RecalculateVisibleTilesWithCustomLights` in `internal/map_state/layered_map.go` bypasses the older source cache and reaches a 512x512 scan on the overworld. Visibility/distance/tint preparation repeats through `gamescene_tiles.go`. | Idle exploration and Ready's underlying world are relevant. Do not assign overworld dimensions to combat or revive the old cache without accounting for custom lights and current invalidation. |
| S4 | `CharacterSummary.Draw` in `internal/ui/mainscreen/character_summary.go` calls uncached `ApplyDynamicOutlineRGBA` in `internal/sprites/tile_effects.go`, with pixel readback, CPU buffers, image creation, and upload for status portraits. | Compare healthy/status-afflicted parties. This is bounded by party size, not monster count. |
| S5 | Surface dither rebuilds/uploads masks; camera start/stop changes overscan dimensions and can replace storage. Inversion allocates a temporary viewport image during its active draw path. | Compare settled idle, continuous motion, discrete steps, map edges, and active inversion separately. |
| S6 | Active event queues rebuild survivor slices and polling callbacks. Enabled logging can format and synchronously print tick diagnostics (`internal/event_queue/event_queue_manager.go`, `internal/logging/logger.go`). | Default logging is disabled. A completed queue can retain release-gated input ownership while CPU work is cheap; record gate state rather than deleting waits. |
| S7 | Global sprite-mask cache in `internal/sprites/tile_overlays.go` retains source images without a bound. Border raster cache retains geometries; `DialogStack` removal does not clear its vacated backing-array slot. | Inspect longer-session retained heap/resource growth. These mechanisms are not yet measured leaks. Avoid disposing borrowed/shared images. |

### Rendering Lifecycle Coverage

| Family | Existing reuse | Remaining or separately triggered work |
|---|---|---|
| Sprites/tile animation | Base/raw sprites and generated effect frames | Lazy first generation, dynamic cache misses, absent-image fallbacks |
| Main borders/sky/wind | Construction/resize/content-change images | HUD/rail/status projection and composition; border readbacks are setup work |
| Ready/general UI text | Rendered-text and engine glyph/shaping caches | Styled Output reconstruction, wrapping/measurements, working-set churn |
| Message/Overlay dialogs | Dirty content and retained dissolve buffers | Fade composition; full-screen buffer footprint and removal lifetime |
| Atlas | Game-owned geography texture | Per-Update dossier/copy/filter, overlay layout, world drawing behind opaque reader; capture/hash before retained-texture check and cold PNG generation/upload |
| Sign/gem/astronomy/symbolic Dungeon View | Prerendered artwork | Frame/hint composition and opening/resize-specific generation |
| Gameplay dungeon | Decoded assets and retained canvases | Snapshot/render plan and scene composition; field stripes repeat within the same 100ms animation phase |
| Intro/character creation | Loaded glyphs/artwork and retained canvases | Static prose rewrap/glyph composition, transition buffers |
| Credits | Size-gated reflow and visible-row body cache | Fractional-scroll composition and fixed chrome |
| Audio/persistence | Embedded assets, retained resources, explicit save boundaries | Decoder/voice creation and seeks; synchronous save/browser/legacy-notebook-import/Doom-recovery work. Excerpt edits are in-memory until saved. No periodic ordinary-frame autosave established |

Dormant helper implementations are not evidence of a hot path. Follow production
callers before assigning pixel loops, focus helpers, or scaled-font creation to
a workload. Numerous vector calls can batch; count actual resource work and
submission behavior rather than treating each Go call as a GPU draw call.

### Current Rendering-Reuse Contracts

- The shared text LRU admits up to **512 entries**, retaining the **16 MiB**
  accounted-byte cap and oversized bypass. The previous 100-entry limit
  saturated during Ready navigation while payload peaked near 1.29 MiB.
  Measure misses, evictions, retained bytes, and actual rendered/action counts
  when evaluating the larger budget; entry capacity alone does not prove a
  frame-time or total GPU-memory improvement.
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
sprite lookup and portrait reuse report allocations independently of live FPS.
Three 200 ms samples on Go 1.27.1/darwin-arm64, Apple M3 Pro, measured approximately
6.5 ns for warmed resolved sprite lookup and 12.4-12.5 ns for warmed portrait
reuse, both with zero bytes/allocations per call. These narrow lookup results
exclude frame preparation, cold generation, composition, and GPU completion.
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
| Combat player boundary | Small and large active crowds with the same loadout/terrain. Stay at player input to expose C1/C2/C3 without enemy action costs. |
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

### First Ready Baseline: Measured Observations

On the same gameplay baseline and an Apple M3 Pro (Go 1.27.1, darwin/arm64),
the first live session captured approximately 20 seconds each of warmed world
idle, unchanged Ready, and Ready Up/Down navigation. The normal optimized binary
used `-webprofile`, without Workbench enabled or optimization patches. CPU and
allocation-delta endpoints overlapped; analysis/heap collection occurred after
CPU capture. Combat capture was deferred by the user.

| Workload | Sampled process CPU / elapsed | Estimated allocation delta |
|---|---|---|
| World idle | 7.07 s / 20.11 s | 2,030 MiB / 20.01 s |
| Ready idle | 14.65 s / 20.09 s | 3,818 MiB / 20.01 s |
| Ready navigation | 12.55 s / 20.10 s | 3,093 MiB / 20.02 s |

These are single-run sampled observations, not frame-time or input-latency
measurements. Exact display settings, scene/save details, rendered frame count,
and input acceptance were not recorded. Do not infer that navigation is faster
than idle Ready, or that the CPU difference is a fixed per-frame dialog cost.
Allocation bytes are churn, not memory held: separate live Go heap samples were
approximately 100 MiB, and do not include the full graphics-memory footprint.

The first captures substantiate several cost-map paths:

- **S3 lighting:** `getAllLightSourcesInRange` accounted for 2.01 s, or 28.4%,
  of world-idle CPU samples. It remained visible behind Ready.
- **S1 discovery:** `refreshDebugDiscoverySnapshot` accounted for approximately
  1,205 MiB of world-idle allocations, and 440-448 MiB in the Ready intervals.
  Merchant preview descendants were absent from Ready as expected.
- **R2 text generation:** `TextCache.renderText` remained active in unchanged
  Ready, with approximately 362 MiB of allocations and 0.70 s of CPU samples.
  This is measured image-generation work, not only speculation about cache
  capacity. Miss/eviction counters are still needed to explain the cause.
- **Graphics submission:** Metal `Graphics.DrawTriangles` rose from 0.34 s in
  world idle to 4.72 s in unchanged Ready. Native Objective-C/reflection
  descendants were also major allocation sites. This supports investigating
  submitted graphics work; it does not measure GPU execution or establish an
  engine defect.

**An additional font-lifecycle problem emerged from allocation profiles.**
`ProvisionChangePanel.Update` in
`internal/ui/mainscreen/provision_change_panel.go` repeatedly reached
`initializeOutputs` -> `NewUltimaFont` -> `NewGoTextFaceSource`, with
approximately 439-485 MiB of allocations per world/Ready-idle interval. Both
`observeProvisionChanges` and `drawDialogs` reach the panel through
`layoutProvisionChanges` in `cmd/ultimav/gamescene_provision_changes.go` and
`cmd/ultimav/gamescene_draw_core.go`. The existence of a dimension guard had
masked this during static review.

Inspect the actual Update/Draw width keys before changing this path. Update
applies `logicalGameBounds` to already published logical dimensions; Draw uses
offscreen bounds directly. Reapplying integer aspect fitting is not always
idempotent (1500x1000 -> 1500x843 -> 1498x843), so alternating widths are a
source-backed hypothesis for repeated initialization. The session did not
record the runtime widths, and therefore does not yet prove that trigger.

The immediate diagnostic questions are now provision-font key stability,
text-cache working-set/key behavior, and rendered-frame timing. The combat
findings remain unmeasured. Preserve the raw captures and local metadata under
`PROFILING/`, and reproduce matching scenarios before interpreting an
optimization as successful.

The user subsequently identified the capture window as 1788x1033 windowed and
arrow navigation as the Ready symptom. Its published logical surface is
1788x1005; applying the fitter again produces 1786x1005. Regression tests now
exercise that exact Layout/Update/Draw mismatch, as well as fractional and
standard sizes. Current provision observation and initialization use the
published logical dimensions directly. Performance improvement still requires
matching captures; the historical baseline above remains unchanged.

## Native Combat Square Test: Measured Results

On 2026-10-17, two serial before/after captures of
`TestCombatFieldsRenderSquareAnchoredThroughActorMotion` on Go 1.27.1,
darwin/arm64 measured a test-only readback optimization against `e853af1`.
Both sides used `GOMAXPROCS=2`, `GOMEMLIMIT=256MiB`, `GOGC=50`, package
parallelism one, and an uncached focused test invocation:

```sh
go test -p 1 ./cmd/ultimav -run TestCombatFieldsRenderSquareAnchoredThroughActorMotion -count=1
```

The supervised runs enforced a 1024 MiB aggregate RSS budget, 25% minimum free
host memory, critical-pressure abort, 0.2-second sampling and a 900-second
deadline. Tooling transitioned from fixed limits to named profiles during the
measurement; the after runs explicitly selected the matching constrained profile.
No resource threshold, native frame cadence, or VSync setting was relaxed.

Temporary test instrumentation collected phase wall times, CPU samples and
process allocation deltas; it was removed after capture. Test-body time includes
fixture cleanup but excludes CPU-profile finalization. Package elapsed includes
the native harness and profiler shutdown, but excludes compilation. Raw captures
remain local and are not source artifacts.

| Measurement | Before, runs 1 / 2 | After, runs 1 / 2 |
|---|---|---|
| Instrumented test body | 21.573 / 21.565 s | 7.573 / 7.643 s |
| Package elapsed | 22.990 / 22.876 s | 8.805 / 8.848 s |
| Fixture setup | 0.105 / 0.132 s | 0.094 / 0.111 s |
| Field target creation | 0.00189 / 0.00228 s | 0.00026 / 0.00029 s |
| Expected artwork acquisition/draw | 1.121 / 1.201 s | 0.942 / 0.991 s |
| Expected readback, including nonempty check | 8.928 / 8.913 s | 0.370 / 0.372 s |
| Actual field readback | 8.132 / 8.017 s | 2.942 / 2.936 s |
| Field comparisons and their invariant checks | 0.00099 / 0.00121 s | 0.00245 / 0.00272 s |
| Sampled process CPU | 1.29 / 1.52 s | 0.56 / 0.53 s |
| Process allocation delta | 290,939,016 / 290,697,832 bytes | 277,880,888 / 277,433,576 bytes |
| Process allocation count | 2,300,156 / 2,289,238 | 1,382,378 / 1,386,723 |

The mean test-body reduction is approximately **65%** (2.84 times faster).
A final focused run after removing all profiling instrumentation passed with
9.112 s package elapsed; no test-body timer remains in the retained test.
Readbacks, rather than fixture I/O or byte comparison, dominated the baseline.
The second read of each unchanged expected image was already cheap; eliminating
repeated oracle rendering and batching actual readbacks provided the useful gain.
Allocation deltas cover all Go activity during the test, including artwork
generation, native-loop work and profiler overhead; they are not retained heap
or GPU-memory measurements. This is evidence for this test, not a gameplay FPS
or full-suite speedup claim.

Coverage remains four field kinds times two logical sizes (400x208, 1366x768),
with fractional shake `(0.35, -0.65)` and artwork time held at 225 ms. Each of
the eight fixtures still exercises two actor identities, both separate and
co-located positions, and recoil, missed lunge, movement start/middle/end.
There are **520 exact field pixel comparisons**, **160 actor-motion comparisons**
and **32 actor nonempty controls** per run, including field-slice reversal and
removal through the real dispel path.

The test caches three independent per-square oracle images as CPU pixels per
fixture. Each is checked nonempty once; its artwork inputs remain fixed while
actor conditions change. Production output is freshly rendered into one cleared,
bounded union target per scenario. Comparison visits every RGBA byte of each
original guarded rectangle using native screen coordinates; ordering never keys
the oracle. Time and rectangle invariants are checked on use. This preserves the
pixel oracle rather than substituting geometry-only assertions. The separate
animation/concealment test retains its fresh per-time oracle.

Field-only image creation falls from **1040 to 32** (24 expected images and eight
reused actual targets); field read calls fall from **1560 to 200** (24 expected
and 176 actual). Actor controls retain their 224 read calls. The largest union
pixel buffer is 460,900 bytes; the target is bounded to these field footprints,
not the whole display. Expected GPU images are disposed immediately after
readback, and fixture cleanup disposes each owned actual target. Borrowed sprite
artwork is never disposed, and no forced GC was added.

The remaining measured costs are approximately 3.31 s of field readbacks,
3.22 s for the unchanged actor baseline/control rendering and readbacks, and
0.94–0.99 s for initial expected artwork acquisition/draw. They remain bounded
test costs; no runtime rendering algorithm was changed.

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
