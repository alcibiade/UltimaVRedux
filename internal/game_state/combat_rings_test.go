package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestCombatRingInitializationUsesLivingPartyOrderAndSleepGate(t *testing.T) {
	state, callbacks := newCombatRingTestState(t)
	if state == nil {
		return
	}
	dead := activateCombatRingTestCharacter(state, 0, references.RingInvisibility, party_state.Dead)
	sleeping := activateCombatRingTestCharacter(state, 1, references.RingInvisibility, party_state.Sleep)
	awake := activateCombatRingTestCharacter(state, 2, references.RingInvisibility, party_state.Good)
	seed := findCombatRingSeed(t, func(rng *beradrng.Source) bool {
		return rng.Intn(combatRingDisappearanceRollMaximum+1) == combatRingDisappearanceRollResult &&
			rng.Intn(combatRingDisappearanceRollMaximum+1) != combatRingDisappearanceRollResult
	})
	state.SetRandomSeed(seed)
	combat := newCombatRingState(state)
	callbacks.Reset()

	state.initializeCombatPartyRings(combat)

	assert.Equal(t, references.RingInvisibility, dead.GetEquippedItem(references.SlotRing),
		"dead wearers neither roll nor lose their ring")
	assert.Equal(t, references.NoEquipment, sleeping.GetEquippedItem(references.SlotRing))
	assert.False(t, combat.IsPartyMemberInvisible(1), "a sleeping wearer never becomes invisible")
	assert.Equal(t, references.RingInvisibility, awake.GetEquippedItem(references.SlotRing))
	assert.True(t, combat.IsPartyMemberInvisible(2))
	assert.Equal(t, []string{"A ring has vanished!"}, callbacks.Messages)
	assert.Equal(t, uint64(2), state.GetRNG().Calls())
}

func TestCombatRingInitializationRunsNestedRegenerationSweeps(t *testing.T) {
	state, _ := newCombatRingTestState(t)
	if state == nil {
		return
	}
	first := activateCombatRingTestCharacter(state, 0, references.RingRegeneration, party_state.Good)
	sleeping := activateCombatRingTestCharacter(state, 1, references.RingRegeneration, party_state.Sleep)
	capped := activateCombatRingTestCharacter(state, 2, references.RingRegeneration, party_state.Good)
	first.SetHp(first.GetMaxHp() - 2)
	sleeping.SetHp(sleeping.GetMaxHp() - 2)
	capped.SetHp(capped.GetMaxHp())
	seed := findCombatRingSeed(t, func(rng *beradrng.Source) bool {
		if rng.Intn(16) == combatRingDisappearanceRollResult {
			return false
		}
		if rng.Intn(8) != 7 || rng.Intn(8) != 7 || rng.Intn(8) != 7 {
			return false
		}
		if rng.Intn(16) == combatRingDisappearanceRollResult ||
			rng.Intn(16) == combatRingDisappearanceRollResult {
			return false
		}
		_ = rng.Intn(8)
		_ = rng.Intn(8)
		_ = rng.Intn(8)

		return true
	})
	expected := beradrng.New(seed)
	_ = expected.Intn(16)
	firstSweep := [3]int{expected.Intn(8), expected.Intn(8), expected.Intn(8)}
	_ = expected.Intn(16)
	_ = expected.Intn(16)
	secondSweep := [3]int{expected.Intn(8), expected.Intn(8), expected.Intn(8)}
	state.SetRandomSeed(seed)
	combat := newCombatRingState(state)

	state.initializeCombatPartyRings(combat)

	assert.Equal(t, first.GetMaxHp()-2+successfulRingRolls(firstSweep[0], secondSweep[0]), first.GetCurrentHp())
	assert.Equal(t, sleeping.GetMaxHp()-2+successfulRingRolls(firstSweep[1], secondSweep[1]), sleeping.GetCurrentHp(),
		"sleeping wearers participate in another wearer's global sweep")
	assert.Equal(t, capped.GetMaxHp(), capped.GetCurrentHp(), "capped wearers still consume their rolls")
	assert.Equal(t, uint64(9), state.GetRNG().Calls(),
		"three disappearance checks and two three-wearer sweeps must run in source order")
}

func TestCombatRingInitializationPrecedesMonsterDexterity(t *testing.T) {
	state, _ := newCombatRingTestState(t)
	if state == nil {
		return
	}
	activateCombatRingTestCharacter(state, 0, references.RingInvisibility, party_state.Good)
	seed := findCombatRingSeed(t, func(rng *beradrng.Source) bool {
		return rng.Intn(16) != combatRingDisappearanceRollResult
	})
	expected := beradrng.New(seed)
	_ = expected.Intn(16)
	expectedDexModifier := expected.Intn(dexModifierRange) + dexModifierMin
	request := newComposerRequest()
	request.CountMode = CombatEncounterCountExact
	request.ExactCount = 1
	request.FriendsEnabled = false
	request.Seed = &seed

	_, err := state.StartCombatEncounter(request)

	require.NoError(t, err)
	monster := state.CombatScene.MonsterManager.GetMonster(0)
	require.NotNil(t, monster)
	assert.Equal(t, expectedDexModifier, monster.dexModifier)
	assert.True(t, state.CombatScene.IsPartyMemberInvisible(0))
	assert.Equal(t, expected.Calls(), state.GetRNG().Calls())
}

func TestOverworldCombatValidationPrecedesRingDisappearance(t *testing.T) {
	state, callbacks := newCombatRingTestState(t)
	if state == nil {
		return
	}
	character := activateCombatRingTestCharacter(state, 0, references.RingInvisibility, party_state.Good)
	source := addOverworldTestEnemy(t, state, references.MonsterOrc, references.Position{X: 51, Y: 50})
	enemyReferences := append(references.EnemyReferences(nil), (*state.GameReferences.EnemyReferences)...)
	enemyReferences[references.MonsterOrc].MaxPerMap = 0
	gameReferences := *state.GameReferences
	gameReferences.EnemyReferences = &enemyReferences
	state.GameReferences = &gameReferences
	seed := findCombatRingSeed(t, func(rng *beradrng.Source) bool {
		return rng.Intn(combatRingDisappearanceRollMaximum+1) == combatRingDisappearanceRollResult
	})
	state.SetRandomSeed(seed)
	callbacks.Reset()

	err := state.StartOverworldCombatEncounter(source)

	assert.ErrorIs(t, err, ErrInvalidOverworldEncounter)
	assert.Nil(t, state.CombatScene)
	assert.Equal(t, references.RingInvisibility, character.GetEquippedItem(references.SlotRing))
	assert.Zero(t, state.GetRNG().Calls())
	assert.Empty(t, callbacks.Messages)
}

func TestCombatActionEndRunsOneRegenerationSweepBeforeFields(t *testing.T) {
	state, _ := newCombatRingTestState(t)
	if state == nil {
		return
	}
	activateCombatRingTestCharacter(state, 0, references.NoEquipment, party_state.Good)
	activateCombatRingTestCharacter(state, 1, references.NoEquipment, party_state.Good)
	require.NoError(t, state.TransitionToCombatMap(
		0, references.BritanniaCombatTerritory, CombatEntryNorth,
	))
	first := state.PartyState.GetCharacter(0)
	second := state.PartyState.GetCharacter(1)
	first.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	second.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	first.SetHp(first.GetMaxHp() - 2)
	second.SetHp(second.GetMaxHp() - 2)
	fieldPosition := references.Position{X: 10, Y: 10}
	if state.CombatScene.GetPartyMemberPosition(0) == fieldPosition {
		fieldPosition = references.Position{}
	}
	require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldSleep, fieldPosition)))
	seed := findCombatRingSeed(t, func(rng *beradrng.Source) bool {
		return rng.Intn(8) == 7 && rng.Intn(8) == 7 && rng.Intn(256) >= 16
	})
	state.SetRandomSeed(seed)

	state.finishCombatAction()

	assert.Equal(t, first.GetMaxHp()-1, first.GetCurrentHp())
	assert.Equal(t, second.GetMaxHp()-1, second.GetCurrentHp())
	assert.Len(t, state.CombatScene.fields, 1)
	assert.Equal(t, uint64(3), state.GetRNG().Calls())
}

func TestCombatInvisibilityRingPersistsForEncounterAndClearsOnExit(t *testing.T) {
	state, _ := newCombatRingTestState(t)
	if state == nil {
		return
	}
	activateCombatRingTestCharacter(state, 0, references.NoEquipment, party_state.Good)
	require.NoError(t, state.TransitionToCombatMap(
		0, references.BritanniaCombatTerritory, CombatEntryNorth,
	))
	character := state.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotRing, references.RingInvisibility)
	identity := state.CombatScene.GetCurrentTurnIdentity()
	randomCalls := state.GetRNG().Calls()

	state.finishCombatAction()

	assert.True(t, state.CombatScene.IsCombatPCInvisible(identity))
	assert.False(t, findNearestPC(references.Position{X: 5, Y: 5}, state.CombatScene).found)
	assert.Equal(t, randomCalls, state.GetRNG().Calls())
	state.TransitionFromCombatMap()
	assert.Nil(t, state.CombatScene)
	assert.Equal(t, references.RingInvisibility, character.GetEquippedItem(references.SlotRing))
}

func TestCampingRingInitializationPrecedesAmbushRNGAndTransfersVisibility(t *testing.T) {
	state, _ := newCombatRingTestState(t)
	if state == nil {
		return
	}
	activateCombatRingTestCharacter(state, 0, references.RingInvisibility, party_state.Good)
	seed := findCombatRingSeed(t, func(rng *beradrng.Source) bool {
		return rng.Intn(16) != combatRingDisappearanceRollResult
	})
	expected := beradrng.New(seed)
	_ = expected.Intn(16)
	expectedAmbushRoll := expected.Intn(64)
	state.SetRandomSeed(seed)
	scene := camping.NewCampingScene(1, NoGuardAssigned, state.MapState.PlayerLocation)
	camp := state.newCampingState(scene, false, false)

	state.initializeCampingPartyRings(camp)
	actualAmbushRoll := state.RandomIntInRange(0, 63)

	assert.Equal(t, expectedAmbushRoll, actualAmbushRoll)
	assert.True(t, camp.partyInvisible[0])
	combat := newCombatRingState(state)
	inheritCampingPartyRings(combat, camp)
	assert.True(t, combat.IsPartyMemberInvisible(0))
}

func newCombatRingTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	if state == nil {
		return nil, callbacks
	}
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapUnrecruited
		character.SetStatus(party_state.Good)
		character.SetHp(character.GetMaxHp())
		character.SetEquippedItem(references.SlotRing, references.NoEquipment)
	}

	return state, callbacks
}

func activateCombatRingTestCharacter(
	state *GameState,
	index int,
	ring references.Equipment,
	status party_state.CharacterStatus,
) *party_state.PlayerCharacter {
	character := state.PartyState.GetCharacter(index)
	character.PartyMap = party_state.PartyMapActive
	character.SetStatus(status)
	character.SetEquippedItem(references.SlotRing, ring)
	if status == party_state.Dead {
		character.SetHp(0)
	}

	return character
}

func newCombatRingState(state *GameState) *CombatState {
	combatMap := references.NewCombatMapReference(0, references.BritanniaCombatTerritory)
	combat := NewCombatState(combatMap, CombatEntryNorth, &state.PartyState)
	state.configureCombatTurnQueue(combat)

	return combat
}

func findCombatRingSeed(t *testing.T, matches func(*beradrng.Source) bool) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 1_000_000; seed++ {
		if matches(beradrng.New(seed)) {
			return seed
		}
	}
	t.Fatal("unable to find deterministic combat-ring seed")

	return 0
}

func successfulRingRolls(rolls ...int) uint16 {
	var successes uint16
	for _, roll := range rolls {
		if roll == regenerationRingRollMaximum {
			successes++
		}
	}

	return successes
}
