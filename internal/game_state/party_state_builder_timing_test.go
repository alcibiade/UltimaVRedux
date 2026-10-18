package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestPartyStateBuilderPresentationTimingKeepsLayersAndProgressCoherent(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	t.Cleanup(state.EventQueueManager.Shutdown)
	flash, invert := 0.0, 0.0
	state.SystemCallbacks.Visual.SetScreenFlashIntensity = func(value float64) { flash = value }
	state.SystemCallbacks.Visual.SetScreenInvert = func(value float64, _ int) { invert = value }
	builder := NewPartyStateBuilder(state).FlashInverted().WithFlash(400, 1, 1, 1, 1).WithShake(200, 6)
	sequence := builder.BuildWithPresentationTiming(func(base uint64) uint64 { return base / 4 })
	// An explicitly timed build must not rewrite later ordinary/noncombat builds.
	assert.Equal(t, uint64(400), builder.flashLayer.Duration)
	assert.Equal(t, uint64(500), builder.invertedLayer.Duration)
	assert.Equal(t, uint64(200), builder.shakeLayer.Duration)
	finished := false
	state.EventQueueManager.Workflow("party_timing_layers").Priority(event_queue.PriorityHigh).
		Sequential().First(func() {}).ThenSequence(sequence).Finally(func() { finished = true }).Start()
	start := state.ElapsedMs
	pump := func(age int64) {
		state.ElapsedMs = start + age
		for range 12 {
			state.EventQueueManager.Update()
		}
	}
	rng := state.GetRNG().Calls()
	pump(0)
	pump(25)
	assert.InDelta(t, 0.5, flash, 0.001, "flash is one quarter through its 100ms effective duration")
	assert.InDelta(t, 0.4, invert, 0.001, "inversion is one fifth through its 125ms effective duration")
	pump(50)
	assert.InDelta(t, 1, flash, 0.001)
	assert.InDelta(t, 0.8, invert, 0.001)
	pump(99)
	require.False(t, finished)
	pump(100)
	require.False(t, finished, "the longer inversion still owns the animation")
	assert.Zero(t, flash)
	assert.InDelta(t, 0.4, invert, 0.001)
	pump(124)
	require.False(t, finished)
	pump(125)
	assert.True(t, finished)
	assert.Zero(t, flash)
	assert.Zero(t, invert)
	assert.Equal(t, rng, state.GetRNG().Calls())
}
