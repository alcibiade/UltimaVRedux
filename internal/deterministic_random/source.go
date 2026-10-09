// Package deterministic_random owns the implementation details of the
// centralized reproducible gameplay random stream.
package deterministic_random

import (
	"encoding/binary"
	"math/bits"

	//nolint:staticcheck // x/exp/rand is retained to preserve the existing gameplay stream algorithm.
	"golang.org/x/exp/rand"
)

const (
	pcgMultiplier     = 47026247687942121848144207491837523525
	pcgIncrement      = 117397592171526113268558934119004209487
	pcgMultiplierHigh = uint64(pcgMultiplier >> 64)
	pcgMultiplierLow  = uint64(pcgMultiplier & (1<<64 - 1))
	pcgIncrementHigh  = uint64(pcgIncrement >> 64)
	pcgIncrementLow   = uint64(pcgIncrement & (1<<64 - 1))
)

// Source wraps the existing x/exp/rand stream and records source consumption
// so snapshots can restore the exact next value.
type Source struct {
	random *rand.Rand
	source *countingSource
	seed   uint64
}

type countingSource struct {
	source rand.Source
	calls  uint64
}

type uint128 struct {
	high uint64
	low  uint64
}

// New constructs a stream at the beginning of seed's sequence.
func New(seed uint64) *Source {
	source := &countingSource{source: rand.NewSource(seed)}

	return &Source{random: rand.New(source), source: source, seed: seed}
}

// Reset returns this source to the beginning of seed's sequence while keeping
// the pointer stable for systems that share the centralized gameplay stream.
func (source *Source) Reset(seed uint64) {
	*source = *New(seed)
}

// Restore constructs a stream after the specified number of source calls.
func Restore(seed, calls uint64) *Source {
	state := advancePCG(uint128{high: seed, low: seed}, calls)
	encoded := make([]byte, 16)
	binary.BigEndian.PutUint64(encoded[:8], state.high)
	binary.BigEndian.PutUint64(encoded[8:], state.low)

	pcg := &rand.PCGSource{}
	if err := pcg.UnmarshalBinary(encoded); err != nil {
		panic(err)
	}

	counted := &countingSource{source: pcg, calls: calls}

	return &Source{random: rand.New(counted), source: counted, seed: seed}
}

func advancePCG(state uint128, calls uint64) uint128 {
	accMultiplier := uint128{low: 1}
	accIncrement := uint128{}
	currentMultiplier := uint128{high: pcgMultiplierHigh, low: pcgMultiplierLow}
	currentIncrement := uint128{high: pcgIncrementHigh, low: pcgIncrementLow}

	for calls > 0 {
		if calls&1 != 0 {
			accMultiplier = multiply128(accMultiplier, currentMultiplier)
			accIncrement = add128(multiply128(accIncrement, currentMultiplier), currentIncrement)
		}

		currentIncrement = multiply128(add128(currentMultiplier, uint128{low: 1}), currentIncrement)
		currentMultiplier = multiply128(currentMultiplier, currentMultiplier)
		calls >>= 1
	}

	return add128(multiply128(accMultiplier, state), accIncrement)
}

func add128(left, right uint128) uint128 {
	low, carry := bits.Add64(left.low, right.low, 0)
	high, _ := bits.Add64(left.high, right.high, carry)

	return uint128{high: high, low: low}
}

func multiply128(left, right uint128) uint128 {
	productHigh, productLow := bits.Mul64(left.low, right.low)
	productHigh += left.low*right.high + left.high*right.low

	return uint128{high: productHigh, low: productLow}
}

func (source *countingSource) Uint64() uint64 {
	source.calls++

	return source.source.Uint64()
}

func (source *countingSource) Seed(seed uint64) {
	source.source.Seed(seed)
	source.calls = 0
}

// Seed returns the seed used to initialize the stream.
func (source *Source) Seed() uint64 { return source.seed }

// Calls returns the underlying source calls consumed so far.
func (source *Source) Calls() uint64 { return source.source.calls }

// Intn delegates without changing the existing stream algorithm.
func (source *Source) Intn(n int) int { return source.random.Intn(n) }

// Float64 delegates without changing the existing stream algorithm.
func (source *Source) Float64() float64 { return source.random.Float64() }

// Shuffle delegates without changing the existing stream algorithm.
func (source *Source) Shuffle(n int, swap func(i, j int)) { source.random.Shuffle(n, swap) }
