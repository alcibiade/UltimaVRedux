// Package deterministic_random keeps the game's shared-stream entry point over
// Berad RNG. GameState owns seed policy, call ordering, and save serialization.
package deterministic_random

import rng "github.com/bradhannah/berad-sdk/modules/berad-rng"

// Source is the SDK stream type, shared by pointer through existing game providers.
type Source = rng.Source

// New constructs a stream at the beginning of the explicitly supplied seed.
func New(seed uint64) *Source { return rng.New(seed) }

// Restore resumes after calls underlying source draws from the game's saved seed.
func Restore(seed, calls uint64) *Source { return rng.Restore(seed, calls) }
