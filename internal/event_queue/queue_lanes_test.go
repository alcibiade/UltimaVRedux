package event_queue

import (
	"reflect"
	"testing"
)

func TestWorkflowRegistersAtomicallyAtStart(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	workflow := manager.Workflow("atomic").
		Priority(PriorityNormal).
		Sequential().
		First(func() {})

	if manager.HasWorkflow("atomic") {
		t.Fatal("unstarted workflow reserved its name")
	}
	workflow.Start()
	if !manager.HasWorkflow("atomic") {
		t.Fatal("started workflow did not reserve its name")
	}
	if !manager.HasOnlyWorkflow("atomic") {
		t.Fatal("sole started workflow was not reported as exclusive")
	}
	manager.Workflow("second").Priority(PriorityLow).Sequential().First(func() {}).Wait(1).Start()
	if manager.HasOnlyWorkflow("atomic") {
		t.Fatal("workflow was reported as exclusive while another name was live")
	}
}

func TestInvalidWorkflowStartDoesNotReserveName(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	workflow := manager.Workflow("invalid").
		Sequential().
		First(func() {})

	assertPanics(t, workflow.Start)
	if manager.HasWorkflow("invalid") {
		t.Fatal("invalid workflow start reserved its name")
	}
}

func TestWorkflowStartedInsideCallbackDefersUntilBoundary(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	var execution []string
	manager.Workflow("parent").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			execution = append(execution, "parent")
			manager.Workflow("child").
				Priority(PriorityHigh).
				Sequential().
				First(func() { execution = append(execution, "child") }).
				Start()
		}).
		Wait(100).
		Then(func() { execution = append(execution, "parent resumed") }).
		Start()

	manager.Update()
	if !reflect.DeepEqual(execution, []string{"parent"}) {
		t.Fatalf("nested workflow executed reentrantly: %v", execution)
	}
	if manager.activeQueue == nil || manager.activeQueue.name != "child" {
		t.Fatalf("deferred high-priority workflow was not dispatched at boundary")
	}

	manager.Update()
	if !reflect.DeepEqual(execution, []string{"parent", "child"}) {
		t.Fatalf("deferred workflow execution = %v", execution)
	}
}

func TestDeferredWorkflowWaitsForFinalizingInputOwner(t *testing.T) {
	callbacks := &MockSystemCallbacks{}
	manager := NewEventQueueManager(func() int64 { return 0 }, callbacks)
	var execution []string
	manager.Workflow("parent").
		Priority(PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			execution = append(execution, "parent")
			manager.Workflow("child").
				Priority(PriorityHigh).
				Sequential().
				First(func() { execution = append(execution, "child") }).
				Start()
		}).
		Start()

	manager.Update()
	if manager.activeQueue == nil || manager.activeQueue.name != "parent" {
		t.Fatal("finalizing input owner should remain active until input cleanup completes")
	}
	if len(manager.pendingQueues) != 1 || manager.pendingQueues[0].name != "child" || !manager.pendingQueues[0].isPaused {
		t.Fatal("deferred child workflow should wait paused behind the finalizing input owner")
	}

	manager.Update()
	if manager.activeQueue == nil || manager.activeQueue.name != "child" {
		t.Fatal("child workflow was not promoted after finalizing input cleanup")
	}
	manager.Update()
	if !reflect.DeepEqual(execution, []string{"parent", "child"}) {
		t.Fatalf("workflow execution = %v", execution)
	}
}

func TestFeedbackCoalescesSameLiveName(t *testing.T) {
	currentTime := int64(0)
	manager := NewEventQueueManager(func() int64 { return currentTime }, nil)
	var execution []string
	build := func(label string) func(*SequentialEventBuilder) {
		return func(sequence *SequentialEventBuilder) {
			sequence.First(func() { execution = append(execution, label+" show") }).
				Wait(100).
				Then(func() { execution = append(execution, label+" hide") }).
				Finally(func() { execution = append(execution, label+" finally") })
		}
	}

	manager.TriggerFeedback("poison", PriorityNormal, build("first"))
	manager.TriggerFeedback("poison", PriorityHigh, build("duplicate"))
	manager.UpdateFeedback()
	currentTime = 100
	manager.UpdateFeedback()

	want := []string{"first show", "first hide", "first finally"}
	if !reflect.DeepEqual(execution, want) {
		t.Fatalf("coalesced feedback execution = %v, want %v", execution, want)
	}
	if manager.HasFeedback("poison") {
		t.Fatal("completed feedback retained its name")
	}
}

func TestFeedbackTriggeredInsideWorkflowStartsSafely(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	var execution []string
	manager.Workflow("parent").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			manager.TriggerFeedback("poison", PriorityNormal, func(sequence *SequentialEventBuilder) {
				sequence.First(func() { execution = append(execution, "feedback") })
			})
		}).
		Start()

	manager.Update()
	if !manager.HasFeedback("poison") {
		t.Fatal("feedback triggered inside workflow was not reserved")
	}
	if len(execution) != 0 {
		t.Fatalf("feedback executed inside workflow callback: %v", execution)
	}

	manager.UpdateFeedback()
	if !reflect.DeepEqual(execution, []string{"feedback"}) {
		t.Fatalf("feedback execution = %v", execution)
	}
}

func TestFeedbackTriggeredTwiceInsideCallbackCoalescesWhileDeferred(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	executions := 0
	build := func(sequence *SequentialEventBuilder) {
		sequence.First(func() { executions++ })
	}
	manager.Workflow("parent").
		Priority(PriorityNormal).
		Sequential().
		First(func() {
			manager.TriggerFeedback("poison", PriorityNormal, build)
			manager.TriggerFeedback("poison", PriorityHigh, build)
		}).
		Start()

	manager.Update()
	manager.UpdateFeedback()
	if executions != 1 {
		t.Fatalf("deferred duplicate feedback executed %d times, want 1", executions)
	}
}

func TestFeedbackSerializesWithoutPreemptingActiveEffect(t *testing.T) {
	currentTime := int64(0)
	manager := NewEventQueueManager(func() int64 { return currentTime }, nil)
	var execution []string
	manager.TriggerFeedback("active", PriorityNormal, func(sequence *SequentialEventBuilder) {
		sequence.First(func() { execution = append(execution, "active show") }).
			Wait(100).
			Then(func() { execution = append(execution, "active hide") })
	})
	manager.TriggerFeedback("low", PriorityLow, func(sequence *SequentialEventBuilder) {
		sequence.First(func() { execution = append(execution, "low") })
	})
	manager.TriggerFeedback("high", PriorityHigh, func(sequence *SequentialEventBuilder) {
		sequence.First(func() { execution = append(execution, "high") })
	})

	manager.UpdateFeedback()
	if !reflect.DeepEqual(execution, []string{"active show"}) {
		t.Fatalf("pending feedback preempted active effect: %v", execution)
	}

	currentTime = 100
	manager.UpdateFeedback()
	manager.UpdateFeedback()
	manager.UpdateFeedback()
	want := []string{"active show", "active hide", "high", "low"}
	if !reflect.DeepEqual(execution, want) {
		t.Fatalf("serialized feedback execution = %v, want %v", execution, want)
	}
}

func TestAwaitFeedbackKeepsWorkflowPausedUntilFeedbackCompletes(t *testing.T) {
	currentTime := int64(0)
	manager := NewEventQueueManager(func() int64 { return currentTime }, nil)
	var execution []string
	manager.Workflow("attack").
		Priority(PriorityHigh).
		Sequential().
		First(func() { execution = append(execution, "attack") }).
		AwaitFeedback("poison", PriorityNormal, func(sequence *SequentialEventBuilder) {
			sequence.First(func() { execution = append(execution, "flash") }).
				Wait(100).
				Then(func() { execution = append(execution, "clear") })
		}).
		Then(func() { execution = append(execution, "finished") }).
		Start()

	manager.Update()
	manager.Update()
	manager.UpdateFeedback()
	if !reflect.DeepEqual(execution, []string{"attack", "flash"}) {
		t.Fatalf("workflow did not pause for feedback: %v", execution)
	}

	currentTime = 100
	manager.UpdateFeedback()
	manager.Update()
	if !reflect.DeepEqual(execution, []string{"attack", "flash", "clear", "finished"}) {
		t.Fatalf("workflow did not resume after feedback: %v", execution)
	}
}

func TestFeedbackCannotBlockInput(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, &MockSystemCallbacks{})
	assertPanics(t, func() {
		manager.feedback("invalid").
			Priority(PriorityNormal).
			Sequential().
			BlockInput().
			First(func() {}).
			Start()
	})
}

func TestFeedbackCannotWaitForInput(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, nil)
	assertPanics(t, func() {
		manager.TriggerFeedback("invalid", PriorityNormal, func(sequence *SequentialEventBuilder) {
			sequence.First(func() {}).WaitForAnyKey()
		})
	})
}
