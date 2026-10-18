package audio

import "time"

// AmbientFailure is the game UI's detached diagnostic for a currently desired
// source. The module owns bounded retry state; this is not an unbounded error log.
type AmbientFailure struct {
	Attempts int
	RetryAt  time.Time
	Message  string
}
