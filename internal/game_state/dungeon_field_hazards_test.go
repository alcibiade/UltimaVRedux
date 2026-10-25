package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDungeonSleepFieldsConsumeFieldAndReplaceNonDeadStatuses(t *testing.T) {
	for _, cell := range []references.DungeonCell{0x80, 0x88} {
		t.Run(fmt.Sprintf("%02x", cell.Raw()), func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, cell, 31)
			characters := configureDungeonHazardParty(
				t,
				state,
				party_state.Good,
				party_state.Poisoned,
				party_state.Dead,
			)
			beforeRNG := state.GetRNG().Calls()
			callbacks.Reset()

			completeDungeonHazardCommand(state)

			assert.True(t, characters[0].IsAsleep())
			assert.True(t, characters[1].IsAsleep(), "sleep field replaces poison")
			assert.True(t, characters[2].IsDead())
			assert.Equal(t, references.DungeonCell(cell.Raw()&0x08), currentDungeonInteractionCell(t, state))
			assert.Equal(t, uint64(len(characters)), state.GetRNG().Calls()-beforeRNG,
				"dead active members still consume a save roll")
			assert.Equal(t, []string{"Sleep spell!"}, callbacks.Messages)
			assert.Equal(t, 2, countEqual(callbacks.SoundEffectsPlayed, SoundWhiteNoiseDamage))
		})
	}
}

func TestDungeonPoisonFieldsPersistAndTickInSameCompletion(t *testing.T) {
	for _, cell := range []references.DungeonCell{0x81, 0x89} {
		t.Run(fmt.Sprintf("%02x", cell.Raw()), func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, cell, 37)
			characters := configureDungeonHazardParty(
				t,
				state,
				party_state.Good,
				party_state.Poisoned,
				party_state.Dead,
			)
			beforeRNG := state.GetRNG().Calls()
			callbacks.Reset()

			completeDungeonHazardCommand(state)

			assert.True(t, characters[0].IsPoisoned())
			assert.True(t, characters[1].IsPoisoned())
			assert.True(t, characters[2].IsDead())
			assert.Equal(t, uint16(99), characters[0].GetCurrentHp())
			assert.Equal(t, uint16(99), characters[1].GetCurrentHp())
			assert.Equal(t, cell, currentDungeonInteractionCell(t, state))
			assert.Equal(t, uint64(len(characters)), state.GetRNG().Calls()-beforeRNG)
			assert.Equal(t, []string{"Poison!"}, callbacks.Messages)
			assert.Zero(t, countEqual(callbacks.SoundEffectsPlayed, SoundWhiteNoiseDamage))
			assert.Equal(t, 1, countEqual(callbacks.SoundEffectsPlayed, SoundPoison),
				"resolved party-wide poison injury requests one semantic cue")
		})
	}
}

func TestDungeonCompletionPoisonsBeforeProvisionResolution(t *testing.T) {
	for _, testCase := range []struct {
		name string
		cell references.DungeonCell
	}{
		{name: "existing poison", cell: 0},
		{name: "new poison field", cell: 0x81},
	} {
		for _, food := range []uint16{0, 10} {
			t.Run(fmt.Sprintf("%s/food-%d", testCase.name, food), func(t *testing.T) {
				state, _ := newDungeonInteractionTestState(t, testCase.cell, 37)
				initialStatus := party_state.Poisoned
				if testCase.cell.Raw() == 0x81 {
					initialStatus = party_state.Good
				}
				characters := configureDungeonHazardParty(t, state, initialStatus, party_state.Good)
				characters[0].SetHp(1)
				state.DateTime.Hour, state.DateTime.Minute = 5, 59
				state.lastProcessedHour = 5
				state.PartyState.Inventory.Provisions.Food.Set(food)
				beforeRNG := state.GetRNG().Calls()

				completeDungeonHazardCommand(state)

				assert.True(t, characters[0].IsDead(), "poison resolves before provisions")
				if food > 0 {
					assert.Equal(t, uint16(8), state.PartyState.Inventory.Provisions.Food.Get(),
						"the cached poisoned diner remains eligible for this update")
				} else {
					fieldRolls := uint64(0)
					if testCase.cell.Raw() == 0x81 {
						fieldRolls = 2
					}
					assert.Equal(t, fieldRolls+1, state.GetRNG().Calls()-beforeRNG,
						"starvation skips the member killed by the preceding poison step")
				}
			})
		}
	}
}

func TestDungeonPoisonFieldReplacesSleep(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x81, 41)
	characters := configureDungeonHazardParty(t, state, party_state.Sleep)

	state.processDungeonCurrentCellField()

	assert.True(t, characters[0].IsPoisoned())
}

func TestDungeonFireFieldsDamageEveryNonDeadMemberAndPersist(t *testing.T) {
	for _, cell := range []references.DungeonCell{0x82, 0x8a} {
		t.Run(fmt.Sprintf("%02x", cell.Raw()), func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, cell, 43)
			characters := configureDungeonHazardParty(
				t,
				state,
				party_state.Good,
				party_state.Poisoned,
				party_state.Sleep,
				party_state.Dead,
			)
			beforeRNG := state.GetRNG().Calls()
			callbacks.Reset()

			state.processDungeonCurrentCellField()

			for index := range 3 {
				assert.Less(t, characters[index].GetCurrentHp(), uint16(100))
			}
			assert.Equal(t, uint16(100), characters[3].GetCurrentHp())
			assert.True(t, characters[0].IsGood())
			assert.True(t, characters[1].IsPoisoned())
			assert.True(t, characters[2].IsAsleep())
			assert.True(t, characters[3].IsDead())
			assert.Equal(t, cell, currentDungeonInteractionCell(t, state))
			assert.Equal(t, uint64(3), state.GetRNG().Calls()-beforeRNG)
			assert.Equal(t, []string{"Fire!!"}, callbacks.Messages)
			assert.Zero(t, countEqual(callbacks.SoundEffectsPlayed, SoundWhiteNoiseDamage))
			assert.Equal(t, 1, countEqual(callbacks.SoundEffectsPlayed, SoundFireBurn))
		})
	}
}

func TestDungeonFireFieldContinuesToPartyWipeCheck(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x82, 47)
	characters := configureDungeonHazardParty(t, state, party_state.Good, party_state.Good)
	for _, character := range characters {
		character.SetHp(1)
	}
	deathSequences := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deathSequences++ }

	completeDungeonHazardCommand(state)

	assert.True(t, state.PartyState.IsAllPartyDead())
	assert.Equal(t, 1, deathSequences)
}

func completeDungeonHazardCommand(state *GameState) {
	token, _ := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
}

func configureDungeonHazardParty(
	t *testing.T,
	state *GameState,
	statuses ...party_state.CharacterStatus,
) []*party_state.PlayerCharacter {
	t.Helper()
	require.NotEmpty(t, statuses)
	for index := 0; index < state.PartyState.GetCharacterCount(); index++ {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	characters := make([]*party_state.PlayerCharacter, len(statuses))
	for index, status := range statuses {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetMaxHp(100)
		character.SetHp(100)
		character.SetStatus(status)
		character.Dexterity = 1
		characters[index] = character
	}
	state.activeDungeon.replaceWanderer(nil)

	return characters
}

func activeDungeonHazardCharacters(state *GameState) []*party_state.PlayerCharacter {
	characters := make([]*party_state.PlayerCharacter, 0, party_state.MaxActiveParty)
	for index := 0; index < party_state.MaxActiveParty; index++ {
		character := state.PartyState.GetCharacter(index)
		if character.PartyMap == party_state.PartyMapActive {
			character.SetMaxHp(100)
			character.SetHp(100)
			character.SetStatus(party_state.Good)
			characters = append(characters, character)
		}
	}

	return characters
}

func dungeonHazardHP(characters []*party_state.PlayerCharacter) []uint16 {
	hp := make([]uint16, len(characters))
	for index, character := range characters {
		hp[index] = character.GetCurrentHp()
	}

	return hp
}
