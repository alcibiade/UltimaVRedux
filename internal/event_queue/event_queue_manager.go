package event_queue

import (
	"fmt"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// NewEventQueueManager creates a new event queue manager
func NewEventQueueManager(timeSource func() int64, systemCallbacks SystemCallbacks) *EventQueueManager {
	//nolint:exhaustruct
	return &EventQueueManager{
		queues:          make(map[string]*NamedEventQueue),
		feedbackQueues:  make(map[string]*NamedEventQueue),
		timeSource:      timeSource,
		systemCallbacks: systemCallbacks,
		inputBlockCount: 0,
	}
}

// Workflow creates an unstarted gameplay workflow. Start atomically reserves
// its name and safely defers activation when called from another queue callback.
func (eqm *EventQueueManager) Workflow(name string) *NamedEventQueue {
	return eqm.newQueue(name, queueLaneWorkflow)
}

func (eqm *EventQueueManager) feedback(name string) *NamedEventQueue {
	return eqm.newQueue(name, queueLaneFeedback)
}

// TriggerFeedback requests presentation-only feedback and continues without waiting.
func (eqm *EventQueueManager) TriggerFeedback(
	name string,
	priority int,
	build func(*SequentialEventBuilder),
) {
	queue := eqm.feedback(name).
		Priority(priority).
		Sequential()
	build(queue)
	queue.Start()
}

func (eqm *EventQueueManager) newQueue(name string, lane queueLane) *NamedEventQueue {
	//nolint:exhaustruct
	return &NamedEventQueue{
		name:             name,
		events:           make([]*DelayedEvent, 0),
		continuousEvents: make([]ContinuousEvent, 0),
		timeSource:       eqm.timeSource,
		manager:          eqm,
		lastUpdateTime:   eqm.timeSource(),
		lane:             lane,
		lifecycle:        queueCreated,
	}
}

func (eqm *EventQueueManager) startQueue(queue *NamedEventQueue, priority int) {
	if eqm.shutdown {
		queue.lifecycle = queueFinalized

		return
	}
	if queue.name == "" {
		panic("EventQueue: queue name cannot be empty")
	}
	if queue.lane == queueLaneFeedback && queue.autoBlockUnblockInput {
		panic(fmt.Sprintf("EventQueue: feedback %q cannot block input", queue.name))
	}

	queue.priority = priority
	if !eqm.reserveQueue(queue) {
		return
	}

	if eqm.executionDepth > 0 {
		queue.lifecycle = queueDeferred
		eqm.deferredStarts = append(eqm.deferredStarts, queue)

		return
	}

	queue.lifecycle = queueStarted
	eqm.dispatchQueue(queue)
}

func (eqm *EventQueueManager) reserveQueue(queue *NamedEventQueue) bool {
	switch queue.lane {
	case queueLaneWorkflow:
		if _, exists := eqm.queues[queue.name]; exists {
			panic(fmt.Sprintf("EventQueue: duplicate live workflow name %q", queue.name))
		}
		eqm.queues[queue.name] = queue
	case queueLaneFeedback:
		if _, exists := eqm.feedbackQueues[queue.name]; exists {
			queue.lifecycle = queueCoalesced

			return false
		}
		eqm.feedbackQueues[queue.name] = queue
	case queueLaneBackground:
		panic("EventQueue: background queue reached normal start routing")
	default:
		panic("EventQueue: queue has unknown execution lane")
	}

	return true
}

func (eqm *EventQueueManager) dispatchQueue(queue *NamedEventQueue) {
	switch queue.lane {
	case queueLaneWorkflow:
		eqm.dispatchWorkflow(queue)
	case queueLaneFeedback:
		eqm.dispatchFeedback(queue)
	case queueLaneBackground:
		panic("EventQueue: background queue reached deferred dispatch")
	default:
		panic("EventQueue: queue has unknown execution lane")
	}
}

func (eqm *EventQueueManager) dispatchWorkflow(queue *NamedEventQueue) {
	switch {
	case eqm.activeQueue == nil:
		eqm.activeQueue = queue
	case eqm.activeQueue.lifecycle != queueStarted:
		// A finishing input owner remains active until its key-release cleanup completes.
		// Queue new workflows behind it rather than trying to pause a finalizing queue.
		eqm.pauseQueue(queue)
		eqm.pendingQueues = append(eqm.pendingQueues, queue)
	case queue.priority < eqm.activeQueue.priority:
		eqm.pauseQueue(eqm.activeQueue)
		eqm.pendingQueues = append(eqm.pendingQueues, eqm.activeQueue)
		eqm.activeQueue = queue
	default:
		eqm.pauseQueue(queue)
		eqm.pendingQueues = append(eqm.pendingQueues, queue)
	}

	if queue.autoBlockUnblockInput && eqm.systemCallbacks != nil {
		eqm.blockInputIfNeeded()
		queue.inputBlockOwned = true
	}
}

func (eqm *EventQueueManager) dispatchFeedback(queue *NamedEventQueue) {
	if eqm.activeFeedbackQueue == nil {
		eqm.activeFeedbackQueue = queue

		return
	}

	eqm.pauseQueue(queue)
	eqm.pendingFeedbackQueues = append(eqm.pendingFeedbackQueues, queue)
}

func (eqm *EventQueueManager) flushDeferredStarts() {
	if eqm.shutdown || eqm.executionDepth != 0 || len(eqm.deferredStarts) == 0 {
		return
	}

	deferred := eqm.deferredStarts
	eqm.deferredStarts = nil
	for _, queue := range deferred {
		if queue.lifecycle != queueDeferred {
			panic(fmt.Sprintf("EventQueue: deferred queue %q has lifecycle %d", queue.name, queue.lifecycle))
		}
		queue.lifecycle = queueStarted
		eqm.dispatchQueue(queue)
	}
}

func (eqm *EventQueueManager) run(action func()) {
	eqm.executionDepth++
	defer func() { eqm.executionDepth-- }()
	action()
}

func (eqm *EventQueueManager) runError(action func() error) error {
	eqm.executionDepth++
	defer func() { eqm.executionDepth-- }()

	return action()
}

func (eqm *EventQueueManager) runContinuous(action func() EventResult) EventResult {
	eqm.executionDepth++
	defer func() { eqm.executionDepth-- }()

	return action()
}

// Priority sets the execution priority for this event queue
// Lower numbers indicate higher priority (can interrupt lower priority queues)
// Use the named constants from priorities.go: PriorityCritical, PriorityHigh,
// PriorityMedium, PriorityNormal, and PriorityLow.
// Must be called before Sequential()
func (neq *NamedEventQueue) Priority(p int) *NamedEventQueue {
	neq.priority = p

	return neq
}

// Sequential returns a sequential event builder for chained events
func (neq *NamedEventQueue) Sequential() *SequentialEventBuilder {
	//nolint:exhaustruct
	return &SequentialEventBuilder{
		queue:    neq,
		priority: neq.priority, // Copy priority from queue to builder
	}
}

// ShouldYield checks if a higher priority queue is waiting to execute
// Returns true if any pending queue has higher priority (lower number) than currentPriority
// Use in long-running loops (While, WhileAwaitingInput) to cooperatively yield to interruptions
func (eqm *EventQueueManager) ShouldYield(currentPriority int) bool {
	// Check all pending queues for higher priority
	for _, pending := range eqm.pendingQueues {
		if pending.priority < currentPriority {
			return true // Higher priority queue waiting
		}
	}

	return false // No higher priority pending
}

// Update processes the active queue and executes ready events
func (eqm *EventQueueManager) Update() {
	if eqm.shutdown {
		return
	}
	eqm.updateWorkflow()
	eqm.flushDeferredStarts()
}

//nolint:cyclop,funlen // Explicit lifecycle phases are clearer together in update order.
func (eqm *EventQueueManager) updateWorkflow() {
	// Only process the active queue - pending queues wait for promotion
	if eqm.activeQueue == nil {
		return // No active queue to process
	}

	queue := eqm.activeQueue
	queueName := queue.name
	currentTime := eqm.timeSource()
	previousContinuousCount := len(queue.continuousEvents)
	queuesToDelete := make([]string, 0)

	// Phase 1: Input waiting state machine
	if eqm.processInputWaiting(queue, currentTime) {
		return
	}

	// Phase 2: Pending input unblock (wait for key release)
	eqm.processPendingInputUnblock(queue, queueName)

	// Phase 3: Execute delayed events
	queue.events = eqm.processDelayedEvents(queue, queueName, currentTime)
	if eqm.shutdown {
		return
	}

	// Phase 3.5: Check for termination request (HandoffTo pattern)
	if queue.terminateRequested {
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Queue '%s' has terminateRequested=true - executing Finally() and marking for deletion",
			queueName)

		// Last cleanup runs while the terminating queue still owns its resources.
		eqm.beginQueueFinalization(queue)
		if eqm.shutdown {
			return
		}

		// Handle automatic input unblocking before deletion
		if queue.inputBlockOwned {
			logging.Debug(logging.CategoryGameState, "EventQueue: Unblocking input for terminated queue '%s'", queueName)
			eqm.releaseInputBlock(queue)
		}

		// Clear remaining events to prevent further execution
		queue.events = nil
		queue.continuousEvents = nil
		queue.waitingForInput = false
		queue.pendingInputUnblock = false

		// CRITICAL: Clear activeQueue BEFORE calling onComplete
		// This prevents the chained queue's Start() from moving this terminated queue to pendingQueues
		if eqm.activeQueue != nil && eqm.activeQueue.name == queueName {
			eqm.activeQueue = nil
		}

		// Remove from queues map BEFORE calling onComplete
		// This ensures the terminated queue can't be referenced during new queue creation
		logging.Debug(logging.CategoryGameState, "EventQueue: Deleting terminated queue '%s'", queueName)
		delete(eqm.queues, queueName)

		// OnComplete may now create the chained queue at depth zero.
		eqm.completeQueueFinalization(queue)

		// Promote next pending queue if no queue became active during onComplete
		if eqm.activeQueue == nil {
			eqm.promoteNextPendingQueue()
		}

		return // Skip remaining phases - queue is deleted
	}

	// Phase 4: Execute continuous events
	queue.continuousEvents = eqm.processContinuousEvents(queue, currentTime, previousContinuousCount)

	// Phase 5: Queue cleanup decision
	eqm.handleQueueCleanup(queue, queueName, &queuesToDelete)
	if eqm.shutdown {
		return
	}

	// Phase 6: Delete empty queues and remove from active list
	activeQueueDeleted := false
	for _, queueName := range queuesToDelete {
		logging.Debug(logging.CategoryGameState, "EventQueue: Deleting empty queue '%s'", queueName)

		queueToDelete := eqm.queues[queueName]
		if queueToDelete == nil {
			panic(fmt.Sprintf("EventQueue: queue %q selected for cleanup is not registered", queueName))
		}

		// Check if we're deleting the active queue
		if eqm.activeQueue != nil && eqm.activeQueue.name == queueName {
			activeQueueDeleted = true
		}

		delete(eqm.queues, queueName)
		eqm.completeQueueFinalization(queueToDelete)
	}

	// Phase 7: Promote next pending queue if active was deleted
	if activeQueueDeleted {
		eqm.promoteNextPendingQueue()
	}
}

// UpdateFeedback advances the serialized presentation lane independently of workflows.
func (eqm *EventQueueManager) UpdateFeedback() {
	if eqm.shutdown {
		return
	}
	eqm.updateFeedback()
	eqm.flushDeferredStarts()
}

func (eqm *EventQueueManager) updateFeedback() {
	queue := eqm.activeFeedbackQueue
	if queue == nil {
		return
	}

	currentTime := eqm.timeSource()
	previousContinuousCount := len(queue.continuousEvents)
	queue.events = eqm.processDelayedEvents(queue, queue.name, currentTime)
	if eqm.shutdown {
		return
	}
	if queue.terminateRequested {
		panic(fmt.Sprintf("EventQueue: feedback %q cannot hand off to another workflow", queue.name))
	}
	queue.continuousEvents = eqm.processContinuousEvents(queue, currentTime, previousContinuousCount)
	if eqm.shutdown {
		return
	}
	if len(queue.events) != 0 || len(queue.continuousEvents) != 0 || queue.waitingForInput {
		return
	}

	eqm.beginQueueFinalization(queue)
	if eqm.shutdown {
		return
	}
	delete(eqm.feedbackQueues, queue.name)
	eqm.activeFeedbackQueue = nil
	eqm.completeQueueFinalization(queue)
	eqm.promoteNextFeedbackQueue()
}

func (eqm *EventQueueManager) promoteNextFeedbackQueue() {
	if len(eqm.pendingFeedbackQueues) == 0 {
		return
	}

	index := highestPriorityQueueIndex(eqm.pendingFeedbackQueues)
	queue := eqm.pendingFeedbackQueues[index]
	eqm.pendingFeedbackQueues = removeQueueAt(eqm.pendingFeedbackQueues, index)
	eqm.resumeQueue(queue)
	eqm.activeFeedbackQueue = queue
}

func highestPriorityQueueIndex(queues []*NamedEventQueue) int {
	index := 0
	for candidate := 1; candidate < len(queues); candidate++ {
		if queues[candidate].priority < queues[index].priority {
			index = candidate
		}
	}

	return index
}

func removeQueueAt(queues []*NamedEventQueue, index int) []*NamedEventQueue {
	copy(queues[index:], queues[index+1:])
	queues[len(queues)-1] = nil

	return queues[:len(queues)-1]
}

// promoteNextPendingQueue promotes the highest priority pending queue to active
// Called when the active queue completes
// NOTE: Caller must hold eqm.mu lock
func (eqm *EventQueueManager) promoteNextPendingQueue() {
	// Clear active queue
	eqm.activeQueue = nil
	logging.Debug(logging.CategoryGameState, "EventQueue: Active queue cleared")

	// If no pending queues, we're done
	if len(eqm.pendingQueues) == 0 {
		logging.Debug(logging.CategoryGameState, "EventQueue: No pending queues to promote")

		return
	}

	// Find highest priority pending queue (lowest priority number), preserving FIFO ties.
	highestPriorityIndex := highestPriorityQueueIndex(eqm.pendingQueues)

	// Promote the highest priority queue
	promotedQueue := eqm.pendingQueues[highestPriorityIndex]
	eqm.resumeQueue(promotedQueue)
	eqm.activeQueue = promotedQueue

	eqm.pendingQueues = removeQueueAt(eqm.pendingQueues, highestPriorityIndex)

	logging.Debug(logging.CategoryGameState,
		"EventQueue: Promoted queue '%s' (priority %d) to active, %d pending remaining",
		promotedQueue.name, promotedQueue.priority, len(eqm.pendingQueues))
}

func (eqm *EventQueueManager) pauseQueue(queue *NamedEventQueue) {
	if queue.lifecycle != queueStarted || queue.isPaused {
		panic(fmt.Sprintf("EventQueue: queue %q cannot pause from lifecycle %d paused=%t",
			queue.name, queue.lifecycle, queue.isPaused))
	}
	queue.pausedAt = eqm.timeSource()
	queue.isPaused = true
}

func (eqm *EventQueueManager) resumeQueue(queue *NamedEventQueue) {
	if queue.lifecycle != queueStarted || !queue.isPaused {
		panic(fmt.Sprintf("EventQueue: queue %q cannot resume from lifecycle %d paused=%t",
			queue.name, queue.lifecycle, queue.isPaused))
	}

	currentTime := eqm.timeSource()
	pausedDuration := currentTime - queue.pausedAt
	if pausedDuration < 0 {
		panic(fmt.Sprintf("EventQueue: time moved backwards while queue %q was paused", queue.name))
	}
	for _, event := range queue.events {
		event.ExecuteAt += pausedDuration
	}
	queue.lastUpdateTime = currentTime
	queue.pausedAt = 0
	queue.isPaused = false
}

// processInputWaiting handles the input waiting state machine for a queue
// Returns true if the queue should skip normal processing this frame
func (eqm *EventQueueManager) processInputWaiting(queue *NamedEventQueue, currentTime int64) bool {
	if !queue.waitingForInput {
		return false
	}
	if eqm.inputPollingSuspended {
		// Losing focus must not manufacture a release edge. On return the
		// owning wait still requires a real release before a fresh press.
		queue.requireKeyRelease = true
		queue.keyWasPressed = false

		return true
	}

	keyPressed := false
	if eqm.systemCallbacks != nil {
		keyPressed = eqm.systemCallbacks.IsAnyKeyPressed()
	}

	// Debounce logic: wait for all keys to be released before accepting new press
	if queue.requireKeyRelease {
		if !keyPressed {
			queue.requireKeyRelease = false
			logging.Debug(logging.CategoryGameState, "EventQueue: Keys released - ready for new input")
		}
		queue.keyWasPressed = keyPressed

		return true
	}

	// Use edge detection: key is pressed AND wasn't pressed last frame
	if keyPressed && !queue.keyWasPressed {
		logging.Debug(logging.CategoryGameState, "EventQueue: Key pressed - resuming from wait point")
		queue.waitingForInput = false
		queue.lastUpdateTime = currentTime

		eqm.schedulePostInputActions(queue, currentTime)
	}

	queue.keyWasPressed = keyPressed

	return queue.waitingForInput
}

// schedulePostInputActions schedules pending actions after input is received
func (eqm *EventQueueManager) schedulePostInputActions(queue *NamedEventQueue, currentTime int64) {
	if queue.pendingNextAction != nil {
		nextExecuteAt := currentTime + queue.pendingNextDelay
		//nolint:exhaustruct
		nextEvent := &DelayedEvent{
			ExecuteAt: nextExecuteAt,
			Action:    queue.pendingNextAction,
		}
		queue.events = append(queue.events, nextEvent)
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Scheduled pending action for %d (delay: %d)",
			nextExecuteAt, queue.pendingNextDelay)

		queue.pendingNextAction = nil
		queue.pendingNextDelay = 0
	}

	// Execute any pending callback that was waiting (legacy support)
	if queue.inputWaitCallback != nil {
		err := eqm.runError(queue.inputWaitCallback)

		if err != nil {
			_ = err
		}
		queue.inputWaitCallback = nil
	}
}

// processPendingInputUnblock handles waiting for key release before unblocking input
func (eqm *EventQueueManager) processPendingInputUnblock(
	queue *NamedEventQueue,
	queueName string,
) {
	if !queue.pendingInputUnblock {
		return
	}

	anyKeyPressed := false
	if queue.inputReleaseReady != nil {
		anyKeyPressed = !queue.inputReleaseReady()
	} else if eqm.systemCallbacks != nil {
		anyKeyPressed = eqm.systemCallbacks.IsAnyKeyPressed()
	}

	if !anyKeyPressed {
		logging.Debug(logging.CategoryGameState,
			"EventQueue: All keys released - unblocking input for queue '%s'", queueName)
		eqm.releaseInputBlock(queue)
		queue.pendingInputUnblock = false
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Queue '%s' ready for deletion after input unblock", queueName)
	} else {
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Queue '%s' waiting for key release before unblocking", queueName)
	}
}

// processDelayedEvents executes ready delayed events and returns remaining events
func (eqm *EventQueueManager) processDelayedEvents(
	queue *NamedEventQueue,
	queueName string,
	currentTime int64,
) []*DelayedEvent {
	remainingEvents := make([]*DelayedEvent, 0)

	//nolint:intrange // Callbacks append chained events; re-read the length so due events run in this update.
	for i := 0; i < len(queue.events); i++ {
		event := queue.events[i]
		if event.ExecuteAt <= currentTime {
			eqm.executeDelayedEvent(queue, queueName, event, currentTime)
			if eqm.shutdown {
				return nil
			}

			// Check for termination request after each event execution
			// (HandoffTo may have been called during the event callback)
			if queue.terminateRequested {
				logging.Debug(logging.CategoryGameState,
					"EventQueue: Termination requested during event execution - skipping remaining events")
				// Return empty slice - no remaining events matter since we're terminating
				return nil
			}
		} else {
			logging.Debug(logging.CategoryGameState,
				"EventQueue: Event in queue '%s' not ready - scheduled for %d (current: %d)",
				queueName, event.ExecuteAt, currentTime)
			remainingEvents = append(remainingEvents, event)
		}
	}

	return remainingEvents
}

// executeDelayedEvent executes a single delayed event and handles chaining
func (eqm *EventQueueManager) executeDelayedEvent(
	queue *NamedEventQueue,
	queueName string,
	event *DelayedEvent,
	currentTime int64,
) {
	logging.Debug(logging.CategoryGameState,
		"EventQueue: Executing event in queue '%s' scheduled for %d (current: %d)",
		queueName, event.ExecuteAt, currentTime)

	// Execute callback (may return error)
	err := eqm.runError(event.Action)
	if eqm.shutdown {
		return
	}

	if err != nil {
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Event in queue '%s' returned error: %v", queueName, err)
		_ = err
	}

	// Don't schedule any more actions if termination was requested
	if queue.terminateRequested {
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Termination requested - skipping next action scheduling")

		return
	}

	if event.NextAction != nil {
		if event.IsWaitPoint {
			logging.Debug(logging.CategoryGameState,
				"EventQueue: Event is wait point - storing next action for input")
			queue.waitingForInput = true
			queue.pendingNextAction = event.NextAction
			queue.pendingNextDelay = event.NextDelay
			queue.keyWasPressed = false
		} else {
			nextExecuteAt := currentTime + event.NextDelay
			//nolint:exhaustruct
			nextEvent := &DelayedEvent{
				ExecuteAt: nextExecuteAt,
				Action:    event.NextAction,
			}
			queue.events = append(queue.events, nextEvent)
			logging.Debug(logging.CategoryGameState,
				"EventQueue: Chained next action scheduled for %d (delay: %d)",
				nextExecuteAt, event.NextDelay)
		}
	}
}

// processContinuousEvents executes continuous events and returns remaining events
func (eqm *EventQueueManager) processContinuousEvents(
	queue *NamedEventQueue,
	currentTime int64,
	previousContinuousCount int,
) []ContinuousEvent {
	//nolint:gosec
	deltaMs := uint64(currentTime - queue.lastUpdateTime)
	queue.lastUpdateTime = currentTime

	remainingContinuousEvents := make([]ContinuousEvent, 0)
	for index, event := range queue.continuousEvents {
		eventDeltaMs := deltaMs
		if index >= previousContinuousCount {
			// Delayed callbacks admitted this event at the current update, not
			// at the previous one. Present its initial frame without charging
			// pre-launch time. Existing continuous events retain their full delta.
			eventDeltaMs = 0
		}
		result := eqm.runContinuous(func() EventResult { return event.Execute(eventDeltaMs) })
		if eqm.shutdown {
			return nil
		}

		if result == EventContinue {
			remainingContinuousEvents = append(remainingContinuousEvents, event)
		}
	}

	return remainingContinuousEvents
}

// handleQueueCleanup determines if a queue should be deleted or enter cleanup state
func (eqm *EventQueueManager) handleQueueCleanup(
	queue *NamedEventQueue,
	queueName string,
	queuesToDelete *[]string,
) {
	isEmpty := len(queue.events) == 0 && len(queue.continuousEvents) == 0 &&
		!queue.waitingForInput && !queue.pendingInputUnblock

	if isEmpty {
		if queue.lifecycle == queueStarted {
			eqm.beginQueueFinalization(queue)
		}
		if queue.autoBlockUnblockInput && eqm.systemCallbacks != nil && queue.inputBlockOwned {
			logging.Debug(logging.CategoryGameState,
				"EventQueue: Queue '%s' is empty - entering pendingInputUnblock state", queueName)
			queue.pendingInputUnblock = true
		} else {
			logging.Debug(logging.CategoryGameState,
				"EventQueue: Queue '%s' is empty - marking for cleanup", queueName)
			*queuesToDelete = append(*queuesToDelete, queueName)
		}
	} else {
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Queue '%s' not empty - delayed: %d, continuous: %d, waitingForInput: %t, pendingInputUnblock: %t",
			queueName, len(queue.events), len(queue.continuousEvents), queue.waitingForInput, queue.pendingInputUnblock)
	}
}

// blockInputIfNeeded blocks input with reference counting
func (eqm *EventQueueManager) blockInputIfNeeded() {
	logging.Debug(logging.CategoryGameState,
		"EventQueue: blockInputIfNeeded called - current count: %d", eqm.inputBlockCount)
	if eqm.inputBlockCount == 0 {
		// First queue requesting block
		logging.Debug(logging.CategoryGameState, "EventQueue: Blocking input (first queue)")
		eqm.systemCallbacks.BlockAllInput(true)
	}
	eqm.inputBlockCount++
	logging.Debug(logging.CategoryGameState,
		"EventQueue: Input block count now: %d", eqm.inputBlockCount)
}

// unblockInputIfNeeded unblocks input with reference counting
func (eqm *EventQueueManager) unblockInputIfNeeded() {
	logging.Debug(logging.CategoryGameState,
		"EventQueue: unblockInputIfNeeded called - current count: %d", eqm.inputBlockCount)

	if eqm.inputBlockCount <= 0 {
		panic(fmt.Sprintf("EventQueue: input block underflow (count %d)", eqm.inputBlockCount))
	}

	eqm.inputBlockCount--
	logging.Debug(logging.CategoryGameState,
		"EventQueue: Input block count now: %d", eqm.inputBlockCount)
	if eqm.inputBlockCount == 0 {
		// Last queue finished - unblock input
		logging.Debug(logging.CategoryGameState, "EventQueue: Unblocking input (last queue finished)")
		eqm.systemCallbacks.BlockAllInput(false)
	}
}

func (eqm *EventQueueManager) releaseInputBlock(queue *NamedEventQueue) {
	if !queue.inputBlockOwned {
		panic(fmt.Sprintf("EventQueue: queue %q released input without ownership", queue.name))
	}
	queue.inputBlockOwned = false
	eqm.unblockInputIfNeeded()
}

// beginQueueFinalization runs Last exactly once when queue work ends.
func (eqm *EventQueueManager) beginQueueFinalization(queue *NamedEventQueue) {
	if queue.lifecycle == queueFinalizing || queue.lifecycle == queueFinalized {
		panic(fmt.Sprintf("EventQueue: queue %q began finalization twice", queue.name))
	}
	if queue.lifecycle != queueStarted {
		panic(fmt.Sprintf("EventQueue: queue %q began finalization before Start", queue.name))
	}

	queue.lifecycle = queueFinalizing
	if queue.lastAction != nil {
		eqm.run(queue.lastAction)
	}
}

// completeQueueFinalization invokes OnComplete after Last and ownership cleanup.
func (eqm *EventQueueManager) completeQueueFinalization(queue *NamedEventQueue) {
	if queue.lifecycle != queueFinalizing {
		panic(fmt.Sprintf("EventQueue: queue %q completed finalization from invalid state %d", queue.name, queue.lifecycle))
	}

	queue.lifecycle = queueFinalized
	if queue.onComplete != nil {
		eqm.run(queue.onComplete)
	}
}

// HasWorkflow reports whether a workflow is deferred, active, pending, or finalizing.
func (eqm *EventQueueManager) HasWorkflow(name string) bool {
	_, exists := eqm.queues[name]

	return exists
}

// HasOnlyWorkflow reports whether name is the sole deferred, active, pending,
// or finalizing gameplay workflow.
func (eqm *EventQueueManager) HasOnlyWorkflow(name string) bool {
	return len(eqm.queues) == 1 && eqm.HasWorkflow(name)
}

// HasFeedback reports whether feedback is active, pending, deferred, or finalizing.
func (eqm *EventQueueManager) HasFeedback(name string) bool {
	_, exists := eqm.feedbackQueues[name]

	return exists
}

// HasInputBlockingWorkflows includes reserved, queued, and finalizing input
// owners, not just the currently executing workflow.
func (eqm *EventQueueManager) HasInputBlockingWorkflows() bool {
	for _, queue := range eqm.queues {
		if queue.autoBlockUnblockInput {
			return true
		}
	}

	return false
}

// HasActiveWorkflows reports whether any workflow is active or pending.
func (eqm *EventQueueManager) HasActiveWorkflows() bool {
	return eqm.activeQueue != nil || len(eqm.pendingQueues) > 0
}

// SetShutdownAction registers owner cleanup for transient state that may outlive
// an individual queue. It must be configured before shutdown begins.
func (eqm *EventQueueManager) SetShutdownAction(action func()) {
	if eqm.shutdown {
		panic("EventQueue: shutdown action registered after shutdown")
	}
	if action == nil {
		panic("EventQueue: shutdown action is nil")
	}
	if eqm.shutdownAction != nil {
		panic("EventQueue: shutdown action is already registered")
	}
	eqm.shutdownAction = action
}

// Shutdown permanently cancels all scene-owned queues. Every accepted queue
// runs Finally at most once before input ownership and registrations are released.
// Repeated shutdowns and later queue starts are harmless.
func (eqm *EventQueueManager) Shutdown() {
	if eqm.shutdown {
		return
	}
	eqm.shutdown = true
	if eqm.shutdownAction != nil {
		eqm.shutdownAction()
		eqm.shutdownAction = nil
	}

	queueNames := make([]string, 0, len(eqm.queues))
	for name := range eqm.queues {
		queueNames = append(queueNames, name)
	}
	sort.Strings(queueNames)
	for _, name := range queueNames {
		eqm.cancelQueueForShutdown(eqm.queues[name])
	}

	feedbackNames := make([]string, 0, len(eqm.feedbackQueues))
	for name := range eqm.feedbackQueues {
		feedbackNames = append(feedbackNames, name)
	}
	sort.Strings(feedbackNames)
	for _, name := range feedbackNames {
		eqm.cancelQueueForShutdown(eqm.feedbackQueues[name])
	}

	clear(eqm.queues)
	clear(eqm.feedbackQueues)
	eqm.activeQueue = nil
	eqm.pendingQueues = nil
	eqm.deferredStarts = nil
	eqm.activeFeedbackQueue = nil
	eqm.pendingFeedbackQueues = nil
	eqm.activeBackgroundQueues = nil
	if eqm.inputBlockCount != 0 {
		panic(fmt.Sprintf("EventQueue: shutdown left %d input blocks owned", eqm.inputBlockCount))
	}
}

func (eqm *EventQueueManager) cancelQueueForShutdown(queue *NamedEventQueue) {
	switch queue.lifecycle {
	case queueDeferred, queueStarted:
		queue.lifecycle = queueFinalizing
		if queue.lastAction != nil {
			eqm.run(queue.lastAction)
		}
	case queueFinalizing:
		// Finally already ran before key-release cleanup or invoked shutdown itself.
	case queueCreated, queueCoalesced, queueFinalized:
		queue.lifecycle = queueFinalized
		queue.onComplete = nil

		return
	default:
		panic(fmt.Sprintf("EventQueue: queue %q has unknown lifecycle %d", queue.name, queue.lifecycle))
	}

	if queue.inputBlockOwned {
		eqm.releaseInputBlock(queue)
	}
	queue.events = nil
	queue.continuousEvents = nil
	queue.waitingForInput = false
	queue.pendingInputUnblock = false
	queue.pendingNextAction = nil
	queue.inputWaitCallback = nil
	queue.onComplete = nil
	queue.lifecycle = queueFinalized
}

// HasBackground reports whether a legacy background queue name is registered.
func (eqm *EventQueueManager) HasBackground(name string) bool {
	_, exists := eqm.queues["bg_"+name]

	return exists
}

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

// UpdateBackground processes all background queues concurrently with the main queue.
// Called from gamescene_update.go after Update() to process visual effects.
// Background queues never block input and run independently of main queue lifecycle.
// All active background queues are processed in parallel - each fades independently.
func (eqm *EventQueueManager) UpdateBackground() {
	if eqm.shutdown {
		return
	}
	eqm.updateBackground()
	eqm.flushDeferredStarts()
}

func (eqm *EventQueueManager) updateBackground() {
	if len(eqm.activeBackgroundQueues) == 0 {
		return
	}

	currentTime := eqm.timeSource()

	// Process ALL active background queues concurrently
	// Collect queues that are still active after processing
	remainingQueues := make([]*NamedEventQueue, 0, len(eqm.activeBackgroundQueues))

	for _, queue := range eqm.activeBackgroundQueues {
		previousContinuousCount := len(queue.continuousEvents)
		logging.Debug(logging.CategoryGameState,
			"EventQueue: UpdateBackground processing '%s' with %d events, currentTime=%d",
			queue.name, len(queue.events), currentTime)

		// Process delayed events (simplified - no input waiting for background queues)
		// Track original length to preserve any events chained during execution
		originalLen := len(queue.events)
		remainingEvents := make([]*DelayedEvent, 0)

		for i := range originalLen {
			event := queue.events[i]
			if event.ExecuteAt <= currentTime {
				eqm.executeBackgroundEvent(queue, event, currentTime)
				if eqm.shutdown {
					return
				}
			} else {
				remainingEvents = append(remainingEvents, event)
			}
		}

		// Append any newly chained events (added during execution) to remaining events
		if len(queue.events) > originalLen {
			remainingEvents = append(remainingEvents, queue.events[originalLen:]...)
		}

		queue.events = remainingEvents

		queue.continuousEvents = eqm.processContinuousEvents(queue, currentTime, previousContinuousCount)
		if eqm.shutdown {
			return
		}

		// Check if this queue is complete
		if len(queue.events) == 0 && len(queue.continuousEvents) == 0 {
			logging.Debug(logging.CategoryGameState,
				"EventQueue: Background queue '%s' completed", queue.name)

			// Remove from queues map
			queueKey := "bg_" + queue.name
			delete(eqm.queues, queueKey)
			eqm.beginQueueFinalization(queue)
			if eqm.shutdown {
				return
			}
			eqm.completeQueueFinalization(queue)
			// Don't add to remainingQueues - this queue is done
		} else {
			// Queue still has work - keep it active
			remainingQueues = append(remainingQueues, queue)
		}
	}

	// Update active queues to only those still running
	eqm.activeBackgroundQueues = remainingQueues
}

// executeBackgroundEvent executes a single background event and handles chaining
func (eqm *EventQueueManager) executeBackgroundEvent(
	queue *NamedEventQueue,
	event *DelayedEvent,
	currentTime int64,
) {
	logging.Debug(logging.CategoryGameState,
		"EventQueue: Executing background event in queue '%s'", queue.name)

	// Execute callback
	if err := eqm.runError(event.Action); err != nil {
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Background event returned error: %v", err)
	}
	if eqm.shutdown {
		return
	}

	// Handle chaining to next action
	if event.NextAction != nil {
		nextExecuteAt := currentTime + event.NextDelay
		//nolint:exhaustruct
		nextEvent := &DelayedEvent{
			ExecuteAt: nextExecuteAt,
			Action:    event.NextAction,
		}
		queue.events = append(queue.events, nextEvent)
		logging.Debug(logging.CategoryGameState,
			"EventQueue: Chained background action scheduled for %d", nextExecuteAt)
	}
}

// HasActiveBackgroundQueues returns true if there are any active background queues
func (eqm *EventQueueManager) HasActiveBackgroundQueues() bool {
	return len(eqm.activeBackgroundQueues) > 0
}
