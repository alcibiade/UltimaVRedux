package helpers

import (
	"math"
	"testing"
)

func TestRandomUInt16InRangeFullWidthAndBoundaries(t *testing.T) {
	// Previously the inclusive width wrapped to zero and every full-width
	// request panicked in rand.Intn. Keep the full-span call in the test.
	_ = RandomUInt16InRange(0, math.MaxUint16)
	if got := RandomUInt16InRange(0, 0); got != 0 {
		t.Fatalf("zero-only range = %d, want zero", got)
	}
	if got := RandomUInt16InRange(math.MaxUint16, math.MaxUint16); got != math.MaxUint16 {
		t.Fatalf("single-value range = %d, want %d", got, math.MaxUint16)
	}
}
