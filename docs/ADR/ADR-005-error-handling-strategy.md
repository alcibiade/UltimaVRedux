# ADR-005: Error Handling Strategy

**Status**: Accepted
**Date**: 2025-08-25

## Context

The codebase previously used fatal termination for unrelated failure classes,
including recoverable user/configuration failures and internal contract bugs.
That made recovery unpredictable and obscured invalid initialization.

## Decision

Classify failures by recovery boundary:

1. Programming errors, impossible states, and violated mandatory contracts panic.
2. Required-resource corruption or unrecoverable startup failure terminates at
   the application boundary, where `log.Fatal` is appropriate.
3. User input, configuration, filesystem, network, and other recoverable
   operations return contextual errors.
4. Expected gameplay rejection returns a domain result and uses validated
   `SystemCallbacks` for player feedback.
5. Recoverable diagnostics use `internal/logging`; logging never replaces error
   propagation or invariant enforcement.

Mandatory callbacks and systems are validated during construction or
initialization. Callers then use them directly. Only capabilities explicitly
documented as optional may be nil-checked at the use site.

## Consequences

- Internal wiring defects fail close to their source.
- User and configuration problems can be reported without terminating the game.
- Fatal exits are limited to boundaries that cannot return or present an error.
- Tests can assert returned errors and contract panics separately.
- Existing ambiguous fatal calls must be classified when touched.

## Alternatives

- **Use returned errors everywhere:** rejected because internal invariant failures
  should not become routine branches that permit invalid state to continue.
- **Use fatal termination everywhere:** rejected because callers lose all ability
  to recover from user and environmental failures.
- **Log and continue:** rejected unless recovery is explicitly defined; otherwise
  it hides corruption and creates later failures.

## References

- [`../ERROR_HANDLING.md`](../ERROR_HANDLING.md)
- [`ADR-003-systemcallbacks-decoupling.md`](ADR-003-systemcallbacks-decoupling.md)
- `internal/game_state/system_callbacks.go`
- `internal/logging/`
