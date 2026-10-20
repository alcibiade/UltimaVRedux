package game_state

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestRNGSDKProviderUsesSharedStreamAcrossReset(t *testing.T) {
	state := createTestGameState()
	state.SetRandomSeed(12)
	var borrowed *rng.Source = state.GetRNG()
	provider := state.AsRandomProvider()
	provider.Intn(13)
	state.SetRandomSeed(8675309)
	if borrowed != state.GetRNG() || borrowed.Calls() != 0 || state.RandomSeed() != 8675309 {
		t.Fatal("SetRandomSeed detached a borrower or failed to restart its stream")
	}
	oracle := rng.New(8675309)
	if provider.Intn(1000) != oracle.Intn(1000) || state.RollDice(30) != oracle.Intn(30)+1 {
		t.Fatal("provider and dice helper did not share one SDK stream")
	}
	if state.RandomIntInRange(-3, 3) != oracle.Intn(7)-3 || state.OneInXOdds(4) != (oracle.Intn(4) == 0) {
		t.Fatal("game range/odds semantics changed at the SDK bridge")
	}
	if state.GetRNG().Calls() != oracle.Calls() {
		t.Fatal("game helpers consumed extra source draws")
	}
	before := borrowed.Calls()
	if state.OneInXOdds(0) || state.RollDice(0) != 1 || borrowed.Calls() != before {
		t.Fatal("game-owned invalid dice/odds policy changed or consumed a draw")
	}
}

func TestRNGSDKSaveHydrationContinuesSharedStream(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
		WithPlayerAt(15, 15).WithSystemCallbacks().WithRandomSeed(8675309).Build()
	state.RollDice(30)
	state.AsRandomProvider().Intn(1000)
	directions := []references.Position{{X: 1, Y: 0}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 0, Y: -1}}
	state.ShuffleDirections(directions)
	before := save_snapshot.RandomSnapshot{Seed: state.GetRNG().Seed(), SourceCalls: state.GetRNG().Calls()}
	snapshot, err := state.CaptureSaveSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Random != before || state.GetRNG().Calls() != before.SourceCalls {
		t.Fatal("capture changed RNG state or the seed/SourceCalls schema")
	}
	encoded, err := json.Marshal(snapshot.Random)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]uint64
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields["seed"] != before.Seed || fields["sourceCalls"] != before.SourceCalls {
		t.Fatalf("RNG save fields changed: %s", encoded)
	}
	hydrated, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	if err != nil {
		t.Fatal(err)
	}
	if hydrated.GetRNG().Seed() != before.Seed || hydrated.GetRNG().Calls() != before.SourceCalls {
		t.Fatal("hydration consumed draws or lost the saved stream position")
	}
	for range 5 {
		if state.RollDice(30) != hydrated.RollDice(30) ||
			state.AsRandomProvider().Intn(1000) != hydrated.AsRandomProvider().Intn(1000) {
			t.Fatal("game provider/dice continuation changed after hydration")
		}
		originalDirections, restoredDirections := slices.Clone(directions), slices.Clone(directions)
		state.ShuffleDirections(originalDirections)
		hydrated.ShuffleDirections(restoredDirections)
		if !slices.Equal(originalDirections, restoredDirections) || state.GetRNG().Calls() != hydrated.GetRNG().Calls() {
			t.Fatal("game shuffle continuation changed after hydration")
		}
	}
}
