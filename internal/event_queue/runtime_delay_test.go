package event_queue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepeatRuntimeDelayReadsAtAdmissionAndPreservesScheduledInterval(t *testing.T) {
	now, delay := int64(0), int64(200)
	reads, finalized := 0, 0
	var times []int64
	manager := NewEventQueueManager(func() int64 { return now }, &MockSystemCallbacks{})
	manager.Workflow("runtime_delay").Priority(PriorityNormal).Sequential().First(func() {}).
		RepeatWithRuntimeDelay(3, func() int64 { reads++; return delay }, func(int) bool {
			times = append(times, now)
			return true
		}).Wait(75).Then(func() { times = append(times, now) }).
		Finally(func() { finalized++ }).Start()
	require.Zero(t, reads, "building does not capture pacing")
	manager.Update()
	require.Equal(t, 1, reads)
	now, delay = 100, 50
	manager.Update()
	require.Empty(t, times, "the admitted 200ms deadline remains intact")
	for _, at := range []int64{200, 250, 300, 374, 375} {
		now = at
		manager.Update()
	}
	require.Equal(t, []int64{200, 250, 300, 375}, times)
	for range 3 {
		manager.Update()
	}
	require.Equal(t, 3, reads)
	require.Equal(t, 1, finalized)
}

func TestRepeatRuntimeDelayHitchEarlyExitAndShutdownFinalizeOnce(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		now := int64(0)
		calls, finalized, continued := 0, 0, 0
		manager := NewEventQueueManager(func() int64 { return now }, &MockSystemCallbacks{})
		manager.Workflow("bounded_runtime_delay").Priority(PriorityNormal).Sequential().BlockInput().First(func() {}).
			RepeatWithRuntimeDelay(96, func() int64 { return 50 }, func(int) bool {
				calls++
				return calls < 2
			}).Then(func() { continued++ }).Finally(func() { finalized++ }).Start()
		manager.Update()
		now = 100000
		manager.Update()
		require.Equal(t, 1, calls, "a hitch cannot drain sleeping cycles")
		if cancel {
			manager.Shutdown()
			manager.Shutdown()
			require.Zero(t, continued)
		} else {
			now += 50
			for range 4 {
				manager.Update()
			}
			require.Equal(t, 2, calls)
			require.Equal(t, 1, continued)
		}
		require.Equal(t, 1, finalized)
		require.False(t, manager.HasInputBlockingWorkflows())
	}
}

func TestWorkflowReleaseGuardDoesNotBypassRawAcknowledgement(t *testing.T) {
	now := int64(0)
	callbacks := &MockSystemCallbacks{KeyPressed: true}
	manager := NewEventQueueManager(func() int64 { return now }, callbacks)
	acknowledged := false
	manager.Workflow("protected_release").Priority(PriorityNormal).Sequential().BlockInput().
		InputReleaseReady(func() bool { return true }).First(func() {}).WaitForAnyKey().
		Then(func() { acknowledged = true }).Start()
	for range 4 {
		manager.Update()
	}
	require.False(t, acknowledged)
	callbacks.KeyPressed = false
	manager.Update()
	callbacks.KeyPressed = true
	for range 4 {
		manager.Update()
	}
	require.True(t, acknowledged)
	require.False(t, manager.HasInputBlockingWorkflows(), "protected key permits handoff after real acknowledgement")
}

func TestSuspendedInputPollingKeepsAcknowledgementFreshAndPresentationRunning(t *testing.T) {
	now := int64(0)
	callbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(func() int64 { return now }, callbacks)
	presented, acknowledged := false, false
	manager.Workflow("unfocused_presentation").Priority(PriorityNormal).Sequential().BlockInput().
		First(func() {}).Wait(100).Then(func() { presented = true }).WaitForAnyKey().
		Then(func() { acknowledged = true }).Start()
	manager.Update()
	restore := manager.SuspendInputPolling()
	now = 100
	manager.Update()
	require.True(t, presented, "elapsed work continues while input is suspended")
	manager.Update() // A masked no-key poll must not arm the wait.
	restore()
	callbacks.KeyPressed = true
	manager.Update()
	require.False(t, acknowledged, "focus return with an already held key is not a fresh acknowledgement")
	callbacks.KeyPressed = false
	manager.Update()
	callbacks.KeyPressed = true
	manager.Update()
	require.True(t, acknowledged)
	callbacks.KeyPressed = false
	manager.Update()
	require.False(t, manager.HasInputBlockingWorkflows())
}
