package event_queue

import "testing"

func TestIsShutdownVisibleBeforeOwnerCleanupAndFinalizers(t *testing.T) {
	manager := NewEventQueueManager(func() int64 { return 0 }, &MockSystemCallbacks{})
	if manager.IsShutdown() {
		t.Fatal("reading the shutdown state must not close a live manager")
	}
	ownerCalls, finallyCalls, rejectedCalls := 0, 0, 0
	manager.SetShutdownAction(func() {
		ownerCalls++
		if !manager.IsShutdown() {
			t.Error("shutdown must be visible before owner cleanup")
		}
	})
	manager.Workflow("accepted").Priority(PriorityNormal).Sequential().BlockInput().
		First(func() { t.Error("shutdown before the first update must not execute the action") }).
		Finally(func() {
			finallyCalls++
			if !manager.IsShutdown() || ownerCalls != 1 {
				t.Error("finalizer must observe shutdown after owner cleanup")
			}
			manager.Shutdown() // Reentrant shutdown remains idempotent.
			manager.Workflow("rejected_during_cleanup").Priority(PriorityNormal).Sequential().
				First(func() { rejectedCalls++ }).Finally(func() { rejectedCalls++ }).Start()
		}).Start()
	if manager.IsShutdown() {
		t.Fatal("workflow admission must not change the shutdown projection")
	}
	manager.Shutdown()
	manager.Shutdown()
	manager.Update()
	if !manager.IsShutdown() || ownerCalls != 1 || finallyCalls != 1 || rejectedCalls != 0 {
		t.Fatalf("shutdown=%t owner=%d finalizer=%d rejected=%d", manager.IsShutdown(), ownerCalls, finallyCalls, rejectedCalls)
	}
	if manager.HasWorkflow("accepted") || manager.HasWorkflow("rejected_during_cleanup") {
		t.Fatal("shutdown must not retain either accepted or rejected queue names")
	}
}
