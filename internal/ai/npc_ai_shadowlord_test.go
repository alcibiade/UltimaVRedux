package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestShadowlordAffectedNPCEligibilityAndRNGOrder(t *testing.T) {
	rolls := []bool{true, false, true}
	rollIndex := 0
	controller := &NPCAIControllerSmallMap{
		mapUnits: map_units.MapUnits{},
	}
	shouldAffect := func() bool {
		result := rolls[rollIndex]
		rollIndex++

		return result
	}

	ordinary := smallMapFleeTestNPC(indexes.Wizard1, 1, [4]byte{0, 0, 0, 5})
	static := smallMapFleeTestNPC(indexes.Wizard1, 1, [4]byte{})
	belowRange := smallMapFleeTestNPC(indexes.LordBritishMirror1, 1, [4]byte{1})
	boundary := smallMapFleeTestNPC(indexes.Apparation1, 1, [4]byte{1})
	merchant := smallMapFleeTestNPC(indexes.Merchant1, byte(references.DialogWeaponDealer), [4]byte{1})
	wanderer := smallMapFleeTestNPC(indexes.TownsPerson1, 1, [4]byte{0, 0, 3})
	controller.mapUnits = map_units.MapUnits{
		ordinary,
		static,
		belowRange,
		boundary,
		merchant,
		wanderer,
	}

	controller.SetNPCsReactToShadowlord(references.Astaroth, shouldAffect)

	assert.Equal(t, len(rolls), rollIndex)
	assert.Equal(t, references.AiType(references.DrudgeWorthThing), ordinary.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceAutomaticAstaroth, ordinary.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, references.Unset, static.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceNone, static.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, references.Unset, belowRange.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceNone, belowRange.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, references.Unset, boundary.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceNone, boundary.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, references.Unset, merchant.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceNone, merchant.MapUnitDetails().AIOverrideProvenance())
	assert.Equal(t, references.AiType(references.DrudgeWorthThing), wanderer.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceAutomaticAstaroth, wanderer.MapUnitDetails().AIOverrideProvenance())
}

func TestAstarothProducerPursuesBeyondThreeThenQueuesTalk(t *testing.T) {
	tests := []struct {
		name   string
		sprite indexes.SpriteIndex
	}{
		{name: "civilian", sprite: indexes.TownsPersonKeyIndex},
		{name: "guard", sprite: indexes.GuardKeyIndex},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedOriginalScheduleRNG{t: t, values: []int{63}}
			controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Fixed, references.Position{X: 10, Y: 10})
			npc.NPCReference.SetKeyIndex(test.sprite)
			npc.MapUnitDetails().SetCurrentPath([]references.Position{{X: 9, Y: 10}})
			npc.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionHostileCombat
			controller.mapState.PlayerLocation.Position = references.Position{X: 14, Y: 10}

			controller.SetNPCsReactToShadowlord(references.Astaroth, func() bool { return true })

			assert.Equal(t, references.AiType(references.DrudgeWorthThing), npc.MapUnitDetails().GetOverriddenAiType())
			assert.Equal(t, references.AIOverrideProvenanceAutomaticAstaroth, npc.MapUnitDetails().AIOverrideProvenance())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
			controller.AdvanceNextTurnCalcAndMoveNPCs()
			assert.Equal(t, references.Position{X: 11, Y: 10}, npc.Pos())
			assert.Equal(t, map_units.PendingNPCInteractionNone, npc.MapUnitDetails().PendingInteraction)
			controller.mapState.PlayerLocation.Position = references.Position{X: 12, Y: 10}
			controller.AdvanceNextTurnCalcAndMoveNPCs()
			assert.Equal(t, map_units.PendingNPCInteractionTalk, npc.MapUnitDetails().PendingInteraction)
			assert.Equal(t, []originalScheduleRNGCall{{0, 63}}, rng.calls)
		})
	}
}

func TestFaulineiDoesNotRollOrOverrideNPCs(t *testing.T) {
	rolls := 0
	controller := &NPCAIControllerSmallMap{}
	shouldAffect := func() bool {
		rolls++

		return true
	}
	npc := smallMapFleeTestNPC(indexes.Wizard1, 1, [4]byte{1})
	controller.mapUnits = map_units.MapUnits{npc}

	controller.SetNPCsReactToShadowlord(references.Faulinei, shouldAffect)

	assert.Zero(t, rolls)
	assert.Equal(t, references.Unset, npc.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, references.AIOverrideProvenanceNone, npc.MapUnitDetails().AIOverrideProvenance())
}
