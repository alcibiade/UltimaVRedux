package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type earthquakeResultRecorder struct {
	results []EarthquakeResult
}

func (r *earthquakeResultRecorder) PresentEarthquake(result EarthquakeResult) {
	r.results = append(r.results, result)
}

func TestEarthquakeSystemCommitsTypedDamageInActiveSlotOrder(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapUnrecruited
		character.SetStatus(party_state.Good)
		character.SetHp(20)
	}
	first := state.PartyState.GetCharacter(0)
	first.PartyMap = party_state.PartyMapActive
	first.SetHp(3)
	second := state.PartyState.GetCharacter(2)
	second.PartyMap = party_state.PartyMapActive
	second.SetStatus(party_state.Poisoned)
	second.SetHp(20)
	dead := state.PartyState.GetCharacter(1)
	dead.PartyMap = party_state.PartyMapActive
	dead.SetHp(0)
	dead.SetStatus(party_state.Dead)

	seed := earthquakeTriggerSeed(t)
	rng := deterministic_random.New(seed)
	expected := deterministic_random.New(seed)
	require.Equal(t, 0x69, expected.Intn(256))
	wantFirst := expected.Intn(8) + 1
	wantSecond := expected.Intn(8) + 1
	recorder := &earthquakeResultRecorder{}
	system := NewEarthquakeSystem(rng, recorder)

	result, triggered := system.CheckForRandomEarthquake(true, &state.PartyState)

	require.True(t, triggered)
	require.Len(t, result.Damage, 2)
	assert.Equal(t, []int{0, 2}, []int{result.Damage[0].CharacterIndex, result.Damage[1].CharacterIndex})
	assert.Equal(t, []int{wantFirst, wantSecond}, []int{result.Damage[0].Damage, result.Damage[1].Damage})
	assert.Equal(t, max(3-wantFirst, 0), int(first.GetCurrentHp()))
	if wantFirst >= 3 {
		assert.Equal(t, party_state.Dead, first.GetStatus())
	}
	assert.Equal(t, 20-wantSecond, int(second.GetCurrentHp()))
	assert.Equal(t, party_state.Poisoned, second.GetStatus())
	assert.Equal(t, result, recorder.results[0])
	assert.Equal(t, uint64(3), rng.Calls())
}

func TestConsumedUnderworldSurfaceTurnOwnsOneEarthquakeAndCleanup(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.UNDERWORLD)).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	state.DebugOptions.MonsterGen = false
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapUnrecruited
	}
	avatar := state.PartyState.GetAvatar()
	avatar.PartyMap = party_state.PartyMapActive
	avatar.SetStatus(party_state.Good)
	avatar.SetHp(1)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 2}
	state.SetRandomSeed(earthquakeTriggerSeed(t))
	deathCalls := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deathCalls++ }
	callbacks.Reset()

	state.completeSurfaceCycleWithActorPhase(references.LargeMapType, 2, false)

	assert.Contains(t, callbacks.Messages, "EARTHQUAKE!")
	assert.Zero(t, avatar.GetCurrentHp())
	assert.Equal(t, party_state.Dead, avatar.GetStatus())
	assert.True(t, state.EventQueueManager.HasFeedback(earthquakeRandomFeedbackName))
	assert.Equal(t, 1, deathCalls)
	state.EventQueueManager.UpdateFeedback()
	state.ElapsedMs += earthquakeRandomDurationMs
	for range 4 {
		state.EventQueueManager.UpdateFeedback()
	}
	assert.False(t, state.EventQueueManager.HasFeedback(earthquakeRandomFeedbackName))
	require.NotEmpty(t, callbacks.ScreenShakeOffsets)
	assert.Equal(t, ScreenShakeOffsetCall{}, callbacks.ScreenShakeOffsets[len(callbacks.ScreenShakeOffsets)-1])
}

func TestEarthquakeEligibilityConsumesNoRNGOutsideUnderworld(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.SetRandomSeed(earthquakeTriggerSeed(t))
	before := state.GetRNG().Calls()

	_, triggered := state.earthquakeSystem.CheckForRandomEarthquake(false, &state.PartyState)

	assert.False(t, triggered)
	assert.Equal(t, before, state.GetRNG().Calls())
}

func earthquakeTriggerSeed(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 100_000; seed++ {
		if deterministic_random.New(seed).Intn(256) == 0x69 {
			return seed
		}
	}
	t.Fatal("unable to find earthquake trigger seed")
	return 0
}
