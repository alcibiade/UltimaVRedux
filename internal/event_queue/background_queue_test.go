package event_queue

import (
	"testing"
)

// TestBackgroundQueue_Registration tests that background queues are registered correctly
func TestBackgroundQueue_Registration(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	// Act - Register a background queue
	queue := manager.RegisterBackgroundQueue("test_bg_queue")

	// Assert
	if queue == nil {
		t.Fatal("RegisterBackgroundQueue should return a queue")
	}

	if queue.lane != queueLaneBackground {
		t.Error("Queue should be marked as background")
	}

	// Verify the queue is stored with "bg_" prefix
	if !manager.HasBackground("test_bg_queue") {
		t.Error("Background queue should be stored with 'bg_' prefix")
	}
}

// TestBackgroundQueue_NeverBlocksInput tests that background queues never block input
func TestBackgroundQueue_NeverBlocksInput(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	executed := false

	// Act - Create background queue with auto-block enabled (should be ignored)
	manager.RegisterBackgroundQueue("test_bg").
		Sequential().
		First(func() {
			executed = true
		}).
		Start()

	// Update to execute the queue
	manager.UpdateBackground()

	// Assert
	if !executed {
		t.Error("Background queue event should execute")
	}

	// Background queues should NEVER block input
	if mockCallbacks.BlockInputCalls > 0 {
		t.Errorf("Background queue should never block input, got %d block calls", mockCallbacks.BlockInputCalls)
	}
}

// TestBackgroundQueue_RunsConcurrently tests background queues run independently of main queue
func TestBackgroundQueue_RunsConcurrently(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	mainQueueExecuted := false
	bgQueueExecuted := false

	// Act - Create main queue
	manager.Workflow("main_queue").
		Priority(PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			mainQueueExecuted = true
		}).
		Start()

	// Create background queue
	manager.RegisterBackgroundQueue("bg_queue").
		Sequential().
		First(func() {
			bgQueueExecuted = true
		}).
		Start()

	// Update both queues
	manager.Update()
	manager.UpdateBackground()

	// Assert - Both should execute
	if !mainQueueExecuted {
		t.Error("Main queue should execute")
	}
	if !bgQueueExecuted {
		t.Error("Background queue should execute concurrently")
	}
}

// TestBackgroundQueue_ConcurrentExecution tests that multiple background queues execute concurrently
func TestBackgroundQueue_ConcurrentExecution(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	var executionOrder []string

	// Act - Register multiple background queues (they should ALL execute concurrently)
	manager.RegisterBackgroundQueue("bg_first").
		Sequential().
		First(func() {
			executionOrder = append(executionOrder, "first")
		}).
		Start()

	manager.RegisterBackgroundQueue("bg_second").
		Sequential().
		First(func() {
			executionOrder = append(executionOrder, "second")
		}).
		Start()

	manager.RegisterBackgroundQueue("bg_third").
		Sequential().
		First(func() {
			executionOrder = append(executionOrder, "third")
		}).
		Start()

	// Single update should execute ALL queues concurrently
	manager.UpdateBackground()

	// All three should have executed in the same update frame
	if len(executionOrder) != 3 {
		t.Errorf("All queues should execute concurrently in single update, got: %v", executionOrder)
	}

	// Order should be based on registration order (slice iteration order)
	expected := []string{"first", "second", "third"}
	for i, v := range expected {
		if executionOrder[i] != v {
			t.Errorf("Expected order %v, got %v", expected, executionOrder)
			break
		}
	}

	// All queues should be complete (no more active)
	if manager.HasActiveBackgroundQueues() {
		t.Error("All background queues should be complete after single update")
	}
}

// TestBackgroundQueue_WithDelays tests that background queues handle delays correctly
func TestBackgroundQueue_WithDelays(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	var executionOrder []string

	// Act - Create background queue with delays
	manager.RegisterBackgroundQueue("bg_delayed").
		Sequential().
		First(func() {
			executionOrder = append(executionOrder, "start")
		}).
		Wait(500).
		Then(func() {
			executionOrder = append(executionOrder, "after_500ms")
		}).
		Wait(300).
		Then(func() {
			executionOrder = append(executionOrder, "after_800ms")
		}).
		Start()

	// Initial update - executes first event
	manager.UpdateBackground()
	if len(executionOrder) != 1 || executionOrder[0] != "start" {
		t.Errorf("First event should execute immediately, got: %v", executionOrder)
	}

	// Before delay completes
	mockTime = 400
	manager.UpdateBackground()
	if len(executionOrder) != 1 {
		t.Errorf("Second event should not execute before delay, got: %v", executionOrder)
	}

	// After first delay
	mockTime = 500
	manager.UpdateBackground()
	if len(executionOrder) != 2 || executionOrder[1] != "after_500ms" {
		t.Errorf("Second event should execute at 500ms, got: %v", executionOrder)
	}

	// After second delay
	mockTime = 800
	manager.UpdateBackground()
	if len(executionOrder) != 3 || executionOrder[2] != "after_800ms" {
		t.Errorf("Third event should execute at 800ms, got: %v", executionOrder)
	}
}

// TestBackgroundQueue_HasActiveBackgroundQueues tests the HasActiveBackgroundQueues method
func TestBackgroundQueue_HasActiveBackgroundQueues(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	// Initially no background queues
	if manager.HasActiveBackgroundQueues() {
		t.Error("Should have no active background queues initially")
	}

	// Register a background queue
	manager.RegisterBackgroundQueue("test_bg").
		Sequential().
		First(func() {}).
		Wait(1000). // Long wait so queue stays active
		Then(func() {}).
		Start()

	// Now should have active background queue
	if !manager.HasActiveBackgroundQueues() {
		t.Error("Should have active background queue after registration")
	}

	// Execute and complete queue
	manager.UpdateBackground() // Execute first
	mockTime = 1000
	manager.UpdateBackground() // Execute second

	// Queue should be cleaned up
	if manager.HasActiveBackgroundQueues() {
		t.Error("Should have no active background queues after completion")
	}
}

// TestBackgroundQueue_Cleanup tests that background queues clean up properly via Finally()
func TestBackgroundQueue_Cleanup(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	cleanupCalled := false

	// Act - Create background queue with Finally() cleanup
	manager.RegisterBackgroundQueue("bg_cleanup").
		Sequential().
		First(func() {}).Finally(

		func() {
			cleanupCalled = true
		}).
		Start()

	// Execute the queue
	manager.UpdateBackground()

	// Assert - Cleanup should be called
	if !cleanupCalled {
		t.Error("Finally() cleanup should be called when background queue completes")
	}
}

// TestBackgroundQueue_ContinuesAfterMainQueue tests background queues continue when main queue finishes
func TestBackgroundQueue_ContinuesAfterMainQueue(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	mainComplete := false
	bgComplete := false

	// Create main queue that completes quickly
	manager.Workflow("main").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			mainComplete = true
		}).
		Start()

	// Create background queue with longer duration
	manager.RegisterBackgroundQueue("bg_long").
		Sequential().
		First(func() {}).
		Wait(1000).
		Then(func() {
			bgComplete = true
		}).
		Start()

	// Execute main queue (completes immediately)
	manager.Update()
	if !mainComplete {
		t.Error("Main queue should complete")
	}

	// Background queue should still be active
	manager.UpdateBackground() // First event
	if bgComplete {
		t.Error("Background queue should not be complete yet")
	}

	// Main queue is done, but background continues
	if manager.activeQueue != nil {
		t.Error("Main queue should be nil after completion")
	}

	// Background queue should still be running
	if !manager.HasActiveBackgroundQueues() {
		t.Error("Background queue should still be active")
	}

	// Complete background queue
	mockTime = 1000
	manager.UpdateBackground()
	if !bgComplete {
		t.Error("Background queue should complete after delay")
	}
}
