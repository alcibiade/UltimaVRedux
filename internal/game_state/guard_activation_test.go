package game_state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestActivateGuardsProcessesActiveNPCsInOriginalSlotOrder(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	require.True(t, ok)

	guard := guardActivationTestNPC(0, indexes.Guard_KeyIndex, 0, false)
	hidden := guardActivationTestNPC(1, indexes.TownsPerson_KeyIndex, 1, true)
	hidden.SetVisible(false)
	firstCivilian := guardActivationTestNPC(2, indexes.TownsPerson_KeyIndex, 1, true)
	staticCivilian := guardActivationTestNPC(3, indexes.TownsPerson_KeyIndex, 1, false)
	invalidShape := guardActivationTestNPC(4, indexes.Apparation1, 1, true)
	daemon := guardActivationTestNPC(5, indexes.Daemon_KeyIndex, 0, false)
	shadowlord := guardActivationTestNPC(6, indexes.ShadowLord_KeyIndex, 0, false)
	lastCivilian := guardActivationTestNPC(7, indexes.MerchantKeyIndex, 1, true)
	for _, npc := range []*map_units.NPCFriendly{guard, firstCivilian, staticCivilian, invalidShape, lastCivilian} {
		npc.MapUnitDetails().SetCurrentPath([]references.Position{{X: 2, Y: 2}, {X: 3, Y: 3}})
		npc.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionTalk
	}
	controller.GetNpcs().ReplaceWith([]map_units.MapUnit{
		lastCivilian, shadowlord, invalidShape, daemon, staticCivilian, firstCivilian, hidden, guard,
	})

	const seed = uint64(1)
	expectedRNG := deterministic_random.New(seed)
	expectedAttempts := make([]bool, 4)
	for index := range expectedAttempts {
		expectedAttempts[index] = expectedRNG.Intn(guardActivationFleeRollMaximum+1) < guardActivationFleeThreshold
	}
	require.NotEqual(t, expectedAttempts[0], expectedAttempts[3], "seed must distinguish slot order")
	expectedRejected := 0
	for _, attempted := range expectedAttempts[1:3] {
		if attempted {
			expectedRejected++
		}
	}
	require.Positive(t, expectedRejected, "seed must exercise rejected setflee attempts")

	state.SetRandomSeed(seed)
	rngCallsBefore := state.GetRNG().Calls()
	result, err := state.ActivateGuards()
	require.NoError(t, err)

	assert.Equal(t, 7, result.ActiveNPCCount)
	assert.Equal(t, 3, result.PotentialGuardCount)
	assert.Zero(t, result.HostileBefore)
	assert.Equal(t, 3, result.HostileAfter)
	assert.Equal(t, 4, result.CivilianRollCount)
	assert.Equal(t, 4, int(state.GetRNG().Calls()-rngCallsBefore))
	assert.Equal(t, expectedRejected, result.FleeRejectedCount)
	assert.Equal(t, countTrue(expectedAttempts), result.FleeAttemptCount)
	assert.Equal(t, countTrue([]bool{expectedAttempts[0], expectedAttempts[3]}), result.FleeAcceptedCount)
	assert.Equal(t, result.FleeAcceptedCount, result.FleeAfter)

	assert.Equal(t, references.AiType(references.DrudgeWorthThing), guard.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AiType(references.DrudgeWorthThing), daemon.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AiType(references.DrudgeWorthThing), shadowlord.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceGuardHostility, guard.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, references.AIOverrideProvenanceGuardHostility, daemon.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, references.AIOverrideProvenanceGuardHostility, shadowlord.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, expectedAttempts[0], firstCivilian.MapUnitDetails().GetOverriddenAiType() == references.AiType(references.FleeFromAvatar))
	assert.Equal(t, expectedAttempts[3], lastCivilian.MapUnitDetails().GetOverriddenAiType() == references.AiType(references.FleeFromAvatar))
	assert.Equal(t, references.Unset, staticCivilian.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.Unset, invalidShape.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.Unset, hidden.MapUnitDetails().GetOverriddenAiType())
	assert.Empty(t, guard.MapUnitDetails().CurrentPath)
	assert.Equal(t, map_units.PendingNPCInteractionNone, guard.MapUnitDetails().PendingInteraction)
	if expectedAttempts[0] {
		assert.Empty(t, firstCivilian.MapUnitDetails().CurrentPath)
		assert.Equal(t, map_units.PendingNPCInteractionNone, firstCivilian.MapUnitDetails().PendingInteraction)
		assert.Equal(t, references.AIOverrideProvenanceGenericFlee, firstCivilian.MapUnitDetails().AIOverrideProvenance())
	}
	assert.NotEmpty(t, staticCivilian.MapUnitDetails().CurrentPath, "rejected flee must retain its current action")
	assert.Equal(t, map_units.PendingNPCInteractionTalk, staticCivilian.MapUnitDetails().PendingInteraction)
	assert.Equal(t,
		expectedRNG.Intn(guardActivationFleeRollMaximum+1),
		state.RandomIntInRange(0, guardActivationFleeRollMaximum),
		"the next caller must receive the value after all ordered per-slot rolls",
	)
}

func TestActivateGuardsPreservesAutomaticShadowlordProvenanceAcrossLawAI(t *testing.T) {
	t.Run("hostility preserves Nosfentor cause", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
		require.True(t, ok)
		guard := guardActivationTestNPC(0, indexes.Guard_KeyIndex, 0, false)
		guard.MapUnitDetails().SetAIOverride(
			references.FleeFromAvatar,
			references.AIOverrideProvenanceAutomaticNosfentor,
		)
		controller.GetNpcs().ReplaceWith([]map_units.MapUnit{guard})

		_, err := state.ActivateGuards()

		require.NoError(t, err)
		assert.Equal(t, references.AiType(references.DrudgeWorthThing), guard.MapUnitDetails().GetOverriddenAiType())
		assert.Equal(t, references.AIOverrideProvenanceAutomaticNosfentor, guard.MapUnitDetails().AIOverrideProvenance())
	})

	t.Run("flee preserves Astaroth cause", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
		require.True(t, ok)
		civilian := guardActivationTestNPC(0, indexes.TownsPerson_KeyIndex, 1, true)
		civilian.MapUnitDetails().SetAIOverride(
			references.DrudgeWorthThing,
			references.AIOverrideProvenanceAutomaticAstaroth,
		)
		controller.GetNpcs().ReplaceWith([]map_units.MapUnit{civilian})
		state.SetRandomSeed(1)
		expectedRNG := deterministic_random.New(1)
		require.Less(t, expectedRNG.Intn(guardActivationFleeRollMaximum+1), guardActivationFleeThreshold)

		result, err := state.ActivateGuards()

		require.NoError(t, err)
		assert.Equal(t, 1, result.FleeAcceptedCount)
		assert.Equal(t, references.AiType(references.FleeFromAvatar), civilian.MapUnitDetails().GetOverriddenAiType())
		assert.Equal(t, references.AIOverrideProvenanceAutomaticAstaroth, civilian.MapUnitDetails().AIOverrideProvenance())
	})
}

func TestActivateGuardsAuthoredActorsPursueBeyondActionSixRange(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		slot     int
	}{
		{name: "Balinor", location: references.Windemere, slot: 10},
		{name: "devil", location: references.Windemere, slot: 11},
		{name: "guard", location: references.Britain, slot: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(test.location).WithSystemCallbacks().Build()
			refs := state.GameReferences.NPCReferences.GetNPCReferencesByLocation(test.location)
			slot := test.slot
			if slot < 0 {
				for index, ref := range *refs {
					if ref.GetSpriteIndex() == indexes.Guard_KeyIndex {
						slot = index
						break
					}
				}
			}
			require.GreaterOrEqual(t, slot, 0, "production location must contain a guard")
			npc := map_units.NewNPCFriendly((*refs)[slot], slot)
			npc.SetVisible(true)
			npc.SetFloor(0)
			start := references.Position{X: 10, Y: 10}
			npc.SetPos(start)
			npc.MapUnitDetails().SetCurrentPath([]references.Position{start.Left()})
			state.CurrentNPCAIController.GetNpcs().ReplaceWith(map_units.MapUnits{npc})
			state.MapState.PlayerLocation.Position = references.Position{X: 14, Y: 10}
			for y := references.Coordinate(8); y <= 15; y++ {
				for x := references.Coordinate(8); x <= 15; x++ {
					position := references.Position{X: x, Y: y}
					state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.BrickFloor)
				}
			}
			state.SetRandomSeed(484)

			result, err := state.ActivateGuards()

			require.NoError(t, err)
			assert.Equal(t, 1, result.HostileAfter)
			assert.Equal(t, references.AiType(references.DrudgeWorthThing), npc.MapUnitDetails().GetOverriddenAiType())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			state.CurrentNPCAIController.AdvanceNextTurnCalcAndMoveNPCs()
			assert.NotEqual(t, start, npc.Pos(), "activated actor must move at distance four")
			assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)

			state.MapState.PlayerLocation.Position = npc.Pos().Right()
			state.CurrentNPCAIController.AdvanceNextTurnCalcAndMoveNPCs()
			want := map_units.PendingNPCInteractionHostileCombat
			if test.name == "guard" {
				want = map_units.PendingNPCInteractionGuardArrest
			}
			assert.Equal(t, want, npc.MapUnitDetails().PendingInteraction, "authored slot %d", slot)
			result, err = state.ActivateGuards()
			require.NoError(t, err)
			assert.Equal(t, 1, result.HostileBefore, "repeat activation recognizes action 7 hostility")
		})
	}
}

func TestActivateGuardsDoesNotOwnCriminalStatus(t *testing.T) {
	for _, criminal := range []bool{false, true} {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.SetCriminalStatus(criminal)
		state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)

		result, err := state.ActivateGuards()
		require.NoError(t, err)
		assert.Equal(t, criminal, state.IsCriminal())
		assert.Equal(t, criminal, result.CriminalBefore)
		assert.Equal(t, criminal, result.CriminalAfter)
	}
}

func TestActivateGuardsContextsDoNotConsumeRNG(t *testing.T) {
	t.Run("large map is a no-op", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithSystemCallbacks().
			Build()
		callsBefore := state.GetRNG().Calls()

		result, err := state.ActivateGuards()

		require.NoError(t, err)
		assert.Equal(t, GuardActivationResult{}, result)
		assert.Equal(t, callsBefore, state.GetRNG().Calls())
	})

	t.Run("combat is rejected", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.MapState.PlayerLocation.Location = references.CombatLocation
		callsBefore := state.GetRNG().Calls()

		_, err := state.ActivateGuards()

		assert.True(t, errors.Is(err, ErrInvalidGuardActivation))
		assert.Equal(t, callsBefore, state.GetRNG().Calls())
	})

	t.Run("missing controller is rejected", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.CurrentNPCAIController = nil
		callsBefore := state.GetRNG().Calls()

		_, err := state.ActivateGuards()

		assert.True(t, errors.Is(err, ErrInvalidGuardActivation))
		assert.Equal(t, callsBefore, state.GetRNG().Calls())
	})
}

func guardActivationTestNPC(
	slot int,
	sprite indexes.SpriteIndex,
	dialogNumber byte,
	movable bool,
) *map_units.NPCFriendly {
	schedule := references.NPCSchedule{
		X: [3]byte{1, 1, 1},
		Y: [3]byte{1, 1, 1},
	}
	if movable {
		schedule.Time[3] = 1
	}
	reference := references.NPCReference{
		DialogNumber: dialogNumber,
		Schedule:     schedule,
		NpcType:      references.NoStatedNpc,
	}
	reference.SetKeyIndex(sprite)

	return map_units.NewNPCFriendly(reference, slot)
}
