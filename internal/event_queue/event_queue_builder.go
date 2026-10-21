// Package event_queue coordinates timed workflows and presentation feedback using injected clocks.
package event_queue

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
)

// Sequence adds reusable runtime steps to an event sequence.
type Sequence func(*SequentialEventBuilder) *SequentialEventBuilder

// SequentialEventBuilder builds sequential (chained) event sequences
// All events use chain mode - legacy pre-calculated timing removed
type SequentialEventBuilder struct {
	queue        *NamedEventQueue
	currentEvent *DelayedEvent // The event currently being built for chaining
	lastAction   func()        // Cleanup action that always runs at sequence end
	priority     int           // Execution priority (lower number = higher priority)
}

// newDelayedEvent creates an unchained event; builders attach continuations explicitly.
func newDelayedEvent(executeAt int64, action func() error) *DelayedEvent {
	return &DelayedEvent{
		ExecuteAt:   executeAt,
		Action:      action,
		NextAction:  nil,
		NextDelay:   0,
		IsWaitPoint: false,
	}
}

// First schedules the first event in a sequential chain
func (seb *SequentialEventBuilder) First(action func()) *SequentialEventBuilder {
	// Create the first event and track it for chaining
	seb.currentEvent = newDelayedEvent(seb.queue.timeSource(), func() error {
		action()

		return nil
	})
	seb.queue.events = append(seb.queue.events, seb.currentEvent)
	logging.Debugf(logging.CategoryGameState, "EventQueue: First event created")

	return seb
}

// Then executes an action immediately after the previous event completes (no delay)
// The action cannot fail - use RepeatWithDelay if you need error-based early termination
// If called on an empty queue (no First() yet), automatically becomes First()
func (seb *SequentialEventBuilder) Then(action func()) *SequentialEventBuilder {
	// Auto-First: If queue is empty, treat Then() as First()
	if seb.currentEvent == nil {
		logging.Debugf(logging.CategoryGameState, "EventQueue: Then() called on empty queue, auto-converting to First()")

		return seb.First(action)
	}

	// Create a placeholder event that will represent this Then action for further chaining
	thenEvent := newDelayedEvent(0, func() error {
		action()

		return nil
	})

	// Create a wrapper action that executes the Then action and schedules further chaining
	wrappedAction := func() error {
		// Execute the Then action
		action()

		// If there's a next action in the chain, schedule it dynamically
		if thenEvent.NextAction != nil {
			// Don't schedule next action if termination was requested
			if seb.queue.terminateRequested {
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: Then - skipping next action scheduling due to terminateRequested")
			} else {
				currentTime := seb.queue.timeSource()
				nextEvent := newDelayedEvent(currentTime+thenEvent.NextDelay, thenEvent.NextAction)
				nextEvent.IsWaitPoint = thenEvent.IsWaitPoint
				seb.queue.events = append(seb.queue.events, nextEvent)
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: Then action scheduled next at t=%d (current=%d + delay=%d)",
					nextEvent.ExecuteAt, currentTime, thenEvent.NextDelay)
			}
		} else if !seb.queue.terminateRequested {
			// No more actions in chain - preserve any terminal delay before Finally.
			// (Skip if terminateRequested - Phase 3.5 will handle lastAction directly)
			seb.scheduleLastAction(thenEvent.NextDelay)
		}

		return nil
	}

	// Set the wrapped action as the NextAction of the current event
	seb.currentEvent.NextAction = wrappedAction
	// Use the delay that was set by Wait(), or 0 if no Wait() was called
	delay := seb.currentEvent.NextDelay
	logging.Debugf(logging.CategoryGameState, "EventQueue: Chained Then action with %d ms delay", delay)

	// Update currentEvent to the placeholder for further chaining
	seb.currentEvent = thenEvent

	return seb
}

// ThenSequence inserts reusable steps when execution reaches this point.
func (seb *SequentialEventBuilder) ThenSequence(sequence Sequence) *SequentialEventBuilder {
	return seb.If(func() bool { return true }, func(builder *SequentialEventBuilder) {
		sequence(builder)
	})
}

// Wait pauses for the specified duration without executing any action
func (seb *SequentialEventBuilder) Wait(delayMs int64) *SequentialEventBuilder {
	// Validate delay parameter - negative delays are programming errors
	if delayMs < 0 {
		logging.Warnf(logging.CategoryGameState, "EventQueue: Wait() called with negative delay %d ms, using 0", delayMs)
		delayMs = 0
	}

	// Store the delay to be applied to the next Then() call
	if seb.currentEvent != nil {
		// ALWAYS use += to accumulate delays, regardless of NextAction state
		// This allows multiple Wait() calls to stack properly
		seb.currentEvent.NextDelay += delayMs
		logging.Debugf(logging.CategoryGameState,
			"EventQueue: Accumulated delay to %d ms (+%d)", seb.currentEvent.NextDelay, delayMs)
	}

	return seb
}

// WaitForAnyKey pauses the event sequence until any key is pressed
func (seb *SequentialEventBuilder) WaitForAnyKey() *SequentialEventBuilder {
	seb.rejectFeedbackOperation("WaitForAnyKey")
	if seb.currentEvent != nil {
		// Create a special wait event that will capture the next action when Then() is called
		waitEvent := newDelayedEvent(0, func() error {
			// This action will be replaced by the actual wait logic when Then() is called
			return nil
		})

		// Create the wait action that sets up input waiting
		waitAction := func() error {
			// Set up input waiting state - next action will be scheduled when key is pressed
			logging.Debugf(logging.CategoryGameState, "EventQueue: WaitForAnyKey triggered - waiting for input")
			seb.queue.waitingForInput = true
			seb.queue.keyWasPressed = false
			seb.queue.requireKeyRelease = true // Debounce: require keys to be released before accepting new press

			// Store the next action to be scheduled when key is pressed
			// This will be set when Then() is called after WaitForAnyKey()
			if waitEvent.NextAction != nil {
				seb.queue.pendingNextAction = waitEvent.NextAction
				seb.queue.pendingNextDelay = waitEvent.NextDelay
				logging.Debugf(logging.CategoryGameState, "EventQueue: Stored pending action for WaitForAnyKey")
			} else {
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: No pending action to store (no Then() called after WaitForAnyKey)")
			}

			return nil
		}

		// Set up the chaining by modifying the current event's NextAction
		seb.currentEvent.NextAction = waitAction
		seb.currentEvent.NextDelay = 0 // Execute immediately

		// Update currentEvent to the wait event for further chaining
		seb.currentEvent = waitEvent
		logging.Debugf(logging.CategoryGameState, "EventQueue: WaitForAnyKey chained")
	}

	return seb
}

// If conditionally executes a block of chained events at runtime.
// Condition is evaluated at EXECUTION time, not build time, allowing conditions
// that depend on state set by previous events in the chain.
//
// If condition() returns true when executed, the closure is called to build the sub-chain
// If condition() returns false, the entire block is skipped
//
// Example:
//
//	.If(func() bool { return wandererAppeared }, func(q *SequentialEventBuilder) {
//	    q.Then(func() error { /* wanderer logic */ }).
//	      WaitForAnyKey()
//	})
//
//nolint:funlen // Runtime sub-chain insertion is one cohesive linked-list operation.
func (seb *SequentialEventBuilder) If(
	condition func() bool,
	build func(*SequentialEventBuilder),
) *SequentialEventBuilder {
	// Create a special event that will do linked list insertion at runtime
	var whenEvent *DelayedEvent

	whenEvent = &DelayedEvent{
		Action: func() error {
			// Capture the original next action at EXECUTION time (it's set by the Then() call that comes AFTER If())
			originalNextAction := whenEvent.NextAction
			originalNextDelay := whenEvent.NextDelay

			if !condition() {
				logging.Debugf(logging.CategoryGameState, "EventQueue: If condition false at execution - skipping block")
				// Condition false - inject the originalNextAction to continue the chain
				if originalNextAction != nil {
					currentTime := seb.queue.timeSource()
					continueEvent := &DelayedEvent{
						ExecuteAt:   currentTime + originalNextDelay,
						Action:      originalNextAction,
						NextAction:  nil,
						NextDelay:   0,
						IsWaitPoint: false,
					}
					seb.queue.events = append(seb.queue.events, continueEvent)
				} else {
					// No more actions - preserve any terminal delay before Finally.
					seb.scheduleLastAction(originalNextDelay)
				}

				return nil
			}

			logging.Debugf(logging.CategoryGameState, "EventQueue: If condition true at execution - building sub-chain")

			// Build the sub-chain dynamically
			tempBuilder := &SequentialEventBuilder{
				queue:        seb.queue,
				currentEvent: nil,
				lastAction:   nil,
				priority:     0,
			}

			// Create a dummy first event to start the chain
			firstEvent := newDelayedEvent(0, func() error { return nil }) // No-op starter
			tempBuilder.currentEvent = firstEvent

			// Execute the build closure - this chains events via Then/Wait/etc
			build(tempBuilder)

			// Check if HandoffTo was called inside the build closure
			// If so, skip wiring the continuation - the queue will be terminated
			if seb.queue.terminateRequested {
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: If sub-chain called HandoffTo - skipping continuation wiring")
			} else if tempBuilder.currentEvent != nil {
				// Wire the end of the sub-chain to continue with the original next action
				tempBuilder.currentEvent.NextAction = originalNextAction
				// Use += to preserve any trailing Wait() delays from sub-chain
				tempBuilder.currentEvent.NextDelay += originalNextDelay
			}

			// INJECT the sub-chain: Add the first sub-chain event directly to the queue
			// The Update() loop will process it, and it will chain to the rest
			if firstEvent.NextAction != nil {
				currentTime := seb.queue.timeSource()
				firstSubEvent := &DelayedEvent{
					ExecuteAt:   currentTime,           // Execute immediately
					Action:      firstEvent.NextAction, // This is the wrapped action from Then()
					NextAction:  nil,                   // Will be handled by the wrapped action itself
					NextDelay:   0,
					IsWaitPoint: false,
				}

				seb.queue.events = append(seb.queue.events, firstSubEvent)
				logging.Debugf(logging.CategoryGameState, "EventQueue: Injected sub-chain first event into queue")
			}

			return nil
		},
		ExecuteAt:   0,
		NextAction:  nil,
		NextDelay:   0,
		IsWaitPoint: false,
	}

	// Chain this conditional event into the builder.
	if seb.currentEvent != nil {
		seb.currentEvent.NextAction = whenEvent.Action
		// Don't zero NextDelay - preserve any accumulated Wait() delay
	} else {
		// First event in the chain - add to queue directly
		currentTime := seb.queue.timeSource()
		seb.queue.events = append(seb.queue.events, newDelayedEvent(currentTime, whenEvent.Action))
	}

	// Update currentEvent to point to the conditional event for further chaining.
	// The next Then/Wait/etc call will set whenEvent.NextAction to the "after" action
	seb.currentEvent = whenEvent

	return seb
}

// whileLoopCore is the common infrastructure for all while-style loops.
// It handles:
// - Creating placeholder event for chaining
// - Priority yielding checks
// - Scheduling next iteration or completion
// - Setting up the chain
//
// The iterationLogic callback is called each iteration and returns:
// - shouldContinue: true to schedule another iteration, false to complete the loop
// - earlyExit: true if loop should exit immediately (e.g., no system callbacks)
//
// The loopName parameter is used for debug logging.
//
//nolint:funlen // Keep yielding, iteration, and late-bound completion together to preserve callback ordering.
func (seb *SequentialEventBuilder) whileLoopCore(
	iterationLogic func() (shouldContinue bool, earlyExit bool),
	intervalMs int64,
	loopName string,
) *SequentialEventBuilder {
	if seb.currentEvent == nil {
		return seb
	}

	// Validate interval parameter - negative intervals are programming errors
	if intervalMs < 0 {
		logging.Warnf(logging.CategoryGameState,
			"EventQueue: %s called with negative interval %d ms, using 0", loopName, intervalMs)
		intervalMs = 0
	}

	// Create a placeholder event that represents the end of the while loop for chaining
	whileEvent := newDelayedEvent(0, func() error {
		// This placeholder should not execute - the while completion handles chaining
		logging.Debugf(logging.CategoryGameState, "EventQueue: %s placeholder executed (unexpected)", loopName)

		return nil
	})

	// scheduleCompletion handles scheduling the next action or Finally() cleanup when loop exits
	scheduleCompletion := func() {
		if whileEvent.NextAction != nil {
			nextEvent := newDelayedEvent(seb.queue.timeSource()+whileEvent.NextDelay, whileEvent.NextAction)
			seb.queue.events = append(seb.queue.events, nextEvent)
			logging.Debugf(logging.CategoryGameState, "EventQueue: %s loop completed, scheduled next action", loopName)
		} else {
			// No more actions - schedule Last cleanup action if present
			seb.scheduleLastAction(whileEvent.NextDelay)
		}
	}

	// Create a recursive while action that uses chain mode chaining
	var createWhileChain func() func() error
	createWhileChain = func() func() error {
		return func() error {
			// Check if higher priority queue wants to interrupt (cooperative yielding)
			if seb.queue.lane == queueLaneWorkflow && seb.queue.manager != nil &&
				seb.queue.manager.ShouldYield(seb.priority) {
				logging.Debugf(logging.CategoryGameState, "EventQueue: %s yielding to higher priority queue", loopName)
				// Schedule another iteration of the loop to resume when promoted
				nextIteration := newDelayedEvent(seb.queue.timeSource()+intervalMs, createWhileChain())
				seb.queue.events = append(seb.queue.events, nextIteration)

				return nil
			}

			// Execute the iteration logic
			shouldContinue, earlyExit := iterationLogic()

			// Handle early exit (e.g., missing system callbacks)
			if earlyExit {
				scheduleCompletion()

				return nil
			}

			// Schedule next iteration or complete the loop
			if shouldContinue {
				nextIteration := newDelayedEvent(seb.queue.timeSource()+intervalMs, createWhileChain())
				seb.queue.events = append(seb.queue.events, nextIteration)
			} else {
				scheduleCompletion()
			}

			return nil
		}
	}

	// Set up the chaining by modifying the current event's NextAction
	seb.currentEvent.NextAction = createWhileChain()
	seb.currentEvent.NextDelay = 0 // Execute immediately

	// Update currentEvent to the placeholder for further chaining
	seb.currentEvent = whileEvent
	logging.Debugf(logging.CategoryGameState, "EventQueue: %s loop chained", loopName)

	return seb
}

// While repeats an action at regular intervals as long as condition returns true.
// On each iteration:
//  1. Checks condition() - if false, exits loop and continues to next event
//  2. Executes action() - called EVERY iteration while condition is true
//  3. Waits intervalMs milliseconds before next iteration
//
// Parameters:
//   - condition: Checked before each iteration. Return false to exit loop.
//   - action: Executed every iteration. Use func() {} for polling without side effects.
//   - intervalMs: Delay between iterations in milliseconds (e.g., 16ms = ~60fps polling rate)
//
// Example - Wait for dialog dismissal:
//
//	While(
//	    func() bool { return !dialog.IsDismissed() }, // Poll every frame
//	    func() {},                                 // No action needed, just polling
//	    16,                                        // Check ~60 times per second
//	)
func (seb *SequentialEventBuilder) While(
	condition func() bool,
	action func(),
	intervalMs int64,
) *SequentialEventBuilder {
	return seb.whileLoopCore(
		func() (bool, bool) {
			// Check condition first
			if !condition() {
				return false, false // Loop complete
			}

			// Execute the action
			action()

			// Check condition again after action
			return condition(), false
		},
		intervalMs,
		"While",
	)
}

// RunScript executes an animation driver until completion.
// The driver is updated every frame (~16ms) until IsComplete() returns true
//
// This method allows script executors to integrate with the event queue without
// requiring callback-based completion handling. The script runs inline with
// normal event queue flow.
//
// Parameters:
//   - driver: Any AnimationDriver implementation (e.g., minimap.ScriptExecutor)
//
// Example:
//
//	executor := minimap.NewScriptExecutor(scene, walkScript)
//	q.First(func() { /* setup */ }).
//	  RunScript(executor).
//	  Then(func() { /* runs after script completes */ })
func (seb *SequentialEventBuilder) RunScript(driver AnimationDriver) *SequentialEventBuilder {
	const frameIntervalMs = 16 // ~60fps update rate

	return seb.While(
		func() bool { return !driver.IsComplete() }, // Continue while not complete
		func() { driver.Update(frameIntervalMs) },   // Update each frame
		frameIntervalMs,
	)
}

// WhileAwaitingInput runs the action repeatedly, checking for key input after each iteration
// It provides the pressed key to the callback for decision-making.
//
// Parameters:
//   - action: Called every iteration before key check (use func() {} for polling without side effects)
//   - keyCallback: Called with the currently pressed key (or nil if no key pressed)
//     Returns true to stop loop, false to continue
//   - intervalMs: Delay between iterations (e.g., 100ms for smooth movement)
//
// The callback receives *semantic_input.Key which can be:
//   - nil: No key currently pressed
//   - Arrow key: Same direction (continue sailing) or different direction (change sailing direction)
//   - Space bar: Stop sailing with "Sheets in irons!" message
//   - Any other key: Stop loop AND process key as normal command (not consumed)
//
// Example - Automatic sailing with direction changes:
//
//	.WhileAwaitingInput(
//	    func() { moveSailingStep() },  // Called every interval
//	    func(key *semantic_input.Key) bool {
//	        if key == nil { return false }  // No key - continue
//	        if *key == semantic_input.KeySpace { return true }  // Space - stop
//	        if isArrowKey(key) {
//	            changeSailingDirection(key)  // Arrow - change direction
//	            return false  // Continue sailing
//	        }
//	        return true  // Other key - stop and process
//	    },
//	    100,  // 100ms between moves
//	)
func (seb *SequentialEventBuilder) WhileAwaitingInput(
	action func(),
	keyCallback func(key *semantic_input.Key) bool,
	intervalMs int64,
) *SequentialEventBuilder {
	seb.rejectFeedbackOperation("WhileAwaitingInput")

	return seb.whileLoopCore(
		func() (bool, bool) {
			// Execute the action first
			action()

			// Get the currently pressed key
			var pressedKey *semantic_input.Key
			if seb.queue.manager != nil && seb.queue.manager.systemCallbacks != nil {
				pressedKey = seb.queue.manager.systemCallbacks.GetPressedKey()
			} else {
				// Safety: If no system callbacks available, stop loop
				logging.Debugf(logging.CategoryGameState, "EventQueue: WhileAwaitingInput - no system callbacks, stopping")

				return false, true // earlyExit = true
			}

			// Call the key callback to determine whether to continue or stop
			shouldStop := keyCallback(pressedKey)

			return !shouldStop, false
		},
		intervalMs,
		"WhileAwaitingInput",
	)
}

// Animate adds a smooth animation event that calls onFrame each frame with progress 0.0-1.0
func (seb *SequentialEventBuilder) Animate(durationMs uint64, onFrame func(float32)) *SequentialEventBuilder {
	if seb.currentEvent != nil {
		// Create a placeholder event that will represent this animation for further chaining
		animateEvent := newDelayedEvent(0, func() error {
			// This placeholder action should not execute - the animation completion handles chaining
			logging.Debugf(logging.CategoryGameState, "EventQueue: Animate placeholder event executed (unexpected)")

			return nil
		})

		// Create the actual animation action that will be executed
		animateAction := func() error {
			// Create animation event and add it to continuous events
			animateEventInternal := NewAnimateEventWithCallback(durationMs, onFrame, func() {
				// Animation completed - schedule next action in chain if there is one
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: Animate completed (duration=%dms), checking for next action", durationMs)
				if animateEvent.NextAction != nil {
					nextEvent := newDelayedEvent(seb.queue.timeSource()+animateEvent.NextDelay, animateEvent.NextAction)
					seb.queue.events = append(seb.queue.events, nextEvent)
					logging.Debugf(logging.CategoryGameState,
						"EventQueue: Animate scheduled next action with delay=%d", animateEvent.NextDelay)
				} else {
					// No more actions in chain - schedule Last cleanup action if present
					// This matches Then() behavior for terminal events
					logging.Debugf(logging.CategoryGameState, "EventQueue: Animate is terminal event, scheduling Last cleanup")
					seb.scheduleLastAction(animateEvent.NextDelay)
				}
			})
			seb.queue.continuousEvents = append(seb.queue.continuousEvents, animateEventInternal)

			return nil
		}

		// Set up the chaining by modifying the current event's NextAction
		seb.currentEvent.NextAction = animateAction
		seb.currentEvent.NextDelay = 0

		// Update currentEvent to the placeholder for potential further chaining
		seb.currentEvent = animateEvent
		logging.Debugf(logging.CategoryGameState, "EventQueue: Animate event chained")
	}

	return seb
}

// RepeatWithDelay executes a function N times with a delay between each iteration
// The function receives the current iteration (0-based) and returns true to continue, false to break early
// Total duration will be approximately: iterations * delayMs
func (seb *SequentialEventBuilder) RepeatWithDelay(
	iterations int, delayMs int64, action func(iteration int) bool,
) *SequentialEventBuilder {
	return seb.RepeatWithRuntimeDelay(iterations, func() int64 { return delayMs }, action)
}

// RepeatWithRuntimeDelay reads delay at admission and after each continuing
// iteration. Existing deadlines are never rewritten; a hitch schedules the next
// interval from execution time rather than catching up missed iterations.
// Providers must be side-effect free. Zero delay retains the fixed API's behavior.
//
//nolint:funlen // Keep runtime delay reads and late-bound continuation in the same recursive scheduling closure.
func (seb *SequentialEventBuilder) RepeatWithRuntimeDelay(
	iterations int, delay func() int64, action func(iteration int) bool,
) *SequentialEventBuilder {
	if iterations <= 0 {
		return seb // No iterations to process
	}
	if delay == nil {
		panic("RepeatWithRuntimeDelay: delay provider is required")
	}
	if seb.currentEvent == nil {
		return seb
	}

	// Create recursive chain where each iteration schedules the next
	// This placeholder is used only for chaining setup, not for execution.
	finalEvent := newDelayedEvent(0, func() error {
		// This should never execute - the last iteration handles chaining directly
		logging.Debugf(logging.CategoryGameState, "EventQueue: RepeatWithDelay finalEvent executed (unexpected)")

		return nil
	})

	// Create a recursive closure that schedules iterations
	var createIterationChain func(currentIteration int) func() error
	createIterationChain = func(currentIteration int) func() error {
		return func() error {
			// Execute this iteration
			shouldContinue := action(currentIteration)
			if !shouldContinue {
				// Break loop early - schedule the final event and stop iterations
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: RepeatWithDelay breaking early at iteration %d", currentIteration)
				if finalEvent.NextAction != nil {
					nextEvent := newDelayedEvent(seb.queue.timeSource()+finalEvent.NextDelay, finalEvent.NextAction)
					nextEvent.IsWaitPoint = finalEvent.IsWaitPoint
					seb.queue.events = append(seb.queue.events, nextEvent)
					logging.Debugf(logging.CategoryGameState,
						"EventQueue: Early break scheduled next action for %d", nextEvent.ExecuteAt)
				} else {
					// No more actions - schedule Last cleanup action if present
					seb.scheduleLastAction(finalEvent.NextDelay)
				}

				return nil // Continue chain after early break
			}

			// If there are more iterations, schedule the next one
			switch {
			case currentIteration < iterations-1:
				nextIteration := createIterationChain(currentIteration + 1)

				// Dynamically create and schedule the next iteration event
				nextEvent := newDelayedEvent(seb.queue.timeSource()+max(0, delay()), nextIteration)
				seb.queue.events = append(seb.queue.events, nextEvent)
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: Scheduled iteration %d for execution at %d", currentIteration+1, nextEvent.ExecuteAt)
			case finalEvent.NextAction != nil:
				// This is the last iteration - directly schedule the next action if present
				nextEvent := newDelayedEvent(seb.queue.timeSource()+finalEvent.NextDelay, finalEvent.NextAction)
				nextEvent.IsWaitPoint = finalEvent.IsWaitPoint
				seb.queue.events = append(seb.queue.events, nextEvent)
				logging.Debugf(logging.CategoryGameState,
					"EventQueue: Last iteration directly scheduled next action for %d", nextEvent.ExecuteAt)
			default:
				// No more actions - schedule Last cleanup action if present
				seb.scheduleLastAction(finalEvent.NextDelay)
			}

			return nil
		}
	}

	// Set the first iteration as the NextAction of the current event
	firstIteration := createIterationChain(0)
	seb.currentEvent.NextAction = func() error {
		seb.queue.events = append(seb.queue.events,
			newDelayedEvent(seb.queue.timeSource()+max(0, delay()), firstIteration))

		return nil
	}
	seb.currentEvent.NextDelay = 0

	// Set the final event as the new current event for further chaining
	seb.currentEvent = finalEvent
	logging.Debugf(logging.CategoryGameState,
		"EventQueue: RepeatWithDelay chained %d iterations in chain mode", iterations)

	return seb
}

// Finally registers guaranteed cleanup that runs exactly once before ownership release.
// This action is guaranteed to run at the end of the sequence
// Use for cleanup: hiding dialogs, stopping sounds, clearing state
//
// Example:
//
//	g.EventQueueManager.Workflow("camping").
//	    First(func() { dialog.Show() }).
//	    Then(func() { dialog.AppendLine("Healing...") }).
//	    WaitForAnyKey().
//	    Finally(func() {
//	        dialog.Hide()           // Cleanup
//	        audio.Stop(sound).Now() // Cleanup
//	    }).
//	    Start()
func (seb *SequentialEventBuilder) Finally(action func()) *SequentialEventBuilder {
	seb.lastAction = action
	seb.queue.lastAction = action // Also store in queue for Update() access
	logging.Debugf(logging.CategoryGameState, "EventQueue: Finally cleanup action registered")

	return seb
}

// HandoffTo marks the current queue for termination and starts another after cleanup.
// The current queue's Finally() will run for cleanup, then the chained queue starts via OnComplete
// Safe to call from inside If()/While() callbacks - termination is deferred until event completes
//
// Use this for internal self-termination with chaining (e.g., sailing → whirlpool transportation)
// This works for internal conditions where a running queue needs to stop itself and chain to another
//
// Example:
//
//	If(func() bool { return landedOnWhirlpool }, func(q *SequentialEventBuilder) {
//	    q.Then(func() { stopSailing() }).
//	      HandoffTo("whirlpool_transport", PriorityHigh, func(wq *SequentialEventBuilder) {
//	          wq.First(func() { showMessage() }).
//	             Then(func() { teleport() })
//	      })
//	})
func (seb *SequentialEventBuilder) HandoffTo(
	newQueueName string,
	newPriority int,
	builderFunc func(*SequentialEventBuilder),
) *SequentialEventBuilder {
	if seb.queue.lane != queueLaneWorkflow {
		panic("EventQueue: HandoffTo is only valid for workflows")
	}
	// Capture references for the closure
	manager := seb.queue.manager
	queue := seb.queue

	logging.Debugf(logging.CategoryGameState,
		"EventQueue: HandoffTo builder called - will terminate queue '%s' and chain to '%s' at execution time",
		seb.queue.name, newQueueName)

	// Add an action that sets the termination flag at EXECUTION time (not build time)
	// This ensures the flag is only set when the previous action in the chain completes
	return seb.Then(func() {
		if queue.onComplete != nil {
			panic("HandoffTo cannot replace an existing completion callback")
		}
		logging.Debugf(logging.CategoryGameState,
			"EventQueue: HandoffTo executing - marking queue '%s' for termination, will chain to '%s'",
			queue.name, newQueueName)

		// Set flag to terminate this queue after this event completes
		queue.terminateRequested = true

		// Register the chain to run after termination via OnComplete
		// This fires when the queue is deleted (after Finally() cleanup runs)
		queue.onComplete = func() {
			logging.Debugf(logging.CategoryGameState,
				"EventQueue: HandoffTo OnComplete - creating new queue '%s' (priority %d)",
				newQueueName, newPriority)

			// Create new queue with specified priority
			newQueue := manager.Workflow(newQueueName).
				Priority(newPriority).
				Sequential()

			// Let the builder function configure the new queue
			builderFunc(newQueue)

			// Start the new queue (safe at depth=0)
			newQueue.Start()
		}
	})
}

// AwaitFeedback triggers serialized feedback and pauses this sequence until it completes.
func (seb *SequentialEventBuilder) AwaitFeedback(
	name string,
	priority int,
	build func(*SequentialEventBuilder),
) *SequentialEventBuilder {
	if seb.queue.lane != queueLaneWorkflow {
		panic("EventQueue: AwaitFeedback is only valid for workflows")
	}

	return seb.Then(func() {
		seb.queue.manager.TriggerFeedback(name, priority, build)
	}).whileLoopCore(
		func() (bool, bool) {
			return seb.queue.manager.HasFeedback(name), false
		},
		1,
		"AwaitFeedback",
	)
}

// scheduleLastAction preserves a terminal delay while leaving Finally execution to queue finalization.
func (seb *SequentialEventBuilder) scheduleLastAction(delay int64) {
	if seb.queue.lastAction != nil && delay > 0 {
		seb.queue.events = append(seb.queue.events,
			newDelayedEvent(seb.queue.timeSource()+delay, func() error { return nil }))
	}
}

// Start executes the sequential event sequence
func (seb *SequentialEventBuilder) Start() {
	if seb.queue.lifecycle != queueCreated {
		panic("EventQueue.Start() called twice for queue '" + seb.queue.name + "'")
	}

	// Safety check: Ensure Priority() was called before Start()
	if seb.priority == 0 {
		logging.Errorf(logging.CategoryGameState,
			"EventQueue.Start() called without Priority() - queue '%s' must have priority set", seb.queue.name)
		panic("EventQueue.Start() called without Priority() - you must call Priority() before Start()")
	}

	// Safety check: Ensure First() was called before Start()
	if len(seb.queue.events) == 0 {
		logging.Errorf(logging.CategoryGameState,
			"EventQueue.Start() called without First() - queue '%s' has no events", seb.queue.name)
		panic("EventQueue.Start() called without First() - you must call First() before Then() or other operations")
	}

	seb.queue.manager.startQueue(seb.queue, seb.priority)
}

// BlockInput gives this workflow ownership of the global input block until finalization.
func (seb *SequentialEventBuilder) BlockInput() *SequentialEventBuilder {
	seb.rejectFeedbackOperation("BlockInput")
	seb.queue.autoBlockUnblockInput = true

	return seb
}

func (seb *SequentialEventBuilder) rejectFeedbackOperation(operation string) {
	if seb.queue.lane == queueLaneFeedback {
		panic("EventQueue: feedback cannot use " + operation)
	}
}
