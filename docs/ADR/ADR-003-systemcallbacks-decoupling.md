# ADR-003: SystemCallbacks for UI Decoupling

**Status**: Accepted
**Date**: 2025-08-25

## Context

Gameplay actions need to emit messages, audio, visual feedback, conversation
requests, combat events, and presentation workflows. Direct calls from gameplay
packages into scene or UI code would couple authoritative rules to Ebitengine,
make tests require presentation setup, and allow presentation concerns to own
gameplay outcomes.

Returning presentation instructions from every action would move the coupling
into result types and make multi-stage effects difficult to compose. A global UI
service would hide dependencies and make tests less reliable.

## Decision

Use `SystemCallbacks` as the explicit dependency-injection boundary from
`GameState` to presentation and external systems.

- Gameplay code owns rules and authoritative mutation.
- Callback families group stable outward concerns such as messages, audio,
  visuals, flow, conversations, UI, and combat events.
- `GameScene` and application adapters supply concrete behavior.
- Tests install controlled callback implementations and assert meaningful
  interactions without initializing the full UI.
- Mandatory callback families are constructed as complete named values and
  validated during initialization.
- Missing mandatory callbacks are programming errors. After successful
  validation, callers use them directly rather than hiding broken wiring behind
  nil checks.
- A callback that loses its final production consumer is removed with its
  constructor requirement, validation, adapter wiring, mocks, and tests.

Gameplay time, resources, randomness, and other durable state do not move into
callbacks merely because their effects are presented externally.

The implementation in `internal/game_state/system_callbacks.go` is
authoritative for current callback families. Current scene wiring is documented
in [Game UI Integration](../ui/GAME_UI_INTEGRATION.md).

## Consequences

### Positive

- Core gameplay remains independent of Ebitengine and UI packages.
- Domain behavior can be tested with deterministic callback adapters.
- Presentation can evolve without moving gameplay authority into scenes.
- Initialization failures expose incomplete production wiring immediately.

### Trade-offs

- Callback evolution requires coordinated changes to contracts, constructors,
  adapters, mocks, and tests.
- Broad callback families need stable concern-based grouping to avoid
  unstructured growth.
- Indirection requires readers to trace from domain invocation to adapter
  implementation.

## Alternatives

### Direct UI Calls

Rejected because they violate package boundaries and require presentation setup
for gameplay tests.

### Event Bus For Every Effect

Rejected as the default because it adds subscription and lifecycle complexity
where an explicit callback is sufficient. Event queues remain appropriate for
owned multi-stage workflows and feedback sequencing.

### Presentation Instructions In Return Values

Rejected as a universal pattern because it couples gameplay APIs to presentation
protocols and complicates nested workflows. Typed domain results remain useful
where callers need gameplay outcomes.

### Global UI Service

Rejected because it hides dependencies and creates mutable global coupling.

## Current References

- `internal/game_state/system_callbacks.go`
- `internal/game_state/mock_callbacks_helpers.go`
- [Architecture Guidelines](../ARCHITECTURE_GUIDELINES.md)
- [Game UI Integration](../ui/GAME_UI_INTEGRATION.md)
- [Event Queue System](../systems/EVENT_QUEUE_SYSTEM.md)
