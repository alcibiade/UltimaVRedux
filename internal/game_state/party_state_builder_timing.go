package game_state

import "github.com/bradhannah/UltimaVRedux/internal/event_queue"

// BuildWithPresentationTiming maps authored visual durations for one sequence.
// The cloned layers keep animation admission and per-layer progress on the same
// timeline. Damage, status, RNG and sound retain Build's existing ownership.
// Ordinary Build callers retain their authored presentation.
func (b *PartyStateBuilder) BuildWithPresentationTiming(duration func(uint64) uint64) event_queue.Sequence {
	if duration == nil {
		panic("party presentation duration policy is required")
	}
	copyBuilder := *b
	if b.flashLayer != nil {
		layer := *b.flashLayer
		layer.Duration = duration(layer.Duration)
		copyBuilder.flashLayer = &layer
	}
	if b.shakeLayer != nil {
		layer := *b.shakeLayer
		layer.Duration = duration(layer.Duration)
		copyBuilder.shakeLayer = &layer
	}
	if b.invertedLayer != nil {
		layer := *b.invertedLayer
		layer.Duration = duration(layer.Duration)
		copyBuilder.invertedLayer = &layer
	}

	return copyBuilder.Build()
}
