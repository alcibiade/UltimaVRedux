package event_queue

import (
	"fmt"
	"testing"
)

// TestChainModeBasicChaining tests the basic First->Then chaining in chain mode
func TestChainModeBasicChaining(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	// Mock system callbacks for debug output
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var executionOrder []string

	// Act - Create a simple chain: First -> Then
	manager.Workflow("chain_test").
		Priority(PriorityNormal).Sequential().
		First(func() {
			executionOrder = append(executionOrder, "first")
		}).
		Then(func() {
			t.Logf("SECOND ACTION EXECUTING")
			executionOrder = append(executionOrder, "second")
			t.Logf("SECOND ACTION COMPLETED, execution order now: %v", executionOrder)
		}).
		Start()

	// Debug: Check what events are in the queue
	queue := manager.queues["chain_test"]
	t.Logf("Events in queue after setup: %d", len(queue.events))
	for i, event := range queue.events {
		hasNext := event.NextAction != nil
		t.Logf("Event %d: ExecuteAt=%d, HasNextAction=%t, NextDelay=%d", i, event.ExecuteAt, hasNext, event.NextDelay)
	}

	// Execute first event
	t.Logf("About to call manager.Update() for first time")
	manager.Update()
	t.Logf("manager.Update() completed")
	t.Logf("After first update - execution order: %v", executionOrder)
	t.Logf("Events remaining in queue: %d", len(queue.events))

	// Debug: Check what events are in the queue after first update
	for i, event := range queue.events {
		hasNext := event.NextAction != nil
		t.Logf("Event %d after first update: ExecuteAt=%d, HasNextAction=%t, NextDelay=%d", i, event.ExecuteAt, hasNext, event.NextDelay)
	}

	// Advance time and execute again
	mockTime = 100
	manager.Update()
	t.Logf("After second update - execution order: %v", executionOrder)
	t.Logf("Events remaining in queue: %d", len(queue.events))

	// Verify execution
	if len(executionOrder) == 0 {
		t.Error("No events executed - chain mode may not be working")
	} else if len(executionOrder) == 1 && executionOrder[0] == "first" {
		t.Error("Only first event executed - Then() chaining is not working")
	} else if len(executionOrder) == 2 && executionOrder[0] == "first" && executionOrder[1] == "second" {
		t.Log("SUCCESS: Both events executed in correct order")
	} else {
		t.Errorf("Unexpected execution order: %v", executionOrder)
	}
}

// TestChainModeRepeatWithDelayChaining tests RepeatWithDelay -> Then chaining
func TestChainModeRepeatWithDelayChaining(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	// Mock system callbacks for debug output
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var executionOrder []string

	// Act - Create a chain: First -> RepeatWithDelay -> Then
	manager.Workflow("repeat_test").
		Priority(PriorityNormal).Sequential().
		First(func() {
			executionOrder = append(executionOrder, "first")
			t.Logf("FIRST ACTION EXECUTED")
		}).
		RepeatWithDelay(2, 0, func(iteration int) bool {
			msg := fmt.Sprintf("repeat_%d", iteration)
			executionOrder = append(executionOrder, msg)
			t.Logf("REPEAT ITERATION %d EXECUTED", iteration)
			return true
		}).
		Then(func() {
			executionOrder = append(executionOrder, "after_repeat")
			t.Logf("AFTER REPEAT ACTION EXECUTED")
		}).
		Start()

	// Debug: Check what events are in the queue after setup
	queue := manager.queues["repeat_test"]
	t.Logf("Events in queue after setup: %d", len(queue.events))
	for i, event := range queue.events {
		hasNext := event.NextAction != nil
		t.Logf("Event %d: ExecuteAt=%d, HasNextAction=%t, NextDelay=%d", i, event.ExecuteAt, hasNext, event.NextDelay)
	}

	// Execute events
	t.Logf("About to call manager.Update()")
	manager.Update()
	t.Logf("manager.Update() completed")
	t.Logf("Execution order: %v", executionOrder)

	// Debug: Check remaining events
	t.Logf("Events remaining after update: %d", len(queue.events))

	// Verify execution
	expected := []string{"first", "repeat_0", "repeat_1", "after_repeat"}
	if len(executionOrder) != len(expected) {
		t.Errorf("Expected %d events, got %d: %v", len(expected), len(executionOrder), executionOrder)
		return
	}

	for i, exp := range expected {
		if i >= len(executionOrder) || executionOrder[i] != exp {
			t.Errorf("Expected event %d to be '%s', got '%s'. Full order: %v", i, exp, executionOrder[i], executionOrder)
			return
		}
	}

	t.Log("SUCCESS: RepeatWithDelay -> Then chaining works correctly")
}

// TestChainModeWhileLoop tests While loop functionality with chaining
func TestChainModeWhileLoop(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var executionOrder []string
	var counter int

	// Act - Create a chain: First -> While (3 iterations) -> Then
	manager.Workflow("while_test").
		Priority(PriorityNormal).Sequential().
		First(func() {
			executionOrder = append(executionOrder, "first")
			counter = 0
			t.Logf("FIRST ACTION EXECUTED")
		}).
		While(func() bool {
			shouldContinue := counter < 3
			t.Logf("WHILE CONDITION CHECK: counter=%d, shouldContinue=%t", counter, shouldContinue)
			return shouldContinue
		}, func() {
			msg := fmt.Sprintf("while_%d", counter)
			executionOrder = append(executionOrder, msg)
			counter++
			t.Logf("WHILE ITERATION %d EXECUTED", counter-1)
		}, 0).
		Then(func() {
			executionOrder = append(executionOrder, "after_while")
			t.Logf("AFTER WHILE ACTION EXECUTED")
		}).
		Start()

	// Debug: Check what events are in the queue after setup
	queue := manager.queues["while_test"]
	t.Logf("Events in queue after setup: %d", len(queue.events))

	// Execute events with advancement
	for i := 0; i < 10; i++ { // Maximum 10 iterations to prevent infinite loops
		t.Logf("Update iteration %d", i)
		manager.Update()
		t.Logf("Execution order after update %d: %v", i, executionOrder)

		if len(queue.events) == 0 {
			t.Logf("No more events in queue, stopping")
			break
		}

		mockTime += 10 // Advance time for next iteration
	}

	// Verify execution
	expected := []string{"first", "while_0", "while_1", "while_2", "after_while"}
	if len(executionOrder) != len(expected) {
		t.Errorf("Expected %d events, got %d: %v", len(expected), len(executionOrder), executionOrder)
		return
	}

	for i, exp := range expected {
		if i >= len(executionOrder) || executionOrder[i] != exp {
			t.Errorf("Expected event %d to be '%s', got '%s'. Full order: %v", i, exp, executionOrder[i], executionOrder)
			return
		}
	}

	t.Log("SUCCESS: While loop chaining works correctly")
}

// TestChainModeWaitForAnyKey tests WaitForAnyKey functionality with chaining
func TestChainModeWaitForAnyKey(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var executionOrder []string

	// Act - Create a chain: First -> WaitForAnyKey -> Then
	manager.Workflow("waitkey_test").
		Priority(PriorityNormal).Sequential().BlockInput().
		First(func() {
			executionOrder = append(executionOrder, "first")
			t.Logf("FIRST ACTION EXECUTED")
		}).
		WaitForAnyKey().
		Then(func() {
			executionOrder = append(executionOrder, "after_wait")
			t.Logf("AFTER WAIT ACTION EXECUTED")
		}).
		Start()

	// Debug: Check initial queue state
	queue := manager.queues["waitkey_test"]
	t.Logf("Events in queue after setup: %d", len(queue.events))

	// Execute first round (first action and start waiting for key)
	t.Logf("First update - should execute 'first' and start waiting for key")
	manager.Update()
	t.Logf("After first update - execution order: %v", executionOrder)
	t.Logf("Events remaining: %d", len(queue.events))

	// Verify we're waiting (should have executed 'first' but not 'after_wait')
	if len(executionOrder) != 1 || executionOrder[0] != "first" {
		t.Errorf("Expected only 'first' to execute, got: %v", executionOrder)
	}

	// Multiple updates while no key is pressed (should remain waiting)
	for i := 0; i < 3; i++ {
		mockTime += 10
		manager.Update()
		t.Logf("Update %d while waiting - execution order: %v", i+2, executionOrder)

		// Should still be waiting
		if len(executionOrder) != 1 {
			t.Errorf("Should still be waiting, but execution order changed: %v", executionOrder)
		}
	}

	// Now simulate key press
	mockCallbacks.KeyPressed = true
	t.Logf("Key pressed - next update should complete the chain")

	// Execute with key pressed (should complete the chain)
	manager.Update()
	t.Logf("After key press update - execution order: %v", executionOrder)
	t.Logf("Events remaining: %d", len(queue.events))

	// Verify execution
	expected := []string{"first", "after_wait"}
	if len(executionOrder) != len(expected) {
		t.Errorf("Expected %d events, got %d: %v", len(expected), len(executionOrder), executionOrder)
		return
	}

	for i, exp := range expected {
		if i >= len(executionOrder) || executionOrder[i] != exp {
			t.Errorf("Expected event %d to be '%s', got '%s'. Full order: %v", i, exp, executionOrder[i], executionOrder)
			return
		}
	}

	t.Log("SUCCESS: WaitForAnyKey chaining works correctly")
}
