package profiling

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDisabledDiagnosticsDoNotReadClockOrRecordSamples(t *testing.T) {
	Disable()
	require.True(t, Begin().IsZero())
	Count(FontCreates)
	EndUpdate(time.Time{})
	EndDraw(time.Time{})
}

func TestSamplesAreBoundedAndSnapshotsDoNotRetainGameState(t *testing.T) {
	metrics := Enable()
	defer Disable()
	for index := range retainedSamples {
		metrics.updates.values[index].Store(int64(index+1) * int64(time.Millisecond))
	}
	metrics.updates.count.Store(retainedSamples * 2)
	Count(ReadySelections)
	snapshot := metrics.Snapshot()
	require.Equal(t, uint64(1), snapshot.Counters["ready_selections"])
	require.Equal(t, uint64(retainedSamples*2), snapshot.Update.Count)
	require.Equal(t, retainedSamples, snapshot.Update.Samples)
	require.InDelta(t, 256, snapshot.Update.P50Ms, 1)
	require.Equal(t, float64(retainedSamples), snapshot.Update.MaxMs)
	snapshot.Counters["ready_selections"] = 999
	require.Equal(t, uint64(1), metrics.Snapshot().Counters["ready_selections"])
	EndReadySelection(time.Time{})
	require.Zero(t, metrics.Snapshot().ReadySelectionNextDraw.Count)
}

func TestCounterPublicationIsSafeForConcurrentReaders(t *testing.T) {
	metrics := Enable()
	defer Disable()
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for range 100 {
				Count(TextHits)
				_ = metrics.Snapshot()
			}
		})
	}
	workers.Wait()
	require.Equal(t, uint64(400), metrics.Snapshot().Counters["text_hits"])
}
