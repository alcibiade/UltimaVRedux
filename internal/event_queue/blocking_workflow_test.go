package event_queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInputBlockingWorkflowsIncludesPendingOwnersButNotBackgroundWork(t *testing.T) {
	var elapsed int64
	manager := NewEventQueueManager(func() int64 { return elapsed }, nil)
	manager.Workflow("background").Priority(PriorityHigh).Sequential().
		First(func() {}).Wait(1000).Start()
	assert.True(t, manager.HasActiveWorkflows())
	assert.False(t, manager.HasInputBlockingWorkflows())
	manager.Workflow("contact").Priority(PriorityNormal).Sequential().BlockInput().
		First(func() {}).Finally(func() {
		assert.True(t, manager.HasInputBlockingWorkflows(), "finalizing contact still owns completion")
	}).Start()
	assert.True(t, manager.HasInputBlockingWorkflows(), "queued contact is an owner even before its first callback")
	manager.Shutdown()
	assert.False(t, manager.HasInputBlockingWorkflows())
}

func TestGuardedWorkflowReleasesInputInItsCompletionUpdate(t *testing.T) {
	for _, test := range []struct {
		name           string
		guard          func(*bool) func() bool
		heldAtFinish   bool
		releaseUpdates int
	}{
		{"guarded and free", func(held *bool) func() bool { return func() bool { return !*held } }, false, 1},
		{"guarded and held", func(held *bool) func() bool { return func() bool { return !*held } }, true, 2},
		{"unguarded", nil, false, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			callbacks := &MockSystemCallbacks{}
			manager := NewEventQueueManager(func() int64 { return 0 }, callbacks)
			held := test.heldAtFinish
			builder := manager.Workflow("turn").Priority(PriorityHigh).Sequential().BlockInput()
			if test.guard != nil {
				builder = builder.InputReleaseReady(test.guard(&held))
			}
			builder.First(func() {}).Start()
			updates := 0
			for callbacks.InputBlocked && updates < 4 {
				manager.Update()
				updates++
				held = false
			}
			assert.Equal(t, test.releaseUpdates, updates)
			assert.False(t, manager.HasWorkflow("turn"))
			assert.Equal(t, 1, callbacks.UnblockInputCalls)
		})
	}
}
