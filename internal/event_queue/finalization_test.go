package event_queue

import (
	"reflect"
	"testing"
)

func TestFinalization_InterruptedQueueResumesAndRunsLastOnce(t *testing.T) {
	mockTime := int64(0)
	manager := NewEventQueueManager(func() int64 { return mockTime }, nil)
	var execution []string

	manager.Workflow("interrupted").
		Priority(PriorityNormal).
		Sequential().
		First(func() { execution = append(execution, "A first") }).
		Wait(100).
		Then(func() { execution = append(execution, "A resumed") }).Finally(

		func() { execution = append(execution, "A last") }).
		Start()
	manager.Update()

	manager.Workflow("interrupting").
		Priority(PriorityHigh).
		Sequential().
		First(func() { execution = append(execution, "B first") }).
		Start()
	manager.Update()

	mockTime = 100
	manager.Update()
	manager.Update()

	want := []string{"A first", "B first", "A resumed", "A last"}
	if !reflect.DeepEqual(execution, want) {
		t.Fatalf("execution order = %v, want %v", execution, want)
	}
}

func TestPriorityPreemptionPreservesRemainingWait(t *testing.T) {
	mockTime := int64(0)
	manager := NewEventQueueManager(func() int64 { return mockTime }, nil)
	var execution []string

	manager.Workflow("waiting").
		Priority(PriorityNormal).
		Sequential().
		First(func() { execution = append(execution, "A first") }).
		Wait(100).
		Then(func() { execution = append(execution, "A resumed") }).
		Start()
	manager.Update()

	mockTime = 10
	manager.Workflow("interrupt").
		Priority(PriorityHigh).
		Sequential().
		First(func() { execution = append(execution, "B first") }).
		Wait(200).
		Then(func() { execution = append(execution, "B finish") }).Finally(

		func() { execution = append(execution, "B last") }).
		Start()
	manager.Update()

	mockTime = 210
	manager.Update()
	manager.Update()
	if !reflect.DeepEqual(execution, []string{"A first", "B first", "B finish", "B last"}) {
		t.Fatalf("queue resumed before its remaining wait elapsed: %v", execution)
	}

	mockTime = 299
	manager.Update()
	if !reflect.DeepEqual(execution, []string{"A first", "B first", "B finish", "B last"}) {
		t.Fatalf("queue resumed early at 299ms: %v", execution)
	}

	mockTime = 300
	manager.Update()
	if !reflect.DeepEqual(execution, []string{"A first", "B first", "B finish", "B last", "A resumed"}) {
		t.Fatalf("queue did not resume with preserved delay: %v", execution)
	}
}

func TestFinalization_HandoffToCleansUpThenHandsOff(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	var execution []string

	manager.Workflow("A").
		Priority(PriorityNormal).
		Sequential().
		First(func() { execution = append(execution, "A action") }).HandoffTo(

		"B", PriorityHigh, func(queue *SequentialEventBuilder) {
			queue.First(func() { execution = append(execution, "B first") })
		}).
		Then(func() { execution = append(execution, "A resumed") }).Finally(

		func() { execution = append(execution, "A last") }).
		Start()

	manager.Update()
	manager.Update()
	manager.Update()

	want := []string{"A action", "A last", "B first"}
	if !reflect.DeepEqual(execution, want) {
		t.Fatalf("execution order = %v, want %v", execution, want)
	}
}

func TestFinalization_FirstAndLastMainQueue(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	var execution []string

	manager.Workflow("first_last").
		Priority(PriorityNormal).
		Sequential().
		First(func() { execution = append(execution, "first") }).Finally(

		func() { execution = append(execution, "last") }).
		Start()

	manager.Update()
	manager.Update()
	want := []string{"first", "last"}
	if !reflect.DeepEqual(execution, want) {
		t.Fatalf("execution order = %v, want %v", execution, want)
	}
}

func TestFinalization_NormalChainedQueueLastRunsOnce(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	var execution []string

	manager.Workflow("chained").
		Priority(PriorityNormal).
		Sequential().
		First(func() { execution = append(execution, "first") }).
		Then(func() { execution = append(execution, "then") }).Finally(

		func() { execution = append(execution, "last") }).
		Start()

	manager.Update()
	manager.Update()
	want := []string{"first", "then", "last"}
	if !reflect.DeepEqual(execution, want) {
		t.Fatalf("execution order = %v, want %v", execution, want)
	}
}

func TestLifecycle_DuplicateLiveQueueNamePanics(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	manager.Workflow("duplicate").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Start()

	assertPanics(t, func() {
		manager.Workflow("duplicate").
			Priority(PriorityNormal).
			Sequential().
			First(func() {}).
			Start()
	})
}

func TestLifecycle_StartTwicePanics(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	builder := manager.Workflow("twice").
		Priority(PriorityNormal).
		Sequential().
		First(func() {})
	builder.Start()

	assertPanics(t, builder.Start)
}

func TestLifecycle_FinalizationTwicePanics(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	queue := manager.Workflow("finalize_twice")
	lastCalls := 0
	completeCalls := 0
	queue.lastAction = func() { lastCalls++ }
	queue.onComplete = func() { completeCalls++ }
	queue.lifecycle = queueStarted
	manager.beginQueueFinalization(queue)
	manager.completeQueueFinalization(queue)

	assertPanics(t, func() { manager.beginQueueFinalization(queue) })
	assertPanics(t, func() { manager.completeQueueFinalization(queue) })
	if lastCalls != 1 || completeCalls != 1 {
		t.Fatalf("callback calls = Last:%d OnComplete:%d, want 1 each", lastCalls, completeCalls)
	}
}

func TestLifecycle_InputBlockReleaseWithoutOwnershipPanics(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, &MockSystemCallbacks{})
	queue := manager.Workflow("no_input_block")

	assertPanics(t, func() { manager.releaseInputBlock(queue) })
}

func TestLifecycle_InputBlockUnderflowPanics(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, &MockSystemCallbacks{})

	assertPanics(t, manager.unblockInputIfNeeded)
}

func TestLifecycle_ShutdownFinalizesEveryLaneAndReleasesInput(t *testing.T) {
	callbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(func() int64 { return 0 }, callbacks)
	finallyCalls := make(map[string]int)

	startWorkflow := func(name string) {
		manager.Workflow(name).
			Priority(PriorityNormal).
			Sequential().
			BlockInput().
			First(func() {}).
			Wait(1000).
			Finally(func() {
				finallyCalls[name]++
				if name == "active" {
					manager.Workflow("started_during_shutdown").
						Priority(PriorityNormal).
						Sequential().
						First(func() {}).
						Finally(func() { finallyCalls["started_during_shutdown"]++ }).
						Start()
				}
			}).
			Start()
	}
	startWorkflow("active")
	startWorkflow("pending")
	manager.TriggerFeedback("feedback", PriorityNormal, func(queue *SequentialEventBuilder) {
		queue.First(func() {}).Wait(1000).Finally(func() { finallyCalls["feedback"]++ })
	})
	manager.run(func() {
		manager.Workflow("deferred").
			Priority(PriorityNormal).
			Sequential().
			First(func() {}).
			Finally(func() { finallyCalls["deferred"]++ }).
			Start()
		manager.TriggerFeedback("deferred_feedback", PriorityNormal, func(queue *SequentialEventBuilder) {
			queue.First(func() {}).Finally(func() { finallyCalls["deferred_feedback"]++ })
		})
	})

	if !callbacks.InputBlocked {
		t.Fatal("started workflows should own input before shutdown")
	}
	manager.Shutdown()
	manager.Shutdown()

	for _, name := range []string{"active", "pending", "feedback", "deferred", "deferred_feedback"} {
		if finallyCalls[name] != 1 {
			t.Errorf("%s Finally calls = %d, want 1", name, finallyCalls[name])
		}
	}
	if callbacks.InputBlocked {
		t.Fatal("shutdown left callback input blocked")
	}
	if manager.HasActiveWorkflows() ||
		manager.HasWorkflow("active") || manager.HasWorkflow("pending") ||
		manager.HasWorkflow("deferred") || manager.HasFeedback("feedback") ||
		manager.HasFeedback("deferred_feedback") {
		t.Fatal("shutdown left queue registrations active")
	}

	manager.Workflow("after_shutdown").
		Priority(PriorityNormal).
		Sequential().
		First(func() {}).
		Finally(func() { finallyCalls["after_shutdown"]++ }).
		Start()
	manager.TriggerFeedback("after_shutdown", PriorityNormal, func(queue *SequentialEventBuilder) {
		queue.First(func() {}).Finally(func() { finallyCalls["after_feedback"]++ })
	})
	manager.Update()
	manager.UpdateFeedback()

	if finallyCalls["after_shutdown"] != 0 || finallyCalls["after_feedback"] != 0 ||
		finallyCalls["started_during_shutdown"] != 0 {
		t.Fatalf("starts after shutdown ran cleanup callbacks: %v", finallyCalls)
	}
	if manager.HasWorkflow("after_shutdown") || manager.HasFeedback("after_shutdown") {
		t.Fatal("starts after shutdown registered live queues")
	}
}

func TestLifecycle_ShutdownRunsOwnerActionExactlyOnce(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	calls := 0
	manager.SetShutdownAction(func() { calls++ })

	manager.Shutdown()
	manager.Shutdown()

	if calls != 1 {
		t.Fatalf("shutdown action calls = %d, want 1", calls)
	}
}

func TestLifecycle_FeedbackCallbackCanShutdownManager(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	finallyCalls := 0
	nextCalls := 0

	manager.TriggerFeedback("feedback", PriorityNormal, func(queue *SequentialEventBuilder) {
		queue.First(manager.Shutdown).
			Then(func() { nextCalls++ }).
			Finally(func() { finallyCalls++ })
	})

	manager.UpdateFeedback()
	if finallyCalls != 1 || nextCalls != 0 {
		t.Fatalf("callback counts = Finally:%d Next:%d, want 1 and 0", finallyCalls, nextCalls)
	}
	if manager.HasFeedback("feedback") {
		t.Fatal("shutdown left feedback queue registered")
	}
}

func assertPanics(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	action()
}
