package event_queue

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// RegisterBackgroundQueue creates a background event queue that runs concurrently with the main queue.
// Background queues are designed for UI-only effects that don't block input or mutate game state.
//
// Key differences from Workflow:
//   - Never blocks input (isBackground=true)
//   - Runs in parallel with main queue (not serialized)
//   - FIFO stacking: multiple background queues execute in order
//   - Independent lifecycle: continues running even after main queue completes
//
// Use cases: Action feedback popups, visual effects, notifications
//
// Example usage:
//
//	eqm.RegisterBackgroundQueue("action_feedback").
//	    Priority(PriorityNormal).
//	    Sequential().
//	    First(func() { dialog.Show() }).
//	    Wait(2000).
//	    Then(func() { dialog.Hide() }).
//	    Start()
func (eqm *EventQueueManager) RegisterBackgroundQueue(name string) *NamedEventQueue {
	logging.Debug(logging.CategoryGameState,
		"EventQueue: RegisterBackgroundQueue('%s') called", name)

	// Background queues allow registration during main queue execution
	// (this is intentional - they run concurrently)

	//nolint:exhaustruct
	queue := &NamedEventQueue{
		name:             name,
		events:           make([]*DelayedEvent, 0),
		continuousEvents: make([]ContinuousEvent, 0),
		timeSource:       eqm.timeSource,
		manager:          eqm,
		lastUpdateTime:   eqm.timeSource(),
		lane:             queueLaneBackground,
		lifecycle:        queueCreated,
	}
	if eqm.shutdown {
		return queue
	}

	// Store in queues map (using "bg_" prefix to avoid collision with main queues)
	queueKey := "bg_" + name
	if _, exists := eqm.queues[queueKey]; exists {
		panic(fmt.Sprintf("EventQueue: duplicate live queue name %q", queueKey))
	}
	eqm.queues[queueKey] = queue

	logging.Debug(logging.CategoryGameState, "EventQueue: Registered background queue '%s'", name)

	return queue
}
