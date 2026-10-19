# Sprite Outline System

## Overview

Sprite outlines are implemented as `EffectOutline` entries in the shared `sprites.TileEffectPipeline`; they are not managed by a separate outline subsystem.

The same processor supports static entity outlines, item highlighting, quest-item treatment, and combat indicators. Overlay glows and auras use the separate `OverlayPipeline` and are not part of this system.

Redux selects tile artwork, semantic colours, authored bounds, timing and presets.
The pinned [Berad Graphics API](https://github.com/bradhannah/berad-sdk/blob/feat/berad-graphics/modules/berad-graphics/docs/API.md)
owns image operations, padding, contour generation and bounded resource reuse.
See [ADR-012](../ADR/ADR-012-berad-graphics.md) for that boundary and its rationale.

## Core API

Build reusable effect pipelines with the fluent builder:

```go
pipeline := sprites.Effect().
	AddBorder().
	AddOutline(sprites.BlackOutline).
	Build()
```

`AddBorder` expands a 16x16 sprite to an 18x18 image with a transparent one-pixel border. Put it first when an outline must remain visible around source pixels that touch the sprite edge.

Apply a complete pipeline through the sprite sheet's processor:

```go
processed := spriteSheet.GetTileEffectProcessor().ApplyDynamic(sprite, pipeline)
```

For a single outline, use one of the convenience methods:

```go
processor.ApplyDynamicSimple(sprite, sprites.YellowOutline)
processor.ApplyDynamicSimpleWithIntensity(sprite, sprites.RedOutline, intensity)
processor.ApplyDynamicMarchingOutline(sprite, phase)
processor.ApplyDynamicOutlineRGBA(sprite, customColor, intensity)
processor.ApplyDynamicOutlineOnly(sprite, faceRGBA)
```

Animated intensity values should be quantized before calling the cached intensity API so they do not churn the bounded cache and repeatedly evict reusable entries.

## Outline Generation

`EffectOutline` finds opaque edge pixels with four-neighbor alpha checks, then paints transparent neighboring pixels around those edges. `WithIntensity` scales the selected outline color.

Predefined colors are represented by `OutlineColorType` in `internal/sprites/tile_effects.go`. They include general EGA colors plus combat-specific active, inactive, and threatened colors.

Dynamic results use an SDK `Processor` with complete source, ordered operation,
colour and padding identity. Each source/derived store is limited to 500 entries
and 32 MiB of accounted payload. This accounting is not total GPU memory.
Results are read-only borrows: never mutate or deallocate them. Cache eviction
drops references without invalidating a retained borrower. Custom portrait
`ApplyDynamicOutlineRGBA` calls instead return owned copies of the cached result;
the portrait owner releases those copies. Reuse source images and bounded phases.

### Outline-only contours

`ApplyDynamicOutlineOnly` returns only a coloured outer contour and black backing.
It reuses the existing edge/neighbor outline algorithm, adds the backing outside
the face, then clears **every original nontransparent source pixel**, including
partial-alpha pixels. It never copies body artwork into the returned contour.
Source dimensions and alignment are preserved; both rings clip at the image bounds.
Use the already bordered 18x18 character frame for normal combat alignment.

The result is borrowed and read-only. It shares the processor's bounded dynamic
LRU, keyed by the immutable source frame, face/backing RGBA and contour operation.
Actor identity, position and animation time are not retained in this cache;
callers select the actual frame first. Repeated frame/colour combinations reuse
their image, while misses allocate and generate pixels. Nil or disposed input,
transparent face colour or an SDK admission error returns nil, **never the original
sprite**. Source bounds are checked before cache lookup, so a disposed source
cannot reuse a cached contour. The bounds guard recovers only Ebitengine's
disposed-image condition; unrelated programming or native-readiness failures still
panic. Native readback requires the running Ebitengine loop. Callers skip nil rather
than falling back to a concealed body.

## Sprite Storage

`SpriteSheet.preGenerateStaticBorders` draws each static atlas region directly
into its cached 18x18 destination. It does not create an intermediate 16x16 GPU
copy. `WrapSpriteWith18x18Border` accepts cropped atlas images with nonzero source
origins; the result retains zero-origin bounds and the one-pixel transparent rim.

Valid `GetRawSprite` lookups lazily cache a detached, zero-origin 16x16 image per
tile, bounded by the 512-tile namespace. These results, and frames returned by
`GetRawAnimatedSprite`, are borrowed and read-only: callers must not dispose them
or modify their pixels. A consumer needing mutable pixels makes its own copy.
Invalid raw indices share the sheet's transparent raw fallback; bordered lookup
uses a separate size-correct fallback.
Custom atlases must be populated before the first cached lookup.

The raw cache belongs to each sheet, not a global scene cache. It has no disk or
save representation. Existing global moongate borrowers can outlive their source
scene, so this optimization does not add broad sheet disposal. Registered effect
generation remains lazy and runs once per processor; it is not regenerated on
every animated lookup.

`internal/sprites/spritesheet_pixels_test.go` covers atlas/border RGBA equivalence,
nonzero source origins, partial alpha, raw-cache identity and bounds, independent
mutable copies, ordinary animation selection and generated-frame borrowing.

## Generated Tile Frames

`tile_effects_registry.go` retains the game's flame, lava, water and field recipes.
`tile_pixel_adapters.go` translates original-palette selections and inclusive
authored bounds into SDK destination masks. Sampling can read an unselected source
colour within those bounds; selectors restrict writes, not source eligibility.
Pure water frames are prepared before mixed tiles use them as explicit replacement
pixels. Background tile choice and baked-ground subtraction remain Redux policy.

Frame preparation is lazy after native initialization. Rise/wave/scroll operations
run during preparation; lookups reuse the resulting 16x16/18x18 frames. SDK cycle
sampling receives game elapsed ticks and spatial phase explicitly. Original
animation families and alternating-column water offsets remain game-owned.

Sleep fields retain alpha-channel-only pulsing, preserving RGB. Protection fields
retain RGB brightness modulation with unchanged alpha. Uncalled opacity/tint/glow
builder methods and unused per-effect interval/bounds fields were retired rather
than exposed as silently ignored options. Dynamic presets accept border/outline;
generated-only operations in that path are programming errors.

## Current Rendering Integration

- The Avatar receives a black outline unless its sprite is integrated with background furniture or terrain.
- Friendly NPCs, enemies, and vehicles receive `StandardNPCPipeline`, which adds an 18x18 border and black outline.
- Background-integrated NPC sprites skip the map-unit pipeline.
- Ground item stacks receive a yellow outline, except gems.
- Lord British artifacts receive a black outline before their overlay glow; shards use their overlay glow without an outline.
- Combat bodies retain their ordinary black outlines; the current actor's animated
  square draws above HP frames and below final aiming indicators. On an arch, only
  the exact current party member or clone additionally receives the cached hollow
  contour with black backing. Its face uses the selection's semantic colour:
  light green normally, the monster-turn colour when possessed, and gray during
  automatic passing. Ordinary floor rendering gets no extra contour.
- The arch contour and square resolve exact identity and admission each draw.
  They cannot persist across handoff, retirement, shared death or combat exit.
  Black terrain, off-screen positions, underground entries and intentional hidden
  teleport/signature phases stay hidden. Body sprites, monsters, HP frames and
  target/recipient admission retain the shared terrain gate. This is the scoped
  Redux presentation enhancement described by `COMBAT-0001` and `COMBAT-0012`.
- Contours share the current sleep, rat and shimmer frame selection and settled
  body-animation clock, including Time Stop. Movement, lunge, recoil, scaling and
  screen shake follow the exact actor/clone. Signature-owned frames retain their
  existing static-frame, fade and dissolve-mask treatment; only the immutable
  unmasked contour is cached, and the existing effect owns mask-buffer cleanup.
- Temporary scene effects can select an outline color through their own renderer-owned state.

Map-unit pipelines are renderer-owned rules. `GameScene.renderMapUnitSprite`
selects and applies `StandardNPCPipeline` after animation selection and before
lighting and drawing; `MapUnitDetails` does not store a per-unit effect
pipeline. Item-stack and Avatar outlines are also renderer rules and are not
persisted per instance in `GameState`.

## Timing

Static outlines need no clock. Callers that pulse intensity or alternate marching phases derive a bounded phase from the scene's shared elapsed milliseconds, then call the processor with that phase or intensity.

## Limitations

- The map-unit default does not color-code friendlies, enemies, and vehicles differently; all currently use the black standard pipeline.
- Ground item stacks do not carry individual outline configuration.
- Outline configuration is not save-game state.
- `ApplyDynamicOutlineRGBA` creates an owned copy on each call; portraits retain
  that copy until their source/status changes.
- The pipeline builder validates non-empty pipelines and bounds, but callers must keep dimension-changing border wrap first.

## Implementation References

- `internal/sprites/tile_effects.go`: game pipeline builder and semantic colours.
- `internal/sprites/tile_effect_processor.go`: frame selection and SDK dynamic-operation adapter.
- `internal/sprites/tile_pixel_adapters.go`: original palette, authored bounds and frame preparation.
- `internal/sprites/tile_outline_only.go`: cached body-free contours and fail-closed admission.
- `internal/sprites/effect_cache.go`: SDK cache facade for game-keyed decorative variants.
- `internal/sprites/effect_presets.go`: standard map-unit pipelines.
- `internal/sprites/sprite_wrapper.go`: 18x18 border wrapping.
- `cmd/ultimav/gamescene_draw_map.go`: map units, item stacks, and quest items.
- `cmd/ultimav/gamescene_draw_entities.go`: Avatar and combat actor rendering.
- `cmd/ultimav/gamescene_combat_active_indicator.go`: combat indicator selection.
- `internal/sprites/tile_effects_test.go`: processor and outline behavior tests.
- `internal/sprites/tile_outline_only_test.go`: transparent interiors, backing, alignment and bounded cache behavior.
- `cmd/ultimav/gamescene_combat_arch_indicator_test.go`: arch composition, frame/identity selection and preserved concealment.
