# Logging System

## Purpose

`internal/logging` provides categorized, leveled diagnostics with runtime filtering, optional stdout output, and a bounded structured history for the browser Debug Workbench.

Logging is disabled by default. A record is retained only when all of these conditions are true:

- Global logging is enabled.
- The record's category is enabled.
- The record's level meets the configured minimum.

## Basic Usage

```go
import "github.com/bradhannah/UltimaVRedux/internal/logging"

logging.Debug(logging.CategoryConversation, "Processing command: %s", command)
logging.Info(logging.CategoryUI, "Dialog opened: %s", title)
logging.Warn(logging.CategoryAudio, "Optional sound unavailable: %v", err)
logging.Error(logging.CategorySystem, "Configuration operation failed: %v", err)
```

Use the most specific existing category. `internal/logging/categories.go` is the
category registry and API authority; do not maintain a copied inventory.
`GetAllCategories()` supplies the current list to Workbench discovery.

Levels are ordered `DEBUG`, `INFO`, `WARN`, `ERROR`, and `FATAL`.

## Fatal Level

`logging.Fatal` records a `FATAL`-level diagnostic; it does not terminate the process. If logging is disabled or filtered, it records nothing.

Use the project's normal error strategy for control flow:

- Return an error for recoverable failures.
- Use `panic` for programming or contract violations.
- Use the standard library's `log.Fatal` only for unrecoverable application startup failures.

Do not rely on `logging.Fatal` to stop execution.

## Runtime Control

Start the browser Debug Workbench and open **System > Logging Control**. The instrument exposes:

- Global enable state.
- Stdout enable state.
- Minimum level.
- Individual category states.
- Enable-all and disable-all operations.

The separate `/logs` route is a read-only viewer for structured history. It has its own local filters and reads the bounded log buffer without changing logger configuration.

Workbench logging mutations update runtime state and persist through `UltimaVConfiguration.SaveLoggingConfiguration`. Direct calls to logger setters change runtime state only unless their caller also persists configuration.

## Configuration API

Package-level helpers cover common runtime operations:

```go
logging.SetGlobalEnabled(true)
logging.SetStdoutEnabled(true)
logging.SetCategoryEnabled(logging.CategoryGameState, true)

enabled := logging.IsGlobalEnabled()
categoryEnabled := logging.IsCategoryEnabled(logging.CategoryGameState)
```

Minimum-level and complete category-state operations are methods on the global logger:

```go
logger := logging.GetGlobalLogger()
logger.SetMinLevel(logging.DEBUG)
states := logger.GetCategoryStates()
logger.SetCategoryStates(states)
```

`IsCategoryEnabled` accounts for the global switch, but it does not test a proposed message level. Avoid expensive argument construction in hot paths; guard it with the available logger state when profiling shows a need.

## Structured History

Accepted records are copied into a bounded in-memory ring. `EntriesAfter` returns a cursor-based `EntryBatch`, including dropped-entry information when a reader falls behind. The Workbench log endpoint consumes this API.

The logger uses `sync.RWMutex` to protect configuration and retained entries. Dynamic effect or gameplay timing rules do not apply to diagnostic timestamps; log timestamps intentionally use wall time.

## Crash Reports

Runtime crash output is separate from categorized logging. `cmd/ultimav/crash_output.go` registers Go runtime crash output at:

```text
~/.ultima_v_redux/crash.log
```

Crash reporting remains active when categorized logging is disabled. On launch, an existing report at the configured size threshold rotates to `crash.log.old` while normal stderr crash behavior remains intact.

## Contributor Checks

Choose the narrow existing domain category before adding another. New categories
use the `Category<SystemName>` naming convention and must be declared
in `categories.go` and returned by `GetAllCategories()` so Logging Control can
discover and persist it.

## Testing

Tests that alter the global logger must restore every changed setting:

```go
logger := logging.GetGlobalLogger()
beforeEnabled := logger.IsEnabled()
beforeStdout := logger.IsStdoutEnabled()
beforeLevel := logger.GetMinLevel()
beforeCategories := logger.GetCategoryStates()
t.Cleanup(func() {
	logger.SetEnabled(beforeEnabled)
	logger.SetStdoutEnabled(beforeStdout)
	logger.SetMinLevel(beforeLevel)
	logger.SetCategoryStates(beforeCategories)
})
```

## Implementation References

- `internal/logging/logger.go`: filtering, configuration, and convenience functions.
- `internal/logging/entries.go`: bounded structured history and cursor reads.
- `internal/logging/log_level.go`: level definitions and parsing.
- `internal/logging/categories.go`: category registry.
- `cmd/ultimav/logging_configuration.go`: runtime application and persistence rollback.
- `cmd/ultimav/crash_output.go`: independent runtime crash reports.

Direct console output remains appropriate for command-line tools and startup failures where the console is the primary interface. Game diagnostics should use `internal/logging`.
