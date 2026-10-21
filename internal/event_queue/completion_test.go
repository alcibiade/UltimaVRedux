package event_queue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOnCompleteWaitsForInputReleaseBeforeRestartingSameName(t *testing.T) {
	callbacks := &MockSystemCallbacks{KeyPressed: true}
	manager := NewEventQueueManager(func() int64 { return 0 }, callbacks)
	var execution []string
	pending := false
	successorCalls := 0
	workflow := manager.Workflow("repeatable").Priority(PriorityNormal).Sequential().BlockInput()
	workflow.First(func() { execution = append(execution, "work") }).
		Finally(func() {
			execution = append(execution, "finally")
			require.True(t, callbacks.InputBlocked)
			require.True(t, workflow.queue.inputBlockOwned)
			require.True(t, manager.HasWorkflow("repeatable"))
		}).
		OnComplete(func() {
			require.Equal(t, []string{"work", "finally"}, execution)
			require.False(t, callbacks.InputBlocked)
			require.False(t, workflow.queue.inputBlockOwned)
			require.Zero(t, manager.inputBlockCount)
			require.Equal(t, 1, callbacks.UnblockInputCalls)
			require.False(t, manager.HasWorkflow("repeatable"))
			require.Equal(t, queueFinalized, workflow.queue.lifecycle)
			require.True(t, pending, "a request arrived while the old name was finalizing")
			execution = append(execution, "complete")
			pending = false
			manager.Workflow("repeatable").Priority(PriorityNormal).Sequential().
				First(func() { successorCalls++ }).Start()
			require.True(t, manager.HasWorkflow("repeatable"), "successor reserves the released name")
			require.Zero(t, successorCalls, "successor must not execute inside completion")
		}).Start()

	manager.Update()
	require.Equal(t, []string{"work", "finally"}, execution)
	require.Equal(t, queueFinalizing, workflow.queue.lifecycle)
	require.True(t, workflow.queue.pendingInputUnblock)
	pending = true
	for range 3 {
		manager.Update()
	}
	require.True(t, manager.HasWorkflow("repeatable"))
	require.True(t, callbacks.InputBlocked)
	require.Equal(t, []string{"work", "finally"}, execution)
	require.Zero(t, successorCalls)

	callbacks.KeyPressed = false
	manager.Update()
	require.Equal(t, []string{"work", "finally", "complete"}, execution)
	require.False(t, pending)
	require.True(t, manager.HasWorkflow("repeatable"))
	require.Zero(t, successorCalls)
	manager.Update()
	manager.Update()
	require.Equal(t, 1, successorCalls)
	require.False(t, manager.HasWorkflow("repeatable"))
	require.False(t, manager.HasActiveWorkflows())
	require.Equal(t, 1, callbacks.BlockInputCalls)
	require.Equal(t, 1, callbacks.UnblockInputCalls)
}

func TestOnCompleteRunsExactlyOnceWithoutInputOwnership(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	var execution []string
	workflow := manager.Workflow("once").Priority(PriorityNormal).Sequential()
	workflow.First(func() { execution = append(execution, "work") }).
		Finally(func() { execution = append(execution, "finally") }).
		OnComplete(func() {
			require.False(t, manager.HasWorkflow("once"))
			execution = append(execution, "complete")
		}).Start()

	manager.Update()
	require.Equal(t, []string{"work", "finally", "complete"}, execution)
	require.Equal(t, queueFinalized, workflow.queue.lifecycle)
	assertPanics(t, func() { manager.completeQueueFinalization(workflow.queue) })
	assertPanics(t, workflow.Start)
	manager.Update()
	manager.Update()
	manager.Shutdown()
	require.Equal(t, []string{"work", "finally", "complete"}, execution)
}

func TestOnCompleteIsSuppressedByShutdown(t *testing.T) {
	for _, phase := range []string{"before_finally", "inside_finally", "after_finally", "waiting_for_key"} {
		t.Run(phase, func(t *testing.T) {
			callbacks := &MockSystemCallbacks{KeyPressed: true}
			manager := NewEventQueueManager(func() int64 { return 0 }, callbacks)
			finallyCalls, completeCalls, successorCalls, afterKeyCalls := 0, 0, 0, 0
			workflow := manager.Workflow("cancelled").Priority(PriorityNormal).Sequential().BlockInput()
			workflow.First(func() {}).Finally(func() {
				finallyCalls++
				require.True(t, callbacks.InputBlocked, "cleanup precedes cancellation's input release")
				if phase == "inside_finally" {
					manager.Shutdown()
				}
			}).OnComplete(func() {
				completeCalls++
				manager.Workflow("successor").Priority(PriorityNormal).Sequential().
					First(func() { successorCalls++ }).Start()
			})
			if phase == "waiting_for_key" {
				workflow.WaitForAnyKey().Then(func() { afterKeyCalls++ })
			}
			workflow.Start()

			switch phase {
			case "before_finally":
				require.Zero(t, finallyCalls)
			case "inside_finally":
				manager.Update()
				require.True(t, manager.shutdown)
				require.Equal(t, 1, finallyCalls)
			case "after_finally":
				manager.Update()
				require.Equal(t, 1, finallyCalls)
				require.True(t, workflow.queue.pendingInputUnblock)
				require.True(t, manager.HasWorkflow("cancelled"))
			case "waiting_for_key":
				manager.Update()
				require.True(t, workflow.queue.waitingForInput)
				require.Zero(t, finallyCalls)
			}
			manager.Shutdown()
			manager.Shutdown()
			callbacks.KeyPressed = false
			manager.Update()
			callbacks.KeyPressed = true
			manager.Update()
			callbacks.KeyPressed = false
			manager.Update()

			require.Equal(t, 1, finallyCalls)
			require.Zero(t, completeCalls, "cancellation must not invoke a gameplay continuation")
			require.Zero(t, successorCalls)
			require.Zero(t, afterKeyCalls)
			require.Nil(t, workflow.queue.onComplete)
			require.Equal(t, queueFinalized, workflow.queue.lifecycle)
			require.False(t, manager.HasWorkflow("cancelled"))
			require.False(t, manager.HasWorkflow("successor"))
			require.False(t, manager.HasActiveWorkflows())
			require.False(t, callbacks.InputBlocked)
			require.Zero(t, manager.inputBlockCount)
			require.Equal(t, 1, callbacks.UnblockInputCalls)
		})
	}
}

func TestOnCompleteRejectsInvalidRegistration(t *testing.T) {
	for _, kind := range []string{"nil_callback", "feedback"} {
		t.Run(kind, func(t *testing.T) {
			manager := NewEventQueueManager(func() int64 { return 0 }, nil)
			queue := manager.Workflow("invalid")
			action := func() {}
			switch kind {
			case "nil_callback":
				action = nil
			case "feedback":
				queue = manager.feedback("invalid")
			}
			require.PanicsWithValue(t, "OnComplete requires a workflow and completion callback", func() {
				queue.Sequential().OnComplete(action)
			})
			require.Nil(t, queue.onComplete)
		})
	}
}

func TestOnCompleteRejectsDuplicateRegistrationWithoutReplacingCallback(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	queue := manager.Workflow("duplicate").Priority(PriorityNormal)
	firstCalls, replacementCalls := 0, 0
	workflow := queue.Sequential().First(func() {})
	require.Same(t, workflow, workflow.OnComplete(func() { firstCalls++ }))
	require.PanicsWithValue(t, "workflow already has a completion callback", func() {
		queue.Sequential().OnComplete(func() { replacementCalls++ })
	})
	workflow.Start()
	manager.Update()
	manager.Update()
	require.Equal(t, 1, firstCalls)
	require.Zero(t, replacementCalls)
}

func TestOnCompleteRejectsRegistrationAfterFinalization(t *testing.T) {
	for _, finish := range []string{"normal", "shutdown", "start_after_shutdown"} {
		t.Run(finish, func(t *testing.T) {
			manager := NewEventQueueManager(func() int64 { return 0 }, nil)
			workflow := manager.Workflow("finished").Priority(PriorityNormal).Sequential().First(func() {})
			if finish == "start_after_shutdown" {
				manager.Shutdown()
			}
			workflow.Start()
			if finish == "shutdown" {
				manager.Shutdown()
			} else {
				manager.Update()
			}
			require.Equal(t, queueFinalized, workflow.queue.lifecycle)
			require.PanicsWithValue(t, "OnComplete cannot register after workflow finalization", func() {
				workflow.OnComplete(func() { t.Fatal("finalized workflow invoked a late callback") })
			})
			require.Nil(t, workflow.queue.onComplete, "a dead workflow must not retain an uncallable callback")
		})
	}
}

func TestOnCompleteAllowsRuntimeRegistrationBeforeFinalizationCompletes(t *testing.T) {
	for _, phase := range []string{"sequence", "finally"} {
		t.Run(phase, func(t *testing.T) {
			manager := NewEventQueueManager(func() int64 { return 0 }, nil)
			completeCalls := 0
			workflow := manager.Workflow("runtime").Priority(PriorityNormal).Sequential().First(func() {})
			if phase == "sequence" {
				workflow.ThenSequence(func(sequence *SequentialEventBuilder) *SequentialEventBuilder {
					require.Equal(t, queueStarted, workflow.queue.lifecycle)
					return sequence.OnComplete(func() { completeCalls++ })
				})
			} else {
				workflow.Finally(func() {
					require.Equal(t, queueFinalizing, workflow.queue.lifecycle)
					workflow.OnComplete(func() { completeCalls++ })
				})
			}
			workflow.Start()
			manager.Update()
			manager.Update()
			require.Equal(t, 1, completeCalls)
			require.False(t, manager.HasWorkflow("runtime"))
		})
	}
}

func TestOnCompleteAndHandoffConflictWhenHandoffExecutes(t *testing.T) {
	for _, order := range []string{"completion_first", "handoff_first"} {
		t.Run(order, func(t *testing.T) {
			manager := NewEventQueueManager(func() int64 { return 0 }, nil)
			firstCalls, completeCalls, handoffCalls := 0, 0, 0
			workflow := manager.Workflow("conflict").Priority(PriorityNormal).Sequential().
				First(func() { firstCalls++ })
			complete := func() { completeCalls++ }
			handoff := func(next *SequentialEventBuilder) {
				handoffCalls++
				next.First(func() {})
			}
			// HandoffTo claims the slot when its event executes, in either declaration order.
			if order == "completion_first" {
				workflow.OnComplete(complete).HandoffTo("successor", PriorityHigh, handoff)
			} else {
				workflow.HandoffTo("successor", PriorityHigh, handoff).OnComplete(complete)
			}
			workflow.Start()
			require.Zero(t, firstCalls)
			require.PanicsWithValue(t, "HandoffTo cannot replace an existing completion callback", manager.Update)
			require.Equal(t, 1, firstCalls)
			require.False(t, workflow.queue.terminateRequested, "conflict fails before termination is requested")
			require.False(t, manager.HasWorkflow("successor"))
			manager.Shutdown()
			require.Zero(t, completeCalls)
			require.Zero(t, handoffCalls)
		})
	}
}

func TestOnCompleteWithUntakenConditionalHandoff(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	completeCalls, handoffCalls := 0, 0
	manager.Workflow("conditional").Priority(PriorityNormal).Sequential().First(func() {}).
		OnComplete(func() { completeCalls++ }).
		If(func() bool { return false }, func(sequence *SequentialEventBuilder) {
			sequence.HandoffTo("successor", PriorityHigh, func(next *SequentialEventBuilder) {
				handoffCalls++
				next.First(func() {})
			})
		}).Start()
	manager.Update()
	require.Equal(t, 1, completeCalls)
	require.Zero(t, handoffCalls)
	require.False(t, manager.HasWorkflow("successor"))
}
