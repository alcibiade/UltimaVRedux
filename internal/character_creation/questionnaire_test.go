package character_creation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestVirtueOrderNamesAndBonuses(t *testing.T) {
	wantNames := [...]string{
		"Honesty", "Compassion", "Valor", "Justice", "Sacrifice", "Honor", "Spirituality", "Humility",
	}
	wantBonuses := [...]AttributeBonuses{
		{Intelligence: 2},
		{Dexterity: 2},
		{Strength: 2},
		{Dexterity: 1, Intelligence: 1},
		{Strength: 1, Dexterity: 1},
		{Strength: 1, Intelligence: 1},
		{Strength: 1, Dexterity: 1, Intelligence: 1},
		{},
	}
	for index := range VirtueCount {
		virtue := Virtue(index)
		if !virtue.Valid() || virtue.String() != wantNames[index] || virtueBonuses(virtue) != wantBonuses[index] {
			t.Fatalf("virtue %d = %q, %#v", index, virtue.String(), virtueBonuses(virtue))
		}
	}
	if Virtue(-1).Valid() || Virtue(VirtueCount).Valid() || Virtue(VirtueCount).String() != "Unknown" {
		t.Fatal("out-of-range virtue reported as valid")
	}
}

func TestQuestionnaireFixedSeedSequenceAndResult(t *testing.T) {
	source := beradrng.New(42)
	questionnaire, err := NewQuestionnaire("Avatar", party_state.Female, source)
	if err != nil {
		t.Fatal(err)
	}
	responses := []rune("AbbaBAa")
	pairs := make([]Dilemma, 0, len(responses))
	for _, response := range responses {
		pair, available := questionnaire.CurrentDilemma()
		if !available {
			t.Fatal("questionnaire completed before seven answers")
		}
		pairs = append(pairs, pair)
		if !questionnaire.Answer(response) {
			t.Fatalf("Answer(%q) was rejected", response)
		}
	}

	wantPairs := []Dilemma{
		{OptionA: Valor, OptionB: Spirituality},
		{OptionA: Sacrifice, OptionB: Humility},
		{OptionA: Justice, OptionB: Honor},
		{OptionA: Honesty, OptionB: Compassion},
		{OptionA: Valor, OptionB: Honor},
		{OptionA: Honesty, OptionB: Humility},
		{OptionA: Honesty, OptionB: Honor},
	}
	if !reflect.DeepEqual(pairs, wantPairs) {
		t.Fatalf("pairs = %#v", pairs)
	}
	result, complete := questionnaire.Result()
	if !complete {
		t.Fatal("Result() is unavailable after seven answers")
	}
	wantResult := Result{
		Name:              "Avatar",
		Gender:            party_state.Female,
		Bonuses:           AttributeBonuses{Strength: 4, Intelligence: 8},
		RandomSeed:        42,
		RandomSourceCalls: 64,
	}
	if result != wantResult {
		t.Fatalf("Result() = %#v", result)
	}
	if err := ValidateResult(result); err != nil {
		t.Fatalf("ValidateResult() error = %v", err)
	}

	restored := beradrng.Restore(result.RandomSeed, result.RandomSourceCalls)
	if got, want := restored.Intn(1000), source.Intn(1000); got != want {
		t.Fatalf("restored next value = %d, want %d", got, want)
	}
	if stable, available := questionnaire.Result(); !available || stable != result {
		t.Fatalf("Result() changed after the caller advanced the random source: %#v", stable)
	}
}

func TestValidateResultRejectsImpossibleTournamentResults(t *testing.T) {
	valid := Result{
		Name:              "Avatar",
		Gender:            party_state.Female,
		Bonuses:           AttributeBonuses{Strength: 4, Intelligence: 8},
		RandomSeed:        42,
		RandomSourceCalls: 64,
	}
	tests := []struct {
		name      string
		mutate    func(*Result)
		wantError error
	}{
		{name: "out of range bonuses", mutate: func(result *Result) {
			result.Bonuses.Strength = maximumResultBonus + 1
		}, wantError: ErrInvalidBonuses},
		{name: "impossible bonus combination", mutate: func(result *Result) {
			result.Bonuses = AttributeBonuses{Strength: 10, Dexterity: 10, Intelligence: 10}
		}, wantError: ErrInconsistentResult},
		{name: "incomplete provenance", mutate: func(result *Result) {
			result.RandomSourceCalls = 0
		}, wantError: ErrIncompleteResult},
		{name: "inconsistent provenance", mutate: func(result *Result) {
			result.RandomSourceCalls++
		}, wantError: ErrInconsistentResult},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := valid
			test.mutate(&result)
			if err := ValidateResult(result); !errors.Is(err, test.wantError) {
				t.Fatalf("ValidateResult() error = %v, want %v", err, test.wantError)
			}
		})
	}
}

func TestQuestionnaireRoundInvariants(t *testing.T) {
	for seed := uint64(1); seed <= 32; seed++ {
		questionnaire, err := NewQuestionnaire("Avatar", party_state.Male, beradrng.New(seed))
		if err != nil {
			t.Fatal(err)
		}
		eligible := map[Virtue]bool{}
		for index := range VirtueCount {
			eligible[Virtue(index)] = true
		}
		for _, roundSize := range []int{4, 2, 1} {
			seen := map[Virtue]bool{}
			winners := map[Virtue]bool{}
			for range roundSize {
				pair, available := questionnaire.CurrentDilemma()
				if !available || pair.OptionA >= pair.OptionB {
					t.Fatalf("seed %d produced invalid pair %#v", seed, pair)
				}
				for _, virtue := range []Virtue{pair.OptionA, pair.OptionB} {
					if !eligible[virtue] || seen[virtue] {
						t.Fatalf("seed %d repeated or revived virtue %v", seed, virtue)
					}
					seen[virtue] = true
				}
				winners[pair.OptionA] = true
				if !questionnaire.Answer('A') {
					t.Fatalf("seed %d rejected a valid response", seed)
				}
			}
			if len(seen) != len(eligible) {
				t.Fatalf("seed %d round used %d virtues, want %d", seed, len(seen), len(eligible))
			}
			eligible = winners
		}
	}
}

func TestQuestionnaireRejectsInvalidAndPostCompletionAnswersAtomically(t *testing.T) {
	source := beradrng.New(99)
	questionnaire, err := NewQuestionnaire("Avatar", party_state.Male, source)
	if err != nil {
		t.Fatal(err)
	}
	wantPair, _ := questionnaire.CurrentDilemma()
	wantCalls := source.Calls()
	for _, response := range []rune{' ', 'C', '1', 0} {
		if questionnaire.Answer(response) {
			t.Fatalf("Answer(%q) accepted invalid response", response)
		}
		gotPair, available := questionnaire.CurrentDilemma()
		if !available || gotPair != wantPair || source.Calls() != wantCalls {
			t.Fatalf("invalid response mutated pair or RNG state")
		}
	}
	if _, complete := questionnaire.Result(); complete {
		t.Fatal("Result() available before completion")
	}
	for range totalAnswers {
		if !questionnaire.Answer('a') {
			t.Fatal("lowercase A was rejected")
		}
	}
	result, complete := questionnaire.Result()
	if !complete {
		t.Fatal("Result() unavailable after completion")
	}
	completedCalls := source.Calls()
	if questionnaire.Answer('B') || source.Calls() != completedCalls {
		t.Fatal("post-completion answer mutated questionnaire")
	}
	if got, available := questionnaire.CurrentDilemma(); available || got != (Dilemma{}) {
		t.Fatalf("CurrentDilemma() after completion = %#v, %t", got, available)
	}
	if got, available := questionnaire.Result(); !available || got != result {
		t.Fatalf("completed result changed: %#v, %t", got, available)
	}
}

func TestNewQuestionnaireValidatesIdentityBeforeConsumingRandomness(t *testing.T) {
	tests := []struct {
		name      string
		character string
		gender    party_state.CharacterGender
		wantError error
	}{
		{name: "empty name", character: "", gender: party_state.Male, wantError: ErrInvalidName},
		{name: "long name", character: "123456789", gender: party_state.Male, wantError: ErrInvalidName},
		{name: "control character", character: "A\n", gender: party_state.Male, wantError: ErrInvalidName},
		{name: "non ASCII", character: "Avatár", gender: party_state.Male, wantError: ErrInvalidName},
		{name: "dash", character: "B-Rad", gender: party_state.Male, wantError: ErrInvalidName},
		{name: "space", character: "B Rad", gender: party_state.Male, wantError: ErrInvalidName},
		{name: "digit", character: "Avatar5", gender: party_state.Male, wantError: ErrInvalidName},
		{name: "unknown gender", character: "Avatar", gender: 0, wantError: ErrInvalidGender},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := beradrng.New(7)
			questionnaire, err := NewQuestionnaire(test.character, test.gender, source)
			if questionnaire != nil || !errors.Is(err, test.wantError) {
				t.Fatalf("NewQuestionnaire() = %#v, %v", questionnaire, err)
			}
			if source.Calls() != 0 {
				t.Fatalf("invalid identity consumed %d random calls", source.Calls())
			}
		})
	}

	for _, character := range []string{"A", "abcdefgh", "Avatar", "BRAD"} {
		if _, err := NewQuestionnaire(character, party_state.Female, beradrng.New(7)); err != nil {
			t.Fatalf("valid name %q rejected: %v", character, err)
		}
	}
}

func TestNewQuestionnaireRequiresRandomSource(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewQuestionnaire() did not panic for a nil random source")
		}
	}()
	_, _ = NewQuestionnaire("Avatar", party_state.Male, nil)
}

func TestNewQuestionnaireRejectsAdvancedRandomSource(t *testing.T) {
	source := beradrng.New(7)
	_ = source.Intn(VirtueCount)
	questionnaire, err := NewQuestionnaire("Avatar", party_state.Male, source)
	if questionnaire != nil || !errors.Is(err, ErrAdvancedRandomSource) {
		t.Fatalf("NewQuestionnaire() = %#v, %v", questionnaire, err)
	}
	if source.Calls() != 1 {
		t.Fatalf("NewQuestionnaire() advanced rejected source to %d calls", source.Calls())
	}
}
