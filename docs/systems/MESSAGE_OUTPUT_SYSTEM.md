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

### Action And Search Feedback History

**Maximum Action Popups** (`Gameplay.Presentation.MaxVisibleActionPopups`) is an
integer from 0 to 3, default 3. Off (0) creates no transient action/Search card;
it does not disable the bottom-right output or its retained history. Values 1–3
share one newest-first stack for both adapters, further bounded by available
space. Admission evicts the oldest exact card instance, including its chest
association, without removing the recorded result.

Each result has one explicit history owner, independent of the popup setting:

| Producer | History owner |
|---|---|
| `ShowActionFeedback(message, true)` | The callback appends to the main `Output` before checking popup admission. |
| `ShowActionFeedback(message, false)` | The caller records the result; the adapter does not add another copy, even at Off. |
| Surface `ShowSearchFeedback` | Search records its full result through `Message.AddRowStr`; the adapter only presents the card. |
| Surface/dungeon Look and dungeon Search | The caller retains the complete or prefixed narration, including darkness and refusal results. |
| Combat corpse/chest Search | The authoritative action records its result before requesting presentation. |
| Location entry/exit and telescope/spyglass outcomes or refusals | `ShowActionFeedback(..., true)` records the result; a command prompt alone is not the outcome. |
| Modal chest contents and structured combat reports | Their workflow/report owner retains the existing history and acknowledgement contract, outside the transient cap. |

Do not deduplicate strings: two harvests may legitimately emit identical results.
The popup cap cannot change crop, theft, loot, turn or RNG effects. A future typed
receipt using `NewItemReceiptDialog` must enter the shared transient admission
path and establish its history owner; constructing a prefab alone does neither.

A successfully applied lower cap retires excess existing cards on the next normal
scene update, without a new message or resize, including cards covered by a modal
or in reveal/hold/hide phases. Raising the limit never restores retired cards.
Staged edits, Cancel and failed persistence leave the live cap and cards intact.
Expiry, eviction and scene cleanup affect presentation only. See
[Action Feedback](../ui/GAME_UI_INTEGRATION.md#action-feedback) for layout and
lifecycle ownership.

### Shared Typography And Rendering Ownership

Redux consumes the pinned
[Berad Text module](https://github.com/bradhannah/berad-sdk/tree/1521d0b6526b/modules/berad-text).
`internal/text` is the game facade: assets, original glyph-byte mapping, command
markers, builders, transcript/history policy and display scaling remain here.
The module owns supplied-font parsing, immutable typography, styled layout/local
geometry, bitmap rasterization and bounded image caching. See
[ADR-011](../ADR/ADR-011-berad-text.md) for the extraction boundary.

`NewUltimaFont` shares one lazily parsed game TTF through the SDK. The game-owned
`TextFace` view remains for engine integrations; Source/features must stay read-only.
Legacy size/language settings are converted to immutable SDK configurations.
TextInput uses this same source and metrics rather than a second OpenType parser.
Runic/IBM bytes map to private-use runes in supplied bitmap descriptors; original
sign decoding and narrative-font policy remain local. Font-file hot reload is
not part of the game contract.

The application-owned SDK renderer retains up to 512 rasters and 16 MiB of logical
pixel payload plus source-text bytes. Fonts, geometry, engine glyph caches and
driver memory are separate; this is not a total GPU-memory guarantee. Equivalent
scaled typography has stable identity rather than a new face-pointer cache key.
Oversized cache entries bypass retention; per-render limits are checked before
allocation. Private cached images never escape. Compatibility `GetCharImage`
returns separately game-owned, fixed-glyph snapshots whose existing borrowers
must not modify/dispose them. Cache eviction cannot invalidate those snapshots.
Final application shutdown closes the SDK renderer; scene lifetimes share it.
Direct bitmap glyph drawing retains native-size rasters and applies scale at final
placement, preserving single-pass nearest sampling for the reduced sky glyphs.
Styled bitmap/TTF runs share palette-alpha semantics; the ordinary game palette
remains opaque.

Output retains bounded physical-row SDK layouts. Ordinary mixed styled rows shape
compatible fonts across colour boundaries; vector command-marker rows preserve
their original compatibility sequence. Recolouring and placement do not rebuild
geometry. Clear/Reset/SetFont invalidate retained rows; wrapped copies detach both
layout and logical-history ownership. Positive column capacity is required, while
zero visible rows remain valid. Local clipping and the simple-row two-pixel anchor
allowance preserve the former game raster canvas without restricting SDK ink bounds.

Measured source reflow delegates to SDK layout. Atlas/save-description/Tavern
controls use SDK source-range/caret geometry while owning native input, selection,
filtering, focus, scrolling and composition projection. Queries are local geometric
candidates, not click handlers: Z order, clipping eligibility and activation belong
to the UI owner. Shaped-cluster snapping is explicit; full bidi/vertical editing
or independent colouring inside a ligature is not promised. The game retains its
existing stored text and input policies.

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
