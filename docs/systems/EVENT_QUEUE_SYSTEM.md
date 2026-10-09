# Event Queue System

**Status**: Implemented and in production use.

The event queue manager provides deterministic, single-threaded orchestration for gameplay workflows and presentation feedback. Both use the same fluent sequence builder but execute in separate lanes with different ownership rules. The lane split and its rationale are recorded in [ADR-008](../ADR/ADR-008-event-queue-execution-lanes.md).

## Decision Guide

| Requirement | API |
|---|---|
| Independent gameplay orchestration or input-owning UI workflow | `Workflow(name)` |
| Presentation-only effect; caller continues | `TriggerFeedback(name, priority, build)` |
| Presentation-only effect; workflow waits | `AwaitFeedback(name, priority, build)` |
| Reusable steps owned by the current workflow | `ThenSequence(sequence)` |

## Workflow Lane

A workflow represents gameplay orchestration where every invocation matters.

```go
manager.Workflow("camping_workflow").
    Priority(event_queue.PriorityNormal).
    Sequential().
    BlockInput().
    First(setupCamping).
    RepeatWithDelay(cycles, cycleDelayMs, processCycle).
    WaitForAnyKey().
    Finally(cleanupCamping).
    Start()
```

### Guarantees

- Construction does not register the name.
- `Start()` validates the sequence and atomically reserves its name.
- Duplicate live workflow names panic.
- Starts requested inside callbacks are deferred until the manager reaches a safe boundary.
- A higher-priority workflow may preempt the active workflow.
- Preempted waits do not age while suspended.
- Equal-priority pending workflows retain FIFO order.
- `Finally()` executes exactly once before input ownership is released.
- Natural completion waits for physical key release before releasing an owned input block.
- `HandoffTo()` finalizes and removes the current workflow before starting its successor.

### Fixed-Name Entry Policy

Strict duplicate-name failure remains the manager invariant. Gameplay owners
must classify each reachable fixed-name entry rather than weakening it:

| Entry | Ownership |
|---|---|
| `skiff_deep_water`, `waterfall_hazard` | Inherently single-flight: successful movement invokes each once and the workflow immediately blocks further movement input. |
| `pirate_attack` | Serialized: every owed broadside is retained in FIFO order. |
| `ship_repair_workflow`, `climbing_fall_effect`, `small_cannon_fire`, `ship_cannon_fire` | Locally guarded before feedback, RNG, movement, flags, or resource mutation. |
| Direct camping APIs | Reject while any gameplay workflow is active. |
| Stonegate death handoff | Presents an already committed town-hazard outcome; the ordinary wipe callback defers while this workflow owns the handoff. Retains any independently queued resurrection workflow. |

Use local guards only when a repeated trigger is redundant. If each invocation
owes gameplay, serialize the requests instead of dropping them.

## Feedback Lane

Feedback is presentation-only work such as a flash, shake, sound, or incidental message.

```go
applyPoisonDamage()
manager.TriggerFeedback("poison", event_queue.PriorityNormal,
    func(sequence *event_queue.SequentialEventBuilder) {
        sequence.First(showPoisonFlash).
            Animate(200, updatePoisonFlash).
            Finally(clearPoisonFlash)
    },
)
```

### Guarantees

- Feedback never owns or blocks input.
- Starting feedback inside a workflow, feedback, finalizer, or background callback is safe.
- Repeated live feedback with the same name coalesces; the first accepted instance wins.
- Coalescing includes deferred, active, pending, and finalizing feedback.
- Different feedback names execute one at a time.
- Priority chooses the next pending feedback; ties are FIFO.
- Active feedback is not preempted because partially suspended visual state could conflict with another effect.
- `Finally()` completes before the feedback name becomes reusable.

### Feedback Restrictions

Feedback may:

- Set or animate visual callbacks.
- Play or stop presentation audio.
- Emit incidental output text.

Feedback must not:

- Apply damage or healing.
- Change inventory, quest, map, or combat state.
- Advance turns or game time.
- Open input-owning dialogs.
- Call `BlockInput()`.

Perform authoritative mutation before requesting feedback. Coalescing can then never discard gameplay.

## Awaited Feedback

`AwaitFeedback` starts feedback on the feedback lane and pauses the parent workflow until that feedback finalizes. The parent workflow remains responsible for input ownership.

```go
manager.Workflow("poison_attack").
    Priority(event_queue.PriorityHigh).
    Sequential().
    BlockInput().
    First(performAttack).
    Then(applyPoison).
    AwaitFeedback("poison", event_queue.PriorityNormal,
        func(sequence *event_queue.SequentialEventBuilder) {
            sequence.First(showPoisonFlash).
                Animate(200, updatePoisonFlash).
                Finally(clearPoisonFlash)
        },
    ).
    Then(finishAttack).
    Finally(cleanupAttack).
    Start()
```

Calling `TriggerFeedback` from a workflow callback is also safe, but the parent continues without waiting.

## Reusable Sequences

`ThenSequence` inserts reusable steps when execution reaches that point. Helpers may use values established by earlier actions.

```go
func showConfirmation(message string) event_queue.Sequence {
    return func(sequence *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
        var dialog *widgets.OverlayDialog
        return sequence.Then(func() {
            dialog = buildConfirmation(message)
            dialogStack.PushModalDialog(dialog)
        }).
            While(func() bool { return !dialog.IsDismissed() }, func() {}, 16).
            Then(func() { keyboard.ResetReleaseDebouncing() })
    }
}

workflow.ThenSequence(showConfirmation("Continue?"))
```

Use `ThenSequence` when the parent must own and wait for the inserted steps. Starting another independent workflow does not make the parent wait.

### Animation Admission And Time

An animation admitted by a delayed callback starts at the current queue update,
not the previous update. Its first `Execute` receives zero elapsed milliseconds
and presents progress zero. Already-running continuous events retain their own
full queue delta; admitting another animation must not reset that delta. The
same rule applies to workflows, feedback, and legacy background queues.

For example, a projectile launched after a 250 ms anticipation must not inherit
those 250 ms and reach its endpoint before it is drawn. Later updates advance
only the flight's elapsed time. Completion schedules the following sequence
steps for a later update, allowing the endpoint state to be drawn first; it does
not guarantee a display frame between updates when the engine is catching up.
Zero-duration animations emit progress one and complete immediately, including
on a zero-delta admission. Post-animation waits retain their own deadlines.

`internal/event_queue/animation_admission_test.go` covers all three lanes,
coexisting continuous events, zero-duration completion, and post-flight waits.

## Conditional Sequences

`If` evaluates its condition once at runtime and conditionally inserts a sequence.

```go
workflow.If(
    func() bool { return result == Success },
    func(sequence *event_queue.SequentialEventBuilder) {
        sequence.Then(showSuccess)
    },
)
```

Use `While` when the condition must be evaluated repeatedly.

## Cleanup And Handoff

`Finally` is a replaceable exact-once cleanup slot, not an ordinary final action.

```go
workflow.Finally(func() {
    stopLoopingAudio()
    closeOwnedDialog()
})
```

`HandoffTo` suppresses remaining work, runs `Finally`, releases ownership, removes the old name, then safely starts the successor.

```go
workflow.HandoffTo(
    "whirlpool_transportation",
    event_queue.PriorityHigh,
    buildWhirlpoolTransportation,
)
```

`Shutdown()` permanently cancels every lane, runs accepted `Finally` callbacks
at most once, and releases input ownership. `GameState` registers one
`SetShutdownAction` callback for transient continuation state, such as a ship
repair paused between its workflow and deferred combat; that owner cleanup does
not belong to an already-finalized queue.

## Fluent API

| Method | Purpose |
|---|---|
| `Priority(level)` | Required execution priority |
| `Sequential()` | Begin sequence construction |
| `BlockInput()` | Give a workflow input ownership |
| `First(action)` | Required initial action |
| `Then(action)` | Add a simple action |
| `ThenSequence(sequence)` | Add reusable runtime-built steps |
| `If(condition, build)` | Conditionally add runtime steps |
| `Wait(ms)` | Delay the next step |
| `WaitForAnyKey()` | Wait for a fresh key press |
| `While(condition, action, ms)` | Repeat while a condition is true |
| `WhileAwaitingInput(...)` | Repeat with key-aware input handling |
| `RepeatWithDelay(count, ms, action)` | Run fixed delayed iterations |
| `Animate(duration, onFrame)` | Run a progress animation |
| `RunScript(driver)` | Update an animation driver until complete |
| `AwaitFeedback(name, priority, build)` | Trigger feedback and wait for finalization |
| `HandoffTo(name, priority, build)` | Transfer to another workflow after cleanup |
| `Finally(action)` | Register exact-once cleanup |
| `Start()` | Validate, reserve, and schedule |

`IntroMenuScene` uses a scene-owned manager and `RunScript` for the startup
Origin/Lord British presentation. The driver owns only presentation progress;
normal menu input begins after workflow `Finally` exposes the menu. Enter marks
the driver complete rather than cancelling the queue, so exact-once cleanup is
preserved without a public cancellation API.

## Priority Levels

| Priority | Value | Typical use |
|---|---:|---|
| `PriorityCritical` | 1 | Fatal or system-critical transitions |
| `PriorityHigh` | 2 | Combat, hazards, moongates, collisions |
| `PriorityMedium` | 3 | Commands and dialogs |
| `PriorityNormal` | 5 | Camping, sailing, routine gameplay |
| `PriorityLow` | 10 | Non-urgent presentation |

## Lifecycle

Queues move through these internal states:

```text
created -> deferred -> started -> finalizing -> finalized
    |
    +-> coalesced  (duplicate feedback only)
```

- `created`: configurable and not registered.
- `deferred`: name reserved; activation waits for a safe callback boundary.
- `started`: active or pending in its lane.
- `finalizing`: `Finally` and ownership cleanup are in progress.
- `finalized`: removed and complete.
- `coalesced`: duplicate feedback discarded without executing callbacks.

Workflow names remain live through finalization and input release. Feedback names remain live through `Finally`.

## Safe Nested Starts

Queue callbacks run with tracked execution depth. A workflow or feedback requested from a callback reserves its name immediately but is dispatched only after the current lane update reaches a safe boundary. This avoids mutating active and pending collections while they are being traversed.

Safe does not mean awaited:

- Start an independent `Workflow` when the parent may continue.
- Use `ThenSequence` when steps belong to the parent.
- Use `AwaitFeedback` when the parent must wait for presentation.

## Input Ownership

Only workflows may call `BlockInput()`.

- Ownership is reference counted across preemption.
- Pausing a workflow does not release its ownership.
- `Finally` runs before release.
- Natural completion waits for all physical keys to be released.
- Feedback never affects the input ownership count.

## Update Order

`GameScene` updates lanes every frame:

```go
manager.Update()           // workflow lane
manager.UpdateBackground() // legacy concurrent background lane
manager.UpdateFeedback()   // serialized feedback lane
```

Workflow-triggered feedback can begin during the same frame without reentrant callback execution. A workflow awaiting feedback observes completion on a subsequent workflow update.

## Legacy Background Queues

`RegisterBackgroundQueue` is a retained legacy compatibility API. `GameScene`
still updates the background lane, but current production code registers no
background queues. New visual/audio presentation should use `TriggerFeedback`
because feedback provides name coalescing, priority serialization,
callback-safe starts, and protection against effects competing over global
flash or shake state.

## Testing Requirements

Event queue changes require focused regression coverage for every affected
lane, ordering, ownership, coalescing, and cleanup guarantee. Cover normal
completion, cancellation, rejection, and interrupted handoffs. Assert that
finalizers run once, release input and presentation resources, and never perform
an owed gameplay continuation. Feedback tests must establish that no gameplay
state, time, or RNG is mutated. Use an injected elapsed clock instead of sleeps;
verify callback-triggered work cannot execute reentrantly in the same callback.

## Source Files

- `internal/event_queue/event_queue_types.go`
- `internal/event_queue/event_queue_manager.go`
- `internal/event_queue/event_queue_builder.go`
- `internal/event_queue/queue_lanes_test.go`
- `internal/event_queue/finalization_test.go`
- `cmd/ultimav/gamescene_update.go`
