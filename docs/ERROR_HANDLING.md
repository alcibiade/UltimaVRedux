# Error Handling

Use the failure mechanism that matches who can recover and whether the state is
valid. Logging is diagnostic context, not a substitute for handling an error.

## Decision Table

| Condition | Mechanism |
|---|---|
| Programming bug, impossible state, or violated internal contract | `panic` |
| Application cannot start or continue because required core resources are unavailable or corrupt | `log.Fatal` at the application boundary |
| User input, configuration, filesystem, network, or other recoverable operation | Return a wrapped `error` |
| Expected gameplay rejection | Return a result and provide player feedback through `SystemCallbacks` |
| Recoverable diagnostic condition | Recover explicitly and log through `internal/logging` |

## Panic For Contract Bugs

Panic when continuing would hide invalid wiring or impossible state:

```go
if callbacks.Message.AddRowStr == nil {
    panic("MessageCallbacks.AddRowStr is required")
}

switch mapType {
case references.LargeMapType, references.SmallMapType:
    // Supported states.
default:
    panic("unhandled map type")
}
```

Constructors and initialization must validate mandatory systems and callbacks.
After successful initialization, callers use them directly without soft nil
checks. Optional callbacks must be explicitly documented as optional.

## Fatal Application Boundaries

Use `log.Fatal` only where the application cannot present or return the failure,
normally startup or the top-level engine boundary:

```go
if err := ebiten.RunGame(game); err != nil {
    log.Fatal(err)
}
```

Do not use `log.Fatal` for invalid user configuration, missing user save files,
ordinary gameplay validation, or internal contract bugs.

## Return Recoverable Errors

Wrap errors with operation context and preserve the cause:

```go
func LoadUserSave(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading save %s: %w", path, err)
    }
    return data, nil
}
```

The caller decides whether to retry, use a documented default, display a dialog,
or abort the current operation. Do not silently invent fallback state.

## Recoverable Diagnostics

When recovery is an explicit part of the contract, use the centralized logger:

```go
if index < 0 || index >= len(entries) {
    logging.Warn(logging.CategoryUI, "Ignoring stale selection index %d", index)
    return
}
```

Use the narrowest category and lowest useful level. See
[`systems/LOGGING_SYSTEM.md`](systems/LOGGING_SYSTEM.md) for category selection,
runtime controls, and logger-state restoration in tests.

## Gameplay And UI Feedback

Invalid player actions are expected outcomes, not process errors:

```go
if !gs.PartyState.Inventory.Provisions.Keys.HasAtLeast(1) {
    gs.SystemCallbacks.Message.AddRowStr("No keys!")
    return false
}
```

An unavailable action can return a result and message. A missing mandatory
dialog stack, callback group, or renderer dependency is an initialization bug
and must fail early instead of degrading silently.

## Review Checklist

- Can the caller reasonably recover? Return an error.
- Is this expected gameplay validation? Return a result and use callbacks.
- Is an internal invariant broken? Panic.
- Is the whole application unable to start or continue? Fail at the application boundary.
- Is recovery documented? Log through `internal/logging`; otherwise do not hide the failure.
- Are mandatory callbacks validated once and then used directly?
