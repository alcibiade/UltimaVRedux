# ADR-008: Event Queue Execution Lanes

**Status**: Accepted

## Context

The event queue builder was used for both gameplay workflows and repeatable presentation effects. The original manager treated every queue as a unique serialized workflow and registered names during construction.

This caused two failure modes:

- Repeatable effects such as poison flashes attempted to register the same live name.
- Queue callbacks attempted to start independent queues while manager collections were being traversed.

Silently replacing duplicate queues corrupted ownership. Generating unique names created presentation backlogs and allowed effects to compete over global flash and shake state.

## Decision

The manager provides separate workflow and feedback lanes while retaining one fluent sequence builder.

### Workflow Lane

- Created with `Workflow(name)`.
- May mutate gameplay and own input through `BlockInput()`.
- Duplicate live names panic.
- Starts requested inside callbacks are deferred to a safe manager boundary.
- Priority preemption pauses lower-priority workflow timing.
- `Finally()` runs exactly once before ownership release.

### Feedback Lane

- Requested with `TriggerFeedback(name, priority, build)`.
- Presentation-only and never owns input.
- Same-name live feedback coalesces.
- Different feedback names serialize by priority with FIFO ties.
- Active feedback is not preempted because suspended effects may retain global visual state.
- Feedback requested inside callbacks starts safely after the callback.

### Composition

- `ThenSequence(sequence)` adds reusable steps owned by the parent workflow.
- `AwaitFeedback(name, priority, build)` triggers feedback and pauses the parent until it finalizes.
- `HandoffTo(name, priority, build)` finalizes and removes a workflow before starting its successor.

Queue names are reserved atomically at `Start()`, not during construction.

## Consequences

### Positive

- Builder readability remains intact.
- Nested independent starts no longer mutate manager collections reentrantly.
- Duplicate workflow names remain hard failures.
- Repeatable presentation does not panic or accumulate redundant effects.
- Feedback cannot accidentally release or acquire gameplay input ownership.
- Shared visual state is protected by serialized feedback execution.

### Negative

- Callers must distinguish authoritative workflow work from presentation-only feedback.
- Feedback cannot contain gameplay mutation.
- Starting an independent workflow from a callback does not make the parent wait; callers must use `ThenSequence` or `AwaitFeedback` when ordering is required.

## Alternatives Rejected

### Silently Replace Duplicate Queues

Rejected because active and registered ownership can diverge, causing incorrect cleanup and deletion.

### Unique Names For Every Effect

Rejected because repeatable effects build an unbounded backlog and compete over global presentation state.

### Expose Policies At Every Call Site

Rejected because callers would repeatedly configure the same conflict and nesting combinations. Workflow and feedback lanes encode the universal policies centrally.

### Remove Event Queues From Feedback

Rejected because deterministic animations, timing, exact cleanup, and builder readability remain valuable for presentation.
