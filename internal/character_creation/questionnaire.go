// Package character_creation implements renderer-independent new-character rules.
package character_creation

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

const (
	// VirtueCount is the number of virtues in the original questionnaire.
	VirtueCount        = 8
	firstRoundAnswers  = 4
	secondRoundAnswers = 6
	totalAnswers       = 7
	pureVirtueBonus    = 2
	minimumResultCalls = totalAnswers * 2
	maximumResultBonus = 10
)

var (
	// ErrInvalidName indicates that the entered identity cannot be stored by character creation.
	ErrInvalidName = errors.New("invalid character creation name")
	// ErrInvalidGender indicates that the entered identity is not an original male or female value.
	ErrInvalidGender = errors.New("invalid character creation gender")
	// ErrInvalidBonuses indicates that attribute modifiers cannot come from the questionnaire.
	ErrInvalidBonuses = errors.New("invalid character creation bonuses")
	// ErrIncompleteResult indicates that the result lacks a completed questionnaire's RNG provenance.
	ErrIncompleteResult = errors.New("incomplete character creation result")
	// ErrInconsistentResult indicates that no answer path produces the result from its recorded seed.
	ErrInconsistentResult = errors.New("inconsistent character creation result")
	// ErrAdvancedRandomSource indicates that a questionnaire did not receive a fresh per-attempt source.
	ErrAdvancedRandomSource = errors.New("advanced character creation random source")
)

// Virtue preserves the original tournament index order; ref:CREATE-0002.
type Virtue int

// The virtue constants retain the original zero-based indexes.
const (
	Honesty Virtue = iota
	Compassion
	Valor
	Justice
	Sacrifice
	Honor
	Spirituality
	Humility
)

// Valid reports whether the value identifies one of the eight virtues.
func (virtue Virtue) Valid() bool { return virtue >= Honesty && virtue < VirtueCount }

// String returns the original character-creation spelling of a virtue.
func (virtue Virtue) String() string {
	switch virtue {
	case Honesty:
		return "Honesty"
	case Compassion:
		return "Compassion"
	case Valor:
		return "Valor"
	case Justice:
		return "Justice"
	case Sacrifice:
		return "Sacrifice"
	case Honor:
		return "Honor"
	case Spirituality:
		return "Spirituality"
	case Humility:
		return "Humility"
	default:
		return "Unknown"
	}
}

// AttributeBonuses contains the questionnaire modifiers to apply to INIT.GAM.
type AttributeBonuses struct {
	Strength     int
	Dexterity    int
	Intelligence int
}

// Dilemma presents the lower virtue index as A and the higher index as B.
type Dilemma struct {
	OptionA Virtue
	OptionB Virtue
}

// Result is the completed, detached input to native new-game preparation.
type Result struct {
	Name              string
	Gender            party_state.CharacterGender
	Bonuses           AttributeBonuses
	RandomSeed        uint64
	RandomSourceCalls uint64
}

// ValidateResult rejects result values that cannot represent a completed questionnaire.
func ValidateResult(result Result) error {
	if err := validateName(result.Name); err != nil {
		return err
	}
	if result.Gender != party_state.Male && result.Gender != party_state.Female {
		return fmt.Errorf("%w: %#x", ErrInvalidGender, byte(result.Gender))
	}
	bonuses := [...]int{result.Bonuses.Strength, result.Bonuses.Dexterity, result.Bonuses.Intelligence}
	for _, bonus := range bonuses {
		if bonus < 0 || bonus > maximumResultBonus {
			return fmt.Errorf("%w: each bonus must be between 0 and %d", ErrInvalidBonuses, maximumResultBonus)
		}
	}
	if result.RandomSourceCalls < minimumResultCalls {
		return fmt.Errorf("%w: random source calls must be at least %d", ErrIncompleteResult, minimumResultCalls)
	}
	if !matchesQuestionnairePath(result) {
		return ErrInconsistentResult
	}

	return nil
}

func matchesQuestionnairePath(result Result) bool {
	for answers := range 1 << totalAnswers {
		questionnaire, err := NewQuestionnaire(
			result.Name,
			result.Gender,
			deterministic_random.New(result.RandomSeed),
		)
		if err != nil {
			panic("character_creation.matchesQuestionnairePath: validated identity was rejected")
		}
		for answerIndex := range totalAnswers {
			response := 'A'
			if answers&(1<<answerIndex) != 0 {
				response = 'B'
			}
			questionnaire.Answer(response)
		}
		candidate, complete := questionnaire.Result()
		if complete && candidate.Bonuses == result.Bonuses &&
			candidate.RandomSourceCalls == result.RandomSourceCalls {
			return true
		}
	}

	return false
}

// Questionnaire owns one deterministic 4/2/1 virtue tournament.
type Questionnaire struct {
	name     string
	gender   party_state.CharacterGender
	source   *deterministic_random.Source
	used     [VirtueCount]bool
	lost     [VirtueCount]bool
	current  Dilemma
	bonuses  AttributeBonuses
	answered int
	result   Result
	complete bool
}

// NewQuestionnaire validates identity and generates the first deterministic
// dilemma from a fresh per-attempt random source.
func NewQuestionnaire(
	name string,
	gender party_state.CharacterGender,
	source *deterministic_random.Source,
) (*Questionnaire, error) {
	if source == nil {
		panic("character_creation.NewQuestionnaire: random source is required")
	}
	if err := validateName(name); err != nil {
		return nil, err
	}
	if gender != party_state.Male && gender != party_state.Female {
		return nil, fmt.Errorf("%w: %#x", ErrInvalidGender, byte(gender))
	}
	if source.Calls() != 0 {
		return nil, fmt.Errorf("%w: source has %d prior calls", ErrAdvancedRandomSource, source.Calls())
	}

	questionnaire := new(Questionnaire)
	questionnaire.name = name
	questionnaire.gender = gender
	questionnaire.source = source
	questionnaire.current = questionnaire.nextDilemma()

	return questionnaire, nil
}

// CurrentDilemma returns the active pair while the questionnaire is incomplete.
func (questionnaire *Questionnaire) CurrentDilemma() (Dilemma, bool) {
	if questionnaire.complete {
		var empty Dilemma

		return empty, false
	}

	return questionnaire.current, true
}

// Answer accepts A or B case-insensitively and advances exactly one dilemma.
func (questionnaire *Questionnaire) Answer(response rune) bool {
	if questionnaire.complete {
		return false
	}

	var winner, loser Virtue
	switch response {
	case 'A', 'a':
		winner, loser = questionnaire.current.OptionA, questionnaire.current.OptionB
	case 'B', 'b':
		winner, loser = questionnaire.current.OptionB, questionnaire.current.OptionA
	default:
		return false
	}

	questionnaire.bonuses = addBonuses(questionnaire.bonuses, virtueBonuses(winner))
	questionnaire.lost[loser] = true
	questionnaire.answered++
	if questionnaire.answered == totalAnswers {
		questionnaire.complete = true
		questionnaire.result = Result{
			Name:              questionnaire.name,
			Gender:            questionnaire.gender,
			Bonuses:           questionnaire.bonuses,
			RandomSeed:        questionnaire.source.Seed(),
			RandomSourceCalls: questionnaire.source.Calls(),
		}

		return true
	}
	if questionnaire.answered == firstRoundAnswers || questionnaire.answered == secondRoundAnswers {
		questionnaire.used = [VirtueCount]bool{}
	}
	questionnaire.current = questionnaire.nextDilemma()

	return true
}

// Result returns the immutable completed result.
func (questionnaire *Questionnaire) Result() (Result, bool) {
	if !questionnaire.complete {
		var empty Result

		return empty, false
	}

	return questionnaire.result, true
}

func (questionnaire *Questionnaire) nextDilemma() Dilemma {
	first := questionnaire.nextVirtue()
	second := questionnaire.nextVirtue()
	if first > second {
		first, second = second, first
	}

	return Dilemma{OptionA: first, OptionB: second}
}

func (questionnaire *Questionnaire) nextVirtue() Virtue {
	for {
		candidate := Virtue(questionnaire.source.Intn(VirtueCount))
		if questionnaire.used[candidate] || questionnaire.lost[candidate] {
			continue
		}
		questionnaire.used[candidate] = true

		return candidate
	}
}

func validateName(name string) error {
	if len(name) < 1 || len(name) > 8 {
		return fmt.Errorf("%w: name must contain one through eight ASCII letters", ErrInvalidName)
	}
	for index := range len(name) {
		if (name[index] < 'A' || name[index] > 'Z') && (name[index] < 'a' || name[index] > 'z') {
			return fmt.Errorf("%w: byte %d is not an ASCII letter", ErrInvalidName, index)
		}
	}

	return nil
}

func virtueBonuses(virtue Virtue) AttributeBonuses {
	switch virtue {
	case Honesty:
		return AttributeBonuses{Strength: 0, Dexterity: 0, Intelligence: pureVirtueBonus}
	case Compassion:
		return AttributeBonuses{Strength: 0, Dexterity: pureVirtueBonus, Intelligence: 0}
	case Valor:
		return AttributeBonuses{Strength: pureVirtueBonus, Dexterity: 0, Intelligence: 0}
	case Justice:
		return AttributeBonuses{Strength: 0, Dexterity: 1, Intelligence: 1}
	case Sacrifice:
		return AttributeBonuses{Strength: 1, Dexterity: 1, Intelligence: 0}
	case Honor:
		return AttributeBonuses{Strength: 1, Dexterity: 0, Intelligence: 1}
	case Spirituality:
		return AttributeBonuses{Strength: 1, Dexterity: 1, Intelligence: 1}
	case Humility:
		return AttributeBonuses{Strength: 0, Dexterity: 0, Intelligence: 0}
	default:
		panic("character_creation.virtueBonuses: invalid virtue")
	}
}

func addBonuses(left, right AttributeBonuses) AttributeBonuses {
	return AttributeBonuses{
		Strength:     left.Strength + right.Strength,
		Dexterity:    left.Dexterity + right.Dexterity,
		Intelligence: left.Intelligence + right.Intelligence,
	}
}
