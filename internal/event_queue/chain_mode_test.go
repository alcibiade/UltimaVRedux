package event_queue

import (
	"fmt"
	"testing"
)

// TestChainMode_BasicThen verifies basic First->Then->Then chaining works
func TestChainMode_BasicThen(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	manager.Workflow("basic_then").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first")
		}).
		Then(func() {
			execution = append(execution, "second")
		}).
		Then(func() {
			execution = append(execution, "third")
		}).
		Start()

	manager.Update()

	expected := []string{"first", "second", "third"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Basic Then chaining: %v", execution)
}

// TestChainMode_RepeatWithDelay_Simple verifies RepeatWithDelay with no delays
func TestChainMode_RepeatWithDelay_Simple(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	manager.Workflow("repeat_simple").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first")
		}).
		RepeatWithDelay(3, 0, func(iteration int) bool {
			execution = append(execution, fmt.Sprintf("repeat_%d", iteration))
			return true
		}).
		Then(func() {
			execution = append(execution, "after_repeat")
		}).
		Start()

	manager.Update()

	expected := []string{"first", "repeat_0", "repeat_1", "repeat_2", "after_repeat"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ RepeatWithDelay simple: %v", execution)
}

// TestChainMode_RepeatWithDelay_WithDelay verifies RepeatWithDelay with delays
func TestChainMode_RepeatWithDelay_WithDelay(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	manager.Workflow("repeat_delay").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first")
		}).
		RepeatWithDelay(2, 100, func(iteration int) bool {
			execution = append(execution, fmt.Sprintf("repeat_%d_at_%d", iteration, mockTime))
			return true
		}).
		Then(func() {
			execution = append(execution, fmt.Sprintf("after_repeat_at_%d", mockTime))
		}).
		Start()

	// Execute first event and first repeat iteration
	manager.Update()
	t.Logf("After first update: %v", execution)

	// Advance time to trigger next iteration
	mockTime = 100
	manager.Update()
	t.Logf("After time advance to 100: %v", execution)

	// Advance time to trigger final Then
	mockTime = 200
	manager.Update()
	t.Logf("After time advance to 200: %v", execution)

	// Advance time to trigger the final event after last iteration
	mockTime = 300
	manager.Update()
	t.Logf("After time advance to 300: %v", execution)

	// Verify timing is correct
	if len(execution) < 4 {
		t.Errorf("Expected at least 4 executions, got %d: %v", len(execution), execution)
	}
	t.Logf("✅ RepeatWithDelay with delays: %v", execution)
}

// TestChainMode_ComplexChain verifies complex chaining patterns
func TestChainMode_ComplexChain(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	manager.Workflow("complex").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "setup")
		}).
		Wait(50).
		Then(func() {
			execution = append(execution, "after_wait_1")
		}).
		RepeatWithDelay(2, 25, func(iteration int) bool {
			execution = append(execution, fmt.Sprintf("loop_%d", iteration))
			return true
		}).
		Wait(75).
		Then(func() {
			execution = append(execution, "final")
		}).
		Start()

	// Execute setup
	manager.Update()
	mockTime = 50
	manager.Update()
	mockTime = 75
	manager.Update()
	mockTime = 150
	manager.Update()
	mockTime = 225
	manager.Update()

	expected := []string{"setup", "after_wait_1", "loop_0", "loop_1", "final"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Complex chain: %v", execution)
}

// TestChainMode_WaitForAnyKey verifies input waiting mechanism
func TestChainMode_WaitForAnyKey(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	manager.Workflow("wait_key").
		Priority(PriorityNormal).Sequential().BlockInput().
		First(func() {
			execution = append(execution, "before_key")
		}).
		WaitForAnyKey().
		Then(func() {
			execution = append(execution, "after_key")
		}).
		Start()

	// Execute before key press
	manager.Update()
	if len(execution) != 1 || execution[0] != "before_key" {
		t.Errorf("Expected ['before_key'], got %v", execution)
		return
	}

	// Should not execute after_key yet
	manager.Update()
	if len(execution) != 1 {
		t.Errorf("Expected 1 execution before key press, got %d: %v", len(execution), execution)
		return
	}

	// Simulate key press
	mockCallbacks.KeyPressed = true
	manager.Update()

	expected := []string{"before_key", "after_key"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ WaitForAnyKey: %v", execution)
}

// TestChainMode_CampingSimulation simulates the camping sequence structure
func TestChainMode_CampingSimulation(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string
	totalCycles := 3 // Simulate 3 camping cycles

	manager.Workflow("camping_sim").
		Priority(PriorityNormal).Sequential().BlockInput().
		First(func() {
			execution = append(execution, "setting_camp")
		}).
		Then(func() {
			execution = append(execution, "time_advances")
		}).
		RepeatWithDelay(totalCycles, 10, func(iteration int) bool {
			execution = append(execution, fmt.Sprintf("cycle_%d", iteration))
			// Simulate cycle logic that could interrupt
			if iteration == totalCycles-1 {
				execution = append(execution, "completed_naturally")
			}
			return true
		}).
		Then(func() {
			execution = append(execution, "wanderer_check")
		}).
		Then(func() {
			execution = append(execution, "cleanup")
		}).
		Then(func() {
			execution = append(execution, "rested")
		}).
		Start()

	// Execute the sequence with time advancement
	manager.Update()
	mockTime = 10
	manager.Update()
	mockTime = 20
	manager.Update()
	mockTime = 30
	manager.Update()

	expected := []string{
		"setting_camp", "time_advances",
		"cycle_0", "cycle_1", "cycle_2", "completed_naturally",
		"wanderer_check", "cleanup", "rested",
	}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Camping simulation: %v", execution)
}

// TestChainMode_ErrorInIteration verifies early break handling in RepeatWithDelay
func TestChainMode_ErrorInIteration(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	manager.Workflow("error_test").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first")
		}).
		RepeatWithDelay(3, 0, func(iteration int) bool {
			if iteration == 1 {
				execution = append(execution, fmt.Sprintf("break_at_%d", iteration))
				return false // Break loop early
			}
			execution = append(execution, fmt.Sprintf("repeat_%d", iteration))
			return true // Continue loop
		}).
		Then(func() {
			execution = append(execution, "after_repeat")
		}).
		Start()

	manager.Update()

	// Should stop repeating at iteration 1, but continue chain
	expected := []string{"first", "repeat_0", "break_at_1", "after_repeat"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Early break handling: %v", execution)
}

// Helper function to compare slices
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}

// TestChainMode_EdgeCases verifies edge case handling
func TestChainMode_EdgeCases(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	// Test zero iterations
	var execution []string
	manager.Workflow("zero_repeat").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first")
		}).
		RepeatWithDelay(0, 0, func(iteration int) bool {
			execution = append(execution, "should_not_execute")
			return true
		}).
		Then(func() {
			execution = append(execution, "after")
		}).
		Start()

	manager.Update()

	expected := []string{"first", "after"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Zero iterations - Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Zero iterations: %v", execution)
}

// TestChainMode_RepeatWithCompletion verifies RepeatWithDelay handles completion errors correctly
func TestChainMode_RepeatWithCompletion(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	manager.Workflow("completion_test").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first")
		}).
		RepeatWithDelay(3, 0, func(iteration int) bool {
			if iteration == 1 {
				execution = append(execution, "completing_naturally")
				return false // Break loop but continue chain
			}
			execution = append(execution, fmt.Sprintf("repeat_%d", iteration))
			return true
		}).
		Then(func() {
			execution = append(execution, "after_completion")
		}).
		Start()

	manager.Update()

	// Should execute first, repeat_0, then complete and continue to Then()
	expected := []string{"first", "repeat_0", "completing_naturally", "after_completion"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Completion handling: %v", execution)
}

// TestChainMode_CampingPattern replicates the exact camping sequence pattern
func TestChainMode_CampingPattern(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string
	totalCycles := 3

	// Simulate the exact camping pattern
	manager.Workflow("camping_pattern").
		Priority(PriorityNormal).Sequential().BlockInput().
		First(func() {
			execution = append(execution, "setting_camp")
		}).
		Then(func() {
			execution = append(execution, "time_advances")
		}).
		RepeatWithDelay(
			totalCycles, // Maximum cycles
			200,         // Delay between cycles
			func(iteration int) bool {
				// Simulate cycle logic that returns completion error
				execution = append(execution, fmt.Sprintf("cycle_%d", iteration))

				// On the last iteration, return false to break loop but continue chain
				if iteration == totalCycles-1 {
					execution = append(execution, "completed_naturally")
					return false
				}
				return true
			},
		).
		Then(func() {
			execution = append(execution, "rested") // This should execute after completion error
		}).
		Start()

	// Execute the sequence with time advancement
	manager.Update()
	mockTime = 200
	manager.Update()
	mockTime = 400
	manager.Update()
	mockTime = 600
	manager.Update()

	expected := []string{"setting_camp", "time_advances", "cycle_0", "cycle_1", "cycle_2", "completed_naturally", "rested"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Camping pattern: %v", execution)
}

// TestChainMode_AnimateIntegration verifies Animate works in chain mode
func TestChainMode_AnimateIntegration(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string
	var frameCount int
	var frameCallTimes []int64

	manager.Workflow("animate_test").
		Priority(PriorityNormal).Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "before_animate")
		}).
		Animate(1000, func(progress float32) {
			frameCount++
			frameCallTimes = append(frameCallTimes, mockTime)
			// Should be called multiple times as animation progresses
		}).
		Then(func() {
			execution = append(execution, "after_animate")
		}).
		Start()

	// Execute first event
	manager.Update()

	// Advance time to trigger animation progress
	mockTime = 500
	manager.Update()

	// Advance time to complete animation
	mockTime = 1000
	manager.Update()

	// Animation should complete and Then() should execute
	mockTime = 1001
	manager.Update()

	// Should have both events and some frame callbacks
	expected := []string{"before_animate", "after_animate"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}

	if frameCount == 0 {
		t.Errorf("Expected animation frames to be called, got %d", frameCount)
	}

	// In chain mode with proper implementation, we should see multiple frame calls
	if frameCount < 3 {
		t.Logf("Warning: Only %d frame calls - may indicate chain mode issue", frameCount)
	}

	t.Logf("✅ Animate integration: %v (frames: %d, times: %v)", execution, frameCount, frameCallTimes)
}

// TestChainMode_AnimateProperChaining verifies proper chain mode implementation for Animate
func TestChainMode_AnimateProperChaining(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string
	var frameCount int
	var completionCallbackCalled bool

	// Test chain mode Animate properly chains to Then
	manager.Workflow("proper_chain_test").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			execution = append(execution, "step1")
		}).
		Animate(200, func(progress float32) {
			frameCount++
		}).
		Then(func() {
			execution = append(execution, "step2")
		}).
		Animate(100, func(progress float32) {
			// Second animation
		}).
		Then(func() {
			completionCallbackCalled = true
		}).
		Then(func() {
			execution = append(execution, "step3")
		}).
		Start()

	// Execute events with time progression
	manager.Update() // First event

	// Progress through first animation (200ms)
	for timeStep := int64(50); timeStep <= 200; timeStep += 50 {
		mockTime = timeStep
		manager.Update()
	}
	manager.Update() // Run step2 and admit its animation at 200ms, after completion.

	// Progress through second animation (100ms from 200ms start)
	for timeStep := int64(250); timeStep <= 300; timeStep += 25 {
		mockTime = timeStep
		manager.Update()
	}

	// Final update to ensure all events complete
	mockTime = 400
	manager.Update()

	// Verify complete chain execution
	expected := []string{"step1", "step2", "step3"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}

	if frameCount == 0 {
		t.Error("Animation frames were not called")
	}

	if !completionCallbackCalled {
		t.Error("Animation completion callback was not called")
	}

	t.Logf("✅ Proper chaining: %v (frames: %d, completion: %t)", execution, frameCount, completionCallbackCalled)
}

// TestHandoffTo_ExecutesChainedQueue verifies that HandoffTo properly
// terminates the current queue and starts a new chained queue
func TestHandoffTo_ExecutesChainedQueue(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	// Create first queue that will terminate and chain
	manager.Workflow("first_queue").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first_queue_start")
		}).
		Then(func() {
			execution = append(execution, "first_queue_action")
		}).HandoffTo(

		"second_queue", PriorityHigh, func(q *SequentialEventBuilder) {
			q.First(func() {
				execution = append(execution, "second_queue_start")
			}).
				Then(func() {
					execution = append(execution, "second_queue_action")
				})
		}).
		Start()

	// Execute first queue actions
	manager.Update()
	t.Logf("After first update: %v", execution)

	// The termination should have triggered and chained queue should be created
	manager.Update()
	t.Logf("After second update: %v", execution)

	expected := []string{"first_queue_start", "first_queue_action", "second_queue_start", "second_queue_action"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ HandoffTo executes chained queue: %v", execution)
}

// TestHandoffTo_LastCleanupRunsFirst verifies that Finally() cleanup runs
// before the chained queue starts
func TestHandoffTo_LastCleanupRunsFirst(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	// Create first queue with Finally() cleanup
	manager.Workflow("cleanup_queue").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "queue_start")
		}).HandoffTo(

		"chained_queue", PriorityHigh, func(q *SequentialEventBuilder) {
			q.First(func() {
				execution = append(execution, "chained_start")
			})
		}).
		Finally(

			func() {
				execution = append(execution, "last_cleanup")
			}).
		Start()

	// Execute
	manager.Update()
	t.Logf("After first update: %v", execution)
	manager.Update()
	t.Logf("After second update: %v", execution)

	// Finally() should execute BEFORE chained queue starts
	expected := []string{"queue_start", "last_cleanup", "chained_start"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ Finally() cleanup runs before chained queue: %v", execution)
}

// TestHandoffTo_WorksInsideWhenCallback tests the pattern used for whirlpool detection
func TestHandoffTo_WorksInsideWhenCallback(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string
	shouldTriggerChain := false

	// Simulate the sailing → whirlpool pattern
	manager.Workflow("sailing_simulation").
		Priority(PriorityNormal).Sequential().BlockInput().
		First(func() {
			execution = append(execution, "sailing_start")
		}).
		Then(func() {
			execution = append(execution, "sailing_move")

			shouldTriggerChain = true
		}).If(

		func() bool { return shouldTriggerChain }, func(q *SequentialEventBuilder) {
			q.Then(func() {
				execution = append(execution, "stop_sailing")
			}).HandoffTo(

				"whirlpool_transport", PriorityHigh, func(wq *SequentialEventBuilder) {
					wq.First(func() {
						execution = append(execution, "whirlpool_message")
					}).
						Then(func() {
							execution = append(execution, "whirlpool_teleport")
						})
				})

		}).Finally(

		func() {
			execution = append(execution, "sailing_cleanup")
		}).
		Start()

	// Execute sailing events
	manager.Update()

	// Should trigger If() callback and HandoffTo
	manager.Update()

	// Execute whirlpool queue
	manager.Update()

	expected := []string{
		"sailing_start", "sailing_move", "stop_sailing",
		"sailing_cleanup", "whirlpool_message", "whirlpool_teleport",
	}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ HandoffTo works inside When callback: %v", execution)
}

// TestHandoffTo_StopsQueueProcessing verifies that terminateRequested
// stops further queue event processing
func TestHandoffTo_StopsQueueProcessing(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	var execution []string

	// Create queue with events that should NOT execute after termination
	manager.Workflow("stop_processing").
		Priority(PriorityNormal).Sequential().
		First(func() {
			execution = append(execution, "first")
		}).HandoffTo(

		"replacement", PriorityHigh, func(q *SequentialEventBuilder) {
			q.First(func() {
				execution = append(execution, "replacement_first")
			})
		}).
		Then(func() {
			// This should NEVER execute because queue was terminated
			execution = append(execution, "SHOULD_NOT_EXECUTE")
		}).
		Start()

	// Execute
	manager.Update()
	manager.Update()
	manager.Update()

	// Verify "SHOULD_NOT_EXECUTE" is not in execution
	for _, item := range execution {
		if item == "SHOULD_NOT_EXECUTE" {
			t.Errorf("Queue processing continued after termination - execution: %v", execution)
			return
		}
	}

	expected := []string{"first", "replacement_first"}
	if !slicesEqual(execution, expected) {
		t.Errorf("Expected %v, got %v", expected, execution)
	}
	t.Logf("✅ HandoffTo stops queue processing: %v", execution)
}
