package event_queue

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
)

// TestWhileAwaitingInput_CallbackReceivesKey tests that callback receives the pressed key
func TestWhileAwaitingInput_CallbackReceivesKey(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	upKey := semantic_input.KeyUp
	mockCallbacks := &MockSystemCallbacks{
		PressedKey: &upKey,
	}

	manager := NewEventQueueManager(timeSource, mockCallbacks)
	var receivedKey *semantic_input.Key
	actionCount := 0

	// Act - Create queue with WhileAwaitingInput
	manager.Workflow("test_receive_key").
		Priority(PriorityNormal).Sequential().
		First(func() {
			// Initial setup
		}).
		WhileAwaitingInput(
			func() {
				actionCount++
			},
			func(key *semantic_input.Key) bool {
				receivedKey = key
				return true // Stop on first key
			},
			100,
		).
		Start()

	// Execute - should run action then check key
	manager.Update()

	// Assert - Callback should have received the Up key
	if receivedKey == nil {
		t.Error("Callback should have received a key")
	} else if *receivedKey != semantic_input.KeyUp {
		t.Errorf("Expected KeyUp, got %v", *receivedKey)
	}

	// Action should have executed once before key check
	if actionCount != 1 {
		t.Errorf("Expected 1 action execution, got %d", actionCount)
	}
}

// TestWhileAwaitingInput_ContinuesOnFalse tests that loop continues when callback returns false
func TestWhileAwaitingInput_ContinuesOnFalse(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	leftKey := semantic_input.KeyLeft
	mockCallbacks := &MockSystemCallbacks{
		PressedKey: &leftKey, // Key pressed but callback says continue
	}

	manager := NewEventQueueManager(timeSource, mockCallbacks)
	actionCount := 0
	callbackCount := 0

	// Act - Create queue that continues for 3 iterations
	manager.Workflow("test_continue").
		Priority(PriorityNormal).Sequential().
		First(func() {
			// Initial setup
		}).
		WhileAwaitingInput(
			func() {
				actionCount++
			},
			func(key *semantic_input.Key) bool {
				callbackCount++
				// Continue (return false) for first 2 calls, stop on 3rd
				return callbackCount >= 3
			},
			100,
		).
		Start()

	// Execute multiple iterations
	for i := 0; i < 5; i++ {
		manager.Update()
		mockTime += 100
		if !manager.HasActiveWorkflows() {
			break
		}
	}

	// Assert - Should have run 3 actions (stopped when callback returned true)
	if actionCount != 3 {
		t.Errorf("Expected 3 action executions, got %d", actionCount)
	}

	// Callback should have been called 3 times
	if callbackCount != 3 {
		t.Errorf("Expected 3 callback calls, got %d", callbackCount)
	}
}

// TestWhileAwaitingInput_StopsOnTrue tests that loop stops when callback returns true
func TestWhileAwaitingInput_StopsOnTrue(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	spaceKey := semantic_input.KeySpace
	mockCallbacks := &MockSystemCallbacks{
		PressedKey: &spaceKey,
	}

	manager := NewEventQueueManager(timeSource, mockCallbacks)
	actionCount := 0

	// Act - Create queue that stops immediately
	manager.Workflow("test_stop").
		Priority(PriorityNormal).Sequential().
		First(func() {
			// Initial setup
		}).
		WhileAwaitingInput(
			func() {
				actionCount++
			},
			func(key *semantic_input.Key) bool {
				return true // Stop immediately
			},
			100,
		).
		Start()

	// Execute
	manager.Update()

	// Assert - Should have run action once before stopping
	if actionCount != 1 {
		t.Errorf("Expected 1 action execution before stop, got %d", actionCount)
	}

	// Queue should be complete
	if manager.HasActiveWorkflows() {
		t.Error("Queue should complete when callback returns true")
	}
}

// TestWhileAwaitingInput_ActionExecutesBeforeKeyCheck tests execution order
func TestWhileAwaitingInput_ActionExecutesBeforeKeyCheck(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{
		PressedKey: nil, // No key initially
	}

	manager := NewEventQueueManager(timeSource, mockCallbacks)
	var executionOrder []string

	// Act - Create queue
	manager.Workflow("test_order").
		Priority(PriorityNormal).Sequential().
		First(func() {
			executionOrder = append(executionOrder, "setup")
		}).
		WhileAwaitingInput(
			func() {
				executionOrder = append(executionOrder, "action")
			},
			func(key *semantic_input.Key) bool {
				executionOrder = append(executionOrder, "callback")
				return true // Stop after first iteration
			},
			100,
		).
		Start()

	// Execute
	manager.Update()

	// Assert - Order should be: setup, action, callback
	expected := []string{"setup", "action", "callback"}
	if len(executionOrder) != len(expected) {
		t.Errorf("Expected %d events, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}

	for i, event := range expected {
		if i >= len(executionOrder) || executionOrder[i] != event {
			t.Errorf("Expected event %d to be '%s', got '%s'", i, event, executionOrder[i])
		}
	}
}

// TestWhileAwaitingInput_IntervalTiming tests proper timing between iterations
func TestWhileAwaitingInput_IntervalTiming(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{
		PressedKey: nil, // No key - rely on iteration count
	}

	manager := NewEventQueueManager(timeSource, mockCallbacks)
	actionCount := 0
	callbackCount := 0

	// Act - Create queue with 200ms interval
	manager.Workflow("test_timing").
		Priority(PriorityNormal).Sequential().
		First(func() {
			// Initial setup
		}).
		WhileAwaitingInput(
			func() {
				actionCount++
			},
			func(key *semantic_input.Key) bool {
				callbackCount++
				return callbackCount >= 3 // Stop after 3 iterations
			},
			200, // 200ms interval
		).
		Start()

	// Execute iterations with proper timing
	manager.Update() // Iteration 1 (immediate)

	mockTime += 200
	manager.Update() // Iteration 2

	mockTime += 200
	manager.Update() // Iteration 3

	// Assert - Should have 3 actions
	if actionCount != 3 {
		t.Errorf("Expected 3 action executions, got %d", actionCount)
	}

	// Should be complete
	if manager.HasActiveWorkflows() {
		t.Error("Queue should be complete after 3 iterations")
	}
}

// TestWhileAwaitingInput_HandlesNilKey tests handling when no key is pressed
func TestWhileAwaitingInput_HandlesNilKey(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{
		PressedKey: nil, // No key pressed
	}

	manager := NewEventQueueManager(timeSource, mockCallbacks)
	actionCount := 0
	var receivedKeys []*semantic_input.Key

	// Act - Create queue
	manager.Workflow("test_nil_key").
		Priority(PriorityNormal).Sequential().
		First(func() {
			// Initial setup
		}).
		WhileAwaitingInput(
			func() {
				actionCount++
			},
			func(key *semantic_input.Key) bool {
				receivedKeys = append(receivedKeys, key)
				return actionCount >= 3 // Stop after 3 iterations
			},
			100,
		).
		Start()

	// Execute iterations
	for i := 0; i < 5; i++ {
		manager.Update()
		mockTime += 100
		if !manager.HasActiveWorkflows() {
			break
		}
	}

	// Assert - Callback should have received nil keys
	if len(receivedKeys) != 3 {
		t.Errorf("Expected 3 callback calls, got %d", len(receivedKeys))
	}

	for i, key := range receivedKeys {
		if key != nil {
			t.Errorf("Expected nil key at index %d, got %v", i, *key)
		}
	}
}

// TestWhileAwaitingInput_DirectionChange tests changing direction during sailing
func TestWhileAwaitingInput_DirectionChange(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	upKey := semantic_input.KeyUp
	rightKey := semantic_input.KeyRight
	mockCallbacks := &MockSystemCallbacks{
		PressedKey: &upKey, // Start with Up
	}

	manager := NewEventQueueManager(timeSource, mockCallbacks)
	actionCount := 0
	var detectedKeys []semantic_input.Key

	// Act - Create queue that detects direction changes
	manager.Workflow("test_direction").
		Priority(PriorityNormal).Sequential().
		First(func() {
			// Initial setup
		}).
		WhileAwaitingInput(
			func() {
				actionCount++
			},
			func(key *semantic_input.Key) bool {
				if key != nil {
					detectedKeys = append(detectedKeys, *key)
					// Simulate direction change after first iteration
					if actionCount == 1 {
						mockCallbacks.PressedKey = &rightKey
					}
				}
				return actionCount >= 3 // Stop after 3 iterations
			},
			100,
		).
		Start()

	// Execute iterations
	for i := 0; i < 5; i++ {
		manager.Update()
		mockTime += 100
		if !manager.HasActiveWorkflows() {
			break
		}
	}

	// Assert - Should have detected both Up and Right keys
	if len(detectedKeys) != 3 {
		t.Errorf("Expected 3 detected keys, got %d", len(detectedKeys))
	}

	// First key should be Up
	if len(detectedKeys) > 0 && detectedKeys[0] != semantic_input.KeyUp {
		t.Errorf("Expected first key to be Up, got %v", detectedKeys[0])
	}

	// Second and third keys should be Right (after direction change)
	if len(detectedKeys) > 1 && detectedKeys[1] != semantic_input.KeyRight {
		t.Errorf("Expected second key to be Right, got %v", detectedKeys[1])
	}
	if len(detectedKeys) > 2 && detectedKeys[2] != semantic_input.KeyRight {
		t.Errorf("Expected third key to be Right, got %v", detectedKeys[2])
	}
}

// ============================================================================
// TDD Cycle 1.1: Priority() Method Tests (RED Phase)
// ============================================================================

// ============================================================================
// TDD Cycle 1.2: Start() Priority Enforcement Tests (RED Phase)
// ============================================================================

// TestStart_PanicsWithoutPriority tests that Start() panics when Priority() not called
func TestStart_PanicsWithoutPriority(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Create queue WITHOUT calling Priority()
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic when Start() called without Priority(), but no panic occurred")
		}
		// Verify the panic message contains "priority"
		if msg, ok := r.(string); ok {
			if !contains(msg, "priority") && !contains(msg, "Priority") {
				t.Errorf("Expected panic message to mention 'priority', got: %s", msg)
			}
		}
	}()

	// Attempt to start without Priority() - should panic
	manager.Workflow("test_no_priority").
		Sequential().
		First(func() { /* no-op */ }).
		Start()
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ============================================================================
// TDD Cycle 2.1: Active Queue Tracking Tests (RED Phase)
// ============================================================================

// TestActiveQueue_SingleQueueExecution tests that a single queue becomes active
func TestActiveQueue_SingleQueueExecution(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	executed := false

	manager.Workflow("test_single").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			executed = true
		}).
		Start()

	// Verify queue is tracked as active
	if manager.activeQueue == nil {
		t.Error("Expected activeQueue to be set after Start(), but it was nil")
	}

	if manager.activeQueue.name != "test_single" {
		t.Errorf("Expected activeQueue name to be 'test_single', got '%s'", manager.activeQueue.name)
	}

	// Execute the queue
	manager.Update()

	if !executed {
		t.Error("Expected queue action to execute")
	}
}

// TestPriorityInterruption_HigherPriorityInterrupts tests that higher priority interrupts lower
func TestPriorityInterruption_HigherPriorityInterrupts(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	normalExecuted := false

	// Start normal priority queue
	manager.Workflow("normal_priority").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			normalExecuted = true
		}).
		Wait(1000). // Long wait to allow interruption
		Then(func() {
			// Should not execute yet - interrupted by high priority
		}).
		Start()

	// Execute first action
	manager.Update()

	if !normalExecuted {
		t.Error("Expected normal priority action to execute")
	}

	// Start high priority queue (should interrupt)
	manager.Workflow("high_priority").
		Priority(PriorityHigh).
		Sequential().
		First(func() {
			// High priority action
		}).
		Start()

	// Verify interruption occurred
	if manager.activeQueue == nil {
		t.Error("Expected activeQueue to exist after high priority Start()")
	}

	if manager.activeQueue.name != "high_priority" {
		t.Errorf("Expected activeQueue to be 'high_priority' (interrupted), got '%s'", manager.activeQueue.name)
	}

	// Verify normal priority queue is in pending
	if len(manager.pendingQueues) != 1 {
		t.Errorf("Expected 1 pending queue, got %d", len(manager.pendingQueues))
	}

	if len(manager.pendingQueues) > 0 && manager.pendingQueues[0].name != "normal_priority" {
		t.Errorf("Expected pending queue to be 'normal_priority', got '%s'", manager.pendingQueues[0].name)
	}
}

// TestPriorityQueuing_EqualPriorityQueues tests FIFO behavior for equal priorities
func TestPriorityQueuing_EqualPriorityQueues(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Start first normal priority queue
	manager.Workflow("first").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Start()

	// Start second normal priority queue (same priority)
	manager.Workflow("second").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Start()

	// Verify first queue is still active
	if manager.activeQueue == nil || manager.activeQueue.name != "first" {
		t.Errorf("Expected activeQueue to remain 'first', got '%v'", manager.activeQueue)
	}

	// Verify second queue is pending
	if len(manager.pendingQueues) != 1 {
		t.Errorf("Expected 1 pending queue, got %d", len(manager.pendingQueues))
	}

	if len(manager.pendingQueues) > 0 && manager.pendingQueues[0].name != "second" {
		t.Errorf("Expected pending queue to be 'second', got '%s'", manager.pendingQueues[0].name)
	}
}

// TestPriorityQueuing_LowerPriorityQueues tests that lower priority always waits
func TestPriorityQueuing_LowerPriorityQueues(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Start high priority queue
	manager.Workflow("high").
		Priority(PriorityHigh).
		Sequential().
		First(func() {}).
		Start()

	// Start low priority queue (should queue, not interrupt)
	manager.Workflow("low").
		Priority(PriorityLow).
		Sequential().
		First(func() {}).
		Start()

	// Verify high priority queue is still active
	if manager.activeQueue == nil || manager.activeQueue.name != "high" {
		t.Errorf("Expected activeQueue to remain 'high', got '%v'", manager.activeQueue)
	}

	// Verify low priority queue is pending
	if len(manager.pendingQueues) != 1 {
		t.Errorf("Expected 1 pending queue, got %d", len(manager.pendingQueues))
	}

	if len(manager.pendingQueues) > 0 && manager.pendingQueues[0].name != "low" {
		t.Errorf("Expected pending queue to be 'low', got '%s'", manager.pendingQueues[0].name)
	}
}

// ============================================================================
// TDD Cycle 3.1: ShouldYield() Method Tests (RED Phase)
// ============================================================================

// TestShouldYield_NoHigherPriority tests that ShouldYield returns false with no higher priority
func TestShouldYield_NoHigherPriority(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Set up: Active normal priority, pending low priority
	manager.Workflow("active_normal").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Start()

	manager.Workflow("pending_low").
		Priority(PriorityLow).
		Sequential().
		First(func() {}).
		Start()

	// Test: Normal priority should NOT yield to lower priority
	shouldYield := manager.ShouldYield(PriorityNormal)
	if shouldYield {
		t.Error("Expected ShouldYield(PriorityNormal) to return false when only lower priority pending")
	}
}

// TestShouldYield_HigherPriorityPending tests that ShouldYield returns true for higher priority
func TestShouldYield_HigherPriorityPending(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Set up: Manually create queue state without using Start() interruption logic
	// This simulates the state during a loop: active normal, higher priority waiting in pending
	normalQueue := &NamedEventQueue{
		name:     "active_normal",
		priority: PriorityNormal,
		manager:  manager,
	}
	highQueue := &NamedEventQueue{
		name:     "pending_high",
		priority: PriorityHigh,
		manager:  manager,
	}

	// Setup test state (no mutex needed in single-threaded model)
	manager.activeQueue = normalQueue
	manager.pendingQueues = []*NamedEventQueue{highQueue}

	// Test: Normal priority SHOULD yield to higher priority in pending
	shouldYield := manager.ShouldYield(PriorityNormal)
	if !shouldYield {
		t.Error("Expected ShouldYield(PriorityNormal) to return true when higher priority pending")
	}
}

// TestShouldYield_EqualPriorityPending tests FIFO behavior (no yield for equal priority)
func TestShouldYield_EqualPriorityPending(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Set up: Active normal priority, pending normal priority
	manager.Workflow("active_normal").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Start()

	manager.Workflow("pending_normal").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Start()

	// Test: Normal priority should NOT yield to equal priority (FIFO)
	shouldYield := manager.ShouldYield(PriorityNormal)
	if shouldYield {
		t.Error("Expected ShouldYield(PriorityNormal) to return false for equal priority (FIFO)")
	}
}

// ============================================================================
// Phase 4: Queue Completion Tests
// ============================================================================

// TestPriorityInterrupt verifies that higher priority queues interrupt lower priority
func TestPriorityInterrupt(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var executionOrder []string

	// Create normal priority queue (becomes active)
	manager.Workflow("normal").
		Priority(PriorityNormal).
		Sequential().
		First(func() { executionOrder = append(executionOrder, "normal") }).
		Start()

	// Verify it's active
	if manager.activeQueue == nil {
		t.Fatal("Expected normal to be active")
	}
	if manager.activeQueue.name != "normal" {
		t.Fatalf("Expected 'normal' to be active, got: %s", manager.activeQueue.name)
	}
	t.Logf("After normal created: active=%s priority=%d", manager.activeQueue.name, manager.activeQueue.priority)

	// Create medium priority queue (should interrupt)
	manager.Workflow("medium").
		Priority(PriorityMedium).
		Sequential().
		First(func() { executionOrder = append(executionOrder, "medium") }).
		Start()

	// Verify medium interrupted normal
	if manager.activeQueue == nil {
		t.Fatal("Expected active queue after medium created")
	}
	t.Logf("After medium created: active=%s priority=%d", manager.activeQueue.name, manager.activeQueue.priority)

	if manager.activeQueue.name != "medium" {
		t.Errorf("Expected 'medium' to interrupt and become active, got: %s", manager.activeQueue.name)
	}

	// Verify normal is in pending
	if len(manager.pendingQueues) != 1 {
		t.Fatalf("Expected 1 pending queue, got: %d", len(manager.pendingQueues))
	}
	if manager.pendingQueues[0].name != "normal" {
		t.Errorf("Expected 'normal' in pending, got: %s", manager.pendingQueues[0].name)
	}
}

// TestQueueCompletion_SingleQueue tests that activeQueue is cleared when a queue finishes
func TestQueueCompletion_SingleQueue(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	executed := false

	// Create a simple queue that completes
	manager.Workflow("single").
		Priority(PriorityNormal).
		Sequential().
		First(func() { executed = true }).
		Start()

	// Verify it's the active queue
	if manager.activeQueue == nil || manager.activeQueue.name != "single" {
		t.Fatal("Expected 'single' to be active queue")
	}

	// Execute the queue
	manager.Update()

	// Verify execution
	if !executed {
		t.Error("Expected queue to execute")
	}

	// Verify activeQueue is cleared
	if manager.activeQueue != nil {
		t.Errorf("Expected activeQueue to be nil after completion, got: %v", manager.activeQueue.name)
	}

	// Verify no pending queues
	if len(manager.pendingQueues) != 0 {
		t.Errorf("Expected no pending queues, got: %d", len(manager.pendingQueues))
	}
}

// TestQueueCompletion_PromotePending tests that next pending queue becomes active
func TestQueueCompletion_PromotePending(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var executionOrder []string

	// Create first queue (becomes active)
	manager.Workflow("first").
		Priority(PriorityNormal).
		Sequential().
		First(func() { executionOrder = append(executionOrder, "first") }).
		Start()

	// Create second queue (goes to pending)
	manager.Workflow("second").
		Priority(PriorityNormal).
		Sequential().
		First(func() { executionOrder = append(executionOrder, "second") }).
		Start()

	// Verify initial state
	if manager.activeQueue.name != "first" {
		t.Errorf("Expected 'first' to be active, got: %v", manager.activeQueue.name)
	}
	if len(manager.pendingQueues) != 1 || manager.pendingQueues[0].name != "second" {
		t.Fatal("Expected 'second' to be pending")
	}

	// Execute first queue (should complete and promote second)
	manager.Update()

	// Verify first executed
	if len(executionOrder) == 0 || executionOrder[0] != "first" {
		t.Errorf("Expected 'first' to execute, got: %v", executionOrder)
	}

	// Verify second was promoted to active
	if manager.activeQueue == nil || manager.activeQueue.name != "second" {
		t.Errorf("Expected 'second' to be promoted to active, got: %v", manager.activeQueue)
	}

	// Execute second queue
	manager.Update()

	// Verify both executed in order
	if len(executionOrder) != 2 || executionOrder[1] != "second" {
		t.Errorf("Expected execution order [first, second], got: %v", executionOrder)
	}

	// Verify activeQueue cleared
	if manager.activeQueue != nil {
		t.Errorf("Expected activeQueue to be nil after all queues complete")
	}
}

// TestQueueCompletion_HighestPriorityPromoted tests that highest priority pending becomes active
func TestQueueCompletion_HighestPriorityPromoted(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var executionOrder []string

	// Create normal priority queue (becomes active)
	manager.Workflow("normal").
		Priority(PriorityNormal).
		Sequential().
		First(func() { executionOrder = append(executionOrder, "normal") }).
		Start()
	t.Logf("After normal: active=%s (priority %d), pending=%d",
		manager.activeQueue.name, manager.activeQueue.priority, len(manager.pendingQueues))

	// Create low priority queue (goes to pending)
	manager.Workflow("low").
		Priority(PriorityLow).
		Sequential().
		First(func() { executionOrder = append(executionOrder, "low") }).
		Start()
	t.Logf("After low: active=%s (priority %d), pending=%d",
		manager.activeQueue.name, manager.activeQueue.priority, len(manager.pendingQueues))

	// Create medium priority queue (goes to pending, but higher than low)
	manager.Workflow("medium").
		Priority(PriorityMedium).
		Sequential().
		First(func() { executionOrder = append(executionOrder, "medium") }).
		Start()
	t.Logf("After medium: active=%s (priority %d), pending=%d",
		manager.activeQueue.name, manager.activeQueue.priority, len(manager.pendingQueues))

	// Verify initial state - medium interrupted normal
	if manager.activeQueue.name != "medium" {
		t.Fatalf("Expected 'medium' to be active (interrupted normal), got: %s", manager.activeQueue.name)
	}
	if len(manager.pendingQueues) != 2 {
		t.Fatalf("Expected 2 pending queues, got: %d", len(manager.pendingQueues))
	}

	// Execute medium queue (highest priority)
	manager.Update()

	// Verify normal (higher priority than low) was promoted
	if manager.activeQueue == nil || manager.activeQueue.name != "normal" {
		t.Errorf("Expected 'normal' (priority 5) to be promoted over 'low' (priority 10), got: %v",
			manager.activeQueue)
	}

	// Execute normal queue
	manager.Update()

	// Verify low was promoted last
	if manager.activeQueue == nil || manager.activeQueue.name != "low" {
		t.Errorf("Expected 'low' to be promoted last, got: %v", manager.activeQueue)
	}

	// Execute low queue
	manager.Update()

	// Verify execution order: medium (interrupted), then normal, then low
	expected := []string{"medium", "normal", "low"}
	if len(executionOrder) != 3 {
		t.Fatalf("Expected 3 executions, got: %d (%v)", len(executionOrder), executionOrder)
	}
	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("Expected execution[%d] = %s, got: %s", i, exp, executionOrder[i])
		}
	}
}

// MockAnimationDriver for testing RunScript
type MockAnimationDriver struct {
	updateCount   int
	completeAfter int // Complete after this many updates
}

func (m *MockAnimationDriver) Update(deltaMs int) {
	m.updateCount++
}

func (m *MockAnimationDriver) IsComplete() bool {
	return m.updateCount >= m.completeAfter
}

// TestRunScript_RunsUntilComplete tests that RunScript updates driver until complete
func TestRunScript_RunsUntilComplete(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	driver := &MockAnimationDriver{completeAfter: 5}
	afterScriptRan := false

	// Act - Create queue with RunScript
	manager.Workflow("test_script").
		Priority(PriorityNormal).
		Sequential().
		First(func() {

		}).RunScript(

		driver).
		Then(func() {
			afterScriptRan = true
		}).
		Start()

	// Execute - should complete after 5 updates (~80ms at 16ms intervals)
	for i := 0; i < 20 && manager.HasActiveWorkflows(); i++ {
		manager.Update()
		mockTime += 16
	}

	// Assert
	if driver.updateCount < 5 {
		t.Errorf("Expected at least 5 updates, got %d", driver.updateCount)
	}

	if !afterScriptRan {
		t.Error("Expected action after RunScript to run")
	}

	if manager.HasActiveWorkflows() {
		t.Error("Expected queue to complete")
	}
}

// TestRunScript_CompletesImmediatelyIfAlreadyComplete tests edge case of already-complete driver
func TestRunScript_CompletesImmediatelyIfAlreadyComplete(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Driver that's already complete (0 updates needed)
	driver := &MockAnimationDriver{completeAfter: 0}
	afterScriptRan := false

	// Act
	manager.Workflow("test_immediate").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).RunScript(

		driver).
		Then(func() {
			afterScriptRan = true
		}).
		Start()

	// Execute - should complete quickly
	for i := 0; i < 5 && manager.HasActiveWorkflows(); i++ {
		manager.Update()
		mockTime += 16
	}

	// Assert
	if !afterScriptRan {
		t.Error("Expected action after RunScript to run")
	}

	if manager.HasActiveWorkflows() {
		t.Error("Expected queue to complete")
	}
}
