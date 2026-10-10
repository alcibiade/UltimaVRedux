# ADR-006: UI Widget System Enhancements

**Status**: Accepted

**Date**: 2025-08-26

## Context

The widget package had reusable borders, buttons, text input, and dialogs, but feature UI still needed common support for loading feedback, visible focus state, reusable list/layout calculations, and resolution-aware sizing. Buttons also needed explicit mouse interaction states rather than only a selected/not-selected presentation.

The decision needed to preserve the existing Ebitengine presentation boundary, `DisplayManager`, percentage-based placement, `text.Output`, and `DialogStack` rather than introducing a second UI framework.

## Decision

Extend `internal/ui/widgets` with focused, opt-in utilities:

- `LoadingIndicator` with spinner, dots, and bars styles
- `FocusIndicator` and `FocusManager` with border, glow, underline, and highlight presentation
- `FocusableItem`, implemented by `Button`
- uniform, flex, and grid layout calculations over `PercentBasedPlacement`
- `ScrollableListState` for reusable selection and viewport movement
- `ResolutionScaler` for font, element, spacing, placement, and grid calculations
- expanded button sizes and normal, hover, pressed, selected, and disabled states
- `DeliberateMouseTracker` so a stationary cursor does not immediately override keyboard selection

Keep these capabilities composable rather than forcing every widget through global managers. Existing dialogs may adopt them when useful; package availability does not imply universal GameScene integration.

Do not introduce a separate generic modal-transition subsystem. Current popup transitions remain part of `MessageDialog`/`OverlayDialog` through the pixel-dissolve effect and their owning lifecycle.

## Rationale

- Small utilities fit the existing package and can be adopted incrementally.
- Percentage-based results compose with current sprite placement and display code.
- Explicit focus state separates focus presentation from input routing.
- `ScrollableListState` removes repeated cursor/viewport arithmetic from complex dialogs.
- Button mouse states improve interaction without replacing keyboard-driven dialog logic.
- Avoiding a second modal animation layer prevents competing ownership with MessageDialog dissolve transitions and the event queue.

## Consequences

### Positive

- Complex dialogs can share tested list and layout calculations.
- Buttons expose a complete interaction-state model and implement `FocusableItem`.
- Owners can select ordinary or high-contrast focus presentation.
- Resolution-aware helpers are available without changing core gameplay packages.
- The additions require no new external UI dependency.

### Trade-offs

- The utilities have different update signatures and do not all implement `Widget`.
- `FocusManager` does not route keyboard input itself; owners call `FocusNext` and `FocusPrevious`.
- `ResolutionScaler` is not a universal layout engine and is not automatically applied to all widgets.
- Loading animation advances by update calls rather than a shared deterministic clock.
- Older components still vary in how strictly they validate optional configuration.

## Scope Clarifications

This decision does not assert standards conformance, complete keyboard coverage, touch support, universal high-DPI behavior, or a specific test-coverage percentage. The code provides focus and sizing primitives that can support accessibility work, but compliance requires product-level design, integration, and validation beyond these types.

Dialog transitions remain owned by `MessageDialog` and `OverlayDialog`; a second generic transition widget is not part of this accepted decision.

## 2026-08-27 Partial Retirement

A production-consumer audit found that the uniform, flex, and grid layout APIs
and `ResolutionScaler` had no consumers outside their own tests. Those
implementations and tests were removed rather than preserving an unadopted
parallel layout system. The original decision bullets above remain as the
historical record, but those two bullets no longer describe available APIs.

Responsive widgets continue to query `DisplayManager` and rebuild derived
geometry with their local dimension-cached `ensureLayout()` methods. Shared
production spacing remains in `dialog_constants.go`, while font scaling remains
in `typography.go` and `internal/text`. No replacement layout framework was
introduced.

## 2026-08-27 Keyboard-Only Retirement

The player-facing Ebitengine game adopted keyboard-only input. Button hover and
pressed states, direct pointer handling, and `DeliberateMouseTracker` were
removed after every retained interaction had a keyboard route. The original
decision and rationale above remain as the historical record, but their mouse
capabilities no longer describe the current implementation.

The browser-based Debug Workbench is developer tooling rather than player-game
input and remains explicitly exempt from this decision.

## Current Implementation

- `internal/ui/widgets/loading_indicator.go`
- `internal/ui/widgets/focus_indicator.go`
- `internal/ui/widgets/scrollable_list_state.go`
- `internal/ui/widgets/button.go`

Focused tests exist beside these files. Current APIs and integration limits are documented in [Widget Components](../ui/WIDGET_COMPONENTS.md) and [UI Widget System Architecture](../ui/UI_WIDGET_SYSTEM.md).
