# ADR-011: Consume Berad Text through the game text facade

**Status:** Accepted; direct SDK extraction and pinned consumer integration

## Context

Supplied-font parsing, styled layout, bitmap rendering, geometry queries and
bounded raster reuse are useful to other Ebitengine applications. The former
`internal/text` implementation mixed these with original glyph encodings,
transcript/journal semantics, semantic palette choices and the game's UI layout.
Moving the whole package would export those application constraints.

## Decision

The independent implementation lives directly at
`github.com/bradhannah/berad-sdk/modules/berad-text`, with its own public contract,
tests, native-render harness and unrelated consumer. The game pins an available
SDK revision; neither a sibling checkout nor a local replacement is required.
SDK guidance was reviewed at `c4bdf2ebb57e50324f3e7184bce2209f5177dcba`.

Keep `internal/text.Output`, `LineBuilder`, font conveniences, original byte/glyph
conversion, sign/narrative content and history/save projection as game-owned
adapters. Generic font/layout/raster mechanics are SDK-owned. Core gameplay keeps
its detached message callbacks and cannot import the renderer module.

Font resources are immutable supplied data; the game retains its embedded TTF
and bitmap assets/notices. Sizes and colours can change live without introducing
a runtime file-watching or hot-replacement service. The SDK exposes deliberately
borrowed Ebitengine drawing targets and read-only geometry, not input/focus/Z
routing. Host controls use its caret/range information and own editing semantics.

Private raster images are renderer-owned. A final application shutdown closes the
shared renderer, while retained geometry and game-owned glyph snapshots have
separate lifetimes. The local facade preserves the old clipped text canvas and
alignment allowance explicitly; generic SDK rendering retains full ink bounds.

## Consequences

- Generic source/cache/layout tests and public-consumer proof run in the SDK.
  Game tests cover adapter pixels, original glyph semantics, editor geometry and
  transcript/save behavior against the actual dependency pin.
- Cache keys use immutable typography/content identity. Repeated scaled text no
  longer creates a new composed-image identity merely by allocating a face.
- TextInput shares the same font source/metrics as its rendered text. Selection
  queries use complete shaped lines rather than independently measured prefixes.
- Panel/scrolling/history policy remains specific to this application; it does
  not constrain another game's UI. Presentation copies cannot append to their
  source's logical history, and invalid zero-column wrapping fails immediately.
- Original narrative rendering and vector command markers remain specialized
  consumers/policy; the extraction does not reinterpret original file formats.
- The API is experimental. Publication, native platform observations and stable
  release promises are distinct from module compilation or passing fake-input tests.

## Production-surface refinement

The experimental SDK surface is narrowed to current production needs. Retained
panel/log containers, annotations and point-hit queries, configurable OpenType
features/variations, bitmap RGB-preservation mode, SDK alignment/explicit layout
spacing and unused draw switches are removed. Font.Measure's unused spacing argument
is removed while preserving the facade's existing zero-spacing engine measurements.
This is an intentional source-compatibility break, not a change to
Redux's text/history/input policy. Width-based wrapping, mixed-font top alignment,
source-range/caret geometry, internal ink preservation and bounded renderer
ownership remain. The existing engine-face compatibility bridge remains for a
separate careful migration. Future expansion needs a real consumer; an SDK demo
alone is not demand.

The retained contract is checked with independently authored bitmap geometry and
pixels, exact source offsets, finite accumulated extents, and LRU promotion under
entry/byte pressure. Consumer smoke tests remain compatibility evidence rather
than a substitute for those semantic assertions.

The consumer facade follows the same required-use boundary. Unused formatting,
sign/debug and direct-drawing conveniences are retired rather than preserved by
tests alone. Consumer tests exercise real sign results, semantic styles, row
retention and legacy clipping; SDK tests own generic cache/layout mechanisms.
The shared-cache clear seam remains an explicit exception for independent native
fixtures and widget-owned composition proof. Removing it would require changing
the application's shared renderer lifetime, not merely deleting a test helper.

See the [game integration contract](../systems/MESSAGE_OUTPUT_SYSTEM.md) and
[SDK API contract](https://github.com/bradhannah/berad-sdk/blob/564225ae5d47/modules/berad-text/docs/API.md).
