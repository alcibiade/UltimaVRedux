# Coding Conventions

This document defines the coding standards for the project. It balances idiomatic Go, public Google Go style guidance, and the project’s practical needs (classic game fidelity, clarity, and incremental development).

Goals:
- Keep core gameplay logic clear and simple.
- Favor readability over cleverness.
- Prefer conventions that make the code testable, deterministic, and reusable.

## Language and Tooling

- Go version: the repository’s go.mod defines the version; keep code compatible with it.
- Formatting: gofmt (via goimports). CI should fail if formatting is not applied.
- Linting: golangci-lint configuration makes final determinations on style not otherwise stated in this document.
- Build tags: use sparingly (e.g., debug vs release).

### Python tooling

Maintained Python, including tests, requires **Python 3.12+**. Annotate function
parameters and returns completely, including nested callbacks. Model structured
records with `TypedDict` or dataclasses and callable contracts with `Protocol`;
use concrete container element types and narrow unions. Inferred local types are
fine. Avoid blanket `Any`, casts that conceal unvalidated assumptions, or empty
annotations added only to satisfy a checker. JSON boundary casts describe known
export schemas; validation still owns runtime error handling.

Use **Black 26.10.1**, stable mode and its default **88-column** layout, and
**Pylint 4.0.4**, including `pylint.extensions.typing`. All enabled diagnostics
must be zero across the full Python inventory, with no changed-line filter or
debt baseline. Scoring is disabled and failures retain nonzero exits. Suppress
only a specific diagnostic at the narrowest useful scope with a reason next to
it; stale suppressions fail. Test names may serve as method documentation, and
explicit binary-layout variables or established CLI signatures can justify local
complexity exceptions after review. Do not disable whole diagnostic categories.

These are the only Python format/lint tools; Mypy and Pyright are not part of the
toolchain. Preserve standard-library-only operational scripts and their CLI
behavior. Keep development dependencies in the hashed quality lock and dedicated
environment, separate from game/runtime dependencies. See
[Python quality workflow](DEVELOPMENT_GUIDE.md#python-quality-workflow) for setup,
inventory rules and check-only hooks.

## Package Structure and Boundaries

- Core logic should be independent of the rendering/input backend. Adapter layers (e.g., Ebiten) call into core, not vice versa.
- Keep packages focused and small; avoid cyclic dependencies.
- Exported types and functions should be stable and usable by external consumers where reasonable.

## Naming

- Exported identifiers: PascalCase. Unexported: camelCase.
- Initialisms: follow Go’s convention (ID, URL, HTTP). Example: UserID, not UserId.
- File names: lowercase with underscores only when it improves clarity; prefer consistent grouping by feature/domain.

## Imports and Dependencies

- **Import grouping**: Standard library first, external packages second, internal packages third; keep groups separated by blank lines:
  ```go
  import (
      "fmt"           // Standard library
      "strings"       // Standard library

      "github.com/hajimehoshi/ebiten/v2"                    // External packages
      "github.com/spf13/viper"

      "github.com/bradhannah/UltimaVRedux/internal/config"    // Internal packages
      "github.com/bradhannah/UltimaVRedux/internal/sprites"
  )
  ```
- **Import aliases**: Avoid unnecessary aliases like `references2`. Only use aliases when:
  - Genuine naming conflicts exist (e.g., `etext` for ebiten text vs internal text package, `ucolor` for project color vs `image/color`)
  - Package names are extremely long and used frequently
  - Standard library conflicts require disambiguation
- **Import formatting**: Use `goimports` to maintain consistent formatting and grouping
- Do not import the adapter layer (e.g., rendering/input libraries) into core logic. Inject via interfaces if needed.
- Keep external dependencies minimal and pinned via go.mod/go.sum.

## Errors and Logging

**Error Strategy**: Use `panic` for programming bugs and impossible states. Use
`log.Fatal` only at the application boundary for unrecoverable startup/resource
failures. Return wrapped errors for user input and recoverable conditions.

**See**: [Error Handling](ERROR_HANDLING.md) for complete guidelines.

## Logging System

**Use centralized logging instead of fmt.Printf, log.Printf, or direct console output.**

```go
// ✅ CORRECT: Use logging system
import "github.com/bradhannah/UltimaVRedux/internal/logging"

logging.Debug(logging.CategoryConversation, "Processing command: %s", cmd)
logging.Info(logging.CategoryUI, "Dialog opened: %s", title)
logging.Warn(logging.CategoryAudio, "Failed to load sound: %v", err)
logging.Error(logging.CategorySystem, "Configuration error: %v", err)

// ❌ WRONG: Direct console output
fmt.Printf("DEBUG: Processing command: %s\n", cmd)
log.Printf("Warning: Failed to load sound: %v", err)
```

### Logging Categories

Use appropriate categories for different systems:
- `CategoryConversation` - Dialog and NPC conversations
- `CategoryUI` - User interface components and interactions
- `CategoryAudio` - Sound and music system
- `CategoryGameState` - Game logic and state management
- `CategorySystem` - File I/O, configuration, system operations
- `CategoryMoongate` - Moongate-specific functionality
- `CategoryAI` - NPC AI behavior

### Creating New Categories

If existing categories don't fit:
1. Add constant to `internal/logging/categories.go`
2. Verify the typed Workbench Logging Control catalog discovers it
3. Use consistent naming: `Category<SystemName>`

### Debug Workbench

Use **System > Logging Control** for global/stdout switches, minimum level, and explicit category controls. Use the separate read-only log viewer for structured history.

## Event Queue System

Use `Workflow(name)` for gameplay orchestration, `TriggerFeedback(...)` for presentation-only effects, `AwaitFeedback(...)` when a workflow must wait for presentation, and `ThenSequence(...)` for reusable steps owned by the current workflow. Only workflows may call `BlockInput()`; feedback must not mutate gameplay.

**See**: [Event Queue System](systems/EVENT_QUEUE_SYSTEM.md) for complete documentation.

## Nil Return Values

Make absence explicit in the API contract. Use a documented optional pointer,
`(value, found)` result, or wrapped error as appropriate; do not invent magic
sentinels when zero is a valid domain value. Avoid typed-nil interface results.
Preserve nil-versus-empty collection distinctions when a wire contract requires
them. Missing mandatory initialization dependencies are programming errors,
not optional values to ignore.

## Time, Randomness, and Ticks

- Do not call `time.Now()` inside core logic or animation helpers that affect gameplay determinism.
- Use the appropriate `GameState` time source:
  - `ElapsedMs` for presentation animation and event-queue scheduling.
  - `DateTime`, turns, and explicit gameplay-minute advancement for schedules, durations, AI cadence, and spawn checks.
- Randomness:
  - Centralize PRNG state and seed per session.
  - For deterministic tests, allow fixed seeds.

## Loops and Collections

- Prefer `range` loops for readability and safety when index is not needed:
  - Good: `for _, v := range items { ... }`
  - Use index-based loops when:
    - You need the index for math or spatial mapping.
    - You need to mutate by index efficiently.
    - You’re iterating over a fixed-size array where index is semantic.
- Preallocate slices with capacity when the size is known (`make(T, 0, n)` or `make(T, n)`).
- Avoid unnecessary allocations in hot paths; reuse buffers where appropriate.

## Data-Driven vs Hardcoded Logic

- Favor data-driven definitions for tile attributes, schedules, and references.
- It is acceptable to hardcode small, well-named rules when it materially improves clarity (e.g., special-case mirror behavior, chair/ladder substitutions).
- When hardcoding, isolate rules behind small helpers so they can be replaced by data later if needed.

### Tile Checking Patterns

**DO NOT create individual Is* functions for single tile matches**. Instead, use a generic `Is(SpriteIndex)` function:

```go
// ❌ AVOID: Creating individual functions for every tile type
func (t *Tile) IsCactus() bool { return t.Index == indexes.Cactus }
func (t *Tile) IsTree() bool { return t.Index == indexes.Tree }
func (t *Tile) IsRock() bool { return t.Index == indexes.Rock }

// ✅ PREFERRED: Generic function for single tile checks
func (t *Tile) Is(spriteIndex indexes.SpriteIndex) bool {
    return t.Index == spriteIndex
}

// Usage examples:
if tile.Is(indexes.Cactus) { /* handle cactus */ }
if tile.Is(indexes.Tree) { /* handle tree */ }
if tile.Is(indexes.Rock) { /* handle rock */ }
```

**Create specific functions only for logical groupings or complex checks**:

```go
// ✅ GOOD: Logical groupings that represent game concepts
func (t *Tile) IsDoor() bool {
    return t.Index == indexes.RegularDoor || t.Index == indexes.LockedDoor || t.Index == indexes.MagicLockDoor
}

func (t *Tile) IsPushable() bool {
    return t.IsChair() || t.IsCannon() || // Logical groupings for multiple variants
           t.Is(indexes.Barrel) || t.Is(indexes.TableMiddle) || t.Is(indexes.Chest) // Single tiles
}

func (t *Tile) IsPassable() bool {
    // Complex logic for determining if tile can be walked through
}
```

This pattern:
- **Reduces code duplication** - One generic function instead of many specific ones
- **Maintains readability** - `tile.Is(indexes.Cactus)` is clear and concise
- **Prevents function proliferation** - Avoids hundreds of single-purpose Is* functions
- **Preserves type safety** - Still uses strongly-typed sprite indexes

## Constructor Patterns and Struct Safety

**Safety-first principle**: Always use `New<StructName>` constructor functions with input validation and safe defaults.

```go
// ✅ GOOD: Constructor with validation
func NewMessageDialog(message string) *MessageDialogBuilder {
    return &MessageDialogBuilder{
        message:          message,
        enableEffects:    true,
        enableDualOutput: false, // Safe default
        fadeInMs:         MessageDialogFadeInMs,
    }
}

// ❌ AVOID: Manual struct creation bypasses validation
dialog := &MessageDialog{config: config} // Missing initialization
```

**Use constructors for**: All exported structs, complex initialization, resource allocation, input validation
**May skip for**: Simple value objects, test fixtures

## Rendering and Animation

- Prefer simple, deterministic animation helpers that accept `(spriteIndex, positionHash, elapsedMs)`.
- Keep per-frame allocations to a minimum (reuse images/options where possible).
- Based on original Ultima V: Water and lava tiles have animation sequences that cycle through multiple frames (e.g., lava sprite 0x8f animates through sequences like `{1,0x8f,2,3,4}`).

## AI and Pathfinding

- Cache computed paths (A*) on the unit and consume steps over subsequent ticks.
- Recompute on cooldowns or when blocked; fall back to simple greedy movement if needed.
- Respect terrain passability by agent type (avatar, vehicle, land/water enemy).
- Do not hard-exit on missing paths; use an explicit fallback and log through the centralized logger when the fallback is diagnostically useful.

## Concurrency

- Prefer single-threaded game-state mutation during the update step.
- If background loading is needed, communicate via channels or synchronized buffers, and apply results during the main update.

## Comments and Documentation

- Use doc comments for exported identifiers; explain the why, not the obvious what.
- Keep comments up-to-date with behavior; incorrect comments are worse than none.
- Add package-level docs for domains with non-trivial rules (e.g., AI, schedule resolution).
- Keep execution results and unfinished work in GitHub Issues. Durable docs
  explain ownership, usage, rationale, and evidence; avoid ticket completion
  ledgers and copied API inventories. Preserve canonical Game Facts references
  and explicit behavioral limits when consolidating them.

### Deliberate Original-Game Deviations

Use the exact searchable tag `REDUX DEVIATION:` beside logic that intentionally
differs from verified original Ultima V behavior:

```go
// REDUX DEVIATION: COMBAT-0009
// The original can divide a surviving monster after a zero-damage hit.
// Redux treats full deflection as an unsuccessful hit, so it does not divide.
```

- State the original rule and its canonical public Game Facts ID; for this
  example see [COMBAT-0009](gamefacts/COMBAT.md#combat-0009). `make gamefacts-check`
  rejects a tagged comment block that names no known ID. Purely presentational
  choices need no tag; describe them in a plain comment.
- State the Redux behavior and why the difference is intentional.
- Add the tag opportunistically when changed or reviewed logic exposes a
  deliberate deviation; do not perform unrelated mass comment churn.
- Do not tag an unknown, accidental mismatch, temporary stub, or open fidelity
  defect. Research it, fix it, or track it as work instead.
- Reflect material gameplay deviations in the owning `docs/gamefacts/` file and
  cover the chosen Redux behavior with a regression test.

## Testing

- Unit tests for deterministic pieces: animation frame selection, schedule resolution, pathfinding.
- Use fixed seeds and a mock clock for deterministic tests.
- Avoid brittle pixel tests; prefer snapshot structures that capture tile IDs, frame indices, and positions.

## Performance and Allocation

- Avoid per-frame heap churn in tight loops.
- Use small structs and pass pointers only when mutation is required or copying is expensive.
- Profile with pprof before optimizing; optimize only bottlenecks.

## Command Input Patterns

Player commands use a two-step Primary/Secondary input state machine (command key → direction key) with SystemCallbacks for UI interaction.

**See**: [Input Handling](systems/INPUT_HANDLING.md) for complete input system architecture, state machine details, and SystemCallbacks patterns.

## UI Modal and Dialog Patterns

Use shared border widgets, fluent builder pattern for dialogs, and dialog stack for input isolation.

**See**: [UI Widget System](ui/UI_WIDGET_SYSTEM.md) and [Modal Dialog System](ui/MODAL_DIALOG_SYSTEM.md) for selection, construction, and lifecycle patterns.

## Text Rendering

**CRITICAL**: Always use the `text.Output` pattern for ALL text rendering. Never call DrawText methods directly.

### Required Pattern
```go
output.Reset()
output.AppendLine().Yellow("Label: ").White("Value").Done()
output.DrawPanelContent(screen, point, width, etext.AlignStart)
```

### Retired compatibility entry points

The unused `Output.DrawText`, `DrawTextCenter`, and `DrawTextRightToLeft` entry
points have been removed. Use panel/scrolling Output rendering instead. The
policy check rejects reintroducing calls to these legacy methods; lower-level
rasterization remains private to the game text adapter and the SDK renderer.

Use [Message Output](systems/MESSAGE_OUTPUT_SYSTEM.md) for ownership and
composition contracts. Keys/chords and action hints use separate semantic
control-key and control-hint colours; labels and values likewise remain distinct.
Preserve selected-row contrast and show availability in text as well as colour.

---

Repository-wide regression rules, including dual-layer collision and Jimmy's
failure-only key consumption, live in `AGENTS.md`. Use
[Testing](TESTING.md#test-review-checklist) for regression review and
[Architecture Guidelines](ARCHITECTURE_GUIDELINES.md) for boundary decisions.
