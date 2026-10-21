package event_queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnimationAdmissionExcludesPreLaunchDeltaInEveryLane(t *testing.T) {
	for _, lane := range []string{"workflow", "feedback"} {
		t.Run(lane, func(t *testing.T) {
			var now int64
			manager := NewEventQueueManager(func() int64 { return now }, nil)
			queue, update := animationAdmissionLane(manager, lane)
			var frames []float32
			var completed, finalized int
			queue.Priority(PriorityNormal).Sequential().
				First(func() {}).Wait(1000).Then(func() {}).
				Animate(200, func(progress float32) { frames = append(frames, progress) }).
				Then(func() { completed++ }).Wait(80).Then(func() {}).
				Finally(func() { finalized++ }).Start()
			pump := func() {
				for range 8 {
					update()
				}
			}
			pump()
			now = 1000
			pump()
			require.NotEmpty(t, frames)
			for _, frame := range frames {
				assert.Zero(t, frame, "admission and repeated same-time updates stay at origin")
			}
			assert.Zero(t, completed)
			now = 1100
			pump()
			assert.Equal(t, float32(0.5), frames[len(frames)-1])
			now = 1200
			pump()
			assert.Equal(t, float32(1), frames[len(frames)-1])
			assert.Equal(t, 1, completed)
			assert.Zero(t, finalized, "post-animation Wait retains its own deadline")
			now = 1279
			pump()
			assert.Zero(t, finalized)
			now = 1280
			pump()
			assert.Equal(t, 1, finalized)
			assert.Equal(t, 1, completed)
		})
	}
}

func TestAnimationAdmissionPreservesExistingContinuousDelta(t *testing.T) {
	for _, lane := range []string{"workflow", "feedback"} {
		t.Run(lane, func(t *testing.T) {
			var now int64
			manager := NewEventQueueManager(func() int64 { return now }, nil)
			queue, update := animationAdmissionLane(manager, lane)
			var existingProgress, admittedProgress float32
			existing := NewAnimateEvent(4000, func(progress float32) { existingProgress = progress })
			queue.continuousEvents = append(queue.continuousEvents, existing)
			queue.Priority(PriorityNormal).Sequential().First(func() {}).Wait(1000).Then(func() {}).
				Animate(2000, func(progress float32) { admittedProgress = progress }).Start()
			for range 8 {
				update()
			}
			now = 1000
			for range 8 {
				update()
			}
			assert.Equal(t, float32(0.25), existingProgress, "admission cannot reset the queue clock for older events")
			assert.Zero(t, admittedProgress)
			now = 1500
			update()
			assert.Equal(t, float32(0.375), existingProgress)
			assert.Equal(t, float32(0.25), admittedProgress)
		})
	}
}

func animationAdmissionLane(manager *EventQueueManager, lane string) (*NamedEventQueue, func()) {
	switch lane {
	case "feedback":
		return manager.feedback("animation"), manager.UpdateFeedback
	default:
		return manager.Workflow("animation"), manager.Update
	}
}

func TestZeroDurationAnimationAdmissionCompletesOnceInEveryLane(t *testing.T) {
	for _, lane := range []string{"workflow", "feedback"} {
		t.Run(lane, func(t *testing.T) {
			var now int64
			manager := NewEventQueueManager(func() int64 { return now }, nil)
			queue, update := animationAdmissionLane(manager, lane)
			var frames []float32
			completed, finalized := 0, 0
			queue.Priority(PriorityNormal).Sequential().First(func() {}).Animate(0, func(progress float32) {
				frames = append(frames, progress)
			}).Then(func() { completed++ }).Finally(func() { finalized++ }).Start()
			for range 8 {
				update()
			}
			now = 1000
			update()
			manager.Shutdown()
			assert.Equal(t, []float32{1}, frames)
			assert.Equal(t, 1, completed)
			assert.Equal(t, 1, finalized)
		})
	}
}
