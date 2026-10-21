package event_queue

import "github.com/bradhannah/UltimaVRedux/internal/logging"

// UntilKeyPressed runs the action repeatedly until any key is pressed
// This is a convenience wrapper around While() for the common pattern of "run until user input"
//
// Parameters:
//   - action: Called every iteration (use func() {} for polling without side effects)
//   - intervalMs: Delay between iterations (e.g., 100ms for smooth movement)
//
// The loop stops when SystemCallbacks.IsAnyKeyPressed() returns true
// The pressed key is NOT consumed - it will be processed by normal input handlers after the loop exits
//
// Example - Automatic sailing:
//
//	.UntilKeyPressed(
//	    func() { moveSailingStep() },  // Called every interval
//	    100,                             // 100ms between moves
//	)
func (seb *SequentialEventBuilder) UntilKeyPressed(
	action func(),
	intervalMs int64,
) *SequentialEventBuilder {
	seb.rejectFeedbackOperation("UntilKeyPressed")
	// Wrap While() with keyboard polling condition
	return seb.While(
		// Condition: continue while NO key pressed
		func() bool {
			// Safety: If no system callbacks available, stop loop
			if seb.queue.manager == nil || seb.queue.manager.systemCallbacks == nil {
				logging.Debug(logging.CategoryGameState, "EventQueue: UntilKeyPressed - no system callbacks, stopping")

				return false
			}

			// Continue loop while no key is pressed
			keyPressed := seb.queue.manager.systemCallbacks.IsAnyKeyPressed()
			if keyPressed {
				logging.Debug(logging.CategoryGameState, "EventQueue: UntilKeyPressed - key detected, stopping loop")
			}

			return !keyPressed
		},
		// Action: execute user-provided action
		action,
		// Interval: user-specified delay
		intervalMs,
	)
}

// HasBackground reports whether a legacy background queue name is registered.
func (eqm *EventQueueManager) HasBackground(name string) bool {
	_, exists := eqm.queues["bg_"+name]

	return exists
}
