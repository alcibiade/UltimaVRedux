package event_queue

// EventResult represents the result of an event execution
type EventResult int

const (
	// EventContinue means the event should continue running next frame
	EventContinue EventResult = iota
	// EventComplete means the event is finished and should be removed from the queue
	EventComplete
)

// ContinuousEvent interface for events that need to run every frame
type ContinuousEvent interface {
	// Execute processes the event and returns whether it should continue
	Execute(deltaMs uint64) EventResult
}

// AnimateEvent provides smooth frame-by-frame animations with progress callbacks
type AnimateEvent struct {
	DurationMs uint64                 // Total animation duration in milliseconds
	ElapsedMs  uint64                 // Time elapsed so far
	OnFrame    func(progress float32) // Called each frame with progress 0.0-1.0
	OnComplete func()                 // Optional callback when animation completes
}

// Execute updates the animation and calls progress callback each frame
func (a *AnimateEvent) Execute(deltaMs uint64) EventResult {
	a.ElapsedMs += deltaMs

	// Calculate progress, clamping to [0.0, 1.0]
	progress := float32(1)
	if a.DurationMs > 0 {
		progress = min(float32(a.ElapsedMs)/float32(a.DurationMs), 1)
	}

	// Call frame callback with current progress
	if a.OnFrame != nil {
		a.OnFrame(progress)
	}

	// Check if animation is complete
	if progress >= 1.0 {
		// Call completion callback if provided
		if a.OnComplete != nil {
			a.OnComplete()
		}

		return EventComplete
	}

	return EventContinue
}

// NewAnimateEventWithCallback creates a new animation event with completion callback
func NewAnimateEventWithCallback(durationMs uint64, onFrame func(float32), onComplete func()) *AnimateEvent {
	return &AnimateEvent{
		DurationMs: durationMs,
		ElapsedMs:  0,
		OnFrame:    onFrame,
		OnComplete: onComplete,
	}
}
