package deterministic_random

import (
	"math"
	"testing"

	"golang.org/x/exp/rand"
)

func TestSourcePreservesExpRandSequenceAndRestoresState(t *testing.T) {
	const seed = uint64(8675309)
	want := rand.New(rand.NewSource(seed))
	got := New(seed)
	for index := range 50 {
		if actual, expected := got.Intn(1000), want.Intn(1000); actual != expected {
			t.Fatalf("roll %d = %d, want %d", index, actual, expected)
		}
	}
	restored := Restore(got.Seed(), got.Calls())
	for index := range 20 {
		if actual, expected := restored.Intn(1000), got.Intn(1000); actual != expected {
			t.Fatalf("restored roll %d = %d, want %d", index, actual, expected)
		}
	}
}

func TestResetPreservesPointerAndRestartsSequence(t *testing.T) {
	const seed = uint64(8675309)
	source := New(1)
	shared := source
	source.Reset(seed)
	want := New(seed)

	if shared != source {
		t.Fatal("Reset replaced the shared source pointer")
	}
	if source.Seed() != seed || source.Calls() != 0 {
		t.Fatalf("Reset state = seed %d, calls %d", source.Seed(), source.Calls())
	}
	for index := range 20 {
		if actual, expected := shared.Intn(1000), want.Intn(1000); actual != expected {
			t.Fatalf("reset roll %d = %d, want %d", index, actual, expected)
		}
	}
}

func TestRestoreMatchesIterativeSourceAcrossSeedsAndCalls(t *testing.T) {
	seeds := []uint64{0, 1, 8675309, 0x0123456789abcdef, math.MaxUint64}
	callCounts := []uint64{0, 1, 2, 3, 7, 8, 31, 64, 255, 1024, 10000}

	for _, seed := range seeds {
		for _, calls := range callCounts {
			iterative := New(seed)
			for range calls {
				iterative.source.Uint64()
			}

			restored := Restore(seed, calls)
			if actual := restored.Calls(); actual != calls {
				t.Fatalf("Restore(%d, %d).Calls() = %d", seed, calls, actual)
			}

			for output := range 10 {
				actual := restored.source.Uint64()
				expected := iterative.source.Uint64()
				if actual != expected {
					t.Fatalf("Restore(%d, %d) output %d = %#x, want %#x", seed, calls, output, actual, expected)
				}
			}
		}
	}
}

func TestRestoreHugeCallCount(t *testing.T) {
	const (
		seed      = uint64(0xfedcba9876543210)
		calls     = uint64(math.MaxUint64)
		stepCalls = uint64(1024)
	)

	restored := Restore(seed, calls)
	if actual := restored.Calls(); actual != calls {
		t.Fatalf("Calls() = %d, want %d", actual, calls)
	}

	stepped := Restore(seed, calls-stepCalls)
	for range stepCalls {
		stepped.source.Uint64()
	}
	if actual := stepped.Calls(); actual != calls {
		t.Fatalf("stepped Calls() = %d, want %d", actual, calls)
	}

	if actual, expected := restored.source.Uint64(), stepped.source.Uint64(); actual != expected {
		t.Fatalf("next output = %#x, want %#x", actual, expected)
	}
}
