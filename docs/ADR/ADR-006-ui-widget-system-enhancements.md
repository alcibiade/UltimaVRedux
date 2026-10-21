# ADR-006: UI Widget System Enhancements

**Status**: Superseded. Only `ScrollableListState` remains in use.

**Date**: 2025-08-26 (retired in stages through 2026-10-20)

## Decision

Extend `internal/ui/widgets` with small, opt-in utilities rather than a second UI
framework, keeping the existing `DisplayManager`, percentage-based placement,
`text.Output` and `DialogStack`. Popup transitions stay owned by
`MessageDialog`/`OverlayDialog`; no generic modal-transition subsystem.

## What remains

`ScrollableListState` (`internal/ui/widgets/scrollable_list_state.go`) shares
selection and viewport movement across list dialogs.

The other utilities this decision added were removed once audits found no
production consumers: layout calculations and `ResolutionScaler`, loading and
focus indicators, `FocusManager`, `Button` and `TabbedInterface`. Mouse states and
`DeliberateMouseTracker` went with the keyboard-only decision; the browser Debug
Workbench is exempt from that decision. Responsive widgets rebuild their own
geometry in `ensureLayout()` from `DisplayManager`, with shared spacing in
`dialog_constants.go` and font scaling in `typography.go` and `internal/text`.

Current APIs are documented in [Widget Components](../ui/WIDGET_COMPONENTS.md)
and [UI Widget System Architecture](../ui/UI_WIDGET_SYSTEM.md).
