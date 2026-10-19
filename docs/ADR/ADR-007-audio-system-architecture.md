# ADR-007: Audio System Architecture

**Status**: Accepted callback/lifecycle rationale; concrete playback realization superseded by [ADR-010](ADR-010-reusable-audio-module.md)

**Date**: 2025-01-24

## Context

Gameplay needs sound effects, location-based music, footsteps, and proximity ambience without importing Ebitengine audio into core packages. Playback may fail for environmental reasons, while missing required embedded assets indicate a broken application build. Audio settings also need one place to coordinate category volumes and previews.

The project already uses `SystemCallbacks` to keep core gameplay independent from presentation adapters. Audio should follow that boundary and use the Ebitengine audio context already present in the application.

## Decision

Adopt a central `AudioManager` in the presentation layer and expose sound intent to core gameplay through `SystemCallbacks.Audio` and `game_state.SoundEffect` identifiers.

The manager owns a shared Ebitengine audio context and coordinates focused components for sound effects, music playback, location/context music selection, walking sounds, proximity ambience, positional attenuation, and audio configuration.

Sound and music metadata map stable identifiers to embedded assets and volume modifiers. Gameplay playback requests choose explicit one-shot or loop mode and builder volume; the audio package chooses the file and applies effective category volume.

Use callback builders as call-site syntax over the callback contract. Builders do not move audio implementation into core logic; each start or stop translates fluent configuration into one typed request.

Coordinate audio with visual effects in the owning event workflow or feedback sequence. Do not create combined audio/visual callbacks such as a screen-shake-with-sound API. This keeps lifecycle ownership explicit and allows `Finally()` to clean up both systems.

Treat the callback contract as mandatory once `SystemCallbacks` is constructed. Sound-disabled mode is represented by the presentation adapter performing no playback, not by missing callback functions.

## Rationale

- Core packages remain independent of Ebitengine and embedded asset details.
- One manager controls the shared context and subsystem lifecycle.
- Stable sound identifiers allow assets and volume tuning to change without gameplay edits.
- Focused playback components isolate effect, music, ambience, and selection concerns.
- Explicit workflow coordination makes audio/visual cleanup visible at the call site.
- Mock callbacks provide deterministic gameplay tests without initializing audio hardware.

## Alternatives Considered

### Direct Audio Calls From Gameplay

Rejected because it would couple `internal/game_state` to Ebitengine and make gameplay tests require presentation resources.

### A Single Monolithic Player

Rejected because sound instances, looping music, proximity ambience, and location selection have different state and lifecycle needs. `AudioManager` remains the facade while focused components own those responsibilities.

### A Global Event Bus

Rejected because it would duplicate `SystemCallbacks`, obscure synchronous ownership, and make sequencing and cleanup less explicit.

### A Combined Audio/Visual Effect API

Rejected because timing belongs to event workflows and presentation feedback. Separate callbacks compose without making either subsystem own the other.

## Consequences

### Positive

- Gameplay depends only on typed sound intent.
- Audio can be disabled or mocked without branching in core gameplay.
- Embedded formats and metadata are centralized.
- Location music and proximity ambience share configuration and context ownership.
- Playback errors can be logged without changing authoritative gameplay results.

### Trade-offs

- `AudioManager` and its components require coordinated locking and cleanup.
- Fluent builders must be kept aligned with the callback implementation.
- Lazy loading moves some asset validation to first playback.
- Presentation timing currently uses wall-clock time and is not replay-deterministic.

## Current Realization

This ADR records the architectural rationale, not the live API reference. The current implementation, supported operations, and known builder limitations are documented in [Audio System](../systems/AUDIO_SYSTEM.md).

Live scenes use `GameAudioManager` over the pinned Berad Audio runtime. The host
context is borrowed, not manager-owned. Shared transport, admission, fades and
authoring/tool mechanisms are SDK-owned; world/music policy and callbacks remain
Redux-owned. The old component graph and test-only implementation copies are retired.

Current game integration locations:

- `internal/audio/audio_manager.go`
- `internal/audio/sound_effects.go`
- `internal/audio/game_audio_routing.go`
- `internal/audio/ambient_projection.go`
- `internal/audio/music_manager.go`
- `internal/audio/configuration.go`
- `internal/game_state/system_callbacks.go`
- `internal/game_state/sound_builder.go`
- `cmd/ultimav/gamescene_audio.go`

Related decisions: [ADR-003: SystemCallbacks Decoupling](ADR-003-systemcallbacks-decoupling.md)
and [ADR-010: SDK adoption](ADR-010-reusable-audio-module.md).
