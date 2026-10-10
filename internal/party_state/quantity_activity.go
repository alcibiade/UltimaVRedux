package party_state

// QuantityActivity retains gross committed movements between observations. A
// balance alone loses a gain followed by a loss within one command/queue update.
// These runtime-only counters are not saved, do not emit callbacks, and must never
// drive gameplay. Observers baseline them after initialization or state replacement.
type QuantityActivity struct {
	Gained uint64
	Lost   uint64
}

func (a *QuantityActivity) record(before, after uint16) {
	if after > before {
		a.Gained += uint64(after - before)
	} else {
		a.Lost += uint64(before - after)
	}
}

// Activity returns a detached observation of actual, post-limit movements.
func (l *ItemQuantityLarge) Activity() QuantityActivity {
	return l.activity
}

// Activity returns a detached observation of actual, post-limit movements.
func (s *ItemQuantitySmall) Activity() QuantityActivity {
	return s.activity
}
