# Modal Dialog System

Choose and construct dialogs, then establish one input and cleanup owner.
`internal/ui/widgets/dialog_stack.go` is the stack implementation; concrete
components and their source entry points are in [Widget Components](WIDGET_COMPONENTS.md).

## Selection Guide

| Need | Use |
|---|---|
| Ordinary map/journal update | Main `text.Output`, with normal History; no popup or acknowledgement. |
| Ammunition threshold/depletion warning | Nonblocking `ShowActionFeedback` with a main-output copy. |
| Short feedback that must not block gameplay | `SystemCallbacks.UI.ShowActionFeedback` |
| Combat pickup result | Structured combat report entry; no transient map dialog. |
| Chest discovery / departure totals | `CombatLootDialog` / `NewCombatSummaryDialog`; see [combat loot feedback](GAME_UI_INTEGRATION.md#combat-loot-feedback). |
| Modal status with a scene-owned deadline | `ShowTimedOverlayMessage` |
| Acknowledged popup | `OverlayDialog`, `AsConfirmation`, `WithUserDismissed` |
| Staged prompt/response transcript | Workflow-owned `InteractionTranscript` |
| Mutable multiline narrative, optionally with a prompt | `MessageDialog` |
| Grammar, regex, yes/no, or numeric input | `InputBox` or its prefabs |
| Filesystem path and platform text entry | `LegacyPathEntryModal` |
| Saves, party, items, equipment, spells, or map selection | Existing domain dialog; see the component guide |
| Personal atlas note authoring / deletion | Scene-owned `AtlasNoteEditor` child inside the atlas pause; core validates the captured draft. |

## Construction

### OverlayDialog

`NewOverlayDialog(...).Build()` requires sizing, position, effects, vertical
content alignment, horizontal text alignment, and exactly one dismissal mode.
`AsConfirmation` and `AsOverlayFeedback` supply appearance only.
`AsBriefFeedback` also supplies a 2000 ms duration.

- `WithDuration(ms)` advances by 16 ms per widget Update, not wall time.
- `WithExternallyManagedDuration(ms)` disables the widget timer; its owner must
  expire and remove the instance. GameScene's timed-overlay installer uses it.
- `WithUserDismissed(keyboard)` accepts Enter/Escape and supplies the shared
  acknowledgement footer, regardless of appearance template.

```go
dialog := widgets.NewOverlayDialog("Quest updated!").
    AsConfirmation().
    WithUserDismissed(keyboard).
    AutoPopFromStack(&stack).
    Build()
dialog.ShowWithoutAnimation()
stack.PushModalDialog(dialog)
```

Here the dialog owns exact-instance removal. For cancellation-safe workflow
ownership, omit auto-pop and use the [workflow pattern](#event-workflow-pattern).
Never assign both cleanup owners. Timed overlays have no acknowledgement footer
and do not auto-remove merely by hiding.

### MessageDialog

Use `NewMessageDialog` for mutable styled content. Prompt methods require
`WithSeparatePromptArea` or `WithInputDismissal`; missing prompt capacity panics.
`WithInputDismissal` supplies the shared Close hint. A workflow accepting any key
instead reserves the prompt area, sets `DialogAnyKeyHint` when ready, and clears
it before hiding; that prompt does not transfer input ownership to the widget.

```go
dialog := widgets.NewMessageDialog("Setting up camp...").
    WithTextLines(6).
    WithCharsPerLine(40).
    WithSeparatePromptArea().
    WithHorizontalPosition(widgets.DialogHorizontalMapOnly).
    WithVerticalPosition(widgets.DialogDefaultVerticalPos).
    Build()
dialog.AppendLineBuilder().White("Found: ").Yellow("50 gold").Done()
```

Always commit line builders with `Done`. Ordinary dissolve transitions use a
wall clock; transient feedback injects its scene clock before `Show` so movement
protection and reveal/hold/hide remain coordinated. Keep a hiding dialog updating
until its fade completes, then remove the exact instance. The camping and action
feedback prefabs have distinct owners; do not substitute an action-feedback
dialog for a modal confirmation.

Message layout retains frame rectangles and wrapped output without constructing
legacy border/background textures. Rendering draws its classic frame directly
into the dialog-owned composition buffer. Resize releases the old composition,
mask buffer, and dissolve resources before replacement. These full-size buffers
use standalone storage rather than sharing an atlas with persistent small glyphs.
`OverlayDialog.PrepareLayout` can resolve wrapping and bounds without drawing,
consuming input, advancing duration, or invoking callbacks.

### InputBox And Transcript

`InputBox` supports grammar commands or regex validation, not both. Its builder
validates required callbacks. `WithResultCallback` communicates a choice to the
owner; `NewYesNoInputBox`, `NewYellDialog`, and `NewSleepDurationDialog` provide
established configurations. For example:

```go
dialog := widgets.NewInputBox("Name thy wish:", nil, keyboard).
    WithRegexValidation(`^[A-Za-z ]*$`).
    WithResultCallback(onResult).
    AutoCloseAndPop(&stack).
    Build()
stack.PushModalDialog(dialog)
```

`InteractionTranscript` captures raw responses and displays explicit Close or
Continue hints only while accepting input. Its workflow owns sequencing,
authoritative mutation, time, and cleanup. Keep hints separate from narrative
text and match the keys actually accepted.

### Popup Placement

Map-relative MessageDialog/OverlayDialog popups use active combat battlefield
bounds, excluding Turn Order. Explicit screen/numeric anchors bypass this.
`WithHorizontalCenter` overrides the horizontal anchor; a later `WithPosition`
resets the override. Viewport changes rewrap a copy of original styled content,
preserving Runic/color segments without changing focus, timing, or ownership.
InputBox and large selection/workspace dialogs do not use this policy.

Action/Search feedback additionally opts into scene-owned full-content
measurement, translation, clipping, and a bounded stack. Do not reuse those
opt-in lifecycle seams for ordinary modal content.

## Stack Model

`DialogStack` stores private entries containing a `Widget` and a modal flag:

```go
type dialogEntry struct {
    widget  Widget
    isModal bool
}
```

Push APIs are explicit:

```go
stack.PushModalDialog(dialog)
stack.PushNonBlockingDialog(dialog)
```

Modal dialogs require exclusive user interaction. Non-blocking dialogs render and update without preventing gameplay input; transient action feedback uses this lane.

Pushing nil or the same widget instance twice panics. A stack entry is an ownership record, not merely a visibility flag.

Retain and remove exact instances instead of blindly removing whatever happens to be on top.

## Input Ownership

Outside the [Journal pause](../systems/TIME_SYSTEM.md#journal-pause),
`GameScene.Update` advances event queues and non-blocking dialogs before modal routing. It then calls `updateDialogs`, which updates only `PeekTopModalDialog`. If a modal exists, main gameplay input does not run that frame.

The effective ownership order is:

1. Event workflows advance and may own the global input block.
2. Non-blocking dialogs advance without consuming modal input.
3. The highest modal dialog receives one `Update` call.
4. If no modal exists and global input is not blocked, secondary then primary game input may run.

Lower modal dialogs remain in the stack and are drawn, but they do not receive `Update` until dialogs above them are removed.

Modal widgets that own platform text focus are notified when they become or
cease to be the highest modal. This private stack contract lets
`LegacyPathEntryModal` focus Ebitengine's native text field only while it owns
input; non-blocking dialogs do not disturb that focus. Clipboard text is read
only after an explicit platform paste shortcut, and failed validation retains
that focus for correction.

`HasOpenDialog` includes non-blocking entries. Use `HasModalDialog` to answer whether the stack owns gameplay input.

## Drawing

`DialogStack.Draw` renders every entry from oldest to newest, producing bottom-to-top visual order. Modal and non-blocking entries share this draw order; modal status controls input, not z-order.

`Dialogs()` returns an allocated snapshot of widget references for diagnostics or exceptional iteration. Normal drawing uses `Draw` directly.

## Removal

Prefer pointer identity:

```go
if !stack.RemoveWidgetInstance(dialog) {
    // The owner may treat an already-removed transient as idempotent cleanup.
}
```

Use `MustRemoveWidgetInstance` when loss of stack ownership is a programming error. It panics if the instance is absent and is used by strict auto-pop contracts.

`RemoveWidget` removes the first matching type and logs if none exists. It is unsuitable when several overlays or message dialogs can coexist.

## Visibility Versus Ownership

`MessageDialog.Hide` may begin a fade while the dialog remains visible and must continue receiving updates. Remove it only after the fade interval or in guaranteed cleanup.

Self-timed `OverlayDialog` dismissal hides the wrapped message dialog but does not remove the stack entry. User-dismissed overlays can use `AutoPopFromStack`; workflow-owned overlays should be removed by the workflow's `Finally` callback.

GameScene timed overlays instead use `WithExternallyManagedDuration`. `OverlayDialog.Update` does not advance those dialogs: GameScene installs one exact instance, tracks its deadline with `GameState.ElapsedMs`, and removes it even when another modal covers it. Replacement, expiry, and scene teardown share the same exact-instance cleanup.

Non-blocking action feedback has separate GameScene ownership. The owner retains stable exact instances, reconciles protected lifecycle and movement before the stack's one non-blocking update, and promptly retires an instance removed elsewhere. Capacity eviction, expiry, scene replacement, and disposal remove only owned feedback and tolerate idempotent cleanup.

## Event Workflow Pattern

```go
var dialog *widgets.OverlayDialog

manager.Workflow("acknowledgement").
    Priority(event_queue.PriorityMedium).
    Sequential().
    BlockInput().
    First(func() {
        dialog = widgets.NewOverlayDialog("Continue?").
            AsConfirmation().
            WithUserDismissed(keyboard).
            Build()
        dialog.ShowWithoutAnimation()
        stack.PushModalDialog(dialog)
    }).
    While(func() bool { return !dialog.IsDismissed() }, func() {}, 16).
    Finally(func() {
        if dialog != nil {
            stack.RemoveWidgetInstance(dialog)
        }
    }).
    Start()
```

Do not use `BlockInput` on feedback. Do not start an unrelated workflow and assume the parent waits for it; insert owned steps with `ThenSequence`.

## Input Debouncing

Dialog transitions must prevent the key that submitted one state from immediately dismissing the next. Use `Keyboard.PrepareForTransition`, advance release state with the keyboard acceptance methods, and accept only a fresh press. Linear conversation and event-queue helpers contain current examples.

Atlas note editors keep the parent mounted and frozen. Only the top child accepts
text; successful commit or cancellation removes that exact child and release-gates
return to the atlas. Parent teardown removes the child first and discards its
single-use draft. Failed validation retains text and focus. See
[Vessels and personal notes](WIDGET_COMPONENTS.md#vessels-and-personal-notes).

Signs retain their own any-key acknowledgement and release gate. Their hidden
maintainer translation view is local to one reading and resets when the passive
Runic reference closes; its chord must be consumed before raw any-key polling.
It never becomes a second modal or an extra command Help entry. See
[Sign reading](../gamefacts/SIGNS_AND_FONTS.md#sign-0008).

## Configuration Transaction

`ConfigurationDialog` is a keyboard-only modal available from both the title
menu and the in-game Escape menu. It keeps detached original and staged copies
of Sound, Display, and Gameplay settings. The title-menu route opens on Display;
the Escape-menu route opens on Sound. Sound changes may preview through the
audio manager, but no persisted configuration model changes before Apply. Tab
changes section; Up/Down changes the active row; Left/Right changes discrete
Sound values or Display choices; Space toggles the selected Gameplay value.
Gameplay includes **Flexible Sextant Use**, an off-by-default Redux option for
daytime and indoor surface-settlement readings. It requires a sextant and excludes
dungeons and the Underworld. It stages the existing
`Gameplay.Navigation.FlexibleSextantUse` preference with the same Apply/Cancel
transaction as the other settings. The seven measured Gameplay rows use compact
vertical padding and gaps, preserving normal typography at supported sizes.
The responsive card uses the
complete host pane, equal-width tabs, and independently measured Display label
and value columns.

Display offers Windowed, Maximized Window, and Borderless Fullscreen. Windowed
mode also offers preset sizes and preserves a valid custom size; presets larger
than the current monitor remain visible as disabled choices. The other modes
show platform-controlled dimensions. Applying a changed display
selection first requests the asynchronous platform transition. Once observed,
the dialog requires Keep or Revert confirmation and automatically reverts after
12 seconds. Transition timeout, explicit Revert, Escape, or persistence failure
requests restoration of the prior mode and normal window size.

`Enter` persists all settings through `SaveConfigurationSettings` after any
display confirmation. Persistence writes a complete candidate configuration to
a same-directory temporary file and atomically replaces the active file before
updating the existing Audio, Display, and Gameplay model pointers. A failure
leaves the models and Viper state unchanged, keeps the dialog open, and offers
retry or cancel after display rollback.

`Escape` writes nothing, restores previewed volumes, stops owned preview
playback, and dismisses the exact Configuration instance. Apply performs the
same preview cleanup after persistence succeeds. When opened in-game, the
underlying Escape menu remains stack-owned, so game audio stays disabled until
that menu closes. When opened from the title, dismissal restores title-menu
audio and returns input ownership to the title menu.

The section and row selections survive section changes and responsive relayout. Each update accepts at most one bound key, and dismissal is guarded so cleanup cannot remove the same instance twice.

## Invariants

- One concrete owner removes each dialog instance.
- Only the highest modal processes input.
- Non-blocking dialogs never imply a gameplay input block.
- A hidden dialog may still be stack-owned.
- Required stack and keyboard dependencies are validated during construction; do not add silent nil fallbacks around mandatory integration.
- Cleanup that must survive cancellation belongs in `Finally`.

See [Game UI Integration](GAME_UI_INTEGRATION.md) for scene wiring and
`internal/ui/widgets/dialog_stack_test.go`, `overlay_dialog_test.go`, and
`message_dialog_test.go` for focused lifecycle/constructor coverage. Scene
ownership tests live in `cmd/ultimav/gamescene_dialog_lifecycle_test.go`.
