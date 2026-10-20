package deterministic_random_test

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestFacadeSharesSDKTypeAndForwardsCheckpoint(t *testing.T) {
	const seed = uint64(8675309)
	var sdkSource *rng.Source = deterministic_random.New(seed)
	var gameSource *deterministic_random.Source = sdkSource
	gameSource.Intn(1000)
	gameSource.Shuffle(4, func(_, _ int) {})
	if sdkSource != gameSource || sdkSource.Seed() != seed || sdkSource.Calls() != 4 {
		t.Fatal("game facade introduced independent state or hidden draws")
	}
	gameRestored := deterministic_random.Restore(sdkSource.Seed(), sdkSource.Calls())
	sdkRestored := rng.Restore(sdkSource.Seed(), sdkSource.Calls())
	if gameRestored.Seed() != seed || gameRestored.Calls() != sdkSource.Calls() {
		t.Fatal("game facade changed checkpoint metadata")
	}
	if gameRestored.Intn(1000) != sdkRestored.Intn(1000) || gameRestored.Calls() != sdkRestored.Calls() {
		t.Fatal("game facade did not forward the SDK checkpoint")
	}
}
