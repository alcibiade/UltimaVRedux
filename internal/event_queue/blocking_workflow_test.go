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
