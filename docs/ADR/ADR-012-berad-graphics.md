# ADR-012: Supplied-image mechanics through Berad Graphics

## Context

Redux's sprite pipeline mixed reusable pixel operations and GPU resource handling
with original tile identities, EGA palette selections, authored regions and scene
presentation. Extracting the whole sprite package would impose those game choices
on other consumers. Shared image pointers also have different lifetime contracts:
cached frames and moongate masks are borrowed, while status portraits own copies.

## Decision

Consume the independently versioned `berad-sdk/modules/berad-graphics` module
through a published `go.mod` pin. The SDK accepts supplied pixels, regions,
colours, masks, phase/progress and presentation randomness. It owns generic
padding, outlines/contours, rise/wave/scroll, channel operations, cycle sampling,
ordered dithering, dissolve ordering and bounded native resource caches/surfaces.

Redux retains its assets/notices, original decoding and indices, palette matching,
effect recipes, water dependencies, frame counts/intervals, spatial phase,
decorative-ground subtraction, quarter-neighborhood coverage, visibility,
lighting/camera policy and workflow ownership. Authoritative gameplay cannot
import the graphics module. There is no automatic reload service or new save state.

Use the existing game facades to keep these choices visible. SDK processors are
host-serialized and return read-only borrows; eviction/closure drops references
without destroying retained images. Game portraits request owned copies. Mutable
dither/dissolve surfaces have explicit owners and disposal. Sprite-sheet scene
shutdown must preserve longer-lived moongate borrowers and shared test sheets.

Preserve sleep-field alpha-only modulation and independent additive RGB/alpha
semantics. Name additive presets accurately rather than silently altering their
appearance. Retire uncalled builder options and duplicate pixel kernels.

The production-surface review narrows the experimental dependency to used
mechanisms. Redux keeps native frame arrays/transforms and draw opacity rather
than adopting unused SDK HSV, Frame/Animation or CPU tint/opacity/crop facilities.
Padding remains native `CopyRegion`/`Insets`; the redundant dynamic border preset
is removed because sprite selection already supplies bordered artwork. This
avoids maintaining a second frame/placement model or future-only API promises.
Explicit cache budgets, invalidation and resource closure remain essential even
when a particular owner currently uses an application-long lifetime.

## Consequences

Generic mechanisms and their independent/native consumer tests live with the SDK;
Redux tests retain palette, authored geometry, phase, concealment, portrait,
terrain and workflow integration. No sibling checkout or local replacement is
needed to build the game. SDK tests are not rerun by game gates.

Immutable inputs require explicit invalidation before reuse after mutation.
Cache payload limits do not bound externally retained results or total VRAM.
Native readiness failures are programming errors; contour admission errors return
no contour rather than revealing the original body. This remains an experimental
dependency with platform-specific acceptance, not a stable SDK release.

See [sprite integration](../systems/SPRITE_OUTLINE_SYSTEM.md),
[VFX ownership](../systems/VISUAL_EFFECTS_SYSTEM.md), and the
[SDK API contract](https://github.com/bradhannah/berad-sdk/blob/main/modules/berad-graphics/docs/API.md).
