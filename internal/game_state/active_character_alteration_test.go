package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAlterActiveCharacterAppliesAtomicSafeChanges(t *testing.T) {
	gameState, callbacks := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	destination := gameState.PartyState.GetCharacter(2)
	target.SetMp(25)
	require.True(t, gameState.SetActivePlayer(2))
	callbacks.Reset()
	strength, dexterity, intelligence := 24, 23, 10
	status := party_state.Poisoned
	destinationSlot := 2

	result, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:               1,
		ExpectedOccupant:   target.GetNameAsString(),
		Strength:           &strength,
		Dexterity:          &dexterity,
		Intelligence:       &intelligence,
		Status:             &status,
		ReorderDestination: &destinationSlot,
	})

	require.NoError(t, err)
	assert.Same(t, target, gameState.PartyState.GetCharacter(2))
	assert.Same(t, destination, gameState.PartyState.GetCharacter(1))
	assert.Equal(t, byte(24), target.Strength)
	assert.Equal(t, byte(23), target.Dexterity)
	assert.Equal(t, byte(10), target.Intelligence)
	assert.Equal(t, byte(10), target.GetCurrentMp())
	assert.Equal(t, party_state.Poisoned, target.GetStatus())
	assert.Same(t, destination, gameState.GetActivePlayer())
	assert.Equal(t, 1, gameState.GetActivePlayerIndex())
	assert.Equal(t, 1, result.CharacterBefore.Slot)
	assert.Equal(t, 2, result.CharacterAfter.Slot)
}

func TestAlterActiveCharacterRejectsInvalidRequestWithoutMutation(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	beforeStrength := target.Strength
	beforeStatus := target.GetStatus()
	invalidStrength := MaxStatValue + 1
	status := party_state.Poisoned

	_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Strength:         &invalidStrength,
		Status:           &status,
	})

	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	assert.Equal(t, beforeStrength, target.Strength)
	assert.Equal(t, beforeStatus, target.GetStatus())
}

func TestAlterActiveCharacterRejectsStaleTargetCombatUnsafeChangesAndAvatarReorder(t *testing.T) {
	t.Run("stale target", func(t *testing.T) {
		gameState, _ := newCharacterAlterationTestState(t)
		strength := 20

		_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
			Slot:             1,
			ExpectedOccupant: "Someone Else",
			Strength:         &strength,
		})

		require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	})

	t.Run("combat unsafe change", func(t *testing.T) {
		gameState, _ := newCharacterAlterationTestState(t)
		target := gameState.PartyState.GetCharacter(1)
		gameState.MapState.PlayerLocation.Location = references.CombatLocation
		strength := 20

		_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
			Slot:             1,
			ExpectedOccupant: target.GetNameAsString(),
			Strength:         &strength,
		})

		require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	})

	t.Run("Avatar reorder", func(t *testing.T) {
		gameState, _ := newCharacterAlterationTestState(t)
		avatar := gameState.PartyState.GetCharacter(0)
		destination := 1

		_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
			Slot:               0,
			ExpectedOccupant:   avatar.GetNameAsString(),
			ReorderDestination: &destination,
		})

		require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	})
}

func TestAlterActiveCharacterAllowsCurrentResourcesDuringCombat(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	target.SetMaxHp(120)
	target.SetHp(80)
	target.Intelligence = 20
	target.SetMp(10)
	gameState.MapState.PlayerLocation.Location = references.CombatLocation
	currentHP, currentMP := 90, 15

	result, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		CurrentHP:        &currentHP,
		CurrentMP:        &currentMP,
	})

	require.NoError(t, err)
	assert.Equal(t, uint16(80), result.CharacterBefore.CurrentHP)
	assert.Equal(t, uint16(90), result.CharacterAfter.CurrentHP)
	assert.Equal(t, byte(10), result.CharacterBefore.CurrentMP)
	assert.Equal(t, byte(15), result.CharacterAfter.CurrentMP)
}

func TestAlterActiveCharacterRejectsUnsafeStatusAndEmptyChange(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	dead := party_state.Dead

	_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Status:           &dead,
	})
	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)

	_, err = gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
	})
	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
}

func TestAlterActiveCharacterSetsExactResourcesWithoutForcingProgression(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	target.SetMaxHp(120)
	target.SetHp(80)
	target.Intelligence = 10
	target.SetMp(10)
	target.Exp = 500
	target.Level = 3
	currentHP, currentMP, experience, intelligence := 90, 25, MaxExperienceValue, 30

	result, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Intelligence:     &intelligence,
		CurrentHP:        &currentHP,
		CurrentMP:        &currentMP,
		Experience:       &experience,
	})

	require.NoError(t, err)
	assert.Equal(t, uint16(90), target.GetCurrentHp())
	assert.Equal(t, byte(25), target.GetCurrentMp())
	assert.Equal(t, uint16(MaxExperienceValue), target.Exp)
	assert.Equal(t, byte(3), target.Level, "exact XP setup must not run level progression")
	assert.Equal(t, uint16(120), target.GetMaxHp())
	assert.Equal(t, uint16(80), result.CharacterBefore.CurrentHP)
}

func TestAlterActiveCharacterSetsAuthoritativeLevelProgressionWithoutHealing(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	target.Level = 3
	target.Exp = 9000
	target.SetMaxHp(90)
	target.SetHp(70)
	level := 6

	result, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Level:            &level,
	})

	require.NoError(t, err)
	assert.Equal(t, byte(6), target.Level)
	assert.Equal(t, uint16(1600), target.Exp)
	assert.Equal(t, uint16(180), target.GetMaxHp())
	assert.Equal(t, uint16(70), target.GetCurrentHp(), "raising level must not heal")
	assert.Equal(t, result.CharacterBefore.Strength, result.CharacterAfter.Strength, "debug progression must not grant random stats")

	level = 2
	result, err = gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Level:            &level,
	})
	require.NoError(t, err)
	assert.Equal(t, byte(2), target.Level)
	assert.Equal(t, uint16(100), target.Exp)
	assert.Equal(t, uint16(60), target.GetMaxHp())
	assert.Equal(t, uint16(60), target.GetCurrentHp(), "lowering level must clamp HP to the new maximum")
	assert.Equal(t, uint16(180), result.CharacterBefore.MaximumHP)
	assert.Equal(t, uint16(60), result.CharacterAfter.MaximumHP)
}

func TestAlterActiveCharacterRejectsInvalidOrAmbiguousLevelWithoutMutation(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	before := snapshotActiveCharacter(1, target)

	invalidLevel := maximumCharacterLevel + 1
	_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Level:            &invalidLevel,
	})
	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)

	level, experience := 4, 400
	_, err = gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Level:            &level,
		Experience:       &experience,
	})
	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	assert.Equal(t, before, snapshotActiveCharacter(1, target))
}

func TestAlterActiveCharacterRejectsInvalidResourcesAtomically(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	target.SetMaxHp(120)
	target.SetHp(80)
	beforeXP := target.Exp
	invalidHP, experience := 0, 9000

	_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		CurrentHP:        &invalidHP,
		Experience:       &experience,
	})

	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	assert.Equal(t, uint16(80), target.GetCurrentHp())
	assert.Equal(t, beforeXP, target.Exp)

	invalidExperience := MaxExperienceValue + 1
	_, err = gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Experience:       &invalidExperience,
	})
	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	assert.Equal(t, beforeXP, target.Exp)

	target.SetStatus(party_state.Dead)
	validHP := 50
	_, err = gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		CurrentHP:        &validHP,
	})
	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	assert.Equal(t, uint16(80), target.GetCurrentHp())
}

func TestAlterActiveCharacterValidatesMPAgainstProposedIntelligence(t *testing.T) {
	gameState, _ := newCharacterAlterationTestState(t)
	target := gameState.PartyState.GetCharacter(1)
	target.Intelligence = 10
	increasedIntelligence, validMP := 20, 18

	_, err := gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Intelligence:     &increasedIntelligence,
		CurrentMP:        &validMP,
	})
	require.NoError(t, err)
	assert.Equal(t, byte(18), target.GetCurrentMp())

	decreasedIntelligence, invalidMP := 5, 6
	_, err = gameState.AlterActiveCharacter(ActiveCharacterAlteration{
		Slot:             1,
		ExpectedOccupant: target.GetNameAsString(),
		Intelligence:     &decreasedIntelligence,
		CurrentMP:        &invalidMP,
	})
	require.ErrorIs(t, err, ErrInvalidCharacterAlteration)
	assert.Equal(t, byte(20), target.Intelligence)
	assert.Equal(t, byte(18), target.GetCurrentMp())
}

func newCharacterAlterationTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	gameState, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	for slot := 0; slot < 3; slot++ {
		character := gameState.PartyState.GetCharacter(slot)
		character.PartyMap = party_state.PartyMapActive
		character.SetStatus(party_state.Good)
	}
	callbacks.Reset()

	return gameState, callbacks
}
