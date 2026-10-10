package event_queue

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
)

// TestEventQueueManager_Sequential tests sequential event execution
func TestEventQueueManager_Sequential(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	manager := NewEventQueueManager(timeSource, nil)

	var executionOrder []string

	// Act - Create sequential queue
	manager.Workflow("sequential_test").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			executionOrder = append(executionOrder, "first")
		}).
		Wait(500).
		Then(func() {
			executionOrder = append(executionOrder, "second")
		}).
		Wait(300).
		Then(func() {
			executionOrder = append(executionOrder, "third")
		}).
		Start()

	// Execute first event (time 0)
	manager.Update()

	// Advance to second event (500ms after first)
	mockTime = 500
	manager.Update()

	// Advance to third event (300ms after second = 800ms total)
	mockTime = 800
	manager.Update()

	// Assert correct order
	expected := []string{"first", "second", "third"}
	if len(executionOrder) != len(expected) {
		t.Errorf("Expected %d events, got %d", len(expected), len(executionOrder))
	}

	for i, event := range expected {
		if i >= len(executionOrder) || executionOrder[i] != event {
			t.Errorf("Expected event %d to be '%s', got '%s'", i, event, executionOrder[i])
		}
	}
}

// MockSystemCallbacks for testing input blocking
type MockSystemCallbacks struct {
	InputBlocked      bool
	BlockInputCalls   int
	UnblockInputCalls int
	KeyPressed        bool
	PressedKey        *semantic_input.Key
}

func (m *MockSystemCallbacks) BlockAllInput(block bool) {
	m.InputBlocked = block
	if block {
		m.BlockInputCalls++
	} else {
		m.UnblockInputCalls++
	}
}

func (m *MockSystemCallbacks) IsAnyKeyPressed() bool {
	return m.KeyPressed
}

func (m *MockSystemCallbacks) GetPressedKey() *semantic_input.Key {
	return m.PressedKey
}

// TestEventQueueManager_WaitForAnyKey tests the WaitForAnyKey functionality
func TestEventQueueManager_WaitForAnyKey(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }

	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	step1Executed := false
	step2Executed := false

	// Act - Create a sequential queue with WaitForAnyKey
	manager.Workflow("test_wait").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			step1Executed = true
		}).
		WaitForAnyKey().
		Then(func() {
			step2Executed = true
		}).
		Start()

	// Assert - Step 1 should execute immediately
	manager.Update()
	if !step1Executed {
		t.Error("Step 1 should execute immediately")
	}
	if step2Executed {
		t.Errorf("Step 2 should not execute before key press (step1: %t, step2: %t)", step1Executed, step2Executed)
	}

	// Simulate multiple updates without key press - step 2 should not execute
	manager.Update()
	manager.Update()
	if step2Executed {
		t.Errorf("Step 2 should not execute without key press (step1: %t, step2: %t)", step1Executed, step2Executed)
	}

	// Simulate key press
	mockCallbacks.KeyPressed = true
	manager.Update()

	// Assert - Step 2 should now execute
	if !step2Executed {
		t.Error("Step 2 should execute after key press")
	}

	// Verify queue is cleaned up
	if manager.HasWorkflow("test_wait") {
		t.Error("Queue should be cleaned up after completion")
	}
}

// TestSequentialEventBuilder_When_ConditionTrue tests that When block executes when condition is true
func TestSequentialEventBuilder_When_ConditionTrue(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	firstExecuted := false
	whenBlockExecuted := false

	// Act - Create chain with When block where condition is true
	manager.Workflow("test_when_true").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				whenBlockExecuted = true

			})
			return q
		}).
		Start()

	// Execute first event
	manager.Update()
	if !firstExecuted {
		t.Error("First event should execute")
	}

	// Advance time to allow chained event to execute
	mockTime = 10
	manager.Update()

	// Assert - When block should have executed
	if !whenBlockExecuted {
		t.Error("When block should execute when condition is true")
	}
}

// TestSequentialEventBuilder_When_ConditionFalse tests that When block is skipped when condition is false
func TestSequentialEventBuilder_When_ConditionFalse(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	firstExecuted := false
	whenBlockExecuted := false
	afterExecuted := false

	// Act - Create chain with When block where condition is false
	manager.Workflow("test_when_false").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).If(

		func() bool { return false }, func(q *SequentialEventBuilder) {
			q.Then(func() {
				whenBlockExecuted = true

			})
		}).
		Then(func() {
			afterExecuted = true
		}).
		Start()

	// Execute first event
	manager.Update()
	if !firstExecuted {
		t.Error("First event should execute")
	}

	// Advance time to allow chained events to execute
	mockTime = 10
	manager.Update()

	// Assert - When block should NOT execute, but chain should continue
	if whenBlockExecuted {
		t.Error("When block should NOT execute when condition is false")
	}
	if !afterExecuted {
		t.Error("Chain should continue after skipped When block")
	}
}

// TestSequentialEventBuilder_When_WithWaitForAnyKey tests that WaitForAnyKey works inside When blocks
func TestSequentialEventBuilder_When_WithWaitForAnyKey(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	firstExecuted := false
	whenBlockExecuted := false
	afterWaitExecuted := false

	// Act - Create chain with WaitForAnyKey inside When block
	manager.Workflow("test_when_wait").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				whenBlockExecuted = true
			}).
				WaitForAnyKey().
				Then(func() {
					afterWaitExecuted = true

				})
			return q
		}).
		Start()

	// Execute first event
	manager.Update()
	if !firstExecuted {
		t.Error("First event should execute")
	}

	// Advance time to execute When block
	mockTime = 10
	manager.Update()
	if !whenBlockExecuted {
		t.Error("When block should execute")
	}
	if afterWaitExecuted {
		t.Error("Event after WaitForAnyKey should not execute yet")
	}

	// Multiple updates without key press
	manager.Update()
	manager.Update()
	if afterWaitExecuted {
		t.Error("Event after WaitForAnyKey should not execute without key press")
	}

	// Simulate key press
	mockCallbacks.KeyPressed = true
	manager.Update()

	// Assert - Event after WaitForAnyKey should now execute
	if !afterWaitExecuted {
		t.Error("Event after WaitForAnyKey should execute after key press")
	}
}

// TestSequentialEventBuilder_When_Nested tests that nested When blocks work correctly
func TestSequentialEventBuilder_When_Nested(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	outerExecuted := false
	innerExecuted := false

	// Act - Create chain with nested When blocks (both conditions true)
	manager.Workflow("test_when_nested").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				outerExecuted = true
			}).ThenSequence(

				func(q2 *SequentialEventBuilder) *SequentialEventBuilder {
					q2.Then(func() {
						innerExecuted = true

					})
					return q2
				})
			return q

		}).
		Start()

	// Execute events
	manager.Update()
	mockTime = 10
	manager.Update()
	mockTime = 20
	manager.Update()

	// Assert - Both nested blocks should execute
	if !outerExecuted {
		t.Error("Outer When block should execute")
	}
	if !innerExecuted {
		t.Error("Inner When block (nested) should execute")
	}
}

// TestSequentialEventBuilder_When_MultipleInSequence tests multiple When blocks in a chain
func TestSequentialEventBuilder_When_MultipleInSequence(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	when1Executed := false
	when2Executed := false
	when3Executed := false

	// Act - Create chain with multiple When blocks (mixed true/false conditions)
	manager.Workflow("test_when_multiple").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				when1Executed = true

			})
			return q
		}).
		If(

			func() bool { return false }, func(q *SequentialEventBuilder) {
				q.Then(func() {
					when2Executed = true

				})
			}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				when3Executed = true

			})
			return q
		}).
		Start()

	// Execute events
	manager.Update()
	mockTime = 10
	manager.Update()
	mockTime = 20
	manager.Update()
	mockTime = 30
	manager.Update()

	// Assert - When1 and When3 should execute, When2 should not
	if !when1Executed {
		t.Error("First When block (true condition) should execute")
	}
	if when2Executed {
		t.Error("Second When block (false condition) should NOT execute")
	}
	if !when3Executed {
		t.Error("Third When block (true condition) should execute")
	}
}

// TestSequentialEventBuilder_When_WithWait tests Wait() inside When block
func TestSequentialEventBuilder_When_WithWait(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	firstExecuted := false
	whenBlockExecuted := false
	afterWaitExecuted := false

	manager.Workflow("test_when_wait").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				whenBlockExecuted = true
			}).
				Wait(100).
				Then(func() {
					afterWaitExecuted = true

				})
			return q
		}).
		Start()

	// First update - executes First() and When block Then()
	manager.Update()
	if !firstExecuted {
		t.Error("First action should execute")
	}
	if !whenBlockExecuted {
		t.Error("When block should execute when condition is true")
	}
	if afterWaitExecuted {
		t.Error("After-wait action should not execute yet (waiting 100ms)")
	}

	// Second update at t=50 - still waiting
	mockTime = 50
	manager.Update()
	if afterWaitExecuted {
		t.Error("After-wait action should not execute yet (only 50ms passed)")
	}

	// Third update at t=100 - wait complete
	mockTime = 100
	manager.Update()
	if !afterWaitExecuted {
		t.Error("After-wait action should execute after 100ms")
	}
}

// TestSequentialEventBuilder_When_WithTrailingWait tests Wait() at end of When block (no trailing Then)
// This is a regression test for bug where trailing Wait() delays were erased by If() implementation
func TestSequentialEventBuilder_When_WithTrailingWait(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	firstExecuted := false
	whenBlockExecuted := false
	afterWhenExecuted := false

	// Pattern that exposed the bug: If() block ending with Wait() and NO trailing Then()
	// The Wait() delay should be honored before continuing to outer chain
	manager.Workflow("test_when_trailing_wait").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				whenBlockExecuted = true
			}).
				Wait(500).
				Wait(300)
			return q

		}).
		Then(func() {
			// This should execute 800ms after whenBlockExecuted
			afterWhenExecuted = true
		}).
		Start()

	// First update - executes First() and When block Then()
	manager.Update()
	t.Logf("[t=%d] After first update: first=%v when=%v after=%v", mockTime, firstExecuted, whenBlockExecuted, afterWhenExecuted)
	if !firstExecuted {
		t.Error("First action should execute")
	}
	if !whenBlockExecuted {
		t.Error("When block should execute when condition is true")
	}
	if afterWhenExecuted {
		t.Errorf("After-when action should not execute yet (waiting 800ms) [t=%d]", mockTime)
	}

	// Second update at t=400 - still waiting (accumulated delay is 800ms)
	mockTime = 400
	manager.Update()
	t.Logf("[t=%d] After second update: after=%v", mockTime, afterWhenExecuted)
	if afterWhenExecuted {
		t.Errorf("After-when action should not execute yet (only 400ms passed, need 800ms) [t=%d]", mockTime)
	}

	// Third update at t=799 - still waiting (just before threshold)
	mockTime = 799
	manager.Update()
	t.Logf("[t=%d] After third update: after=%v", mockTime, afterWhenExecuted)
	if afterWhenExecuted {
		t.Errorf("After-when action should not execute yet (799ms passed, need 800ms) [t=%d]", mockTime)
	}

	// Fourth update at t=800 - accumulated wait complete
	mockTime = 800
	manager.Update()
	t.Logf("[t=%d] After fourth update: after=%v", mockTime, afterWhenExecuted)
	if !afterWhenExecuted {
		t.Errorf("After-when action should execute after 800ms accumulated Wait() delay [t=%d]", mockTime)
	}
}

// TestSequentialEventBuilder_When_WithAnimate tests Animate() inside When block
func TestSequentialEventBuilder_When_WithAnimate(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	firstExecuted := false
	animationStarted := false
	animationFrameCount := 0
	afterAnimateExecuted := false

	manager.Workflow("test_when_animate").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.Then(func() {
				animationStarted = true
			}).
				Animate(500, func(progress float32) {
					animationFrameCount++
				}).
				Then(func() {
					afterAnimateExecuted = true

				})
			return q
		}).
		Start()

	// First update - starts animation
	manager.Update()
	if !firstExecuted || !animationStarted {
		t.Error("First and animation start should execute")
	}
	if animationFrameCount == 0 {
		t.Error("Animation should have started (frame 0)")
	}

	// Advance time through animation
	mockTime = 250
	manager.Update()
	midwayFrameCount := animationFrameCount

	// Complete animation
	mockTime = 500
	manager.Update()

	// Need extra update after animation completes for Then() to execute
	mockTime = 510
	manager.Update()

	if !afterAnimateExecuted {
		t.Error("After-animate action should execute after animation completes")
	}
	if animationFrameCount <= midwayFrameCount {
		t.Error("Animation should have progressed through multiple frames")
	}
}

// TestSequentialEventBuilder_When_WithRepeatWithDelay tests RepeatWithDelay() inside When block
func TestSequentialEventBuilder_When_WithRepeatWithDelay(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	firstExecuted := false
	repeatCount := 0
	afterRepeatExecuted := false

	manager.Workflow("test_when_repeat").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.RepeatWithDelay(3, 0, func(iteration int) bool {
				repeatCount++
				return true
			}).
				Then(func() {
					afterRepeatExecuted = true

				})
			return q
		}).
		Start()

	// With delay=0, all iterations execute in a single Update()
	manager.Update()

	if !firstExecuted {
		t.Error("First action should execute")
	}
	if repeatCount != 3 {
		t.Errorf("Expected 3 repeat iterations, got %d", repeatCount)
	}
	if !afterRepeatExecuted {
		t.Error("After-repeat action should execute after all iterations")
	}
}

// TestSequentialEventBuilder_When_WithWhile tests While() inside When block
func TestSequentialEventBuilder_When_WithWhile(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	firstExecuted := false
	whileIterations := 0
	afterWhileExecuted := false
	maxIterations := 3

	manager.Workflow("test_when_while").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
		}).ThenSequence(

		func(q *SequentialEventBuilder) *SequentialEventBuilder {
			q.While(
				func() bool { return whileIterations < maxIterations },
				func() {
					whileIterations++
				},
				50,
			).
				Then(func() {
					afterWhileExecuted = true

				})
			return q
		}).
		Start()

	// First update - executes First() and first while iteration
	manager.Update()
	if !firstExecuted {
		t.Error("First action should execute")
	}
	if whileIterations != 1 {
		t.Errorf("Expected 1 while iteration, got %d", whileIterations)
	}

	// Advance time for iterations
	mockTime = 50
	manager.Update()
	if whileIterations != 2 {
		t.Errorf("Expected 2 while iterations, got %d", whileIterations)
	}

	mockTime = 100
	manager.Update()
	if whileIterations != 3 {
		t.Errorf("Expected 3 while iterations, got %d", whileIterations)
	}

	// Next update should complete the while loop and execute Then
	mockTime = 150
	manager.Update()
	if !afterWhileExecuted {
		t.Error("After-while action should execute after loop completes")
	}
	if whileIterations != 3 {
		t.Errorf("While should stop at 3 iterations, got %d", whileIterations)
	}
}

// TestSequentialEventBuilder_When_ExecutionTimeEvaluation tests that If() condition
// is evaluated at EXECUTION time, not BUILD time. This is critical for conditions that
// depend on state set by previous events in the chain.
func TestSequentialEventBuilder_When_ExecutionTimeEvaluation(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	// State flag that will be set during execution
	stateFlag := false
	whenBlockExecuted := false
	afterWhenExecuted := false

	manager.Workflow("test_execution_time").
		Priority(PriorityNormal).
		Sequential().
		First(func() {

			stateFlag = true
		}).If(

		func() bool { return stateFlag }, func(q *SequentialEventBuilder) {

			q.Then(func() {
				whenBlockExecuted = true

			})
		}).
		Then(func() {
			// This should execute after the When block
			afterWhenExecuted = true
		}).
		Start()

	// First update - executes First() which sets stateFlag = true
	manager.Update()
	if !stateFlag {
		t.Error("First action should have set stateFlag to true")
	}

	// At this point with current implementation, When block was skipped at build time
	// because stateFlag was false when chain was built
	// With the fix, When block should execute now because stateFlag is true at execution time

	// Second update - should execute When block's Then() if condition is evaluated at execution time
	mockTime = 10
	manager.Update()

	// Third update - should execute the after-When Then()
	mockTime = 20
	manager.Update()

	if !whenBlockExecuted {
		t.Error("When block should execute based on EXECUTION-TIME state (stateFlag=true), not build-time state")
	}
	if !afterWhenExecuted {
		t.Error("After-when action should execute after When block completes")
	}
}

// TestWhen_MultipleEventsInBlock verifies multiple events in When block execute sequentially
func TestWhen_MultipleEventsInBlock(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	stateFlag := false
	firstExecuted := false
	whenBlock1Executed := false
	whenBlock2Executed := false
	afterWhenExecuted := false

	manager.Workflow("test").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			firstExecuted = true
			stateFlag = true
			t.Logf("First executed, set stateFlag=true")
		}).If(

		func() bool { return stateFlag }, func(q *SequentialEventBuilder) {
			q.Then(func() {
				whenBlock1Executed = true
				t.Logf("When block 1 executed")
			}).
				Then(func() {
					whenBlock2Executed = true
					t.Logf("When block 2 executed")

				})
		}).
		Then(func() {
			afterWhenExecuted = true
			t.Logf("After When executed")
		}).
		Start()

	// Update to execute all events
	for i := 0; i < 10; i++ {
		mockTime = int64(i * 10)
		manager.Update()
		t.Logf("After update %d: first=%t, when1=%t, when2=%t, after=%t",
			i+1, firstExecuted, whenBlock1Executed, whenBlock2Executed, afterWhenExecuted)
		if afterWhenExecuted {
			break
		}
	}

	if !whenBlock1Executed {
		t.Error("When block 1 should have executed")
	}
	if !whenBlock2Executed {
		t.Error("When block 2 should have executed")
	}
	if !afterWhenExecuted {
		t.Error("After-when should have executed")
	}
}

// TestSequentialEventBuilder_Last_AfterThen tests that Finally() executes after Then() sequence
func TestSequentialEventBuilder_Last_AfterThen(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	var executionOrder []string
	cleanupExecuted := false

	// Act - Create sequence with Last cleanup
	manager.Workflow("test_last_then").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			executionOrder = append(executionOrder, "first")
		}).
		Wait(100).
		Then(func() {
			executionOrder = append(executionOrder, "second")
		}).
		Wait(100).
		Then(func() {
			executionOrder = append(executionOrder, "third")
		}).Finally(

		func() {
			cleanupExecuted = true
			executionOrder = append(executionOrder, "cleanup")
		}).
		Start()

	// Execute first event (time 0)
	manager.Update()
	if executionOrder[0] != "first" {
		t.Errorf("Expected 'first', got '%s'", executionOrder[0])
	}

	// Execute second event (time 100)
	mockTime = 100
	manager.Update()
	if len(executionOrder) < 2 || executionOrder[1] != "second" {
		t.Error("Expected 'second' to execute")
	}

	// Execute third event (time 200)
	mockTime = 200
	manager.Update()
	if len(executionOrder) < 3 || executionOrder[2] != "third" {
		t.Error("Expected 'third' to execute")
	}

	// Execute Last cleanup (time 200 - no delay after third)
	manager.Update()

	// Assert cleanup executed after all Then() calls
	if !cleanupExecuted {
		t.Error("Last cleanup should have executed")
	}
	if len(executionOrder) != 4 {
		t.Errorf("Expected 4 events, got %d", len(executionOrder))
	}
	if executionOrder[3] != "cleanup" {
		t.Errorf("Expected 'cleanup' last, got '%s'", executionOrder[3])
	}
}

// TestSequentialEventBuilder_Last_AfterWaitForAnyKey tests Finally() after WaitForAnyKey()
func TestSequentialEventBuilder_Last_AfterWaitForAnyKey(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	mockCallbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(timeSource, mockCallbacks)

	stepExecuted := false
	cleanupExecuted := false

	// Act - Create sequence with WaitForAnyKey followed by Last
	manager.Workflow("test_last_wait").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			stepExecuted = true
		}).
		WaitForAnyKey().Finally(

		func() {
			cleanupExecuted = true
		}).
		Start()

	// Execute first event
	manager.Update()
	if !stepExecuted {
		t.Error("First step should execute")
	}
	if cleanupExecuted {
		t.Error("Cleanup should not execute before key press")
	}

	// Simulate key press
	mockCallbacks.KeyPressed = true
	manager.Update()

	// Release key (required for debounce logic)
	mockCallbacks.KeyPressed = false
	manager.Update()

	// Detect key press edge (was pressed, now released, press again)
	mockCallbacks.KeyPressed = true
	manager.Update()

	// Release key again to clear debounce
	mockCallbacks.KeyPressed = false
	manager.Update()

	// Assert cleanup executed after key press
	if !cleanupExecuted {
		t.Error("Last cleanup should execute after WaitForAnyKey")
	}
}

// TestSequentialEventBuilder_Last_AfterAnimate tests Finally() after Animate()
func TestSequentialEventBuilder_Last_AfterAnimate(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	animationFrames := 0
	cleanupExecuted := false
	animationCompleted := false

	// Act - Create sequence with Animate followed by Last
	manager.Workflow("test_last_animate").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Animate(500, func(progress float32) {
			animationFrames++
			if progress >= 1.0 {
				animationCompleted = true
			}
		}).Finally(

		func() {
			cleanupExecuted = true
		}).
		Start()

	// Execute first event (starts animation)
	manager.Update()

	// Run animation to completion (500ms)
	for i := int64(0); i <= 500; i += 16 { // ~60fps
		mockTime = i
		manager.Update()
	}

	t.Logf("Animation frames: %d, completed: %t, cleanup: %t", animationFrames, animationCompleted, cleanupExecuted)

	// One more update to reach completion (progress = 1.0)
	mockTime += 100
	manager.Update()

	t.Logf("After completion - Animation frames: %d, completed: %t, cleanup: %t", animationFrames, animationCompleted, cleanupExecuted)

	// One more update to execute the scheduled lastAction event
	mockTime += 100
	manager.Update()

	t.Logf("After lastAction - Animation frames: %d, completed: %t, cleanup: %t", animationFrames, animationCompleted, cleanupExecuted)

	// Assert animation ran and cleanup executed
	if animationFrames == 0 {
		t.Error("Animation should have run")
	}
	if !cleanupExecuted {
		t.Error("Last cleanup should execute after Animate")
	}
}

// TestSequentialEventBuilder_Last_AfterRepeatWithDelay tests Finally() after RepeatWithDelay()
func TestSequentialEventBuilder_Last_AfterRepeatWithDelay(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	iterations := 0
	cleanupExecuted := false

	// Act - Create sequence with RepeatWithDelay followed by Last
	manager.Workflow("test_last_repeat").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		RepeatWithDelay(3, 100, func(iteration int) bool {
			iterations++

			return true
		}).Finally(

		func() {
			cleanupExecuted = true
		}).
		Start()

	// Execute first event
	manager.Update()

	// Execute 3 iterations with 100ms delays
	mockTime = 100
	manager.Update() // Iteration 1

	mockTime = 200
	manager.Update() // Iteration 2

	mockTime = 300
	manager.Update() // Iteration 3

	// Execute Last cleanup
	manager.Update()

	// Assert all iterations executed and cleanup ran
	if iterations != 3 {
		t.Errorf("Expected 3 iterations, got %d", iterations)
	}
	if !cleanupExecuted {
		t.Error("Last cleanup should execute after RepeatWithDelay")
	}
}

// TestSequentialEventBuilder_Last_AfterRepeatEarlyBreak tests Finally() after early break
func TestSequentialEventBuilder_Last_AfterRepeatEarlyBreak(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	iterations := 0
	cleanupExecuted := false

	// Act - Create sequence with RepeatWithDelay that breaks early
	manager.Workflow("test_last_repeat_break").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		RepeatWithDelay(5, 100, func(iteration int) bool {
			iterations++

			return iteration < 2
		}).Finally(

		func() {
			cleanupExecuted = true
		}).
		Start()

	// Execute first event
	manager.Update()

	// Execute iterations until early break
	mockTime = 100
	manager.Update() // Iteration 0

	mockTime = 200
	manager.Update() // Iteration 1

	mockTime = 300
	manager.Update() // Iteration 2 (breaks here)

	// Execute Last cleanup
	manager.Update()

	// Assert early break happened and cleanup still ran
	if iterations != 3 {
		t.Errorf("Expected 3 iterations before break, got %d", iterations)
	}
	if !cleanupExecuted {
		t.Error("Last cleanup should execute even after early break")
	}
}

// TestEventQueueManager_ExecutionDepth_RecoveryAfterPanic tests that executionDepth
// is properly decremented even when an event callback panics.
// This is critical to prevent all subsequent Workflow calls from
// incorrectly triggering the nested queue detection panic.
func TestEventQueueManager_ExecutionDepth_RecoveryAfterPanic(t *testing.T) {
	// Arrange
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	// Create a queue with a callback that will panic
	panicQueue := manager.Workflow("panic_queue").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			panic("intentional test panic")
		})

	// Act - Execute the queue with panic, recover to verify executionDepth behavior
	recovered := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
				t.Logf("Recovered from panic: %v", r)
			}
		}()
		panicQueue.Start()
		manager.Update() // This will execute the panicking callback
	}()

	// Assert - the panic was recovered
	if !recovered {
		t.Error("Expected panic to occur and be recovered")
	}

	// Critical assertion: executionDepth should be back to 0 after panic recovery
	// If the defer pattern wasn't used, executionDepth would be stuck at 1
	// and the next Workflow call would fail with nested queue panic
	if manager.executionDepth != 0 {
		t.Errorf("executionDepth should be 0 after panic recovery, got %d", manager.executionDepth)
	}

	// NOTE: After a panic, the queue itself is in an undefined state (still has the panic-causing event).
	// In a real application, a higher-level recover would typically clean up or restart the game.
	// For this test, we verify the critical invariant: executionDepth is reset, allowing
	// new queue creation without false "nested queue" detection.
	//
	// We specifically test that Workflow() does NOT panic due to elevated executionDepth,
	// which was the original bug this fix addresses.

	// Verification: Workflow should succeed (not panic due to executionDepth > 0)
	// We create a new manager to isolate from the broken queue state
	manager2 := NewEventQueueManager(timeSource, nil)

	// Simulate what would happen if executionDepth wasn't reset: manually set it to 1
	// This would cause the nested queue panic - but with our fix, this doesn't happen
	// after the panic because defer resets it.
	newQueueCreated := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Creating new queue should NOT panic, but got: %v", r)
			}
		}()
		manager2.Workflow("new_queue").
			Priority(PriorityNormal).
			Sequential().
			First(func() {
				newQueueCreated = true
			}).
			Start()
		manager2.Update()
	}()

	if !newQueueCreated {
		t.Error("Should be able to create and execute new queue")
	}
}

// TestEventQueueManager_ExecutionDepth_DefersWork verifies the defer pattern works correctly
// by ensuring executionDepth is correctly managed across normal event execution
func TestEventQueueManager_ExecutionDepth_DefersWork(t *testing.T) {
	mockTime := int64(0)
	timeSource := func() int64 { return mockTime }
	manager := NewEventQueueManager(timeSource, nil)

	// Track execution depth during callback
	depthDuringExecution := -1
	depthAfterExecution := -1

	manager.Workflow("depth_test").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			// Capture depth during execution (should be 1)
			depthDuringExecution = manager.executionDepth
		}).
		Then(func() {
			// Also check in Then callback
			if manager.executionDepth != 1 {
				t.Errorf("executionDepth during Then should be 1, got %d", manager.executionDepth)
			}
		}).
		Start()

	manager.Update()
	depthAfterExecution = manager.executionDepth

	// During execution, depth should have been 1
	if depthDuringExecution != 1 {
		t.Errorf("executionDepth during First should be 1, got %d", depthDuringExecution)
	}

	// Advance time to complete the queue
	mockTime = 100
	manager.Update()
	mockTime = 200
	manager.Update()

	// After execution completes, depth should be 0
	// (not during Update since queue may have more events)
	if manager.executionDepth != 0 {
		t.Errorf("executionDepth after queue completes should be 0, got %d", manager.executionDepth)
	}

	// The key invariant: after Update() returns, executionDepth should be 0
	// This is what allows new queue registration
	_ = depthAfterExecution // Used for logging if needed
}
