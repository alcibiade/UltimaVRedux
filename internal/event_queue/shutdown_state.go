package event_queue

// IsShutdown reports whether shutdown has begun, including while owner cleanup
// and queue finalizers are running. Shutdown is permanent; new Start requests
// are rejected without running their finalizers.
func (eqm *EventQueueManager) IsShutdown() bool {
	return eqm.shutdown
}
