# ADR-002: DisplayManager Pattern

**Status**: Accepted
**Date**: 2025-08-25

## Context

Screen size, resolution, fullscreen, and display-change behavior originally
leaked direct Ebitengine calls into packages that should remain independently
testable. This coupled non-rendering code to graphics initialization and made
display policy inconsistent across callers.

The project needs one application-level owner for mutable display state without
attempting to abstract the entire Ebitengine presentation stack.

## Decision

Centralize game-window display state and operations in `internal/display`
behind `DisplayManager`.

- New shared display-state reads and mutations use the manager rather than
  introducing independent Ebitengine display policy.
- Core gameplay packages remain independent of display and rendering packages.
- Windowed, maximized-window, and borderless-fullscreen requests are validated
  and coordinated through the manager's application boundary. Mode changes are
  asynchronous: callers request a desired mode and observe completion through
  `Update`, `RequestedMode`, and `IsTransitionPending`.
- Startup uses `InitializeMode`, which is restricted to Ebitengine operations
  valid before the main loop. Runtime changes use `RequestMode`, which may also
  preserve the existing window center.
- The minimum decorated window size is 1280x720. `DisplayManager` remembers the
  last normal window size across maximized and fullscreen transitions. It
  separately tracks the player-persistable normal size: observed manual drags
  and successful configuration commits update it, while runtime-only display
  requests do not.
- `Game.Layout` publishes the aspect-safe logical draw size to the manager.
  `GetCurrentSize` is the logical-size compatibility API used by layout
  consumers; window, monitor, and outer-container dimensions use the explicit
  `WindowSize`, `MonitorSize`, and `ContainerSize` APIs.
- Deprecated compatibility helpers are not the pattern for new callers.

The current implementation in `internal/display/` is authoritative for API
details. The application startup path and scene-resolution invalidation use the
same manager identity; scenes do not independently force fullscreen state.
[Architecture Guidelines](../ARCHITECTURE_GUIDELINES.md) defines the current
package boundary.

## Consequences

### Positive

- Non-rendering logic remains testable without graphics initialization.
- Shared display policy has a designated manager and notification path.
- Display-mode and sizing behavior remain consistent across UI consumers.
- Logical 16:9 rendering remains distinct from platform window and monitor
  dimensions, so non-16:9 containers can use centered black bars without
  distorting or cropping game content.
- Core packages do not acquire presentation dependencies for convenience.

### Trade-offs

- Callers depend on an application-level manager rather than direct Ebitengine
  functions.
- The manager must avoid becoming a general presentation service.
- Compatibility helpers may temporarily exist at integration boundaries and
  require deliberate retirement.

## Alternatives

### Direct Ebitengine Calls

Rejected because they spread mutable display policy and violate core package
boundaries.

### Full Rendering Abstraction

Rejected because replacing or abstracting all Ebitengine behavior is not a
project requirement. The narrower manager addresses the actual coupling.

### Display Interface Through Every Layer

Rejected because core gameplay should not need display state at all. Passing a
display dependency through those layers would preserve unnecessary coupling in
a different form.

## Current References

- `internal/display/manager.go`
- `internal/display/resolution.go`
- [Architecture Guidelines](../ARCHITECTURE_GUIDELINES.md)
- [Game UI Integration](../ui/GAME_UI_INTEGRATION.md)
