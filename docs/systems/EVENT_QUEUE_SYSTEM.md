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
  An unguarded workflow checks release on the update after `Finally()`. A workflow
  with an `InputReleaseReady` guard checks it in the same update, so a passing
  guard releases and removes the workflow immediately; a failing guard keeps the
  block until a later update passes it. Only the combat scheduler's enemy-turn
  and auto-pass workflows use the guard; they add their own scaled hand-off pause.
- `HandoffTo()` finalizes and removes the current workflow before starting its successor.

### Fixed-Name Entry Policy

Strict duplicate-name failure remains the manager invariant. Gameplay owners
must classify each reachable fixed-name entry rather than weakening it:

| Entry | Ownership |
|---|---|
| `skiff_deep_water` | Serialized: one high-priority owner drains every admitted rough-seas hit, including programmatic requests received while presentation is active. |
| `waterfall_hazard` | One logical `activeWaterfall` cascade, with at most one live physical workflow. The guard also covers actor/contact handoffs between workflow instances. |
| `automatic_sailing` | Locally guarded through finalization; each cadence boundary revalidates sailing authority. Stopping travel does not discard another owner's owed work. |
| `pirate_attack` | Serialized: retained pirate broadside and outdoor fireball results present in FIFO order; damage has already resolved in actor order. |
| `ship_repair_workflow`, `climbing_fall_effect`, `small_cannon_fire`, `ship_cannon_fire` | Locally guarded before feedback, RNG, movement, flags, or resource mutation. |
| Direct camping APIs | Reject while any gameplay workflow is active. |
| Stonegate death handoff | Presents an already committed town-hazard outcome; the ordinary wipe callback defers while this workflow owns the handoff. Retains any independently queued resurrection workflow. |

Use local guards only when a repeated trigger is redundant. If each invocation
owes gameplay, serialize the requests instead of dropping them.

### Rough-seas and travel ownership

`BlockInput` excludes ordinary player input, not programmatic movement or repeated
core calls. `ProcessSkiffDeepWaterDamage` therefore increments a pending-hit count
for each qualifying request before checking for the existing fixed-name owner.
That high-priority workflow preempts normal-priority travel and drains one damage
sequence per request. Hits are not coalesced as cosmetic feedback. Its finalizer
clears pending state, flash and shake without applying abandoned damage.

No production workflow awaits rough seas. One that must would compose the damage
sequence with `ThenSequence` and own `ClearSkiffDeepWaterEffects` in its own
`Finally`. Starting an independent child
and waiting for its name can deadlock when the child has equal or lower priority;
composition avoids that dependency even for critical-priority parents. This is
an ownership contract, not an expansion of the current successful-movement-only
hazard admission described in [SHIP-0003](../gamefacts/SHIP_MECHANICS.md#ship-0003).

Automatic sailing's state is separate from the lifetime of `automatic_sailing`.
An interruption stops future waits/movement immediately; the live workflow still
finalizes and releases its name. Successful ranged hits retain their ordered
`pirate_attack` presentation, and the already-started outdoor actor phase retains
its owed lower-slot work. Cleanup must not restart sailing, replay damage, or add
another actor phase. See [automatic sailing boundaries](TIME_SYSTEM.md#automatic-sailing-boundaries).

### Retained Waterfall Cascade

`GameState.activeWaterfall` owns the exact landing, visited trigger states and
initiating actor-tail completion independently of the `waterfall_hazard` queue.
`IsWaterfallHazardActive` includes that logical owner as well as pending, active
and finalizing physical workflow ownership. Repeated admission cannot recharge
the initiating command or create another cascade during an actor handoff.

The first complete descent performs its two southward steps, presentation,
damage and any canonical plane transition before running the initiating command's
owed actor tail **once**. The tail is marked done before invoking callbacks.
Further qualifying falls compose through `ThenSequence` and add no ordinary
command, player update or actor phase. An already admitted high-priority rough-sea
hit runs before the normal-priority waterfall workflow; the waterfall never
requests a duplicate hit.
Only the short interval between forced steps remains; there is no post-step
presentation-only dwell. The falls notice is nonblocking and may outlive the
cascade. Damage, transitions and owed actor processing retain input ownership
until safe completion, including physical key release.

If the owed phase produces projectile presentation, tactical contact or a
whirlpool, the physical waterfall workflow reaches its normal end and releases
presentation/input ownership. Its finalizer retains the logical cascade only for
that explicit actor handoff. The outdoor cursor binds to the same cascade and
preserves lower-slot work through successive contacts. It may accept a new landing
only from a validated owned whirlpool return, not unrelated relocation.

`ResumeWaterfallContinuation` runs after pending contacts, combat scheduling and
auto-talk, before ordinary input. It waits for the old physical name and relevant
interaction owners to clear, then reviews the current landing and starts another
workflow instance if needed. It does not wait for passive feedback work,
resolve contact itself, or replay the initiating tail. Owned tactical combat keeps
its ordinary input. Starting a workflow from `Finally`, or keeping the waterfall
block while waiting on an independent equal/lower-priority child, would break this
ownership contract. Strict duplicate-name failure remains unchanged.

Visited `(location, floor, position)` triggers terminate a wrapped cycle at its
committed landing with one nonmodal notice. Death, terminal takeover, invalid
ownership or shutdown discards remaining work while retaining committed steps and
damage. Each physical finalizer clears only its owned fade/flash; the manager's
shutdown action also clears a logical cascade between physical instances. Native
capture rejects that retained continuation even when no waterfall queue is live.
The raw south-tile admission and original standing-tile/visible-buffer comparison
limits remain explicit in [ENV-0003](../gamefacts/ENVIRONMENTAL_HAZARDS.md#env-0003).
See [waterfall timing](TIME_SYSTEM.md#waterfall-completion-and-resumption) for the
first-fall boundary and outdoor-window integration.

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
- Starting feedback inside a workflow, feedback, or finalizer callback is safe.
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

Inserted steps run on the parent's queue, so a `Finally` inside a sequence or `If`
block replaces the parent's cleanup when that step runs. Clear sequence-local state
with `Then` and leave cancellation cleanup to the parent's `Finally`. Camping is the
deliberate exception: its sequence takes over the slot and calls the caller's
completion from its own cleanup.

### Animation Admission And Time

An animation admitted by a delayed callback starts at the current queue update,
not the previous update. Its first `Execute` receives zero elapsed milliseconds
and presents progress zero. Already-running continuous events retain their own
full queue delta; admitting another animation must not reset that delta. The
same rule applies to workflows and feedback.

For example, a projectile launched after a 250 ms anticipation must not inherit
those 250 ms and reach its endpoint before it is drawn. Later updates advance
only the flight's elapsed time. Completion schedules the following sequence
steps for a later update, allowing the endpoint state to be drawn first; it does
not guarantee a display frame between updates when the engine is catching up.
Zero-duration animations emit progress one and complete immediately, including
on a zero-delta admission. Post-animation waits retain their own deadlines.

`internal/event_queue/animation_admission_test.go` covers both lanes,
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
`SetShutdownAction` callback for transient continuation state, including a logical
waterfall cascade between physical workflows and ship repair paused for deferred
combat. That owner cleanup does not belong to an already-finalized queue and
never resumes abandoned gameplay.

### Normal-Completion Continuations

`OnComplete(action)` registers a workflow's single normal-completion callback.
It runs exactly once after `Finally`, release of that workflow's input ownership
(including any key-release wait), and removal of its name. The callback may start
a new workflow with the same name; its start reserves the name immediately and
activation is deferred to the manager's safe callback boundary. Other workflows
may still own an input block.

Use `Finally` for cleanup and `OnComplete` for owed continuation after normal
completion. `Shutdown` runs cleanup but discards completion callbacks, including
when shutdown occurs inside `Finally` or while input release is pending.

Nil callbacks, feedback queues, duplicate registration, and registration
after finalization panic. Runtime sequence construction and `Finally` may register
the callback before finalization completes. `HandoffTo` uses the same callback
slot: if its event executes with an existing completion callback, it panics rather
than replacing it, regardless of declaration order. An untaken conditional handoff
does not claim the slot.

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
| `OnComplete(action)` | Continue after normal cleanup, input release, and name removal |
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
manager.Update()         // workflow lane
manager.UpdateFeedback() // serialized feedback lane
```

Workflow-triggered feedback can begin during the same frame without reentrant callback execution. A workflow awaiting feedback observes completion on a subsequent workflow update.

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
