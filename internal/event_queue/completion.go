package event_queue

// OnComplete registers the workflow's single normal-completion continuation.
// It runs after Finally, input release and name removal; shutdown omits it.
// Do not combine it with HandoffTo, which owns the same continuation slot.
// Registration after finalization panics because the callback could never run.
func (seb *SequentialEventBuilder) OnComplete(action func()) *SequentialEventBuilder {
	if seb.queue.lane != queueLaneWorkflow || action == nil {
		panic("OnComplete requires a workflow and completion callback")
	}
	if seb.queue.lifecycle == queueFinalized {
		panic("OnComplete cannot register after workflow finalization")
	}
	if seb.queue.onComplete != nil {
		panic("workflow already has a completion callback")
	}
	seb.queue.onComplete = action

	return seb
}
