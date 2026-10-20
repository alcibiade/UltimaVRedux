package audio

import (
	"sync"
	"time"
)

// presentationClock preserves the monotonic component of native time.Now and
// rejects backward samples from injected clocks. Every runtime audio lane shares
// this timeline; it never reads or advances gameplay time.
type presentationClock struct {
	mutex  sync.Mutex
	source func() time.Time
	latest time.Time
}

func newPresentationClock(source func() time.Time) *presentationClock {
	if source == nil {
		panic("audio presentation clock is required")
	}
	clock := new(presentationClock)
	clock.source = source
	clock.latest = source()

	return clock
}

func (clock *presentationClock) Now() time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	if sample := clock.source(); sample.After(clock.latest) {
		clock.latest = sample
	}

	return clock.latest
}
