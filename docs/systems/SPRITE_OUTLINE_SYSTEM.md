# Sprite Outline System

## Overview

Sprite outlines are implemented as `EffectOutline` entries in the shared `sprites.TileEffectPipeline`; they are not managed by a separate outline subsystem.

The same processor supports static entity outlines, item highlighting, quest-item treatment, and combat indicators. Overlay glows and auras use the separate `OverlayPipeline` and are not part of this system.

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
```

Animated intensity values should be quantized before calling the cached intensity API so they do not churn the bounded cache and repeatedly evict reusable entries.

## Outline Generation

`EffectOutline` finds opaque edge pixels with four-neighbor alpha checks, then paints transparent neighboring pixels around those edges. `WithIntensity` scales the selected outline color.

Predefined colors are represented by `OutlineColorType` in `internal/sprites/tile_effects.go`. They include general EGA colors plus combat-specific active, inactive, and threatened colors.

Dynamic results are cached by source image and pipeline in a bounded LRU `EffectCache`. Reuse pipeline values and source images instead of rebuilding semantically unique effects every frame.

## Sprite Storage

`SpriteSheet.preGenerateStaticBorders` draws each static atlas region directly
into its cached 18x18 destination. It does not create an intermediate 16x16 GPU
copy. `WrapSpriteWith18x18Border` accepts cropped atlas images with nonzero source
origins; the result retains zero-origin bounds and the one-pixel transparent rim.

Valid `GetRawSprite` lookups lazily cache a detached, zero-origin 16x16 image per
tile, bounded by the 512-tile namespace. These results, and frames returned by
`GetRawAnimatedSprite`, are borrowed and read-only: callers must not dispose them
or modify their pixels. A consumer needing mutable pixels makes its own copy.
Invalid raw indices retain the fresh transparent fallback and are not cached.
Custom atlases must be populated before the first cached lookup.

The raw cache belongs to each sheet, not a global scene cache. It has no disk or
save representation. Existing global moongate borrowers can outlive their source
scene, so this optimization does not add broad sheet disposal. Registered effect
generation remains lazy and runs once per processor; it is not regenerated on
every animated lookup.

`internal/sprites/spritesheet_pixels_test.go` covers atlas/border RGBA equivalence,
nonzero source origins, partial alpha, raw-cache identity and bounds, independent
mutable copies, ordinary animation selection and generated-frame borrowing.

## Current Rendering Integration

- The Avatar receives a black outline unless its sprite is integrated with background furniture or terrain.
- Friendly NPCs, enemies, and vehicles receive `StandardNPCPipeline`, which adds an 18x18 border and black outline.
- Background-integrated NPC sprites skip the map-unit pipeline.
- Ground item stacks receive a yellow outline, except gems.
- Lord British artifacts receive a black outline before their overlay glow; shards use their overlay glow without an outline.
- Combat rendering uses dynamic outlines and marching or pulsing indicators for active, inactive, and threatened actors.
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
- `ApplyDynamicOutlineRGBA` is an uncached custom-color path.
- The pipeline builder validates non-empty pipelines and bounds, but callers must keep dimension-changing border wrap first.

## Implementation References

- `internal/sprites/tile_effects.go`: pipeline builder, outline algorithms, and dynamic application.
- `internal/sprites/effect_cache.go`: bounded dynamic-effect cache.
- `internal/sprites/effect_presets.go`: standard map-unit pipelines.
- `internal/sprites/sprite_wrapper.go`: 18x18 border wrapping.
- `cmd/ultimav/gamescene_draw_map.go`: map units, item stacks, and quest items.
- `cmd/ultimav/gamescene_draw_entities.go`: Avatar and combat actor rendering.
- `cmd/ultimav/gamescene_combat_active_indicator.go`: combat indicator selection.
- `internal/sprites/tile_effects_test.go`: processor and outline behavior tests.
