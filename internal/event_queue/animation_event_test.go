package event_queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnimateEvent_Execute(t *testing.T) {
	tests := []struct {
		name           string
		durationMs     uint64
		deltaMs        uint64
		expectedResult EventResult
		expectedCalls  int
	}{
		{
			name:           "Single frame partial progress",
			durationMs:     1000,
			deltaMs:        250,
			expectedResult: EventContinue,
			expectedCalls:  1,
		},
		{
			name:           "Single frame completion",
			durationMs:     1000,
			deltaMs:        1000,
			expectedResult: EventComplete,
			expectedCalls:  1,
		},
		{
			name:           "Single frame over-completion",
			durationMs:     1000,
			deltaMs:        1500,
			expectedResult: EventComplete,
			expectedCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var progressValues []float32
			var completionCalled bool

			event := NewAnimateEventWithCallback(tt.durationMs, func(progress float32) {
				progressValues = append(progressValues, progress)
			}, func() {
				completionCalled = true
			})

			result := event.Execute(tt.deltaMs)

			assert.Equal(t, tt.expectedResult, result)
			assert.Len(t, progressValues, tt.expectedCalls)

			if tt.expectedResult == EventComplete {
				assert.True(t, completionCalled, "Completion callback should be called when animation completes")
				assert.Equal(t, float32(1.0), progressValues[len(progressValues)-1], "Final progress should be 1.0")
			} else {
				assert.False(t, completionCalled, "Completion callback should not be called when animation continues")
				expectedProgress := float32(tt.deltaMs) / float32(tt.durationMs)
				assert.Equal(t, expectedProgress, progressValues[0])
			}
		})
	}
}

func TestAnimateEvent_MultiFrameSequence(t *testing.T) {
	var progressValues []float32
	var completionCalled bool

	event := NewAnimateEventWithCallback(1000, func(progress float32) {
		progressValues = append(progressValues, progress)
	}, func() {
		completionCalled = true
	})

	// First frame: 25% progress
	result1 := event.Execute(250)
	assert.Equal(t, EventContinue, result1)
	assert.False(t, completionCalled)
	assert.Len(t, progressValues, 1)
	assert.Equal(t, float32(0.25), progressValues[0])

	// Second frame: 50% total progress
	result2 := event.Execute(250)
	assert.Equal(t, EventContinue, result2)
	assert.False(t, completionCalled)
	assert.Len(t, progressValues, 2)
	assert.Equal(t, float32(0.5), progressValues[1])

	// Third frame: 100% total progress (completion)
	result3 := event.Execute(500)
	assert.Equal(t, EventComplete, result3)
	assert.True(t, completionCalled)
	assert.Len(t, progressValues, 3)
	assert.Equal(t, float32(1.0), progressValues[2])
}

func TestAnimateEvent_ProgressClamping(t *testing.T) {
	var progressValues []float32

	event := NewAnimateEvent(100, func(progress float32) {
		progressValues = append(progressValues, progress)
	})

	// Execute with deltaMs much larger than duration
	result := event.Execute(1000)

	assert.Equal(t, EventComplete, result)
	assert.Len(t, progressValues, 1)
	assert.Equal(t, float32(1.0), progressValues[0], "Progress should be clamped to 1.0")
}

func TestAnimateEvent_NilCallbacks(t *testing.T) {
	// Test that nil callbacks don't cause crashes
	event := NewAnimateEvent(1000, nil)

	// Should not panic with nil OnFrame callback
	result1 := event.Execute(500)
	assert.Equal(t, EventContinue, result1)

	event2 := NewAnimateEventWithCallback(1000, nil, nil)

	// Should not panic with nil OnFrame and OnComplete callbacks
	result2 := event2.Execute(1000)
	assert.Equal(t, EventComplete, result2)
}

func TestAnimateEvent_ZeroDuration(t *testing.T) {
	var progressValues []float32
	var completionCalled bool

	event := NewAnimateEventWithCallback(0, func(progress float32) {
		progressValues = append(progressValues, progress)
	}, func() {
		completionCalled = true
	})

	// With zero duration, any deltaMs should complete immediately
	result := event.Execute(0)

	assert.Equal(t, EventComplete, result)
	assert.True(t, completionCalled)
	assert.Len(t, progressValues, 1)
	assert.Equal(t, float32(1.0), progressValues[0])
}

// NewAnimateEvent creates a new animation event
func NewAnimateEvent(durationMs uint64, onFrame func(float32)) *AnimateEvent {
	return &AnimateEvent{
		DurationMs: durationMs,
		ElapsedMs:  0,
		OnFrame:    onFrame,
		OnComplete: nil,
	}
}
