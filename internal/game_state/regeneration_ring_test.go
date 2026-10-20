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

func TestProcessRegenerationRingSweepUsesActivePartyOrderAndReportsEveryWearer(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetStatus(party_state.Good)
		character.SetHp(character.GetMaxHp())
		character.SetEquippedItem(references.SlotRing, references.NoEquipment)
	}

	first := state.PartyState.GetCharacter(0)
	first.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	first.SetHp(first.GetMaxHp() - 2)
	dead := state.PartyState.GetCharacter(1)
	dead.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	dead.SetHp(0)
	dead.SetStatus(party_state.Dead)
	capped := state.PartyState.GetCharacter(3)
	capped.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	miss := state.PartyState.GetCharacter(4)
	miss.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	miss.SetHp(miss.GetMaxHp() - 2)
	inactive := state.PartyState.GetCharacter(5)
	inactive.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	inactive.PartyMap = party_state.PartyMapUnrecruited

	seed := regenerationSeedForRolls(t, 7, 7, 0)
	state.SetRandomSeed(seed)

	results := state.ProcessRegenerationRingSweep()

	require.Len(t, results, 3)
	assert.Equal(t, []int{0, 3, 4}, []int{
		results[0].CharacterIndex,
		results[1].CharacterIndex,
		results[2].CharacterIndex,
	})
	assert.Equal(t, first.GetNameAsString(), results[0].CharacterName)
	assert.Equal(t, results[0].HPBefore+1, results[0].HPAfter)
	assert.True(t, results[0].RollSucceeded)
	assert.Equal(t, capped.GetMaxHp(), results[1].HPBefore)
	assert.Equal(t, capped.GetMaxHp(), results[1].HPAfter)
	assert.True(t, results[1].RollSucceeded)
	assert.Equal(t, results[2].HPBefore, results[2].HPAfter)
	assert.False(t, results[2].RollSucceeded)
	assert.Equal(t, uint16(0), dead.GetCurrentHp())
	assert.Equal(t, uint64(3), state.GetRNG().Calls())
}

func TestOrdinarySurfaceCompletionRunsOneRegenerationRingSweep(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	state.DebugOptions.MonsterGen = false
	character := state.PartyState.GetAvatar()
	character.SetStatus(party_state.Good)
	character.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	character.SetHp(character.GetMaxHp() - 2)
	seed := regenerationSeedForRolls(t, 7)
	state.SetRandomSeed(seed)

	state.completeSurfaceCycleWithActorPhase(references.LargeMapType, 1, false)

	assert.Equal(t, character.GetMaxHp()-1, character.GetCurrentHp())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestCombatTurnEffectsDoNotRunNoncombatRegenerationRingSweep(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	character := state.PartyState.GetAvatar()
	character.SetStatus(party_state.Good)
	character.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	character.SetHp(character.GetMaxHp() - 2)
	before := character.GetCurrentHp()
	state.MapState.PlayerLocation.Location = references.CombatLocation
	state.SetRandomSeed(regenerationSeedForRolls(t, 7))

	state.processNonCombatPlayerTurnEffects()

	assert.Equal(t, before, character.GetCurrentHp())
	assert.Zero(t, state.GetRNG().Calls())
}

func TestOrdinaryDungeonCompletionRunsOneRegenerationRingSweep(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithPlayerAt(1, 1).
		WithSystemCallbacks().
		Build()
	character := state.PartyState.GetAvatar()
	character.SetStatus(party_state.Good)
	character.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	character.SetHp(character.GetMaxHp() - 2)
	state.SetRandomSeed(regenerationSeedForRolls(t, 7))

	state.processDungeonEndPlayerFinalEffects()

	assert.Equal(t, character.GetMaxHp()-1, character.GetCurrentHp())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestOutdoorHoleUpIterationRunsOneRegenerationRingSweep(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(42, 73).
		WithSystemCallbacks().
		Build()
	for index := range party_state.MaxActiveParty {
		member := state.PartyState.GetCharacter(index)
		member.PartyMap = party_state.PartyMapActive
		member.SetStatus(party_state.Good)
		member.SetHp(member.GetMaxHp())
	}
	character := state.PartyState.GetAvatar()
	character.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	character.SetHp(character.GetMaxHp() - 2)
	scene := camping.NewCampingScene(1, NoGuardAssigned, state.MapState.PlayerLocation)
	campState := state.newCampingState(scene, true, false)
	seed := regenerationSeedForRolls(t, 7)
	state.SetRandomSeed(seed)

	assert.False(t, state.processSingleCampingCycle(scene, campState))

	assert.Equal(t, character.GetMaxHp()-1, character.GetCurrentHp())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestDungeonHoleUpIterationRunsOneRegenerationRingSweep(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithPlayerAt(1, 1).
		WithSystemCallbacks().
		Build()
	character := state.PartyState.GetAvatar()
	character.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	character.SetHp(character.GetMaxHp() - 2)
	scene := camping.NewDungeonCampingScene(1, NoGuardAssigned, state.MapState.PlayerLocation, nil)
	campState := state.newCampingState(scene, false, false)
	seed := regenerationSeedForRolls(t, 7)
	state.SetRandomSeed(seed)

	assert.False(t, state.processDungeonCampingCycle(scene, campState, &DungeonCampingPlan{}))

	assert.Equal(t, character.GetMaxHp()-1, character.GetCurrentHp())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func regenerationSeedForRolls(t *testing.T, rolls ...int) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 100_000; seed++ {
		rng := beradrng.New(seed)
		matches := true
		for _, roll := range rolls {
			if rng.Intn(regenerationRingRollMaximum+1) != roll {
				matches = false
				break
			}
		}
		if matches {
			return seed
		}
	}
	t.Fatal("unable to find deterministic regeneration-ring seed")

	return 0
}
