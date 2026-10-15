package event_queue

// InputReleaseReady supplies a narrow workflow handoff guard. A scene may treat
// presses protected through physical release as safe, without changing raw
// WaitForAnyKey acknowledgement. The guard must not mutate gameplay or input.
func (seb *SequentialEventBuilder) InputReleaseReady(ready func() bool) *SequentialEventBuilder {
	if seb.queue.lane != queueLaneWorkflow || ready == nil {
		panic("InputReleaseReady requires a workflow and release guard")
	}
	seb.queue.inputReleaseReady = ready

	return seb
}

// SuspendInputPolling retains raw acknowledgement waits while allowing elapsed
// presentation work to progress. The input adapter must also mask ordinary key
// reads for workflow callbacks. Restore once after the non-input update.
func (eqm *EventQueueManager) SuspendInputPolling() func() {
	previous := eqm.inputPollingSuspended
	eqm.inputPollingSuspended = true

	return func() { eqm.inputPollingSuspended = previous }
}
