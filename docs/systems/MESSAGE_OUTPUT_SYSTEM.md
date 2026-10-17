# Message Output System

The message output system carries gameplay feedback from `GameState` to the presentation layer without importing UI packages into core game logic.

## Callback Groups

`GameState.SystemCallbacks` separates three kinds of player-facing output:

| Group | Purpose | Typical calls |
|---|---|---|
| `Message` | Persistent command transcript and prompts | `AddRowStr`, `AppendToCurrentRowStr`, `ShowCommandPrompt` |
| `UI` | Transient or modal presentation | `ShowActionFeedback`, `ShowTimedOverlayMessage`, dialog-specific callbacks |
| `Combat` | Structured combat results and post-combat conclusions | `PublishEvent`, `PublishConclusion` |

Game logic should use `GameState.SystemCallbacks.Message` for ordinary text. It should use `UI` when the feedback has an explicit popup lifecycle and `Combat` when presentation needs typed combat data rather than a preformatted string.

```go
gs.SystemCallbacks.Message.ShowCommandPrompt("Open-")
gs.SystemCallbacks.Message.AddRowStr("Opened!")
gs.SystemCallbacks.UI.ShowActionFeedback("Unlocked!", true)
gs.SystemCallbacks.Combat.PublishEvent(event)
```

`MessageCallbacks.AppendStyledMessage` accepts detached `StyledMessage` values:
ordered text segments with spoken, quotation, or highlight semantics and optional
leading spacing. Core code exposes no `text.LineBuilder`. The production scene
adapts these values to colored `MessageDialog` output and a plain persistent
transcript; leading blank rows apply only to the primary presentation.

The gameplay transcript retains 64 circular-buffer rows and displays the newest
11 rows at 16 characters wide, including the current command prompt. The
[four-row provisions panel](../ui/GAME_UI_INTEGRATION.md#ammunition-summary-and-shortages)
grows upward into the message region. Rendering uses a reusable panel-sized
image, so text is clipped to the bounds-driven right-rail output region;
visible capacity comes from the shortened clip's measured height. Resolution
changes update font metrics without discarding transcript history.

## UI Adapter

`GameScene` owns the right-side `text.Output` and wires it in `wireSystemCallbacks`:

- `addRowStr` always appends to the output pane. If the top modal is a visible, non-static `MessageDialog`, it mirrors the line there.
- `appendToCurrentRowStr` appends to the current transcript line, commonly completing a command with its direction.
- `showCommandPrompt` starts an original-style command line through `Output.AddCommandPrompt`.
- `appendStyledMessage` adapts semantic segments while recording a plain transcript.

Core actions do not know whether a dialog is open and do not manipulate the dialog stack.

### Shared Typography And Rendering Ownership

`NewUltimaFont` shares one lazily parsed embedded TTF source, including the
engine's source-owned shaping/glyph caches. Each caller retains a separate face
with its own size/direction/language settings. TextInput separately shares its
parsed OpenType source while retaining independent faces for cursor metrics.
Bitmap Runic/IBM and original narrative fonts keep their distinct pipelines.

The shared rendered-text LRU uses comparable content/face/layout/colour keys;
warm lookups do not format strings. Keys retain strong face identities and
snapshot public GoTextFace rendering fields. Opaque OpenType feature/variation
settings must remain immutable while cached: replace the face or explicitly
clear the text cache after changing them. Colours are numeric RGBA tuples, and
layout values, not option-pointer addresses, identify entries.

The default budget is 512 entries, additionally bounded to 16 MiB of
cache-held logical pixel payload plus owned content strings. This is not a total
GPU-memory limit. Oversized results bypass admission rather than evicting the
entire working set. Returned images retain their existing escaping lifetime:
eviction drops cache references and must not dispose borrowed images.

Output retains bounded physical-row segment advances, not raster image pointers.
Unchanged content/typography reuses geometry; colour/alignment changes affect
paint/placement. Editing and circular overwrite are checked against the actual
row content; Clear/Reset and SetFont invalidate retained geometry. Wrapped copies
own independent layout storage. Short styled lines without newlines take a
rune-aware detached-copy fast path, preserving history and caller isolation.

CharacterSummary retains separate left/right text outputs per party slot until
visible values or semantic colour change. It still draws the current portraits
and uses the same measured header/loadout/status layout.

### Popup Reflow

`Output.WrappedCopyForWidth` creates a non-mutating chronological copy of the
retained rows, using measured glyph widths for normal, scaled, and bitmap fonts.
`WrappedCopy` provides the character-capacity variant. Both preserve original
row boundaries, interior blank rows, segment colors/fonts, and line alignment;
they do not join already-wrapped source rows when widening. Empty edge rows are
omitted. A glyph wider than the requested width still occupies its own row.

Map-relative message dialogs retain their source output and rebuild only the
display copy when content or viewport geometry changes. This avoids losing
styled text, prompts, or earlier rows when the combat battlefield is narrower
than the ordinary map pane. Normal Output insertion and scrolling capacity
remain unchanged.

### Opt-In Logical History

`Output.RetainLogicalHistory` retains detached already-inserted text rather than
only wrapped rows. `ReflowRetained` updates typography and replays those text
insertions only when wrapping changes; it never invokes gameplay callbacks.
`RetainedHistoryPanel` provides chronological physical rows for keyboard paging.
Retention is explicitly bounded by insertion count and text bytes, trims at
UTF-8 boundaries, and reports truncation. A font-only resize preserves even live
rows whose older logical insertions have been evicted. This is opt-in for the
current Talk dialog, separate from the 64-row gameplay transcript and the saved
plain Journal projection below. Talk controls and limits are documented in
[Conversation system](LINEAR_CONVERSATION_SYSTEM.md#runtime-path).

### Journal History

The main right-rail Output separately opts into `EnableJournalHistory`.
`JournalHistory` returns a detached canonical plain string captured at Output
mutation seams, before 16-character rail wrapping. Modal-only speech is excluded;
dual-output callbacks record only the mutation reaching the main Output. The
live rail and Talk's styled history retain their existing behavior.
The API retains its historical name, but Journal no longer has a History reader
or search source.

- Retain at most 64 nonblank logical entries, 4,096 UTF-8 bytes per logical line,
  and 65,536 bytes of canonical retained text including separators, not JSON
  escaping/DTO overhead. Repeated identical emissions remain separate entries.
- LF separates entries; double LF preserves at most one explicitly emitted blank
  separator. Empty History is `""`. Remove blank edge padding; separators consume
  display rows but not entry slots. Preserve ordinary authored spaces within
  nonblank lines and invent no spacing between styled segments.
- Plain Add APIs discard one terminal empty LF component; empty input is a no-op.
  Styled Add and builder Done preserve terminal logical rows. Append amends the
  current logical target, not the last nonblank entry across a blank or prompt.
  Builder NewLine commits its actual spacing. A marker-only command prompt
  creates a new append target but is neither a message nor a blank separator;
  graphical prompt-marker segments are omitted from History.
- Oversized lines retain a complete-rune prefix plus the 16-byte
  `[text shortened]` marker within the line limit. Appends cannot regrow the
  discarded suffix, but subsequent logical lines are still processed. Count or
  aggregate overflow evicts oldest complete entries and exposed leading spacing.
- Only this live projection normalizes malformed UTF-8 to replacement runes and
  controls/Unicode line separators other than LF to ordinary spaces. Saved
  History must already be canonical; hydration rejects rather than repairs it.

For example, plain `AddText("A\n"); AppendText("B")` retains `AB`, whereas
styled Add of `A\n` followed by Append of `B` retains `A\nB`, and followed by
Add of `B` retains `A\n\nB`. These API distinctions are covered in
`internal/text/output_journal_history_test.go`.

`RestoreJournalHistory` validates before replacing the collector and rail,
rebuilding monochrome wrapped rail rows without callback mirroring or recording
them again. Accepted canonical source recaptures exactly. The scene snapshots
this source in saves for main-rail restoration; it is not passed into the Journal
widget. Permanent Chronicle discoveries are a separate
authority and are never subject to History retention limits. See
[native presentation](../details/SAVE_SNAPSHOT_SCHEMA.md#presentation) for save
ownership and strict byte-aware validation.

## Initialization Contract

Callbacks are constructed and validated before gameplay begins:

1. `NewMessageCallbacks` requires row, append, command-prompt, and `AppendStyledMessage` functions.
2. `NewCombatCallbacks` requires both `PublishEvent` and `PublishConclusion`.
3. `NewSystemCallbacks` validates the mandatory cross-group contract, including message output, action/timed UI feedback, combat event publication, combat impact presentation, audio, and stats display.
4. `GameScene.wireSystemCallbacks` constructs the named callback groups first; violations of those programming contracts panic. It wraps and returns an initialization error if aggregate cross-group validation fails.
5. `initializeResizeableVisualElements` installs scaled transcript typography and preserves existing transcript contents. Styled output support is wired separately through `AppendStyledMessage`.

Mandatory callbacks are invariants. Gameplay code calls them directly; it must not add runtime nil checks or silently skip required feedback. Tests should install `MockSystemCallbacks` or construct the required callback groups instead of relying on partial structs.

Some explicitly documented extension callbacks are optional. Code that requires one must establish that stronger contract during feature initialization, not repeatedly probe it at each message site.

## Ownership Rules

Keyboard guidance separates key/chord ink (`DialogControlKeyColor`, white) from
action descriptions (`DialogControlHintColor`, light grey). Shared merchant
footers accept bracketed keys and recognized colon-style controls, preserving
text, spacing and logical newlines before wrapping; ordinary status prose with a
colon is not a shortcut. `SetAcknowledgementPrompt` composes these semantic
segments in its reserved output without taking workflow input ownership.
Generic `SetPromptWithColor` remains a single-colour prose API. Error/status text
retains its meaning; retry/cancel controls live in the visible styled footer or
an explicitly separate hint, not duplicated inside the error.

- Authoritative gameplay outcomes are decided in `GameState`; callbacks only present those outcomes.
- Do not duplicate an action result in both `GameState` and its input handler.
- Use `Message` for the durable transcript, `UI` for lifecycle-owned overlays, and `Combat` for typed combat events.
- Command prompts are started by primary input and completed by secondary input.
- Presentation adapters use `text.Output` and `LineBuilder`; core producers send plain or styled message values. Do not call low-level text drawing functions directly.

## Key Files

- `internal/game_state/message_callbacks.go`
- `internal/game_state/system_callbacks.go`
- `internal/game_state/combat_events.go`
- `cmd/ultimav/gamescene_callbacks.go`
- `cmd/ultimav/gamescene_message_dialog.go`
- `cmd/ultimav/gamescene_initialization.go`
- `cmd/ultimav/gamescene_right_rail.go`
- `internal/text/output_journal_history.go`
- `internal/text/journal_history_wrapping.go`
- `internal/journal/history.go`
