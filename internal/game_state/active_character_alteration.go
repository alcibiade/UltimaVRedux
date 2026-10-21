package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

// MinStatValue is the minimum valid STR, DEX, or INT value.
const MinStatValue = 1

// MaxExperienceValue is the highest supported gameplay-oriented experience value.
const MaxExperienceValue = 9999

// ErrInvalidCharacterAlteration indicates that an active-character update would violate party invariants.
var ErrInvalidCharacterAlteration = errors.New("invalid active character alteration")

// ActiveCharacterAlteration describes one atomic character update.
type ActiveCharacterAlteration struct {
	Slot               int
	ExpectedOccupant   string
	Strength           *int
	Dexterity          *int
	Intelligence       *int
	CurrentHP          *int
	CurrentMP          *int
	Experience         *int
	Level              *int
	Status             *party_state.CharacterStatus
	ReorderDestination *int
}

// ActiveCharacterSnapshot captures the editable state of one active character.
type ActiveCharacterSnapshot struct {
	Slot         int
	Name         string
	Class        party_state.CharacterClass
	Status       party_state.CharacterStatus
	Strength     byte
	Dexterity    byte
	Intelligence byte
	CurrentMP    byte
	CurrentHP    uint16
	MaximumHP    uint16
	Experience   uint16
	Level        byte
}

// ActivePartyCharacters returns editable snapshots in current party order.
func (g *GameState) ActivePartyCharacters() []ActiveCharacterSnapshot {
	characters := make([]ActiveCharacterSnapshot, 0, party_state.MaxActiveParty)
	for slot := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(slot)
		if character == nil || !character.IsInParty() {
			continue
		}
		characters = append(characters, snapshotActiveCharacter(slot, character))
	}

	return characters
}

// ActivePartySlotSnapshot identifies one occupied active-party slot.
type ActivePartySlotSnapshot struct {
	Slot int    `json:"slot"`
	Name string `json:"name"`
}

// ActiveCharacterAlterationResult reports the exact state before and after an update.
type ActiveCharacterAlterationResult struct {
	CharacterBefore ActiveCharacterSnapshot
	CharacterAfter  ActiveCharacterSnapshot
	OrderBefore     []ActivePartySlotSnapshot
	OrderAfter      []ActivePartySlotSnapshot
}

// AlterActiveCharacter validates and applies one atomic character update.
// During combat, only current HP and MP may be changed so tactical structures remain stable.
func (g *GameState) AlterActiveCharacter(input ActiveCharacterAlteration) (ActiveCharacterAlterationResult, error) {
	var empty ActiveCharacterAlterationResult
	if g.IsInCombat() && !isCombatResourceAlteration(input) {
		return empty, fmt.Errorf("%w: only current HP and MP may be altered during combat", ErrInvalidCharacterAlteration)
	}
	target, err := g.validateCharacterAlteration(input)
	if err != nil {
		return empty, err
	}

	result := ActiveCharacterAlterationResult{
		CharacterBefore: snapshotActiveCharacter(input.Slot, target),
		CharacterAfter:  snapshotActiveCharacter(input.Slot, target),
		OrderBefore:     g.snapshotActivePartyOrder(),
		OrderAfter:      nil,
	}
	if input.Strength != nil {
		target.Strength = byte(*input.Strength) //nolint:gosec // Validated stat range is 1..30.
	}
	if input.Dexterity != nil {
		target.Dexterity = byte(*input.Dexterity) //nolint:gosec // Validated stat range is 1..30.
	}
	if input.Intelligence != nil {
		target.Intelligence = byte(*input.Intelligence) //nolint:gosec // Validated stat range is 1..30.
		if input.CurrentMP == nil {
			target.SetMp(target.GetCurrentMp())
		}
	}
	if input.CurrentMP != nil {
		target.SetMp(byte(*input.CurrentMP)) //nolint:gosec // Validated MP range is 0..30.
	}
	if input.Level != nil {
		target.Level = byte(*input.Level) //nolint:gosec // Validated level range is 1..8.
		target.Exp = uint16(minimumExperienceForLevel(*input.Level))
		target.SetMaxHp(uint16(*input.Level * maximumHPPerLevel)) //nolint:gosec // Validated level range is 1..8.
	}
	if input.CurrentHP != nil {
		target.SetHp(uint16(*input.CurrentHP)) //nolint:gosec // Validated against the stored maximum HP.
	}
	if input.Experience != nil {
		target.Exp = uint16(*input.Experience) //nolint:gosec // Validated experience range is 0..9999.
	}
	if input.Status != nil {
		target.SetStatus(*input.Status)
	}

	finalSlot := input.Slot
	if input.ReorderDestination != nil && *input.ReorderDestination != input.Slot {
		activePlayer := g.GetActivePlayer()
		if err := g.PartyState.SwapMembers(input.Slot, *input.ReorderDestination); err != nil {
			panic("validated active character reorder failed: " + err.Error())
		}
		g.restoreActivePlayerAfterSwap(activePlayer, input.Slot, *input.ReorderDestination)
		finalSlot = *input.ReorderDestination
	}
	result.CharacterAfter = snapshotActiveCharacter(finalSlot, target)
	result.OrderAfter = g.snapshotActivePartyOrder()

	return result, nil
}

func isCombatResourceAlteration(input ActiveCharacterAlteration) bool {
	return (input.CurrentHP != nil || input.CurrentMP != nil) &&
		input.Strength == nil && input.Dexterity == nil && input.Intelligence == nil &&
		input.Experience == nil && input.Level == nil && input.Status == nil && input.ReorderDestination == nil
}

func (g *GameState) restoreActivePlayerAfterSwap(activePlayer *party_state.PlayerCharacter, source, destination int) {
	if activePlayer == nil {
		return
	}
	switch activePlayer {
	case g.PartyState.GetCharacter(source):
		g.activePlayerIndex = source
	case g.PartyState.GetCharacter(destination):
		g.activePlayerIndex = destination
	}
}

func (g *GameState) validateCharacterAlteration(input ActiveCharacterAlteration) (*party_state.PlayerCharacter, error) {
	target, err := g.validateCharacterAlterationTarget(input)
	if err != nil {
		return nil, err
	}
	if !hasCharacterAlteration(input) {
		return nil, fmt.Errorf("%w: at least one change is required", ErrInvalidCharacterAlteration)
	}
	if err := validateAlteredStat("strength", input.Strength); err != nil {
		return nil, err
	}
	if err := validateAlteredStat("dexterity", input.Dexterity); err != nil {
		return nil, err
	}
	if err := validateAlteredStat("intelligence", input.Intelligence); err != nil {
		return nil, err
	}
	if err := validateAlteredResources(target, input); err != nil {
		return nil, err
	}
	if err := validateAlteredStatus(target, input.Status); err != nil {
		return nil, err
	}
	if err := g.validateAlteredPartyPosition(input.Slot, input.ReorderDestination); err != nil {
		return nil, err
	}

	return target, nil
}

func (g *GameState) validateCharacterAlterationTarget(
	input ActiveCharacterAlteration,
) (*party_state.PlayerCharacter, error) {
	if input.Slot < 0 || input.Slot >= party_state.MaxActiveParty {
		return nil, fmt.Errorf(
			"%w: character slot %d is outside the active party",
			ErrInvalidCharacterAlteration,
			input.Slot+1,
		)
	}
	target := g.PartyState.GetCharacter(input.Slot)
	if target == nil || !target.IsInParty() {
		return nil, fmt.Errorf("%w: character slot %d is not occupied", ErrInvalidCharacterAlteration, input.Slot+1)
	}
	if input.ExpectedOccupant == "" || target.GetNameAsString() != input.ExpectedOccupant {
		return nil, fmt.Errorf("%w: character slot %d changed after selection", ErrInvalidCharacterAlteration, input.Slot+1)
	}

	return target, nil
}

func hasCharacterAlteration(input ActiveCharacterAlteration) bool {
	return input.Strength != nil || input.Dexterity != nil || input.Intelligence != nil ||
		input.CurrentHP != nil || input.CurrentMP != nil || input.Experience != nil || input.Level != nil ||
		input.Status != nil || input.ReorderDestination != nil
}

func validateAlteredResources(target *party_state.PlayerCharacter, input ActiveCharacterAlteration) error {
	if input.Level != nil && (*input.Level < 1 || *input.Level > maximumCharacterLevel) {
		return fmt.Errorf(
			"%w: level must be between 1 and %d",
			ErrInvalidCharacterAlteration,
			maximumCharacterLevel,
		)
	}
	if input.Level != nil && input.Experience != nil {
		return fmt.Errorf("%w: level and experience cannot be set together", ErrInvalidCharacterAlteration)
	}
	if input.CurrentHP != nil {
		if target.IsDead() {
			return fmt.Errorf("%w: dead characters require resurrection", ErrInvalidCharacterAlteration)
		}
		maximumHP := int(target.GetMaxHp())
		if input.Level != nil {
			maximumHP = *input.Level * maximumHPPerLevel
		}
		if *input.CurrentHP < 1 || *input.CurrentHP > maximumHP {
			return fmt.Errorf(
				"%w: current HP must be between 1 and %d",
				ErrInvalidCharacterAlteration,
				maximumHP,
			)
		}
	}
	proposedIntelligence := int(target.Intelligence)
	if input.Intelligence != nil {
		proposedIntelligence = *input.Intelligence
	}
	if input.CurrentMP != nil && (*input.CurrentMP < 0 || *input.CurrentMP > proposedIntelligence) {
		return fmt.Errorf(
			"%w: current MP must be between 0 and %d",
			ErrInvalidCharacterAlteration,
			proposedIntelligence,
		)
	}
	if input.Experience != nil && (*input.Experience < 0 || *input.Experience > MaxExperienceValue) {
		return fmt.Errorf(
			"%w: experience must be between 0 and %d",
			ErrInvalidCharacterAlteration,
			MaxExperienceValue,
		)
	}

	return nil
}

func minimumExperienceForLevel(level int) int {
	if level <= 1 {
		return 0
	}

	return 100 << (level - 2)
}

func validateAlteredStatus(target *party_state.PlayerCharacter, status *party_state.CharacterStatus) error {
	if status == nil {
		return nil
	}
	if target.IsDead() {
		return fmt.Errorf("%w: dead characters require resurrection", ErrInvalidCharacterAlteration)
	}
	switch *status {
	case party_state.Good, party_state.Poisoned, party_state.Sleep, party_state.Charmed:
		return nil
	case party_state.Dead:
		return fmt.Errorf("%w: death requires a synchronized alteration", ErrInvalidCharacterAlteration)
	default:
		return fmt.Errorf("%w: status %q is not a safe alteration", ErrInvalidCharacterAlteration, byte(*status))
	}
}

func validateAlteredStat(name string, value *int) error {
	if value != nil && (*value < MinStatValue || *value > MaxStatValue) {
		return fmt.Errorf("%w: %s must be between %d and %d", ErrInvalidCharacterAlteration, name, MinStatValue, MaxStatValue)
	}

	return nil
}

func (g *GameState) validateAlteredPartyPosition(source int, destination *int) error {
	if destination == nil || *destination == source {
		return nil
	}
	if source == party_state.AvatarCharacterIndex || *destination == party_state.AvatarCharacterIndex {
		return fmt.Errorf("%w: the Avatar must remain in party position 1", ErrInvalidCharacterAlteration)
	}
	if *destination < 0 || *destination >= party_state.MaxActiveParty {
		return fmt.Errorf("%w: party position %d is outside the active party", ErrInvalidCharacterAlteration, *destination+1)
	}
	character := g.PartyState.GetCharacter(*destination)
	if character == nil || !character.IsInParty() {
		return fmt.Errorf("%w: party position %d is not occupied", ErrInvalidCharacterAlteration, *destination+1)
	}

	return nil
}

func snapshotActiveCharacter(slot int, character *party_state.PlayerCharacter) ActiveCharacterSnapshot {
	return ActiveCharacterSnapshot{
		Slot:         slot,
		Name:         character.GetNameAsString(),
		Class:        character.Class,
		Status:       character.GetStatus(),
		Strength:     character.Strength,
		Dexterity:    character.Dexterity,
		Intelligence: character.Intelligence,
		CurrentMP:    character.GetCurrentMp(),
		CurrentHP:    character.GetCurrentHp(),
		MaximumHP:    character.GetMaxHp(),
		Experience:   character.Exp,
		Level:        character.Level,
	}
}

func (g *GameState) snapshotActivePartyOrder() []ActivePartySlotSnapshot {
	order := make([]ActivePartySlotSnapshot, 0, party_state.MaxActiveParty)
	for slot := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(slot)
		if character == nil || !character.IsInParty() {
			continue
		}
		order = append(order, ActivePartySlotSnapshot{Slot: slot, Name: character.GetNameAsString()})
	}

	return order
}
